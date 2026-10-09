// Форматы импорта:
//
//   - Текст с разделителем (CSV / TSV / TXT): табуляция, ';', '|' или ',',
//     определяется автоматически. Строка — ru_word; ru_example; en_word; en_example
//     или ru_word; en_word. Необязательная строка заголовков (ru_word, en_word…
//     или русские и английские синонимы) задаёт другой порядок колонок.
//     Пустые строки и строки с '#' пропускаются.
//   - JSON: массив объектов или {"cards": [...]} с ключами ru_word, ru_example,
//     en_word, en_example (или короткими ru, en).
package cardio

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/worddeck/internal/domain"
)

type Card = domain.CardFields

// Добавляется при экспорте для Excel и отбрасывается при импорте.
const UTF8BOM = "\xef\xbb\xbf"

// С ';' файл корректно открывается в Excel.
const CSVComma = ';'

const delimiterSampleLines = 20

// Line — номер строки файла или элемента JSON-массива, начиная с 1.
type LineError struct {
	Line  int
	Error *domain.ValidationError
}

type Result struct {
	Format    string // "json" или "text"
	Delimiter string // для текста: "tab", ";", "|" или ","
	Cards     []Card
	Errors    []LineError
}

func fileError(code string) error {
	return domain.Invalid("file", code)
}

func lineError(line int, err error) LineError {
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		ve = domain.Invalid("line", domain.CodeBadLine)
	}
	return LineError{Line: line, Error: ve}
}

func Parse(data []byte) (Result, error) {
	data = bytes.TrimPrefix(data, []byte(UTF8BOM))
	if !utf8.Valid(data) {
		return Result{}, fileError(domain.CodeNotUTF8)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Result{}, fileError(domain.CodeEmptyFile)
	}
	var res Result
	var err error
	if trimmed[0] == '[' || trimmed[0] == '{' {
		res, err = parseJSON(trimmed)
	} else {
		res = parseDelimited(string(data))
	}
	if err != nil {
		return res, err
	}
	if len(res.Cards) == 0 && len(res.Errors) == 0 {
		return res, fileError(domain.CodeEmptyFile)
	}
	if len(res.Cards) > domain.MaxCardsPerRequest {
		return res, domain.TooMany("file", domain.MaxCardsPerRequest)
	}
	return res, nil
}

type jsonCard struct {
	RuWord    string `json:"ru_word"`
	Ru        string `json:"ru"`
	RuExample string `json:"ru_example"`
	EnWord    string `json:"en_word"`
	En        string `json:"en"`
	EnExample string `json:"en_example"`
}

func parseJSON(data []byte) (Result, error) {
	res := Result{Format: "json", Cards: []Card{}, Errors: []LineError{}}
	var items []jsonCard
	if data[0] == '{' {
		var wrapper struct {
			Cards []jsonCard `json:"cards"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return res, fileError(domain.CodeInvalidJSON)
		}
		items = wrapper.Cards
	} else if err := json.Unmarshal(data, &items); err != nil {
		return res, fileError(domain.CodeInvalidJSON)
	}
	for i, it := range items {
		c := Card{RuWord: first(it.RuWord, it.Ru), RuExample: it.RuExample, EnWord: first(it.EnWord, it.En), EnExample: it.EnExample}
		c, err := c.Normalize()
		if err != nil {
			res.Errors = append(res.Errors, lineError(i+1, err))
			continue
		}
		res.Cards = append(res.Cards, c)
	}
	return res, nil
}

func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

var headerAliases = map[string]string{
	"ru_word": "ru_word", "ru": "ru_word", "russian": "ru_word", "русский": "ru_word", "слово": "ru_word", "рус": "ru_word",
	"ru_example": "ru_example", "ru_context": "ru_example", "пример": "ru_example", "пример_ru": "ru_example", "пример на русском": "ru_example",
	"en_word": "en_word", "en": "en_word", "english": "en_word", "английский": "en_word", "word": "en_word", "англ": "en_word",
	"en_example": "en_example", "en_context": "en_example", "example": "en_example", "пример_en": "en_example", "пример на английском": "en_example",
}

func DetectDelimiter(text string) rune {
	lines := strings.Split(text, "\n")
	counts := map[rune]int{'\t': 0, ';': 0, '|': 0, ',': 0}
	seen := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		for r := range counts {
			if strings.ContainsRune(l, r) {
				counts[r]++
			}
		}
		if seen++; seen >= delimiterSampleLines {
			break
		}
	}
	for _, r := range []rune{'\t', ';', '|', ','} { // по приоритету
		if counts[r] > 0 {
			return r
		}
	}
	return ';'
}

func delimiterName(r rune) string {
	switch r {
	case '\t':
		return "tab"
	default:
		return string(r)
	}
}

func parseDelimited(text string) Result {
	delim := DetectDelimiter(text)
	res := Result{Format: "text", Delimiter: delimiterName(delim), Cards: []Card{}, Errors: []LineError{}}

	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.Comment = '#'
	r.ReuseRecord = false

	var mapping []string // индекс колонки -> имя поля
	for rowNum := 0; ; rowNum++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line, _ := r.FieldPos(0)
		if err != nil {
			res.Errors = append(res.Errors, LineError{Line: line, Error: domain.Invalid("line", domain.CodeBadLine)})
			continue
		}
		if isBlank(rec) {
			continue
		}
		if mapping == nil && rowNum == 0 {
			if m, ok := headerMapping(rec); ok {
				mapping = m
				continue
			}
		}
		c, err := recordToCard(rec, mapping)
		if err == nil {
			c, err = c.Normalize()
		}
		if err != nil {
			res.Errors = append(res.Errors, lineError(line, err))
			continue
		}
		res.Cards = append(res.Cards, c)
	}
	return res
}

func isBlank(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

func headerMapping(rec []string) ([]string, bool) {
	m := make([]string, len(rec))
	found := map[string]bool{}
	for i, f := range rec {
		name, ok := headerAliases[strings.ToLower(strings.TrimSpace(f))]
		if !ok {
			return nil, false
		}
		m[i] = name
		found[name] = true
	}
	return m, found["ru_word"] && found["en_word"]
}

func recordToCard(rec []string, mapping []string) (Card, error) {
	get := func(i int) string {
		if i < len(rec) {
			return rec[i]
		}
		return ""
	}
	if mapping != nil {
		var c Card
		for i, name := range mapping {
			v := get(i)
			switch name {
			case "ru_word":
				c.RuWord = v
			case "ru_example":
				c.RuExample = v
			case "en_word":
				c.EnWord = v
			case "en_example":
				c.EnExample = v
			}
		}
		return c, nil
	}
	// Табличные редакторы часто добавляют пустые колонки в конце.
	n := len(rec)
	for n > 0 && strings.TrimSpace(rec[n-1]) == "" {
		n--
	}
	switch {
	case n == 2:
		return Card{RuWord: rec[0], EnWord: rec[1]}, nil
	case n == 4 || (n == 3 && len(rec) >= 4): // 4 колонки, последняя (en_example) может быть пустой
		return Card{RuWord: get(0), RuExample: get(1), EnWord: get(2), EnExample: get(3)}, nil
	default:
		return Card{}, domain.BadColumns("line", n)
	}
}

func WriteCSV(w io.Writer, cards []Card) error {
	cw := csv.NewWriter(w)
	cw.Comma = CSVComma
	if err := cw.Write([]string{"ru_word", "ru_example", "en_word", "en_example"}); err != nil {
		return err
	}
	for _, c := range cards {
		if err := cw.Write([]string{c.RuWord, c.RuExample, c.EnWord, c.EnExample}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
