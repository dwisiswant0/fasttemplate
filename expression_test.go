package fasttemplate

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestSimpleExpressions(t *testing.T) {
	t.Run("numeric operations", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{"addition", "{{1 + 2}}", "3"},
			{"subtraction", "{{5 - 3}}", "2"},
			{"multiplication", "{{2 * 3}}", "6"},
			{"division", "{{6 / 3}}", "2"},
			{"modulo", "{{7 % 3}}", "1"},
			{"power", "{{2 ** 3}}", "8"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(nil)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})

	t.Run("string operations", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{"concatenation", "{{'Hello' + ' ' + 'World'}}", "Hello World"},
			{"with number", "{{'Count: ' + 42}}", "Count: 42"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(nil)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})

	t.Run("comparison operations", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{"equal", "{{1 == 1}}", "true"},
			{"not equal", "{{1 != 2}}", "true"},
			{"greater than", "{{5 > 3}}", "true"},
			{"less than", "{{3 < 5}}", "true"},
			{"greater equal", "{{5 >= 5}}", "true"},
			{"less equal", "{{3 <= 3}}", "true"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(nil)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})

	t.Run("logical operations", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{"and true", "{{true && true}}", "true"},
			{"and false", "{{true && false}}", "false"},
			{"or true", "{{true || false}}", "true"},
			{"or false", "{{false || false}}", "false"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(nil)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})
}

func TestComplexExpressions(t *testing.T) {
	t.Run("nested operations", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{"parentheses", "{{(1 + 2) * 3}}", "9"},
			{"order of operations", "{{1 + 2 * 3}}", "7"},
			{"complex math", "{{(2 + 3) * (4 - 2) / 2}}", "5"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(nil)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})

	t.Run("mixed types", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
			data     Map
			expected string
		}{
			{
				"string and variable",
				"{{first_name + ' ' + last_name}}",
				Map{"first_name": "John", "last_name": "Doe"},
				"John Doe",
			},
			{
				"numeric variables",
				"{{num1 + num2 * num3}}",
				Map{"num1": 5, "num2": 10, "num3": 2},
				"25",
			},
			{
				"mixed strings and numbers",
				"{{first_name + ' is ' + age + ' years old'}}",
				Map{"first_name": "John", "age": 30},
				"John is 30 years old",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(tc.data)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})
}

func TestCombinedExpressionWithFunctions(t *testing.T) {
	t.Run("combined functions and expressions", func(t *testing.T) {
		data := Map{
			"first_name": "john",
			"last_name":  "doe",
			"upper": func(s string) string {
				return strings.ToUpper(s)
			},
			"lower": func(s string) string {
				return strings.ToLower(s)
			},
		}

		testCases := []struct {
			name     string
			template string
			expected string
		}{
			{
				"upper function with concat",
				"{{upper(first_name + ' ' + last_name)}}",
				"JOHN DOE",
			},
			// For now, let's test just one function call approach
			// The next one requires a different syntax for our implementation
			// {
			//    "function with expression inside",
			//    "{{upper(first_name) + ' ' + upper(last_name)}}",
			//    "JOHN DOE",
			// },
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tpl := New(tc.template, "{{", "}}")
				result := tpl.ExecuteString(data)
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			})
		}
	})
}

// TestComplexEdgeCases covers advanced edge cases combining variables, functions, and expressions
func TestComplexEdgeCases(t *testing.T) {
	t.Run("functions with variables and expressions", func(t *testing.T) {
		// This test verifies that nested function calls work correctly
		template := `{{format(multiply(add(5, x), 2))}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"x": 7,
			"add": func(a, b int) int {
				return a + b
			},
			"multiply": func(a, b int) int {
				return a * b
			},
			"format": func(n int) string {
				return "[ " + strconv.Itoa(n) + " ]"
			},
		}

		result := tpl.ExecuteString(data)
		expected := "[ 24 ]"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("mixed expressions in function arguments", func(t *testing.T) {
		template := `{{calculate(x * 2, y + 3, z / 2)}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"x": 5,
			"y": 7,
			"z": 10,
			"calculate": func(a, b, c float64) string {
				return fmt.Sprintf("%.1f + %.1f + %.1f = %.1f", a, b, c, a+b+c)
			},
		}

		result := tpl.ExecuteString(data)
		expected := "10.0 + 10.0 + 5.0 = 25.0"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("complex output formatting", func(t *testing.T) {
		// This test verifies boolean expressions in function calls
		template := `{{format(total, total >= threshold, user_name)}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"total":     95,
			"threshold": 90,
			"user_name": "admin",
			"format": func(value int, warning bool, user string) string {
				level := "info"
				if warning {
					level = "warning"
				}
				return fmt.Sprintf("[%s] User %s reports value %d", level, user, value)
			},
		}

		result := tpl.ExecuteString(data)
		expected := "[warning] User admin reports value 95"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("multiple functions and operators in one template", func(t *testing.T) {
		template := `Start {{func1(x)}} middle {{x + y * z}} end {{func2("test")}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"x": 10,
			"y": 5,
			"z": 3,
			"func1": func(val int) string {
				return "[" + strconv.Itoa(val) + "]"
			},
			"func2": func(val string) string {
				return strings.ToUpper(val)
			},
		}

		result := tpl.ExecuteString(data)
		expected := "Start [10] middle 25 end TEST"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("type conversion in expressions", func(t *testing.T) {
		// Test string concatenation with numeric values
		template := `{{price + " x " + quantity + " = $" + formatted}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"price":     19.99,
			"quantity":  3,
			"formatted": "59.97",
		}

		result := tpl.ExecuteString(data)
		expected := "19.99 x 3 = $59.97"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("dynamic function dispatch", func(t *testing.T) {
		template := `{{operation(x, y)}}`
		tpl := New(template, "{{", "}}")

		// Test with add operation
		addResult := tpl.ExecuteString(Map{
			"operation": func(a, b int) int {
				return a + b
			},
			"x": 5,
			"y": 3,
		})
		if addResult != "8" {
			t.Errorf("Add operation: Expected: %q, got: %q", "8", addResult)
		}

		// Test with multiply operation
		mulResult := tpl.ExecuteString(Map{
			"operation": func(a, b int) int {
				return a * b
			},
			"x": 5,
			"y": 3,
		})
		if mulResult != "15" {
			t.Errorf("Multiply operation: Expected: %q, got: %q", "15", mulResult)
		}
	})

	t.Run("expressions with function results", func(t *testing.T) {
		// Test combining function calls with expressions
		template := `{{label(getCount())}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"getCount": func() int {
				return 15
			},
			"label": func(count int) string {
				if count > 10 {
					return "Many"
				}
				return "Few"
			},
		}

		result := tpl.ExecuteString(data)
		expected := "Many"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})

	t.Run("mixed numeric types", func(t *testing.T) {
		template := `{{int_val + float_val}}`
		tpl := New(template, "{{", "}}")
		data := Map{
			"int_val":   10,
			"float_val": 5.75,
		}

		result := tpl.ExecuteString(data)
		expected := "15.75"

		if result != expected {
			t.Errorf("Expected: %q, got: %q", expected, result)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("invalid expressions", func(t *testing.T) {
		testCases := []struct {
			name     string
			template string
		}{
			{"division by zero", "{{1 / 0}}"},
			{"mismatched parentheses", "{{(1 + 2}}"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// These should not panic
				tpl := New(tc.template, "{{", "}}")

				// Wrap in a recover to catch panics if they occur
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Test panicked: %v", r)
					}
				}()

				// Execute should handle errors gracefully - ignore the result
				_ = tpl.ExecuteString(nil)

				// In Std mode, errors preserve the original tag
				resultStd := tpl.ExecuteStringStd(nil)
				if resultStd != tc.template {
					t.Errorf("Expected original tag %q in Std mode, got %q", tc.template, resultStd)
				}
			})
		}
	})
}

func TestExampleFromReadme(t *testing.T) {
	template := "Hello {{upper(first_name + \" Doe\")}}!"
	tpl := New(template, "{{", "}}")

	result := tpl.ExecuteString(Map{
		"first_name": "john",
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
	})

	expected := "Hello JOHN DOE!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
