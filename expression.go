package fasttemplate

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// expressionCache is a cache for storing parsed expressions
type expressionCache struct {
	mu      sync.RWMutex
	postfix map[string][]token
}

// exprCache is the global expression cache
var exprCache = expressionCache{
	postfix: make(map[string][]token),
}

// List of operators in order of precedence (low to high)
var operators = map[string]int{
	"||": 1,
	"&&": 2,
	"==": 3, "!=": 3,
	">": 4, ">=": 4, "<": 4, "<=": 4,
	"+": 5, "-": 5,
	"*": 6, "/": 6, "%": 6,
	"**": 7, // Power operator
}

// Common operators for quick detection (single char)
var commonOps = map[byte]bool{
	'+': true,
	'-': true,
	'*': true,
	'/': true,
	'=': true,
	'<': true,
	'>': true,
	'!': true,
	'&': true,
	'|': true,
	'%': true,
}

// isExpression checks if a tag contains an expression (operators)
func isExpression(tag string) bool {
	if isFunctionCall(tag) {
		return false
	}

	tag = strings.TrimSpace(tag)

	// scan for common operators first
	inSingleQuote := false
	inDoubleQuote := false
	for i := 0; i < len(tag); i++ {
		// Handle quotes
		if tag[i] == '\'' && (i == 0 || tag[i-1] != '\\') {
			inSingleQuote = !inSingleQuote
			continue
		}
		if tag[i] == '"' && (i == 0 || tag[i-1] != '\\') {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		// Skip characters inside quotes
		if inSingleQuote || inDoubleQuote {
			continue
		}

		// check for common single-char operators first
		if commonOps[tag[i]] {
			return true
		}

		// Check for multi-char operators only when we see a potential start
		if (tag[i] == '&' || tag[i] == '|' || tag[i] == '=' || tag[i] == '!' ||
			tag[i] == '<' || tag[i] == '>' || tag[i] == '*') && i+1 < len(tag) {
			// Check for ** (power)
			if tag[i] == '*' && tag[i+1] == '*' {
				return true
			}
			// Check for && and ||
			if (tag[i] == '&' && tag[i+1] == '&') || (tag[i] == '|' && tag[i+1] == '|') {
				return true
			}
			// Check for ==, !=, >=, <=
			if (tag[i] == '=' && tag[i+1] == '=') ||
				(tag[i] == '!' && tag[i+1] == '=') ||
				(tag[i] == '>' && tag[i+1] == '=') ||
				(tag[i] == '<' && tag[i+1] == '=') {
				return true
			}
		}
	}

	return false
}

// evalExpression evaluates an expression and returns the result
func evalExpression(expression string, data Map) (interface{}, error) {
	// check if it's a simple function call that doesn't need tokenization
	if isFunctionCall(expression) {
		funcCall, err := parseFunctionCall(expression)
		if err != nil {
			return nil, err
		}
		result, err := funcCall.execute(data, data)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	exprCache.mu.RLock()
	postfixTokens, found := exprCache.postfix[expression]
	exprCache.mu.RUnlock()

	if !found {
		tokens, err := tokenize(expression)
		if err != nil {
			return nil, err
		}

		postfixTokens, err = toPostfix(tokens)
		if err != nil {
			return nil, err
		}

		// Store in cache
		exprCache.mu.Lock()
		exprCache.postfix[expression] = postfixTokens
		exprCache.mu.Unlock()
	}

	// Evaluate the postfix expression
	return evaluatePostfix(postfixTokens, data)
}

// Token types
const (
	tokenOperator = iota
	tokenNumber
	tokenString
	tokenIdentifier
	tokenLeftParen
	tokenRightParen
	tokenFunctionCall
)

// Token structure
type token struct {
	typ   int
	value string
}

// pre-alloc token slice size - a reasonable estimate for most expressions
const initialTokenCapacity = 16

// Fast lookup for single-char operators
var singleCharOps = map[byte]bool{
	'+': true,
	'-': true,
	'*': true,
	'/': true,
	'%': true,
	'>': true,
	'<': true,
	'=': true,
	'!': true,
}

// Multi-char operators mapping for quick lookup
var multiCharOps = map[string]bool{
	"||": true,
	"&&": true,
	"==": true,
	"!=": true,
	">=": true,
	"<=": true,
	"**": true,
}

// tokenPool is a pool for reusing token slices
var tokenPool = sync.Pool{
	New: func() any {
		tokens := make([]token, 0, initialTokenCapacity)
		return &tokens
	},
}

// tokenize converts a string expression into tokens
func tokenize(expr string) ([]token, error) {
	// Get token slice from pool
	tokensPtr := tokenPool.Get().(*[]token)
	tokens := *tokensPtr
	tokens = tokens[:0] // Clear but keep capacity

	for i := 0; i < len(expr); {
		c := expr[i]

		// Skip whitespace (fast path using direct byte comp)
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		// Handle numbers (fast path using direct byte comp)
		if c >= '0' && c <= '9' {
			start := i
			hasDot := false

			// Fast digit scanning
			for i < len(expr) {
				if expr[i] >= '0' && expr[i] <= '9' {
					i++
				} else if expr[i] == '.' {
					if hasDot {
						return nil, fmt.Errorf("invalid number format: multiple decimal points")
					}
					hasDot = true
					i++
				} else {
					break
				}
			}

			tokens = append(tokens, token{tokenNumber, expr[start:i]})
			continue
		}

		// Handle identifiers and function calls (variable names) - fast path
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			start := i
			i++
			// Fast scan for identifier chars
			for i < len(expr) {
				ch := expr[i]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
					(ch >= '0' && ch <= '9') || ch == '_' {
					i++
				} else {
					break
				}
			}

			// check if this is a function call
			if i < len(expr) && expr[i] == '(' {
				// Find the matching closing parenthesis
				parenDepth := 1
				funcStart := start
				i++ // Skip the opening parenthesis

				for parenDepth > 0 && i < len(expr) {
					if expr[i] == '(' {
						parenDepth++
					} else if expr[i] == ')' {
						parenDepth--
					} else if expr[i] == '"' || expr[i] == '\'' {
						// Skip quoted strings
						quote := expr[i]
						i++
						for i < len(expr) && expr[i] != quote {
							if expr[i] == '\\' && i+1 < len(expr) {
								i += 2 // Skip escaped chars
							} else {
								i++
							}
						}
					}

					if parenDepth > 0 && i < len(expr) {
						i++
					}
				}

				if parenDepth > 0 {
					return nil, fmt.Errorf("unclosed function call")
				}

				// Include the closing parenthesis
				i++

				// Add as func call token
				tokens = append(tokens, token{tokenFunctionCall, expr[funcStart:i]})
				continue
			}

			// Regular identifier
			tokens = append(tokens, token{tokenIdentifier, expr[start:i]})
			continue
		}

		// Handle strings - optimized path
		if c == '"' || c == '\'' {
			quote := c
			start := i
			i++ // Skip the opening quote

			// Fast string scanning
			for i < len(expr) && expr[i] != quote {
				if expr[i] == '\\' && i+1 < len(expr) {
					i += 2 // Skip escaped characters
				} else {
					i++
				}
			}

			if i >= len(expr) {
				return nil, fmt.Errorf("unterminated string")
			}
			i++ // Skip the closing quote
			tokens = append(tokens, token{tokenString, expr[start:i]})
			continue
		}

		// Handle parentheses
		if c == '(' {
			tokens = append(tokens, token{tokenLeftParen, "("})
			i++
			continue
		}
		if c == ')' {
			tokens = append(tokens, token{tokenRightParen, ")"})
			i++
			continue
		}

		// Handle multi-char operators (fast path using direct lookup)
		if i+1 < len(expr) {
			possibleOp := expr[i : i+2]
			if multiCharOps[possibleOp] {
				tokens = append(tokens, token{tokenOperator, possibleOp})
				i += 2
				continue
			}
		}

		// Handle single char operators
		if singleCharOps[c] {
			tokens = append(tokens, token{tokenOperator, string(c)})
			i++
			continue
		}

		// Fallback for unrecognized characters
		return nil, fmt.Errorf("unexpected character: %c at position %d", c, i)
	}

	result := make([]token, len(tokens))
	copy(result, tokens)

	// Put original back in pool
	tokenPool.Put(tokensPtr)

	return result, nil
}

// toPostfix converts infix tokens to postfix notation using the Shunting-yard
// algorithm
func toPostfix(infix []token) ([]token, error) {
	output := make([]token, 0, len(infix))
	stack := make([]token, 0, len(infix)/2)

	for _, t := range infix {
		switch t.typ {
		case tokenNumber, tokenString, tokenIdentifier, tokenFunctionCall:
			output = append(output, t)
		case tokenLeftParen:
			stack = append(stack, t)
		case tokenRightParen:
			for len(stack) > 0 && stack[len(stack)-1].typ != tokenLeftParen {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses")
			}
			// Pop the left parenthesis
			stack = stack[:len(stack)-1]
		case tokenOperator:
			for len(stack) > 0 && stack[len(stack)-1].typ == tokenOperator &&
				operators[stack[len(stack)-1].value] >= operators[t.value] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, t)
		}
	}

	for len(stack) > 0 {
		if stack[len(stack)-1].typ == tokenLeftParen {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

// evaluatePostfix evaluates a postfix expression with variable substitution
func evaluatePostfix(postfix []token, data Map) (interface{}, error) {
	// pre-alloc stack with reasonable capacity based on postfix length
	stackCapacity := len(postfix) / 2
	if stackCapacity < 4 {
		stackCapacity = 4
	}
	stack := make([]interface{}, 0, stackCapacity)

	for _, t := range postfix {
		switch t.typ {
		case tokenNumber:
			// Fast path for integers (most common)
			hasDot := false
			for i := 0; i < len(t.value); i++ {
				if t.value[i] == '.' {
					hasDot = true
					break
				}
			}

			if hasDot {
				// Parse float only when necessary
				val, err := strconv.ParseFloat(t.value, 64)
				if err != nil {
					return nil, err
				}
				stack = append(stack, val)
			} else {
				// Fast integer parsing
				val, err := strconv.Atoi(t.value)
				if err != nil {
					return nil, err
				}
				stack = append(stack, val)
			}

		case tokenString:
			// Optimized string handling
			s := t.value

			// Strings are at least 2 chars with quotes
			if len(s) >= 2 {
				s = s[1 : len(s)-1] // Remove quotes

				// Only process escape sequences if we have backslashes
				if strings.IndexByte(s, '\\') != -1 {
					s = strings.ReplaceAll(s, "\\\"", "\"")
					s = strings.ReplaceAll(s, "\\'", "'")
				}
			}

			stack = append(stack, s)

		case tokenFunctionCall:
			// Parse and execute the function call
			funcCall, err := parseFunctionCall(t.value)
			if err != nil {
				return nil, err
			}

			// Execute the function with access to all data
			result, err := funcCall.execute(data, data)
			if err != nil {
				return nil, err
			}

			stack = append(stack, result)

		case tokenIdentifier:
			// Variable lookup optimization
			val, ok := data[t.value]
			if !ok {
				// it looks like a variable
				if isLikelyVariable(t.value) {
					return nil, fmt.Errorf("%w: %s", errVariableNotFound, t.value)
				}
				// otherwise, use as string literal
				stack = append(stack, t.value)
				continue
			}

			// check only if it's a function (optimized)
			if funcVal, isFunc := val.(func(string) string); isFunc && len(stack) > 0 {
				// Apply function to top of stack if it's a string
				if argVal, isStr := stack[len(stack)-1].(string); isStr {
					stack[len(stack)-1] = funcVal(argVal)
					continue
				}
			}

			// Regular variable
			stack = append(stack, val)

		case tokenOperator:
			// Error check for stack underflow
			if len(stack) < 2 {
				return nil, fmt.Errorf("not enough operands for operator %s", t.value)
			}

			// Optimized operator application
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2] // Reduce stack

			// Apply operator (which already handles type conversion)
			result, err := applyOperator(t.value, a, b)
			if err != nil {
				return nil, err
			}

			stack = append(stack, result)
		}
	}

	// Final stack validation
	if len(stack) != 1 {
		return nil, fmt.Errorf("invalid expression: expected 1 result, got %d", len(stack))
	}

	return stack[0], nil
}

// applyOperator applies the operator to the operands with type conversions
func applyOperator(op string, a, b interface{}) (interface{}, error) {
	switch op {
	case "+":
		// Try to convert both to numbers if possible
		if isNumeric(a) && isNumeric(b) {
			va, vb := toFloat64(a), toFloat64(b)
			return va + vb, nil
		}

		aStr := toString(a)
		bStr := toString(b)
		return aStr + bStr, nil

	case "-":
		if !isNumeric(a) || !isNumeric(b) {
			return nil, fmt.Errorf("cannot subtract non-numeric values")
		}
		return toFloat64(a) - toFloat64(b), nil

	case "*":
		if !isNumeric(a) || !isNumeric(b) {
			return nil, fmt.Errorf("cannot multiply non-numeric values")
		}
		return toFloat64(a) * toFloat64(b), nil

	case "/":
		if !isNumeric(a) || !isNumeric(b) {
			return nil, fmt.Errorf("cannot divide non-numeric values")
		}
		if toFloat64(b) == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return toFloat64(a) / toFloat64(b), nil

	case "%":
		if !isNumeric(a) || !isNumeric(b) {
			return nil, fmt.Errorf("cannot perform modulo on non-numeric values")
		}
		if toFloat64(b) == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return int(toFloat64(a)) % int(toFloat64(b)), nil

	case "**":
		if !isNumeric(a) || !isNumeric(b) {
			return nil, fmt.Errorf("exponentiation requires numeric values")
		}
		return pow(toFloat64(a), toFloat64(b)), nil

	case ">":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) > toFloat64(b), nil
		}
		return toString(a) > toString(b), nil

	case "<":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) < toFloat64(b), nil
		}
		return toString(a) < toString(b), nil

	case ">=":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) >= toFloat64(b), nil
		}
		return toString(a) >= toString(b), nil

	case "<=":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) <= toFloat64(b), nil
		}
		return toString(a) <= toString(b), nil

	case "==":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) == toFloat64(b), nil
		}
		return toString(a) == toString(b), nil

	case "!=":
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) != toFloat64(b), nil
		}
		return toString(a) != toString(b), nil

	case "&&":
		return toBool(a) && toBool(b), nil

	case "||":
		return toBool(a) || toBool(b), nil

	default:
		return nil, fmt.Errorf("unsupported operator: %s", op)
	}
}

// Helper functions for type conversion

func isNumeric(v interface{}) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	}
	return false
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int8:
		return float64(val)
	case int16:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint8:
		return float64(val)
	case uint16:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case float32:
		return float64(val)
	case float64:
		return val
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f
		}
		return 0
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return toFloat64(val) != 0
	case float32, float64:
		return toFloat64(val) != 0
	case string:
		return val != "" && val != "0" && val != "false"
	default:
		return false
	}
}

func pow(a, b float64) float64 {
	result := 1.0
	for i := 0; i < int(b); i++ {
		result *= a
	}
	return result
}
