package cardio

import (
	"errors"
	"strings"
	"testing"

	"github.com/worddeck/internal/domain"
)

func TestParseSemicolonFourColumns(t *testing.T) {
	in := "дом; Я иду домой.; house; I'm going home.\n\n# comment\nкот;;cat;\n"
	res, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if res.Delimiter != ";" || len(res.Cards) != 2 || len(res.Errors) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Cards[0].RuExample != "Я иду домой." || res.Cards[0].EnWord != "house" {
		t.Fatalf("bad card: %+v", res.Cards[0])
	}
	if res.Cards[1].RuWord != "кот" || res.Cards[1].EnWord != "cat" {
		t.Fatalf("bad card: %+v", res.Cards[1])
	}
}

func TestParseTabTwoColumns(t *testing.T) {
	res, err := Parse([]byte("собака\tdog\nптица\tbird"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Delimiter != "tab" || len(res.Cards) != 2 || res.Cards[1].EnWord != "bird" {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestParseHeaderReordersColumns(t *testing.T) {
	in := "en_word,ru_word,en_example\n" +
		`apple,яблоко,"An apple, a day"` + "\n"
	res, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Cards) != 1 {
		t.Fatalf("unexpected: %+v", res)
	}
	c := res.Cards[0]
	if c.RuWord != "яблоко" || c.EnWord != "apple" || c.EnExample != "An apple, a day" {
		t.Fatalf("bad mapping: %+v", c)
	}
}

func TestParseReportsBadLines(t *testing.T) {
	in := "дом;house\nодно поле\n;пустое\n"
	res, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Cards) != 1 || len(res.Errors) != 2 {
		t.Fatalf("unexpected: %+v", res)
	}
	if res.Errors[0].Line != 2 || res.Errors[1].Line != 3 {
		t.Fatalf("wrong line numbers: %+v", res.Errors)
	}
	if res.Errors[0].Error.Code != domain.CodeBadColumns || res.Errors[1].Error.Code != domain.CodeRequired || res.Errors[1].Error.Field != "ru_word" {
		t.Fatalf("wrong error codes: %+v %+v", res.Errors[0].Error, res.Errors[1].Error)
	}
}

func TestParseJSON(t *testing.T) {
	in := `{"cards":[{"ru_word":"вода","en_word":"water","en_example":"Drink water."},{"ru":"огонь","en":"fire"},{"ru":"x"}]}`
	res, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "json" || len(res.Cards) != 2 || len(res.Errors) != 1 {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestParseRejectsEmptyAndTooLong(t *testing.T) {
	var ve *domain.ValidationError
	if _, err := Parse([]byte("   \n")); !errors.As(err, &ve) || ve.Code != domain.CodeEmptyFile {
		t.Fatalf("expected empty_file, got %v", err)
	}
	res, err := Parse([]byte(strings.Repeat("я", 201) + ";word"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) != 1 || res.Errors[0].Error.Code != domain.CodeTooLong {
		t.Fatalf("expected too_long error, got %+v", res)
	}
}

func TestParseBOM(t *testing.T) {
	res, err := Parse([]byte("\xef\xbb\xbfru_word;en_word\nсолнце;sun\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Cards) != 1 || res.Cards[0].RuWord != "солнце" {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestExportRoundTrip(t *testing.T) {
	cards := []Card{{RuWord: "дом", RuExample: "Мой дом; большой", EnWord: "house", EnExample: `A "big" house`}}
	var sb strings.Builder
	if err := WriteCSV(&sb, cards); err != nil {
		t.Fatal(err)
	}
	res, err := Parse([]byte(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Cards) != 1 || res.Cards[0] != cards[0] {
		t.Fatalf("round trip mismatch: %+v", res)
	}
}

func TestParseRejectsBadEncodingAndJSON(t *testing.T) {
	var ve *domain.ValidationError
	if _, err := Parse([]byte{0xff, 0xfe, 'a'}); !errors.As(err, &ve) || ve.Code != domain.CodeNotUTF8 {
		t.Fatalf("expected not_utf8, got %v", err)
	}
	if _, err := Parse([]byte(`[{"ru": "x",`)); !errors.As(err, &ve) || ve.Code != domain.CodeInvalidJSON {
		t.Fatalf("expected invalid_json, got %v", err)
	}
}
