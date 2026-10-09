package domain

import (
	"errors"
	"fmt"
)

var (
	// Возвращается и для чужих сущностей, чтобы по ответу нельзя было узнать,
	// существует ли чужой id.
	ErrNotFound           = errors.New("not found")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const (
	CodeRequired      = "required"
	CodeTooShort      = "too_short"
	CodeTooLong       = "too_long"
	CodeInvalid       = "invalid"
	CodeTaken         = "taken"
	CodeWrongPassword = "wrong_password"
	CodeTooMany       = "too_many"
	CodeEmptyFile     = "empty_file"
	CodeNotUTF8       = "not_utf8"
	CodeInvalidJSON   = "invalid_json"
	CodeBadColumns    = "bad_columns"
	CodeBadLine       = "bad_line"
	CodeNoValidCards  = "no_valid_cards"
)

type ValidationError struct {
	Field  string
	Code   string
	Params ValidationParams
}

type ValidationParams struct {
	Min       int
	Max       int
	Found     int
	Number    int
	ItemField string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s %+v", e.Field, e.Code, e.Params)
}

func Invalid(field, code string) *ValidationError {
	return &ValidationError{Field: field, Code: code}
}

func TooShort(field string, min int) *ValidationError {
	return &ValidationError{Field: field, Code: CodeTooShort, Params: ValidationParams{Min: min}}
}

func TooLong(field string, max int) *ValidationError {
	return &ValidationError{Field: field, Code: CodeTooLong, Params: ValidationParams{Max: max}}
}

func TooMany(field string, max int) *ValidationError {
	return &ValidationError{Field: field, Code: CodeTooMany, Params: ValidationParams{Max: max}}
}

func BadColumns(field string, found int) *ValidationError {
	return &ValidationError{Field: field, Code: CodeBadColumns, Params: ValidationParams{Found: found}}
}

// InItem превращает ошибку поля элемента в ошибку списка: Field становится
// именем списка, а номер элемента и исходное поле переходят в Params.
func (e *ValidationError) InItem(list string, number int) *ValidationError {
	out := *e
	out.Field = list
	out.Params.Number, out.Params.ItemField = number, e.Field
	return &out
}
