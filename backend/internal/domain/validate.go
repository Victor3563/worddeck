package domain

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Длины в символах. Те же значения заданы в CHECK-ограничениях таблиц.
const (
	MaxDisplayNameLen  = 60
	MaxEmailLen        = 254 // RFC 5321
	MaxNameLen         = 100
	MaxDescriptionLen  = 500
	MaxWordLen         = 200
	MaxExampleLen      = 1000
	MaxSessionTitleLen = 200
	MinPasswordLen     = 8
	MaxPasswordLen     = 128
)

func Text(field, v string, required bool, max int) (string, error) {
	v = strings.TrimSpace(v)
	if required && v == "" {
		return v, Invalid(field, CodeRequired)
	}
	if utf8.RuneCountInString(v) > max {
		return v, TooLong(field, max)
	}
	return v, nil
}

func OptionalText(field string, v *string, required bool, max int) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s, err := Text(field, *v, required, max)
	return &s, err
}

func Email(field, v string) (string, error) {
	v, err := Text(field, v, true, MaxEmailLen)
	if err != nil {
		return v, err
	}
	if addr, err := mail.ParseAddress(v); err != nil || addr.Address != v || !strings.Contains(v, ".") {
		return v, Invalid(field, CodeInvalid)
	}
	return v, nil
}

// Пароль не обрезается: пробелы — его часть.
func Password(field, v string) error {
	n := utf8.RuneCountInString(v)
	switch {
	case n < MinPasswordLen:
		return TooShort(field, MinPasswordLen)
	case n > MaxPasswordLen:
		return TooLong(field, MaxPasswordLen)
	}
	return nil
}
