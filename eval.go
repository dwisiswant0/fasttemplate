package fasttemplate

import (
	"fmt"
	"reflect"
	"strconv"
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

	targetType := reflect.TypeOf((*T)(nil)).Elem()

	switch targetType.Kind() {
	case reflect.String:
		// Convert to string
		s := toString(val)
		return any(s).(T), nil // no need to convertible check
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Convert to int
		var i int64
		switch v := val.(type) {
		case int:
			i = int64(v)
		case int8:
			i = int64(v)
		case int16:
			i = int64(v)
		case int32:
			i = int64(v)
		case int64:
			i = v
		case uint:
			i = int64(v)
		case uint8:
			i = int64(v)
		case uint16:
			i = int64(v)
		case uint32:
			i = int64(v)
		case uint64:
			i = int64(v)
		case float32:
			i = int64(v)
		case float64:
			i = int64(v)
		case string:
			var err error
			i, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return zero, fmt.Errorf("cannot convert string %q to int", v)
			}
		case bool:
			if v {
				i = 1
			} else {
				i = 0
			}
		default:
			return zero, fmt.Errorf("cannot convert %T to int", val)
		}
		v := reflect.ValueOf(i)
		if v.Type().ConvertibleTo(targetType) {
			return v.Convert(targetType).Interface().(T), nil
		}
	case reflect.Float32, reflect.Float64:
		// Convert to float
		var f float64
		switch v := val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			f = toFloat64(v)
		case float32:
			f = float64(v)
		case float64:
			f = v
		case string:
			var err error
			f, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return zero, fmt.Errorf("cannot convert string %q to float", v)
			}
		case bool:
			if v {
				f = 1.0
			} else {
				f = 0.0
			}
		default:
			return zero, fmt.Errorf("cannot convert %T to float", val)
		}
		v := reflect.ValueOf(f)
		if v.Type().ConvertibleTo(targetType) {
			return v.Convert(targetType).Interface().(T), nil
		}
	case reflect.Bool:
		// Convert to bool
		b := toBool(val)
		v := reflect.ValueOf(b)
		if v.Type().ConvertibleTo(targetType) {
			return v.Convert(targetType).Interface().(T), nil
		}
	}

	// Try with reflection as a last resort
	valValue := reflect.ValueOf(val)
	if valValue.Type().ConvertibleTo(targetType) {
		convertedValue := valValue.Convert(targetType)
		return convertedValue.Interface().(T), nil
	}

	return zero, fmt.Errorf("cannot convert value of type %T to %v", val, targetType)
}
