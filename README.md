fasttemplate
============

Simple and fast template engine for Go.

Fasttemplate performs template substitution with high efficiency. This fork adds significant new capabilities:

1. **Basic substitution**: Replaces placeholders with user-defined values.
2. **Function support**: Enables function calls within templates for data transformation. 🆕
3. **Expression evaluation**: Supports mathematical, logical, and comparison operators. 🆕

All at high speed :)

> [!WARNING]
> **fasttemplate** does NOT do any escaping on template values unlike [html/template](http://golang.org/pkg/html/template/) do. So values must be properly escaped before passing them to `fasttemplate`.

Fasttemplate is faster than [text/template](http://golang.org/pkg/text/template/),
[strings.Replace](http://golang.org/pkg/strings/#Replace),
[strings.Replacer](http://golang.org/pkg/strings/#Replacer)
and [fmt.Fprintf](https://golang.org/pkg/fmt/#Fprintf) on placeholders' substitution.

Below are benchmark results comparing fasttemplate performance to `text/template`,
`strings.Replace`, `strings.Replacer` and `fmt.Fprintf`:

```console
$ go test -benchmem -run=^$ -bench .
goos: linux
goarch: amd64
pkg: github.com/dwisiswant0/fasttemplate
cpu: 11th Gen Intel(R) Core(TM) i9-11900H @ 2.50GHz
BenchmarkFmtFprintf-16                           	13713362	        96.64 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringsReplace-16                       	 3393204	       446.4 ns/op	    1168 B/op	       7 allocs/op
BenchmarkStringsReplacer-16                      	 1211418	       993.5 ns/op	    2552 B/op	      24 allocs/op
BenchmarkTextTemplate-16                         	 1717138	       675.9 ns/op	     352 B/op	      20 allocs/op
BenchmarkFastTemplateExecuteFunc-16              	 4194436	       292.5 ns/op	      64 B/op	       3 allocs/op
BenchmarkFastTemplateExecute-16                  	 4164301	       289.5 ns/op	      64 B/op	       3 allocs/op
BenchmarkFastTemplateExecuteStd-16               	 5574823	       218.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastTemplateExecuteString-16            	 2329857	       507.0 ns/op	     673 B/op	       7 allocs/op
BenchmarkFastTemplateExecuteStringStd-16         	 2734778	       444.8 ns/op	     625 B/op	       4 allocs/op
BenchmarkNewTemplate-16                          	 4027020	       299.0 ns/op	     768 B/op	       3 allocs/op
BenchmarkTemplateReset-16                        	16813707	        73.24 ns/op	       0 B/op	       0 allocs/op
BenchmarkExecuteFunc-16                          	 3263163	       354.9 ns/op	      64 B/op	       3 allocs/op
BenchmarkFastTemplateFunctionCall-16             	 2346298	       514.0 ns/op	     408 B/op	      19 allocs/op
BenchmarkFastTemplateNestedFunctions-16          	 2494704	       478.1 ns/op	     417 B/op	      19 allocs/op
BenchmarkFastTemplateVariableArguments-16        	 1701109	       724.9 ns/op	     905 B/op	      31 allocs/op
BenchmarkFastTemplateFunctionWithVariables-16    	 2042143	       585.7 ns/op	     352 B/op	      18 allocs/op
BenchmarkFastTemplateComplexFunctions-16         	  373095	      3276 ns/op	    2904 B/op	      91 allocs/op
# TRUNCATED BenchmarkFastTemplateEval* RESULTS
```

Docs
====

See http://pkg.go.dev/github.com/dwisiswant0/fasttemplate.

Usage
=====

## Basic variable substitution

```go
template := "http://{{host}}/?q={{query}}&foo={{bar}}{{bar}}"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "host":  "google.com",
    "query": url.QueryEscape("hello=world"),
    "bar":   "foobar",
})
fmt.Printf("%s", s)

// Output:
// http://google.com/?q=hello%3Dworld&foo=foobarfoobar
```

## Using function calls in templates

```go
template := "Hello, {{capitalize(name)}}! Today is {{concat(day, ', ', month, ' ', year)}}."
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "name":  "john",
    "day":   "Monday",
    "month": "January",
    "year":  "2023",
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
```

## Nested function calls

```go
template := "Result: {{multiply(add(5, 10), 2)}}"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
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
```

## Using variables as function arguments

```go
template := "{{upper(name)}}'s total: {{formatNumber(total)}}"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
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
```

## Keeping unknown placeholders with `ExecuteStd`

```go
template := "Hello, {{name}}! {{unknown(value)}}"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteStringStd(fasttemplate.Map{
    "name": "John",
})
fmt.Printf("%s", s)

// Output:
// Hello, John! {{unknown(value)}}
```

## Error handling with functions

```go
template := "Result: {{validateData(input)}}"
t := fasttemplate.New(template, "{{", "}}")

// Set up a variable to capture output
var buf bytes.Buffer

// Try with valid data
_, err := t.Execute(&buf, fasttemplate.Map{
    "input": "valid-input",
    "validateData": func(s string) (string, error) {
        if strings.HasPrefix(s, "valid") {
            return "Validation passed!", nil
        }
        return "", fmt.Errorf("validation failed for input: %q", s)
    },
})

if err != nil {
    fmt.Println("Error:", err)
} else {
    fmt.Println(buf.String()) // Output: Result: Validation passed!
}

// Try with invalid data
buf.Reset()
_, err = t.Execute(&buf, fasttemplate.Map{
    "input": "incorrect-format",
    "validateData": func(s string) (string, error) {
        if strings.HasPrefix(s, "valid") {
            return "Validation passed!", nil
        }
        return "", fmt.Errorf("validation failed for input: %q", s)
    },
})

if err != nil {
    fmt.Println("Error:", err) // Output: Error: validation failed for input: "incorrect-format"
}
```

> [!NOTE]
> `ExecuteStd` doesn't return errors from function calls - it preserves the original tag text instead.

## Using expressions with operators

```go
template := "The total price is: ${{price * quantity * (1 - discount)}} ({{quantity}} items)"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "price":    29.99,
    "quantity": 3, 
    "discount": 0.15,
})
fmt.Printf("%s", s)

// Output:
// The total price is: $76.47 (3 items)
```

## Comparisons and logical operations

```go
template := "Is adult: {{age >= 18}} | Is senior: {{age > 65}} | Can purchase: {{age >= 18 && has_id}}"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "age": 70,
    "has_id": true,
})
fmt.Printf("%s", s)

// Output:
// Is adult: true | Is senior: true | Can purchase: true
```

## String operations

```go
template := "Hello, {{first_name + ' ' + last_name}}!"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "first_name": "John",
    "last_name":  "Doe",
})
fmt.Printf("%s", s)

// Output:
// Hello, John Doe!
```

## Combining functions and expressions

```go
template := "Hello {{upper(first_name + \" \" + last_name)}}!"
t := fasttemplate.New(template, "{{", "}}")
s := t.ExecuteString(fasttemplate.Map{
    "first_name": "john",
    "last_name":  "doe",
    "upper": func(s string) string {
        return strings.ToUpper(s)
    },
})
fmt.Printf("%s", s)

// Output:
// Hello JOHN DOE!
```

## Validating templates before execution

```go
template := "Hello, {{name}}! Your discount is {{calculateDiscount(total, membership)}}."
// Check if all template variables and functions are available
err := fasttemplate.Validate(template, "{{", "}}", fasttemplate.Map{
    "name": "John",
    "total": 100.0,
    "membership": "gold",
    "calculateDiscount": func(total float64, tier string) float64 {
        switch tier {
        case "gold":
            return total * 0.15
        case "silver":
            return total * 0.10
        default:
            return 0
        }
    },
})

if err != nil {
    // Handle validation error
    fmt.Println("Template validation failed:", err)
} else {
    // Template is valid, proceed with execution
    t := fasttemplate.New(template, "{{", "}}")
    result := t.ExecuteString(/* same map as used for validation */)
    fmt.Println(result)
}

// Output:
// Hello, John! Your discount is 15.
```

## Direct expression evaluation with typed results

```go
// Simple variable lookup with strong typing
name, err := fasttemplate.Eval[string]("name", fasttemplate.Map{
    "name": "John",
})
fmt.Println("Name:", name) // Output: Name: John

// Expression evaluation with correct return type
total, err := fasttemplate.Eval[float64]("price * quantity * (1 - discount)", fasttemplate.Map{
    "price":    29.99,
    "quantity": 3,
    "discount": 0.15,
})
fmt.Printf("Total: $%.2f\n", total) // Output: Total: $76.47

// Function call evaluation 
greeting, err := fasttemplate.Eval[string]("greet(name)", fasttemplate.Map{
    "name": "Alice",
    "greet": func(name string) string {
        return "Hello, " + name + "!"
    },
})
fmt.Println(greeting) // Output: Hello, Alice!

// Boolean expression evaluation
isEligible, err := fasttemplate.Eval[bool]("age >= 21 && hasID", fasttemplate.Map{
    "age": 25,
    "hasID": true,
})
fmt.Println("Is eligible:", isEligible) // Output: Is eligible: true
```

License
=======

MIT.