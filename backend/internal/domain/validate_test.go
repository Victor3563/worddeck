package domain

import (
	"errors"
	"strings"
	"testing"
)

func code(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("not a ValidationError: %v", err)
	}
	return ve.Code
}

func TestText(t *testing.T) {
	cases := []struct {
		in       string
		required bool
		want     string
		code     string
	}{
		{"  hello  ", true, "hello", ""},
		{"   ", true, "", CodeRequired},
		{"", false, "", ""},
		{strings.Repeat("я", 5), true, strings.Repeat("я", 5), ""}, // символы, а не байты
		{strings.Repeat("я", 6), true, "", CodeTooLong},
	}
	for _, c := range cases {
		got, err := Text("f", c.in, c.required, 5)
		if code(t, err) != c.code || (c.code == "" && got != c.want) {
			t.Errorf("Text(%q) = %q, %v; want %q, %q", c.in, got, err, c.want, c.code)
		}
	}
}

func TestEmail(t *testing.T) {
	for in, want := range map[string]string{
		"a@b.co":         "",
		" a@b.co ":       "",
		"no-at.example":  CodeInvalid,
		"Name <a@b.co>":  CodeInvalid,
		"user@localhost": CodeInvalid,
		"":               CodeRequired,
	} {
		if _, err := Email("email", in); code(t, err) != want {
			t.Errorf("Email(%q): got %v, want %q", in, err, want)
		}
	}
}

func TestPassword(t *testing.T) {
	if c := code(t, Password("p", "1234567")); c != CodeTooShort {
		t.Errorf("7 chars: %q", c)
	}
	if c := code(t, Password("p", "12345678")); c != "" {
		t.Errorf("8 chars: %q", c)
	}
	if c := code(t, Password("p", strings.Repeat("x", MaxPasswordLen+1))); c != CodeTooLong {
		t.Errorf("too long: %q", c)
	}
}

func TestCardNormalize(t *testing.T) {
	c, err := CardFields{RuWord: " дом ", EnWord: "house ", EnExample: " A house. "}.Normalize()
	if err != nil || c.RuWord != "дом" || c.EnWord != "house" || c.EnExample != "A house." {
		t.Fatalf("got %+v, %v", c, err)
	}
	_, err = CardFields{RuWord: "дом"}.Normalize()
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "en_word" || ve.Code != CodeRequired {
		t.Fatalf("missing en_word: %v", err)
	}
}

func TestEnums(t *testing.T) {
	if !Color("violet").Valid() || Color("black").Valid() {
		t.Error("Color.Valid")
	}
	if !ModeMatch.Valid() || StudyMode("cram").Valid() {
		t.Error("StudyMode.Valid")
	}
	if !ResourceCard.Valid() || ResourceType("user").Valid() {
		t.Error("ResourceType.Valid")
	}
}
