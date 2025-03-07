package fasttemplate

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Map is a map of function names to functions that can be called from templates.
type Map map[string]any

// FunctionCall represents a parsed function call in a template.
type functionCall struct {
	Name string
	Args []any // can be string, int, float64, bool, or anything else
}

// expressionPlaceholder represents an expression that needs to be evaluated
type expressionPlaceholder struct {
	expression string
}

// executeFunctionCall executes the function represented by this call.
func (fc *functionCall) execute(funcs, data Map) (interface{}, error) {
	fn, ok := funcs[fc.Name]
	if !ok {
		return nil, fmt.Errorf("function not found: %s", fc.Name)
	}

	// Prepare arguments
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s is not a function", fc.Name)
	}

	// Pre-allocate args slice to exact needed size
	argsLen := len(fc.Args)
	args := make([]reflect.Value, argsLen)
	
	for i, arg := range fc.Args {
		// Fast path for simple types (most common case)
		switch typedArg := arg.(type) {
		case string:
			// Handle variable lookup for strings
			if data != nil {
				if val, exists := data[typedArg]; exists {
					args[i] = reflect.ValueOf(val)
					continue
				}
			}
			args[i] = reflect.ValueOf(typedArg)
		case int, float64, bool:
			// Fast path for primitive types
			args[i] = reflect.ValueOf(arg)
		case *functionCall:
			// Handle nested function calls
			result, err := typedArg.execute(funcs, data)
			if err != nil {
				return nil, err
			}
			args[i] = reflect.ValueOf(result)
		case *expressionPlaceholder:
			// Handle expressions
			result, err := evalExpression(typedArg.expression, data)
			if err != nil {
				return nil, err
			}
			args[i] = reflect.ValueOf(result)
		default:
			args[i] = reflect.ValueOf(arg)
		}
	}

	// Call the function with panic recovery
	var panicErr error
	var result []reflect.Value

	// Use pre-calculated reflection value to avoid repeated reflections
	fnValue := reflect.ValueOf(fn)
	
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("%s: %v", fc.Name, r)
			}
		}()
		result = fnValue.Call(args)
	}()

	if panicErr != nil {
		return nil, panicErr
	}

	// Fast path for functions with no return value
	if len(result) == 0 {
		return nil, nil
	}

	// Fast path for single return value (most common case)
	if len(result) == 1 {
		return result[0].Interface(), nil
	}

	// Handle error return value if present
	if !result[1].IsNil() && result[1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, result[1].Interface().(error)
	}

	return result[0].Interface(), nil
}

// parseFunctionCall parses a string into a function call structure.
func parseFunctionCall(s string) (*functionCall, error) {
	s = strings.TrimSpace(s)

	// find function name
	parenIdx := strings.IndexByte(s, '(')
	if parenIdx < 1 {
		return nil, fmt.Errorf("invalid function call format: %s", s)
	}

	name := strings.TrimSpace(s[:parenIdx])
	if !isValidFunctionName(name) {
		return nil, fmt.Errorf("invalid function name: %s", name)
	}

	// check for matching closing parenthesis
	if !strings.HasSuffix(s, ")") {
		return nil, fmt.Errorf("missing closing parenthesis: %s", s)
	}

	// Extract arguments string
	argsStr := s[parenIdx+1 : len(s)-1]
	args, err := parseArgs(argsStr)
	if err != nil {
		return nil, err
	}

	return &functionCall{
		Name: name,
		Args: args,
	}, nil
}

// parseArgs parses a comma-separated list of arguments.
func parseArgs(s string) ([]interface{}, error) {
	var args []interface{}
	var currentArg strings.Builder
	var inSingleQuote, inDoubleQuote bool
	var parenDepth int

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		// Handle quotes
		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			currentArg.WriteRune(r)
			continue
		} else if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			currentArg.WriteRune(r)
			continue
		}

		// Handle nested func calls
		if r == '(' && !inSingleQuote && !inDoubleQuote {
			parenDepth++
			currentArg.WriteRune(r)
			continue
		} else if r == ')' && !inSingleQuote && !inDoubleQuote {
			parenDepth--
			currentArg.WriteRune(r)
			continue
		}

		// Handle arg separation
		if r == ',' && !inSingleQuote && !inDoubleQuote && parenDepth == 0 {
			argStr := strings.TrimSpace(currentArg.String())
			if argStr != "" {
				arg, err := parseArg(argStr)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
			currentArg.Reset()
			continue
		}

		// Add character to current arg
		currentArg.WriteRune(r)
	}

	// Add the last arg
	argStr := strings.TrimSpace(currentArg.String())
	if argStr != "" {
		arg, err := parseArg(argStr)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

// parseArg parses a single arg value.
func parseArg(s string) (interface{}, error) {
	// Fast path for quoted strings (common case)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			// Remove quotes
			return s[1 : len(s)-1], nil
		}
	}

	// Check if it's a nested func call
	if strings.IndexByte(s, '(') > 0 && s[len(s)-1] == ')' {
		return parseFunctionCall(s)
	}

	// Try to parse as a number (fast path for common integer cases)
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		// Try integer first (most common case)
		if i, err := strconv.Atoi(s); err == nil {
			return i, nil
		}
		// Then try float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, nil
		}
	}

	// Fast path for boolean literals
	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}

	// Check for expressions with operators 
	// (this is expensive, so do it last)
	if isExpression(s) {
		return &expressionPlaceholder{expression: s}, nil
	}

	// Return as is (likely a variable name)
	return s, nil
}

// isValidFunctionName checks if a function name is valid.
func isValidFunctionName(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// isFunctionCall checks if a tag might be a function call.
func isFunctionCall(tag string) bool {
	tag = strings.TrimSpace(tag)
	parenIdx := strings.IndexByte(tag, '(')
	return parenIdx > 0 && strings.HasSuffix(tag, ")")
}
