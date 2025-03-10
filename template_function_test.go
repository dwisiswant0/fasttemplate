package fasttemplate

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTemplateFunctions(t *testing.T) {
	// Define test functions
	functions := Map{
		"concat": func(s ...string) string {
			return strings.Join(s, "")
		},
		"join": func(sep string, s ...string) string {
			return strings.Join(s, sep)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"nested": func(s string) string {
			return "nested(" + s + ")"
		},
		"error": func() (string, error) {
			return "", errors.New("test error")
		},
		"identity": func(v interface{}) interface{} {
			return v
		},
		// Add a function that returns both string and error
		"validate": func(s string) (string, error) {
			if s == "invalid" {
				return "", errors.New("validation error")
			}
			return "valid: " + s, nil
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple function call",
			template: "Hello {{concat(\"world\")}}!",
			expected: "Hello world!",
		},
		{
			name:     "Function with multiple arguments",
			template: "{{join(\"-\", \"a\", \"b\", \"c\")}}",
			expected: "a-b-c",
		},
		{
			name:     "Function with numeric arguments",
			template: "The answer is {{add(40, 2)}}",
			expected: "The answer is 42",
		},
		{
			name:     "Nested function calls",
			template: "{{nested(concat(\"foo\", \"bar\"))}}",
			expected: "nested(foobar)",
		},
		{
			name:     "Single quoted strings",
			template: "{{concat('hello', ' ', 'world')}}",
			expected: "hello world",
		},
		{
			name:     "Mixed quotes",
			template: "{{concat(\"hello\", ' ', 'world')}}",
			expected: "hello world",
		},
		{
			name:     "Multiple function calls in one template",
			template: "{{concat(\"foo\", \"bar\")}} and {{add(1, 2)}}",
			expected: "foobar and 3",
		},
		{
			name:     "Function call with function result as argument",
			template: "{{concat(\"prefix-\", identity(\"value\"))}}",
			expected: "prefix-value",
		},
		{
			name:     "Function with string and error return (success case)",
			template: "Result: {{validate(\"user\")}}",
			expected: "Result: valid: user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")
			result := tpl.ExecuteString(functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestTemplateFunctionWithStringAndError(t *testing.T) {
	// Test successful case
	tpl := New("Status: {{validate(\"good\")}}", "{{", "}}")
	var buf bytes.Buffer
	_, err := tpl.Execute(&buf, Map{
		"validate": func(s string) (string, error) {
			if s == "invalid" {
				return "", errors.New("validation failed")
			}
			return "valid: " + s, nil
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if buf.String() != "Status: valid: good" {
		t.Fatalf("Unexpected result: %q", buf.String())
	}

	// Test error case
	tpl = New("Status: {{validate(\"invalid\")}}", "{{", "}}")
	buf.Reset()
	_, err = tpl.Execute(&buf, Map{
		"validate": func(s string) (string, error) {
			if s == "invalid" {
				return "", errors.New("validation failed")
			}
			return "valid: " + s, nil
		},
	})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "validation failed" {
		t.Fatalf("Expected 'validation failed' error, got: %v", err)
	}
}

// Add test for ExecuteStd with error function
func TestExecuteStdWithErrorFunction(t *testing.T) {
	tpl := New("Status: {{validate(\"invalid\")}}", "{{", "}}")

	// ExecuteStd should keep the tag when the function returns an error
	var buf bytes.Buffer
	_, err := tpl.ExecuteStd(&buf, Map{
		"validate": func(s string) (string, error) {
			if s == "invalid" {
				return "", errors.New("validation failed")
			}
			return "valid: " + s, nil
		},
	})
	if err != nil {
		t.Fatalf("ExecuteStd should not return error: %v", err)
	}

	expected := "Status: {{validate(\"invalid\")}}"
	if buf.String() != expected {
		t.Fatalf("Expected: %q, got: %q", expected, buf.String())
	}
}

func TestErrorFunction(t *testing.T) {
	tpl := New("{{error()}}", "{{", "}}")

	var buf bytes.Buffer
	_, err := tpl.Execute(&buf, Map{
		"error": func() (string, error) {
			return "", errors.New("test error")
		},
	})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestNestedFunctions(t *testing.T) {
	functions := Map{
		"concat": func(s ...string) string {
			return strings.Join(s, "")
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"repeat": func(s string, n int) string {
			return strings.Repeat(s, n)
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Double nested functions",
			template: "{{upper(lower(\"Hello\"))}}",
			expected: "HELLO",
		},
		{
			name:     "Triple nested functions",
			template: "{{repeat(upper(lower(\"x\")), 3)}}",
			expected: "XXX",
		},
		{
			name:     "Complex nested functions",
			template: "{{concat(upper(\"a\"), lower(\"B\"), repeat(\"c\", 3))}}",
			expected: "Abccc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")
			result := tpl.ExecuteString(functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestInvalidFunctionCalls(t *testing.T) {
	functions := Map{
		"test": func(s string) string { return s },
	}

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "Missing closing parenthesis",
			template: "{{test(\"hello\"}}",
		},
		{
			name:     "Invalid function name",
			template: "{{123test(\"hello\")}}",
		},
		{
			name:     "Unknown function",
			template: "{{unknown(\"hello\")}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")
			result := tpl.ExecuteString(functions)
			// With invalid function calls, we should get empty strings
			if result != "" {
				t.Errorf("Expected empty string for invalid function call, got: %q", result)
			}
		})
	}
}

func TestMixedTemplateAndFunctions(t *testing.T) {
	template := "Hello {{upper(name)}}, welcome to {{location}}!"
	tpl := New(template, "{{", "}}")

	result := tpl.ExecuteString(Map{
		"name":     "john",
		"location": "Earth",
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	})

	expected := "Hello JOHN, welcome to Earth!"
	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}

func TestFunctionsInDataMap(t *testing.T) {
	template := "Hello {{upper(name)}}, welcome to {{location}}!"
	tpl := New(template, "{{", "}}")

	result := tpl.ExecuteString(Map{
		"name":     "john",
		"location": "Earth",
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	})

	expected := "Hello JOHN, welcome to Earth!"
	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}

func TestNestedFunctionsInDataMap(t *testing.T) {
	template := "{{upper(concat(\"he\", \"llo\"))}} {{repeat(lower(\"WORLD\"), 2)}}"
	tpl := New(template, "{{", "}}")

	result := tpl.ExecuteString(Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"concat": func(a, b string) string {
			return a + b
		},
		"repeat": func(s string, n int) string {
			return strings.Repeat(s, n)
		},
	})

	expected := "HELLO worldworld"
	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}

func TestExecuteStdWithFunctions(t *testing.T) {
	// Define functions
	functions := Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple function call",
			template: "Hello {{upper(\"world\")}}!",
			expected: "Hello WORLD!",
		},
		{
			name:     "Unknown function - should keep tag",
			template: "Hello {{unknown(\"world\")}}!",
			expected: "Hello {{unknown(\"world\")}}!",
		},
		{
			name:     "Mixed known and unknown functions",
			template: "{{upper(\"hello\")}} {{unknown(\"world\")}} {{add(1, 2)}}",
			expected: "HELLO {{unknown(\"world\")}} 3",
		},
		{
			name:     "Invalid function name - should keep tag",
			template: "Hello {{123invalid(\"world\")}}!",
			expected: "Hello {{123invalid(\"world\")}}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_ExecuteStd", func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")

			var buf bytes.Buffer
			_, err := tpl.ExecuteStd(&buf, functions)
			if err != nil {
				t.Fatalf("ExecuteStd error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})

		t.Run(tt.name+"_ExecuteStringStd", func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")

			result := tpl.ExecuteStringStd(functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestExecuteStdWithNestedFunctions(t *testing.T) {
	functions := Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"repeat": func(s string, n int) string {
			return strings.Repeat(s, n)
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Nested functions with known outer function",
			template: "{{upper(lower(\"Hello\"))}}",
			expected: "HELLO",
		},
		{
			name:     "Nested functions with unknown inner function",
			template: "{{upper(unknown(\"Hello\"))}}",
			expected: "{{upper(unknown(\"Hello\"))}}",
		},
		{
			name:     "Nested functions with unknown outer function",
			template: "{{unknown(lower(\"Hello\"))}}",
			expected: "{{unknown(lower(\"Hello\"))}}",
		},
		{
			name:     "Complex mix of known and unknown",
			template: "{{upper(\"a\")}} {{unknown(lower(\"B\"))}} {{repeat(\"c\", 3)}}",
			expected: "A {{unknown(lower(\"B\"))}} ccc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")

			result := tpl.ExecuteStringStd(functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}
}

func TestExecuteStdWithVariablesAndFunctions(t *testing.T) {
	template := "Hello {{upper(name)}}, {{unknown(name)}} and {{missing}}"
	tpl := New(template, "{{", "}}")

	data := Map{
		"name": "john",
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	}

	result := tpl.ExecuteStringStd(data)
	expected := "Hello JOHN, {{unknown(name)}} and {{missing}}"

	if result != expected {
		t.Errorf("Expected: %q, got: %q", expected, result)
	}
}

func TestWrongParameterCount(t *testing.T) {
	// Define functions that expect specific parameter counts
	functions := Map{
		"add": func(a, b int) int {
			return a + b
		},
		"join": func(sep string, items ...string) string {
			return strings.Join(items, sep)
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Too few arguments",
			template: "{{add(1)}}",
			expected: "",
		},
		{
			name:     "Required args missing for variadic function",
			template: "{{join()}}",
			expected: "",
		},
		{
			name:     "Too many arguments",
			template: "{{add(1, 2, 3)}}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Execute", func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")
			result := tpl.ExecuteString(functions)
			// With incorrect parameter count, we should get empty string
			if result != tt.expected {
				t.Errorf("Expected empty string for wrong parameter count, got: %q", result)
			}

			// When using ExecuteStd, the tag should be preserved
			resultStd := tpl.ExecuteStringStd(functions)
			if resultStd != tt.template {
				t.Errorf("ExecuteStringStd: Expected original tag %q to be preserved, got: %q", tt.template, resultStd)
			}
		})
	}
}

func TestMissingParameterWithVariable(t *testing.T) {
	// Define a template with a function call missing a parameter
	template := "{{formatRange(min)}} - missing max parameter"
	tpl := New(template, "{{", "}}")

	data := Map{
		"min": 10,
		"max": 20,
		"formatRange": func(min, max int) string {
			return fmt.Sprintf("%d to %d", min, max)
		},
	}

	// ExecuteStd should preserve the tag
	resultStd := tpl.ExecuteStringStd(data)
	expected := "{{formatRange(min)}} - missing max parameter"
	if resultStd != expected {
		t.Errorf("Expected %q, got: %q", expected, resultStd)
	}

	// Now test a successful call with both parameters
	template2 := "Range: {{formatRange(min, max)}}"
	tpl2 := New(template2, "{{", "}}")

	result2 := tpl2.ExecuteString(data)
	if result2 != "Range: 10 to 20" {
		t.Errorf("Expected 'Range: 10 to 20', got: %q", result2)
	}
}

func TestWrongParameterType(t *testing.T) {
	// Define functions that expect specific parameter types
	functions := Map{
		"add": func(a, b int) int {
			return a + b
		},
		"concat": func(a, b string) string {
			return a + b
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "String instead of int",
			template: "{{add(\"1\", 2)}}",
			expected: "",
		},
		{
			name:     "Int instead of string",
			template: "{{concat(1, \"2\")}}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Execute", func(t *testing.T) {
			tpl := New(tt.template, "{{", "}}")
			result := tpl.ExecuteString(functions)
			// With incorrect parameter type, we should get empty string
			if result != tt.expected {
				t.Errorf("Expected empty string for wrong parameter type, got: %q", result)
			}

			// When using ExecuteStd, the tag should be preserved
			resultStd := tpl.ExecuteStringStd(functions)
			if resultStd != tt.template {
				t.Errorf("ExecuteStringStd: Expected original tag %q to be preserved, got: %q", tt.template, resultStd)
			}
		})
	}
}

// TestTopLevelExecuteFunctions tests function calls with the top-level Execute functions
func TestTopLevelExecuteFunctions(t *testing.T) {
	functions := Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"join": func(sep string, s ...string) string {
			return strings.Join(s, sep)
		},
		"error": func() (string, error) {
			return "", errors.New("test error")
		},
	}

	tests := []struct {
		name      string
		template  string
		expected  string
		expectErr bool
	}{
		{
			name:      "Simple function call",
			template:  "Hello {{upper(\"world\")}}!",
			expected:  "Hello WORLD!",
			expectErr: false,
		},
		{
			name:      "Function with multiple arguments",
			template:  "{{join(\"-\", \"a\", \"b\", \"c\")}}",
			expected:  "a-b-c",
			expectErr: false,
		},
		{
			name:      "Function returning error",
			template:  "{{error()}}",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "Multiple function calls",
			template:  "{{upper(\"hello\")}} {{add(1, 2)}}",
			expected:  "HELLO 3",
			expectErr: false,
		},
		{
			name:      "Unknown function",
			template:  "{{unknown(\"test\")}}",
			expected:  "",
			expectErr: true,
		},
	}

	// Test Execute
	for _, tt := range tests {
		t.Run("Execute_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			n, err := Execute(tt.template, "{{", "}}", &buf, functions)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}

			if n != int64(len(result)) {
				t.Errorf("Expected n=%d, got: %d", len(result), n)
			}
		})
	}

	// Test ExecuteString
	for _, tt := range tests {
		t.Run("ExecuteString_"+tt.name, func(t *testing.T) {
			if tt.expectErr {
				// ExecuteString doesn't return errors, so skip error tests
				return
			}

			result := ExecuteString(tt.template, "{{", "}}", functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}

	// Test ExecuteStd
	for _, tt := range tests {
		t.Run("ExecuteStd_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			n, err := ExecuteStd(tt.template, "{{", "}}", &buf, functions)

			if err != nil {
				t.Fatalf("ExecuteStd returned error: %v", err)
			}

			var expected string
			if tt.name == "Unknown function" {
				expected = "{{unknown(\"test\")}}"
			} else if tt.name == "Function returning error" {
				expected = "{{error()}}"
			} else {
				expected = tt.expected
			}

			result := buf.String()
			if result != expected {
				t.Errorf("Expected: %q, got: %q", expected, result)
			}

			if n != int64(len(result)) {
				t.Errorf("Expected n=%d, got: %d", len(result), n)
			}
		})
	}

	// Test ExecuteStringStd
	for _, tt := range tests {
		t.Run("ExecuteStringStd_"+tt.name, func(t *testing.T) {
			result := ExecuteStringStd(tt.template, "{{", "}}", functions)

			var expected string
			if tt.name == "Unknown function" {
				expected = "{{unknown(\"test\")}}"
			} else if tt.name == "Function returning error" {
				expected = "{{error()}}"
			} else {
				expected = tt.expected
			}

			if result != expected {
				t.Errorf("Expected: %q, got: %q", expected, result)
			}
		})
	}
}

// TestTopLevelExecuteWithNestedFunctions tests complex nested function calls with top-level Execute functions
func TestTopLevelExecuteWithNestedFunctions(t *testing.T) {
	functions := Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"repeat": func(s string, n int) string {
			return strings.Repeat(s, n)
		},
		"concat": func(s ...string) string {
			return strings.Join(s, "")
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Double nested functions",
			template: "{{upper(lower(\"Hello\"))}}",
			expected: "HELLO",
		},
		{
			name:     "Triple nested functions",
			template: "{{repeat(upper(lower(\"x\")), 3)}}",
			expected: "XXX",
		},
		{
			name:     "Complex mixed nested functions",
			template: "{{concat(upper(\"a\"), lower(\"B\"), repeat(\"c\", 3))}}",
			expected: "Abccc",
		},
		{
			name:     "Multiple nested function calls",
			template: "{{upper(lower(\"Hi\"))}} and {{repeat(lower(\"X\"), 2)}}",
			expected: "HI and xx",
		},
		{
			name:     "Nested with unknown inner function",
			template: "{{upper(unknown(\"test\"))}}",
			expected: "",
		},
	}

	// Test Execute
	for _, tt := range tests {
		t.Run("Execute_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := Execute(tt.template, "{{", "}}", &buf, functions)

			// Skip error checking for the "Nested with unknown inner function" test
			// since it is expected to fail with an error
			if tt.name != "Nested with unknown inner function" && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}

	// Test ExecuteString
	for _, tt := range tests {
		t.Run("ExecuteString_"+tt.name, func(t *testing.T) {
			result := ExecuteString(tt.template, "{{", "}}", functions)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})
	}

	// Test ExecuteStd and ExecuteStringStd
	for _, tt := range tests {
		t.Run("ExecuteStd_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := ExecuteStd(tt.template, "{{", "}}", &buf, functions)
			if err != nil {
				t.Fatalf("ExecuteStd returned error: %v", err)
			}

			var expected string
			if tt.name == "Nested with unknown inner function" {
				expected = "{{upper(unknown(\"test\"))}}"
			} else {
				expected = tt.expected
			}

			result := buf.String()
			if result != expected {
				t.Errorf("Expected: %q, got: %q", expected, result)
			}

			// Also test ExecuteStringStd
			resultString := ExecuteStringStd(tt.template, "{{", "}}", functions)
			if resultString != expected {
				t.Errorf("ExecuteStringStd: Expected: %q, got: %q", expected, resultString)
			}
		})
	}
}

// TestTopLevelExecuteWithMixedVarsAndFunctions tests using both variables and function calls
// with top-level Execute* functions
func TestTopLevelExecuteWithMixedVarsAndFunctions(t *testing.T) {
	data := Map{
		"name":     "john",
		"city":     "New York",
		"greeting": "Hello",
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"join": func(sep string, s ...string) string {
			return strings.Join(s, sep)
		},
		"combine": func(parts ...interface{}) string {
			var result string
			for _, part := range parts {
				result += fmt.Sprintf("%v", part)
			}
			return result
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Mix variables and functions",
			template: "{{greeting}} {{upper(name)}} from {{city}}!",
			expected: "Hello JOHN from New York!",
		},
		{
			name:     "Function with variable as argument",
			template: "{{upper(greeting)}} {{name}}!",
			expected: "HELLO john!",
		},
		{
			name:     "Function with multiple variables as arguments",
			template: "{{join(\", \", name, city)}}",
			expected: "john, New York",
		},
		{
			name:     "Functions with variables and explicit values",
			template: "{{combine(greeting, \" \", upper(name), \"!\")}}",
			expected: "Hello JOHN!",
		},
		{
			name:     "Mix known and unknown variables with functions",
			template: "{{upper(name)}} {{unknown}} {{city}}",
			expected: "JOHN  New York",
		},
	}

	// Test all four Execute* functions
	for _, tt := range tests {
		// Test Execute
		t.Run("Execute_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := Execute(tt.template, "{{", "}}", &buf, data)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})

		// Test ExecuteString
		t.Run("ExecuteString_"+tt.name, func(t *testing.T) {
			result := ExecuteString(tt.template, "{{", "}}", data)
			if result != tt.expected {
				t.Errorf("Expected: %q, got: %q", tt.expected, result)
			}
		})

		// Test ExecuteStd and ExecuteStringStd
		t.Run("ExecuteStd_"+tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := ExecuteStd(tt.template, "{{", "}}", &buf, data)
			if err != nil {
				t.Fatalf("ExecuteStd returned error: %v", err)
			}

			var expected string
			if tt.name == "Mix known and unknown variables with functions" {
				expected = "JOHN {{unknown}} New York"
			} else {
				expected = tt.expected
			}

			result := buf.String()
			if result != expected {
				t.Errorf("Expected: %q, got: %q", expected, result)
			}

			// Also test ExecuteStringStd
			resultString := ExecuteStringStd(tt.template, "{{", "}}", data)
			if resultString != expected {
				t.Errorf("ExecuteStringStd: Expected: %q, got: %q", expected, resultString)
			}
		})
	}
}
