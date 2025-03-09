package fasttemplate

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		startTag string
		endTag   string
		m        Map
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "No tags",
			template: "Hello, world!",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{},
			wantErr:  false,
		},
		{
			name:     "All tags resolved",
			template: "Hello, {{name}}!",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{"name": "John"},
			wantErr:  false,
		},
		{
			name:     "Multiple tags all resolved",
			template: "Hello, {{name}}! You are {{age}} years old.",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{"name": "John", "age": 30},
			wantErr:  false,
		},
		{
			name:     "Unresolved tag",
			template: "Hello, {{name}}!",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{},
			wantErr:  true,
			errMsg:   "unresolved tag \"name\"",
		},
		{
			name:     "Nil map with tags",
			template: "Hello, {{name}}!",
			startTag: "{{",
			endTag:   "}}",
			m:        nil,
			wantErr:  true,
			errMsg:   "unresolved tag \"name\": nil map provided",
		},
		{
			name:     "Nil map without tags",
			template: "Hello, world!",
			startTag: "{{",
			endTag:   "}}",
			m:        nil,
			wantErr:  false,
		},
		{
			name:     "Function call resolved",
			template: "Result: {{uppercase(name)}}",
			startTag: "{{",
			endTag:   "}}",
			m: Map{
				"name": "john",
				"uppercase": func(s string) string {
					return strings.ToUpper(s)
				},
			},
			wantErr: false,
		},
		{
			name:     "Function call unresolved",
			template: "Result: {{uppercase(name)}}",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{"name": "john"},
			wantErr:  true,
			errMsg:   "unresolved function \"uppercase\"",
		},
		{
			name:     "Invalid function call",
			template: "Result: {{uppercase(}}",
			startTag: "{{",
			endTag:   "}}",
			m: Map{
				"uppercase": func(s string) string {
					return strings.ToUpper(s)
				},
			},
			wantErr: true,
			errMsg:  "unresolved tag \"uppercase(\"",
		},
		{
			name:     "Expression tag",
			template: "Result: {{10 + 20}}",
			startTag: "{{",
			endTag:   "}}",
			m:        Map{},
			wantErr:  false, // Expressions are not validated in detail
		},
		{
			name:     "Complex template with mixed tags - all resolved",
			template: "Hello, {{name}}! {{uppercase(name)}} is {{age + 5}} in 5 years.",
			startTag: "{{",
			endTag:   "}}",
			m: Map{
				"name": "john",
				"age":  25,
				"uppercase": func(s string) string {
					return strings.ToUpper(s)
				},
			},
			wantErr: false,
		},
		{
			name:     "Complex template with mixed tags - partially resolved",
			template: "Hello, {{name}}! {{uppercase(name)}} is {{age + 5}} in 5 years.",
			startTag: "{{",
			endTag:   "}}",
			m: Map{
				"name": "john",
				"uppercase": func(s string) string {
					return strings.ToUpper(s)
				},
			},
			wantErr: false, // We don't validate expression variables
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := New(tt.template, tt.startTag, tt.endTag)
			err := tmpl.Validate(tt.m)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateWithComplexExpressions(t *testing.T) {
	// Define a complex template with nested functions and expressions
	template := `{{greet(name)}} Your score is {{score > 80 ? "excellent" : "good"}}. 
	Age next year: {{age + 1}}. Full name: {{firstname + " " + lastname}}`

	tmpl := New(template, "{{", "}}")

	// Test with all variables and functions resolved
	m := Map{
		"name":      "John",
		"score":     90,
		"age":       30,
		"firstname": "John",
		"lastname":  "Doe",
		"greet": func(name string) string {
			return "Hello, " + name + "!"
		},
	}

	if err := tmpl.Validate(m); err != nil {
		t.Errorf("Validate() should not error with complete map, got: %v", err)
	}

	// Test with function missing
	m = Map{
		"name":      "John",
		"score":     90,
		"age":       30,
		"firstname": "John",
		"lastname":  "Doe",
	}

	if err := tmpl.Validate(m); err == nil {
		t.Error("Validate() should error when function is missing")
	} else if !strings.Contains(err.Error(), "unresolved function \"greet\"") {
		t.Errorf("Validate() error = %v, should mention unresolved function", err)
	}
}

func TestEmptyTemplate(t *testing.T) {
	tpl := New("", "[", "]")

	s := tpl.ExecuteString(Map{"foo": "bar", "aaa": "bbb"})
	if s != "" {
		t.Fatalf("unexpected string returned %q. Expected empty string", s)
	}
}

func TestEmptyTagStart(t *testing.T) {
	expectPanic(t, func() { NewTemplate("foobar", "", "]") })
}

func TestEmptyTagEnd(t *testing.T) {
	expectPanic(t, func() { NewTemplate("foobar", "[", "") })
}

func TestNoTags(t *testing.T) {
	template := "foobar"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "bar", "aaa": "bbb"})
	if s != template {
		t.Fatalf("unexpected template value %q. Expected %q", s, template)
	}
}

func TestEmptyTagName(t *testing.T) {
	template := "foo[]bar"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"": "111", "aaa": "bbb"})
	result := "foo111bar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestOnlyTag(t *testing.T) {
	template := "[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestStartWithTag(t *testing.T) {
	template := "[foo]barbaz"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "111barbaz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestEndWithTag(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "foobar111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestTemplateReset(t *testing.T) {
	template := "foo{bar}baz"
	tpl := New(template, "{", "}")
	s := tpl.ExecuteString(Map{"bar": "111"})
	result := "foo111baz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}

	template = "[xxxyyyzz"
	if err := tpl.Reset(template, "[", "]"); err == nil {
		t.Fatalf("expecting error for unclosed tag on %q", template)
	}

	template = "[xxx]yyy[zz]"
	if err := tpl.Reset(template, "[", "]"); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	s = tpl.ExecuteString(Map{"xxx": "11", "zz": "2222"})
	result = "11yyy2222"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestDuplicateTags(t *testing.T) {
	template := "[foo]bar[foo][foo]baz"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "111bar111111baz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestMultipleTags(t *testing.T) {
	template := "foo[foo]aa[aaa]ccc"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "foo111aabbbccc"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestLongDelimiter(t *testing.T) {
	template := "foo{{{foo}}}bar"
	tpl := New(template, "{{{", "}}}")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "foo111bar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestIdenticalDelimiter(t *testing.T) {
	template := "foo@foo@foo@aaa@"
	tpl := New(template, "@", "@")

	s := tpl.ExecuteString(Map{"foo": "111", "aaa": "bbb"})
	result := "foo111foobbb"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestDlimitersWithDistinctSize(t *testing.T) {
	template := "foo<?phpaaa?>bar<?phpzzz?>"
	tpl := New(template, "<?php", "?>")

	s := tpl.ExecuteString(Map{"zzz": "111", "aaa": "bbb"})
	result := "foobbbbar111"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestEmptyValue(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": "", "aaa": "bbb"})
	result := "foobar"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestNoValue(t *testing.T) {
	template := "foobar[foo]x[aaa]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"aaa": "bbb"})
	result := "foobarxbbb"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestNoEndDelimiter(t *testing.T) {
	template := "foobar[foo"
	_, err := NewTemplate(template, "[", "]")
	if err == nil {
		t.Fatalf("expected non-nil error. got nil")
	}

	expectPanic(t, func() { New(template, "[", "]") })
}

func TestNumericValue(t *testing.T) {
	template := "foobar[foo]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{"foo": 123, "aaa": "bbb"})
	result := "foobar123"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestMixedValues(t *testing.T) {
	template := "foo[foo]bar[bar]baz[baz]"
	tpl := New(template, "[", "]")

	s := tpl.ExecuteString(Map{
		"foo": "111",
		"bar": []byte("bbb"),
		"baz": func(w io.Writer, tag string) (int, error) { return w.Write([]byte(tag)) },
	})
	result := "foo111barbbbbazbaz"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestExecute(t *testing.T) {
	testExecute(t, "", "")
	testExecute(t, "a", "a")
	testExecute(t, "abc", "abc")
	testExecute(t, "{foo}", "xxxx")
	testExecute(t, "a{foo}", "axxxx")
	testExecute(t, "{foo}a", "xxxxa")
	testExecute(t, "a{foo}bc", "axxxxbc")
	testExecute(t, "{foo}{foo}", "xxxxxxxx")
	testExecute(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecute(t, "{unclosed", "{unclosed")
	testExecute(t, "{{unclosed", "{{unclosed")
	testExecute(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecute(t, "{unknown}", "")
	testExecute(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxqbarxxxx")
}

func testExecute(t *testing.T, template, expectedOutput string) {
	var bb bytes.Buffer
	Execute(template, "{", "}", &bb, Map{"foo": "xxxx"})
	output := bb.String()
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func TestExecuteStd(t *testing.T) {
	testExecuteStd(t, "", "")
	testExecuteStd(t, "a", "a")
	testExecuteStd(t, "abc", "abc")
	testExecuteStd(t, "{foo}", "xxxx")
	testExecuteStd(t, "a{foo}", "axxxx")
	testExecuteStd(t, "{foo}a", "xxxxa")
	testExecuteStd(t, "a{foo}bc", "axxxxbc")
	testExecuteStd(t, "{foo}{foo}", "xxxxxxxx")
	testExecuteStd(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecuteStd(t, "{unclosed", "{unclosed")
	testExecuteStd(t, "{{unclosed", "{{unclosed")
	testExecuteStd(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecuteStd(t, "{unknown}", "{unknown}")
	testExecuteStd(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxq{unexpected}{missing}barxxxx")
}

func testExecuteStd(t *testing.T, template, expectedOutput string) {
	var bb bytes.Buffer
	ExecuteStd(template, "{", "}", &bb, Map{"foo": "xxxx"})
	output := bb.String()
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func TestExecuteString(t *testing.T) {
	testExecuteString(t, "", "")
	testExecuteString(t, "a", "a")
	testExecuteString(t, "abc", "abc")
	testExecuteString(t, "{foo}", "xxxx")
	testExecuteString(t, "a{foo}", "axxxx")
	testExecuteString(t, "{foo}a", "xxxxa")
	testExecuteString(t, "a{foo}bc", "axxxxbc")
	testExecuteString(t, "{foo}{foo}", "xxxxxxxx")
	testExecuteString(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecuteString(t, "{unclosed", "{unclosed")
	testExecuteString(t, "{{unclosed", "{{unclosed")
	testExecuteString(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecuteString(t, "{unknown}", "")
	testExecuteString(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxqbarxxxx")
}

func testExecuteString(t *testing.T, template, expectedOutput string) {
	output := ExecuteString(template, "{", "}", Map{"foo": "xxxx"})
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func TestExecuteStringStd(t *testing.T) {
	testExecuteStringStd(t, "", "")
	testExecuteStringStd(t, "a", "a")
	testExecuteStringStd(t, "abc", "abc")
	testExecuteStringStd(t, "{foo}", "xxxx")
	testExecuteStringStd(t, "a{foo}", "axxxx")
	testExecuteStringStd(t, "{foo}a", "xxxxa")
	testExecuteStringStd(t, "a{foo}bc", "axxxxbc")
	testExecuteStringStd(t, "{foo}{foo}", "xxxxxxxx")
	testExecuteStringStd(t, "{foo}bar{foo}", "xxxxbarxxxx")

	// unclosed tag
	testExecuteStringStd(t, "{unclosed", "{unclosed")
	testExecuteStringStd(t, "{{unclosed", "{{unclosed")
	testExecuteStringStd(t, "{un{closed", "{un{closed")

	// test unknown tag
	testExecuteStringStd(t, "{unknown}", "{unknown}")
	testExecuteStringStd(t, "{foo}q{unexpected}{missing}bar{foo}", "xxxxq{unexpected}{missing}barxxxx")
}

func testExecuteStringStd(t *testing.T, template, expectedOutput string) {
	output := ExecuteStringStd(template, "{", "}", Map{"foo": "xxxx"})
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}

func expectPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("missing panic")
		}
	}()
	f()
}
