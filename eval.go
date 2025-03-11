package fasttemplate

import (
	"fmt"
	"reflect"
)

// EvalType is a type constraint that only allows numeric types, string, and
// bool
type EvalType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~string | ~bool
}

// Eval evaluates an expression directly without requiring template tags.
// It accepts a type parameter T that must be a number, string, or boolean.
//
// The expression can be a simple variable lookup, a function call, or a complex
// expression with arithmetic, comparison, and logical operators.
func Eval[T EvalType](expression string, m Map) (T, error) {
	var zero T

	// Handle function calls
	if isFunctionCall(expression) {
		fnCall, err := parseFunctionCall(expression)
		if err != nil {
			return zero, err
		}

		result, err := fnCall.execute(m, m)
		if err != nil {
			// Forward all errors from function execution
			return zero, err
		}

		return convertToType[T](result)
	}

	// Handle expressions
	if isExpression(expression) {
		result, err := evalExpression(expression, m)
		if err != nil {
			return zero, err
		}

		return convertToType[T](result)
	}

	// Handle simple variable lookup
	if val, ok := m[expression]; ok {
		return convertToType[T](val)
	}

	return zero, fmt.Errorf("%w: %s", errVariableNotFound, expression)
}

// convertToType handles converting a value to the desired type T
func convertToType[T any](val any) (T, error) {
	var zero T

	if v, ok := val.(T); ok {
		return v, nil
	}

	// Fast path for common types
	switch any(zero).(type) {
	case string:
		s := toString(val)
		return any(s).(T), nil

	case float64:
		f := toFloat64(val)
		return any(f).(T), nil

	case int:
		i := int(toFloat64(val))
		return any(i).(T), nil

	case bool:
		b := toBool(val)
		return any(b).(T), nil
	}

	targetType := reflect.TypeOf((*T)(nil)).Elem()

	// Try with reflection as a last resort
	valValue := reflect.ValueOf(val)
	if valValue.Type().ConvertibleTo(targetType) {
		convertedValue := valValue.Convert(targetType)
		return convertedValue.Interface().(T), nil
	}

	return zero, fmt.Errorf("cannot convert value of type %T to %v", val, targetType)
}
