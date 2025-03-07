package fasttemplate

import (
	"bytes"
	"fmt"
	"strings"
)

func ExampleTemplate() {
	// Define a variety of functions to demonstrate the capabilities
	template := `
Items: {{join(", ", "apple", "banana", "orange")}}
Price: {{formatPrice(price)}}
Status: {{choose(inStock, "In Stock", "Out of Stock")}}
`

	t := New(template, "{{", "}}")

	// Execute the template with data and functions
	s := t.ExecuteString(Map{
		"price":   29.99,
		"inStock": true,
		// Functions defined in the data map
		"join": func(sep string, items ...string) string {
			return strings.Join(items, sep)
		},
		"formatPrice": func(price float64) string {
			return fmt.Sprintf("$%.2f", price)
		},
		"choose": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
	})
	fmt.Printf("%s", s)

	// Output:
	//
	// Items: apple, banana, orange
	// Price: $29.99
	// Status: In Stock
}

func ExampleTemplate_variables() {
	// Create a template that uses functions with variable arguments
	template := "{{upper(name)}}'s total: {{formatNumber(total)}}"

	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with data - notice how variables are referenced by name directly in function calls
	s := t.ExecuteString(Map{
		"name":  "alice",
		"total": 42.75,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
	})
	fmt.Printf("%s", s)

	// Output:
	// ALICE's total: 42.75
}

func ExampleTemplate_function() {
	// Create a template with function calls
	template := "Hello, {{capitalize(name)}}! Today is {{concat(day, ', ', month, ' ', year)}}."

	// Create a new template instance
	t := New(template, "{{", "}}")

	// Execute the template with data and functions
	s := t.ExecuteString(Map{
		"name":  "john",
		"day":   "Monday",
		"month": "January",
		"year":  "2023",
		// Functions are defined directly in the data map
		"capitalize": func(s string) string {
			if s == "" {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
		"concat": func(parts ...string) string {
			return strings.Join(parts, "")
		},
	})
	fmt.Printf("%s", s)

	// Output:
	// Hello, John! Today is Monday, January 2023.
}

func ExampleTemplate_nested_function_call() {
	// Create a template with nested function calls
	template := "Result: {{multiply(add(5, 10), 2)}}"

	// Create a new template instance
	t := New(template, "{{", "}}")

	// Execute the template with functions in data map
	s := t.ExecuteString(Map{
		"add": func(a, b int) int {
			return a + b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
	})
	fmt.Printf("%s", s)

	// Output:
	// Result: 30
}

func ExampleTemplate_variable_function_args() {
	// Create a template that uses functions with variable arguments
	template := "{{upper(name)}}'s total: {{formatNumber(total)}}"

	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with data - notice how variables are referenced by name directly in function calls
	s := t.ExecuteString(Map{
		"name":  "alice",
		"total": 42.75,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
	})
	fmt.Printf("%s", s)

	// Output:
	// ALICE's total: 42.75
}

func ExampleTemplate_error_handling() {
	// Create a template with a function that may return an error
	template := "Processing: {{validateInput(data)}}"
	t := New(template, "{{", "}}")

	// Function that validates input and returns error for invalid data
	validateFunc := func(input string) (string, error) {
		if input == "invalid" {
			return "", fmt.Errorf("invalid input: %q", input)
		}
		return "Valid: " + input, nil
	}

	// Set up data maps for success and failure cases
	successData := Map{
		"data":          "good-data",
		"validateInput": validateFunc,
	}

	errorData := Map{
		"data":          "invalid",
		"validateInput": validateFunc,
	}

	// Success case
	var buf bytes.Buffer
	_, err := t.Execute(&buf, successData)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println(buf.String())
	}

	// Error case
	buf.Reset()
	_, err = t.Execute(&buf, errorData)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println(buf.String())
	}

	// Output:
	// Processing: Valid: good-data
	// Error: invalid input: "invalid"
}

func ExampleTemplate_arithmetic_expression() {
	// Create a template with arithmetic expressions
	template := `
Price: ${{price}}
Quantity: {{quantity}}
Discount: {{discount * 100}}%
Subtotal: ${{price * quantity}}
Total: ${{total}}
`
	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with numeric values
	s := t.ExecuteString(Map{
		"price":    24.99,
		"quantity": 3,
		"discount": 0.15,
		"total":    63.72, // Pre-calculated for consistent output
	})
	fmt.Printf("%s", s)

	// Output:
	//
	// Price: $24.99
	// Quantity: 3
	// Discount: 15%
	// Subtotal: $74.97
	// Total: $63.72
}

func ExampleTemplate_comparison_expression() {
	// Create a template with comparison expressions
	template := `
Age: {{age}}
Is adult: {{age >= 18}}
Is senior: {{age >= 65}}
Can purchase: {{age >= 18 && has_id}}
Will receive discount: {{(age < 13) || (age >= 65)}}
`
	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with data
	s := t.ExecuteString(Map{
		"age":    70,
		"has_id": true,
	})
	fmt.Printf("%s", s)

	// Output:
	//
	// Age: 70
	// Is adult: true
	// Is senior: true
	// Can purchase: true
	// Will receive discount: true
}

func ExampleTemplate_string_expressions() {
	// Create a template with string expressions
	template := `
First name: {{first_name}}
Last name: {{last_name}}
Full name: {{first_name + " " + last_name}}
Biography preview: {{bio + "..."}}
`
	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with string values
	s := t.ExecuteString(Map{
		"first_name": "John",
		"last_name":  "Doe",
		"bio":        "John is a software engineer",
	})
	fmt.Printf("%s", s)

	// Output:
	//
	// First name: John
	// Last name: Doe
	// Full name: John Doe
	// Biography preview: John is a software engineer...
}

func ExampleTemplate_expression_with_function() {
	// Create a template that combines expressions with function calls
	template := `
Original name: {{first_name + " " + last_name}}
Uppercase: {{upper(first_name + " " + last_name)}}
Formatted price: ${{formatNumber(price)}}
`
	// Create template instance
	t := New(template, "{{", "}}")

	// Execute with data and functions
	s := t.ExecuteString(Map{
		"first_name": "john",
		"last_name":  "doe",
		"price":      19.99,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
	})
	fmt.Printf("%s", s)

	// Output:
	//
	// Original name: john doe
	// Uppercase: JOHN DOE
	// Formatted price: $19.99
}
