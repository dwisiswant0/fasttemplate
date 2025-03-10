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

// Merge combines the contents of another Map into this Map.
// Values from the other Map will overwrite values in this Map if keys conflict.
func (m Map) Merge(other Map) Map {
	for k, v := range other {
		m[k] = v
	}
	return m
}

// FunctionCall represents a parsed function call in a template.
type functionCall struct {
	Name string
	Args []any // can be string, int, float64, bool, or anything else
}

// expressionPlaceholder represents an expression that needs to be evaluated
type expressionPlaceholder struct {
	expression string
}

// literalString represents a string that was quoted in the original template
// and should be treated as a literal value, not a variable reference.
type literalString string

// executeFunctionCall executes the function represented by this call.
func (fc *functionCall) execute(funcs, data Map) (interface{}, error) {
	fn, ok := funcs[fc.Name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errFunctionNotFound, fc.Name)
	}

	// Prepare args
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s is not a function", fc.Name)
	}

	reflectArgs := make([]reflect.Value, 0, len(fc.Args))

	for _, arg := range fc.Args {
		// Fast path for simple types (most common case)
		switch typedArg := arg.(type) {
		case literalString:
			// This string was quoted in the original template, so it's a literal
			// Convert back to a regular string for the function call
			reflectArgs = append(reflectArgs, reflect.ValueOf(string(typedArg)))

		case string:
			// Handle variable lookup for strings
			if data != nil {
				if val, exists := data[typedArg]; exists {
					reflectArgs = append(reflectArgs, reflect.ValueOf(val))
					continue
				}

				// For unquoted variables like in upper(last_name)
				// We need to check if this string is likely a variable name
				// rather than a literal string value
				if isLikelyVariable(typedArg) {
					return nil, fmt.Errorf("%w: %s", errVariableNotFound, typedArg)
				}
			}

			// If not a variable or if data is nil, treat as literal
			reflectArgs = append(reflectArgs, reflect.ValueOf(typedArg))

		case int, float64, bool:
			// Fast path for primitive types
			reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
		case *functionCall:
			// Handle nested function calls
			result, err := typedArg.execute(funcs, data)
			if err != nil {
				// Bubble up the error for proper handling in Std mode
				return nil, err
			}
			reflectArgs = append(reflectArgs, reflect.ValueOf(result))
		case *expressionPlaceholder:
			// Handle expressions
			result, err := evalExpression(typedArg.expression, data)
			if err != nil {
				// Bubble up the error for proper handling in Std mode
				return nil, err
			}
			reflectArgs = append(reflectArgs, reflect.ValueOf(result))
		default:
			reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
		}
	}

	// Call the function with panic recovery
	var panicErr error
	var result []reflect.Value

	// Use pre-calc reflection value to avoid repeated reflections
	fnValue := reflect.ValueOf(fn)

	// For variadic funcs, we need to handle the arguments differently
	var callResult []reflect.Value
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("%s: %v", fc.Name, r)
			}
		}()

		// Special handling for variadic functions
		if fnType.IsVariadic() {
			// Get the required non-variadic arguments count
			nonVariadicCount := fnType.NumIn() - 1

			// For variadic funcs, we need to ensure we have at least the
			// required arguments
			if len(reflectArgs) >= nonVariadicCount {
				// For variadic funcs, just call with args as is
				callResult = fnValue.Call(reflectArgs)
			} else {
				// Not enough arguments for variadic function
				callResult = fnValue.Call(reflectArgs)
			}
		} else {
			// For non-variadic funcs, check if we have the right argument count
			if len(reflectArgs) != fnType.NumIn() {
				// Wrong number of arguments
				panicErr = fmt.Errorf("invalid argument count: expected %d, got %d", fnType.NumIn(), len(reflectArgs))
				return
			}
			// For non-variadic funcs, just call normally
			callResult = fnValue.Call(reflectArgs)
		}
	}()

	if panicErr != nil {
		return nil, panicErr
	}

	result = callResult

	// Fast path for functions with no return value
	if len(result) == 0 {
		return nil, nil
	}

	// Fast path for single return value (most common case)
	if len(result) == 1 {
		return result[0].Interface(), nil
	}

	// Handle error return value if present
	if len(result) >= 2 && !result[1].IsNil() && result[1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
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
	var escaped bool

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		// Handle escape sequences
		if escaped {
			escaped = false
			currentArg.WriteRune(r)
			continue
		}

		// Handle backslash for escaping
		if r == '\\' && (inSingleQuote || inDoubleQuote) {
			escaped = true
			currentArg.WriteRune(r)
			continue
		}

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

	// Check for unterminated quotes or parentheses
	if inSingleQuote || inDoubleQuote || parenDepth != 0 {
		return nil, fmt.Errorf("unterminated quotes or parentheses in arguments: %s", s)
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
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			// Remove single quotes and return as a literal string
			// Mark as literal by using the literalString type
			return literalString(s[1 : len(s)-1]), nil
		}
		if s[0] == '"' && s[len(s)-1] == '"' {
			// Remove double quotes and return as a literal string
			// Mark as literal by using the literalString type
			return literalString(s[1 : len(s)-1]), nil
		}
	}

	// Check if it's a nested func call
	if strings.IndexByte(s, '(') > 0 && s[len(s)-1] == ')' {
		funcCall, err := parseFunctionCall(s)
		if err != nil {
			// If parsing failed but it has parentheses, treat it as a string
			// This helps prevent treating invalid function calls as variables
			return s, nil
		}
		return funcCall, nil
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

// isLikelyVariable determines if a string is likely a variable name rather than
// a literal string.
// This helps distinguish between variables that should be looked up and literal
// strings that should be used as-is.
func isLikelyVariable(s string) bool {
	// If it's quoted, it's not a variable
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return false
		}
	}

	// If it starts with a number, it's likely a numeric value
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		return false
	}

	// If it's a boolean literal, it's not a variable
	if s == "true" || s == "false" {
		return false
	}

	// If it contains spaces or operators, it's likely an expr, not a variable
	if strings.ContainsAny(s, " \t\n\r+-*/=<>!&|%") {
		return false
	}

	// Otherwise, it's likely a variable name
	return true
}
