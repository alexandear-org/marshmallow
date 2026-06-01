// Copyright 2026 PerimeterX. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package lexer

import (
	"reflect"
	"testing"
)

func TestLexerError(t *testing.T) {
	e := &LexerError{Reason: "syntax error", Offset: 5, Data: "hello"}
	want := "parse error: syntax error near offset 5 of 'hello'"
	if got := e.Error(); got != want {
		t.Errorf("LexerError.Error() = %q; want %q", got, want)
	}
}

func TestString(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		want      string
		wantError bool
	}{
		{toParse: `"simple string"`, want: "simple string"},
		{toParse: " \r\r\n\t  " + `"test"`, want: "test"},
		{toParse: `"\n\t\"\/\\\f\r"`, want: "\n\t\"/\\\f\r"},
		{toParse: `" "`, want: " "},
		{toParse: `" -\t"`, want: " -\t"},
		{toParse: `"��"`, want: "��"},
		{toParse: `"😀"`, want: "😀"},
		{toParse: `"😈"`, want: "😈"},
		{toParse: `"\ud8"`, wantError: true},
		{toParse: `"test"junk`, want: "test"},
		{toParse: `5`, wantError: true},    // not a string
		{toParse: `"\x"`, wantError: true}, // invalid escape
		{toParse: `"\ud800"`, want: "�"},   // lone surrogate → Unicode replacement char
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.String()
		if got != tt.want {
			t.Errorf("[%d, %q] String() = %v; want %v", i, tt.toParse, got, tt.want)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] String() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] String() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestUnsafeFieldName(t *testing.T) {
	for i, tt := range []struct {
		toParse      string
		skipUnescape bool
		want         string
		wantError    bool
	}{
		{toParse: `"field"`, want: "field"},
		{toParse: `"field\nname"`, want: "field\nname"},
		{toParse: `"field\nname"`, skipUnescape: true, want: `field\nname`},
		{toParse: `"A"`, want: "A"},
		{toParse: `123`, wantError: true},
		{toParse: `true`, wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.UnsafeFieldName(tt.skipUnescape)
		if !tt.wantError && got != tt.want {
			t.Errorf("[%d, %q] UnsafeFieldName() = %q; want %q", i, tt.toParse, got, tt.want)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] UnsafeFieldName() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] UnsafeFieldName() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestNumber(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		want      string
		wantError bool
	}{
		{toParse: "123", want: "123"},
		{toParse: "-123", want: "-123"},
		{toParse: "\r\n12.35", want: "12.35"},
		{toParse: "12.35e+1", want: "12.35e+1"},
		{toParse: "12.35e-15", want: "12.35e-15"},
		{toParse: "12.35E-15", want: "12.35E-15"},
		{toParse: "12.35E15", want: "12.35E15"},
		{toParse: `"a"`, wantError: true},
		{toParse: "123junk", wantError: true},
		{toParse: "1.2.3", wantError: true},
		{toParse: "1e2e3", wantError: true},
		{toParse: "1e2.3", wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.number()
		if got != tt.want {
			t.Errorf("[%d, %q] number() = %v; want %v", i, tt.toParse, got, tt.want)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] number() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] number() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestBool(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		want      bool
		wantError bool
	}{
		{toParse: "true", want: true},
		{toParse: "false", want: false},
		{toParse: "1", wantError: true},
		{toParse: "truejunk", wantError: true},
		{toParse: `false"junk"`, wantError: true},
		{toParse: "True", wantError: true},
		{toParse: "False", wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.Bool()
		if got != tt.want {
			t.Errorf("[%d, %q] Bool() = %v; want %v", i, tt.toParse, got, tt.want)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] Bool() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] Bool() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestSkipRecursive(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		left      string
		wantError bool
	}{
		{toParse: "5, 4", left: ", 4"},
		{toParse: "[5, 6], 4", left: ", 4"},
		{toParse: "[5, [7,8]]: 4", left: ": 4"},
		{toParse: `{"a":1}, 4`, left: ", 4"},
		{toParse: `{"a":1, "b":{"c": 5}, "e":[12,15]}, 4`, left: ", 4"},
		// array start/end chars in a string
		{toParse: `[5, "]"], 4`, left: ", 4"},
		{toParse: `[5, "\"]"], 4`, left: ", 4"},
		{toParse: `[5, "["], 4`, left: ", 4"},
		{toParse: `[5, "\"["], 4`, left: ", 4"},
		// object start/end chars in a string
		{toParse: `{"a}":1}, 4`, left: ", 4"},
		{toParse: `{"a\"}":1}, 4`, left: ", 4"},
		{toParse: `{"a{":1}, 4`, left: ", 4"},
		{toParse: `{"a\"{":1}, 4`, left: ", 4"},
		// object with double slashes at end of string
		{toParse: `{"a":"hey\\"}, 4`, left: ", 4"},
		// invalid JSON inside nested structure
		{toParse: `{"a": [ ##invalid json## ]}, 4`, wantError: true},
		{toParse: `{"a": [ [1], [ ##invalid json## ]]}, 4`, wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		l.SkipRecursive()
		got := string(l.Data[l.pos:])
		if got != tt.left {
			t.Errorf("[%d, %q] SkipRecursive() left = %v; want %v", i, tt.toParse, got, tt.left)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] SkipRecursive() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] SkipRecursive() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestInterface(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		want      interface{}
		wantError bool
	}{
		{toParse: "null", want: nil},
		{toParse: "true", want: true},
		{toParse: `"a"`, want: "a"},
		{toParse: "5", want: float64(5)},
		{toParse: `{}`, want: map[string]interface{}{}},
		{toParse: `[]`, want: []interface{}{}},
		{toParse: `{"a": "b"}`, want: map[string]interface{}{"a": "b"}},
		{toParse: `[5]`, want: []interface{}{float64(5)}},
		{toParse: `{"a":5 , "b" : "string"}`, want: map[string]interface{}{"a": float64(5), "b": "string"}},
		{toParse: `["a", 5 , null, true]`, want: []interface{}{"a", float64(5), nil, true}},
		{toParse: `{"a" "b"}`, wantError: true},
		{toParse: `{"a": "b",}`, wantError: true},
		{toParse: `{"a":"b","c" "b"}`, wantError: true},
		{toParse: `{"a": "b","c":"d",}`, wantError: true},
		{toParse: `{,}`, wantError: true},
		{toParse: `[1, 2,]`, wantError: true},
		{toParse: `[1  2]`, wantError: true},
		{toParse: `[,]`, wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.Interface()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("[%d, %q] Interface() = %v; want %v", i, tt.toParse, got, tt.want)
		}
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] Interface() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] Interface() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestConsumed(t *testing.T) {
	for i, tt := range []struct {
		toParse   string
		wantError bool
	}{
		{toParse: "", wantError: false},
		{toParse: "   ", wantError: false},
		{toParse: "\r\n", wantError: false},
		{toParse: "\t\t", wantError: false},
		{toParse: "{", wantError: true},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		l.Consumed()
		if tt.wantError && l.Ok() {
			t.Errorf("[%d, %q] Consumed() ok; want error", i, tt.toParse)
		}
		if !tt.wantError && !l.Ok() {
			t.Errorf("[%d, %q] Consumed() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestFetchStringUnterminated(t *testing.T) {
	for _, tt := range []struct {
		data []byte
	}{
		{data: []byte(`"string without trailing quote`)},
		{data: []byte(`"\"`)},
		{data: []byte{'"'}},
	} {
		l := Lexer{Data: tt.data}
		l.fetchString()
		if l.pos > len(l.Data) {
			t.Errorf("fetchString(%s): pos=%v must not exceed len(Data)=%v", tt.data, l.pos, len(l.Data))
		}
		if l.Error() == nil {
			t.Errorf("fetchString(%s): expected parse error, got none", tt.data)
		}
	}
}

func TestIsNull(t *testing.T) {
	for i, tt := range []struct {
		toParse  string
		wantNull bool
	}{
		{toParse: "null", wantNull: true},
		{toParse: "  null", wantNull: true},
		{toParse: `"null"`, wantNull: false},
		{toParse: "true", wantNull: false},
		{toParse: "{}", wantNull: false},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.IsNull()
		if got != tt.wantNull {
			t.Errorf("[%d, %q] IsNull() = %v; want %v", i, tt.toParse, got, tt.wantNull)
		}
	}
}

func TestSkip(t *testing.T) {
	for i, tt := range []struct {
		toParse string
	}{
		{toParse: "null"},
		{toParse: "true"},
		{toParse: "false"},
		{toParse: "42"},
		{toParse: `"str"`},
		{toParse: `{}`},
		{toParse: `[]`},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		l.Skip()
		if !l.Ok() {
			t.Errorf("[%d, %q] Skip() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestIsDelim(t *testing.T) {
	for i, tt := range []struct {
		toParse string
		delim   byte
		want    bool
	}{
		{toParse: "{}", delim: '{', want: true},
		{toParse: "[]", delim: '[', want: true},
		{toParse: "{}", delim: '[', want: false},
		{toParse: "42", delim: '{', want: false},
		{toParse: `"str"`, delim: '{', want: false},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.IsDelim(tt.delim)
		if got != tt.want {
			t.Errorf("[%d, %q] IsDelim(%c) = %v; want %v", i, tt.toParse, tt.delim, got, tt.want)
		}
	}
}

func TestDelim(t *testing.T) {
	t.Run("correct delimiter consumed without error", func(t *testing.T) {
		l := Lexer{Data: []byte(`{"key":"val"}`)}
		l.Delim('{')
		if !l.Ok() {
			t.Errorf("Delim('{'): unexpected error: %v", l.Error())
		}
	})
	t.Run("wrong delimiter sets error", func(t *testing.T) {
		l := Lexer{Data: []byte(`{"key":"val"}`)}
		l.Delim('[')
		if l.Ok() {
			t.Error("Delim('['): expected error on '{' input, got none")
		}
	})
}

func TestRaw(t *testing.T) {
	for i, tt := range []struct {
		toParse string
		want    string
	}{
		{toParse: `"hello"`, want: `"hello"`},
		{toParse: `42`, want: `42`},
		{toParse: `true`, want: `true`},
		{toParse: `null`, want: `null`},
		{toParse: `{"a":1}`, want: `{"a":1}`},
		{toParse: `[1,2,3]`, want: `[1,2,3]`},
		{toParse: `{"a":[1,{"b":2}]}`, want: `{"a":[1,{"b":2}]}`},
	} {
		l := Lexer{Data: []byte(tt.toParse)}
		got := l.Raw()
		if string(got) != tt.want {
			t.Errorf("[%d, %q] Raw() = %q; want %q", i, tt.toParse, got, tt.want)
		}
		if !l.Ok() {
			t.Errorf("[%d, %q] Raw() error: %v", i, tt.toParse, l.Error())
		}
	}
}

func TestNonFatalErrors(t *testing.T) {
	// With UseMultipleErrors=true, non-fatal errors are collected without halting.
	// Use different offsets so de-duplication does not apply.
	l := Lexer{Data: []byte(`{"a":1,"b":2}`), UseMultipleErrors: true}
	l.addNonfatalError(&LexerError{Reason: "first", Offset: 0})
	l.addNonfatalError(&LexerError{Reason: "second", Offset: 5})

	errs := l.GetNonFatalErrors()
	if len(errs) != 2 {
		t.Fatalf("GetNonFatalErrors() = %d errors; want 2", len(errs))
	}
	if errs[0].Reason != "first" || errs[1].Reason != "second" {
		t.Errorf("unexpected reasons: %q, %q", errs[0].Reason, errs[1].Reason)
	}
	// Non-fatal errors must not set fatalError.
	if !l.Ok() {
		t.Error("Ok() = false; want true when only non-fatal errors present")
	}
}

func TestNonFatalErrorsFallbackWhenDisabled(t *testing.T) {
	// With UseMultipleErrors=false, a non-fatal error becomes the fatal error.
	l := Lexer{Data: []byte(`{}`), UseMultipleErrors: false}
	l.AddNonFatalError(&LexerError{Reason: "test"})
	if l.Ok() {
		t.Error("Ok() = true; want false")
	}
	if l.Error() == nil {
		t.Error("Error() = nil; want non-nil")
	}
}

func TestAddError(t *testing.T) {
	l := Lexer{Data: []byte(`{}`)}
	l.AddError(&LexerError{Reason: "fatal"})
	if l.Ok() {
		t.Error("Ok() = true after AddError; want false")
	}
	// Second AddError must not overwrite the first.
	l.AddError(&LexerError{Reason: "second"})
	lexErr, ok := l.Error().(*LexerError)
	if !ok || lexErr.Reason != "fatal" {
		t.Errorf("first error should be preserved; got %v", l.Error())
	}
}

func TestDeduplicateNonFatalErrors(t *testing.T) {
	// Two errors at the same offset must be stored only once.
	l := Lexer{Data: []byte(`{}`), UseMultipleErrors: true}
	l.addNonfatalError(&LexerError{Reason: "first", Offset: 5})
	l.addNonfatalError(&LexerError{Reason: "duplicate", Offset: 5})

	if errs := l.GetNonFatalErrors(); len(errs) != 1 {
		t.Errorf("expected 1 deduplicated error; got %d", len(errs))
	}
}

func TestFullObjectParse(t *testing.T) {
	// Simulate the token sequence marshmallow uses when walking a JSON object.
	input := []byte(`{"name":"alice","age":30,"active":true,"score":null}`)
	l := Lexer{Data: input}

	l.Delim('{')

	key := l.UnsafeFieldName(false)
	l.WantColon()
	val := l.Interface()
	l.WantComma()
	if key != "name" || val != "alice" {
		t.Errorf(`field "name": got key=%q val=%v`, key, val)
	}

	key = l.UnsafeFieldName(false)
	l.WantColon()
	val = l.Interface()
	l.WantComma()
	if key != "age" || val != float64(30) {
		t.Errorf(`field "age": got key=%q val=%v`, key, val)
	}

	key = l.UnsafeFieldName(false)
	l.WantColon()
	val = l.Interface()
	l.WantComma()
	if key != "active" || val != true {
		t.Errorf(`field "active": got key=%q val=%v`, key, val)
	}

	key = l.UnsafeFieldName(false)
	l.WantColon()
	if !l.IsNull() {
		t.Error("IsNull() = false; want true for null value")
	}
	l.Skip()
	l.WantComma()
	if key != "score" {
		t.Errorf(`field key = %q; want "score"`, key)
	}

	l.Delim('}')
	l.Consumed()

	if !l.Ok() {
		t.Errorf("full object parse error: %v", l.Error())
	}
}
