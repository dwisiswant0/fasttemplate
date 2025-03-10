// Package fasttemplate implements simple and fast template library.
//
// Fasttemplate is faster than text/template, strings.Replace
// and strings.Replacer.
//
// Fasttemplate ideally fits for fast and simple placeholders' substitutions,
// function calls, and expression evaluation.
package fasttemplate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/valyala/bytebufferpool"
)

// Execute substitutes template tags (placeholders) with the corresponding
// values from the map m and writes the result to the given writer w.
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// Returns the number of bytes written to w.
//
// This function is optimized for constantly changing templates.
// Use Template.Execute for frozen templates. For validating templates, use
// the [Validate] function.
func Execute(template, startTag, endTag string, w io.Writer, m Map) (int64, error) {
	s := unsafeString2Bytes(template)
	a := unsafeString2Bytes(startTag)
	b := unsafeString2Bytes(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		tag := unsafeBytes2String(s[:n])
		ni, err = processTag(w, tag, m)
		nn += int64(ni)
		if err != nil {
			// Always propagate func call errors, but maintain backward
			// compatibility for simple variable errors
			if isFunctionCall(tag) || !errors.Is(err, errVariableNotFound) {
				return nn, err
			}
			// for simple variable not found, ignore for backward compatibility
		}
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
}

// ExecuteStd works the same way as Execute, but keeps the unknown placeholders.
// This can be used as a drop-in replacement for strings.Replacer
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// Returns the number of bytes written to w.
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteStd for frozen templates. For validating templates, use
// the [Validate] function.
func ExecuteStd(template, startTag, endTag string, w io.Writer, m Map) (int64, error) {
	s := unsafeString2Bytes(template)
	a := unsafeString2Bytes(startTag)
	b := unsafeString2Bytes(endTag)

	var nn int64
	var ni int
	var err error
	for {
		n := bytes.Index(s, a)
		if n < 0 {
			break
		}
		ni, err = w.Write(s[:n])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			// cannot find end tag - just write it to the output.
			ni, _ = w.Write(a)
			nn += int64(ni)
			break
		}

		tag := unsafeBytes2String(s[:n])
		ni, err = processTagStd(w, tag, startTag, endTag, m)
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
		s = s[n+len(b):]
	}
	ni, err = w.Write(s)
	nn += int64(ni)

	return nn, err
}

// Validate checks if all tags in the template can be resolved by the provided
// [Map].
//
// It returns nil if all tags are resolvable, otherwise it returns an error with
// details about the first unresolved tag found.
//
// This function creates a temporary template and might be less efficient than
// creating a [Template] instance and using its [Validate] method.
func Validate(template, startTag, endTag string, m Map) error {
	t, err := NewTemplate(template, startTag, endTag)
	if err != nil {
		return err
	}
	return t.Validate(m)
}

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteString for frozen templates. For validating templates,
// create a [Template] instance and use the [Validate] method before execution.
func ExecuteString(template, startTag, endTag string, m Map) string {
	var bb bytes.Buffer
	Execute(template, startTag, endTag, &bb, m)
	return bb.String()
}

// ExecuteStringStd works the same way as ExecuteString, but keeps the unknown placeholders.
// This can be used as a drop-in replacement for strings.Replacer
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// This function is optimized for constantly changing templates.
// Use Template.ExecuteStringStd for frozen templates. For validating templates,
// create a [Template] instance and use the [Validate] method before execution.
func ExecuteStringStd(template, startTag, endTag string, m Map) string {
	var bb bytes.Buffer
	ExecuteStd(template, startTag, endTag, &bb, m)
	return bb.String()
}

// var byteBufferPool bytebufferpool.Pool

// Template implements simple template engine, which can be used for fast
// tags' (aka placeholders) substitution.
type Template struct {
	template string
	startTag string
	endTag   string

	texts          [][]byte
	tags           []string
	byteBufferPool bytebufferpool.Pool
}

// New parses the given template using the given startTag and endTag
// as tag start and tag end.
//
// The returned template can be executed by concurrently running goroutines
// using Execute* methods.
//
// New panics if the given template cannot be parsed. Use NewTemplate instead
// if template may contain errors.
func New(template, startTag, endTag string) *Template {
	t, err := NewTemplate(template, startTag, endTag)
	if err != nil {
		panic(err)
	}
	return t
}

// NewTemplate parses the given template using the given startTag and endTag
// as tag start and tag end.
//
// The returned template can be executed by concurrently running goroutines
// using Execute* methods.
func NewTemplate(template, startTag, endTag string) (*Template, error) {
	var t Template
	err := t.Reset(template, startTag, endTag)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Reset resets the template t to new one defined by
// template, startTag and endTag.
//
// Reset allows Template object re-use.
//
// Reset may be called only if no other goroutines call t methods at the moment.
func (t *Template) Reset(template, startTag, endTag string) error {
	// Keep these vars in t, so GC won't collect them and won't break
	// vars derived via unsafe*
	t.template = template
	t.startTag = startTag
	t.endTag = endTag
	t.texts = t.texts[:0]
	t.tags = t.tags[:0]

	if len(startTag) == 0 {
		panic("startTag cannot be empty")
	}
	if len(endTag) == 0 {
		panic("endTag cannot be empty")
	}

	s := unsafeString2Bytes(template)
	a := unsafeString2Bytes(startTag)
	b := unsafeString2Bytes(endTag)

	tagsCount := bytes.Count(s, a)
	if tagsCount == 0 {
		return nil
	}

	if tagsCount+1 > cap(t.texts) {
		t.texts = make([][]byte, 0, tagsCount+1)
	}
	if tagsCount > cap(t.tags) {
		t.tags = make([]string, 0, tagsCount)
	}

	for {
		n := bytes.Index(s, a)
		if n < 0 {
			t.texts = append(t.texts, s)
			break
		}
		t.texts = append(t.texts, s[:n])

		s = s[n+len(a):]
		n = bytes.Index(s, b)
		if n < 0 {
			return fmt.Errorf("cannot find end tag=%q in the template=%q starting from %q", endTag, template, s)
		}

		t.tags = append(t.tags, unsafeBytes2String(s[:n]))
		s = s[n+len(b):]
	}

	return nil
}

// Execute substitutes template tags (placeholders) with the corresponding
// values from the map m and writes the result to the given writer w.
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// Returns the number of bytes written to w.
//
// Note: It is advised to call [Validate] before Execute to ensure all tags can
// be resolved or use ExecuteStd if you want to keep the unknown placeholders.
func (t *Template) Execute(w io.Writer, m Map) (int64, error) {
	var nn int64

	n := len(t.texts) - 1
	if n == -1 {
		ni, err := w.Write(unsafeString2Bytes(t.template))
		return int64(ni), err
	}

	for i := 0; i < n; i++ {
		ni, err := w.Write(t.texts[i])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		ni, err = processTag(w, t.tags[i], m)
		nn += int64(ni)
		// Special handling for errors:
		// - For function calls, propagate all errors
		// - For variables, only propagate non-"variable not found" errors
		//   (backward compatibility)
		if err != nil {
			if isFunctionCall(t.tags[i]) || !errors.Is(err, errVariableNotFound) {
				return nn, err
			}
			// for simple variable not found, ignore for backward compatibility
		}
	}
	ni, err := w.Write(t.texts[n])
	nn += int64(ni)
	return nn, err
}

// ExecuteStd works the same way as Execute, but keeps the unknown placeholders.
// This can be used as a drop-in replacement for strings.Replacer
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// Returns the number of bytes written to w.
//
// Note it doesn't return errors from function calls - it preserves the original
// tag text instead.
//
// Note: It is advised to call [Validate] before ExecuteStd if you want to
// ensure all tags can be resolved.
func (t *Template) ExecuteStd(w io.Writer, m Map) (int64, error) {
	var nn int64

	n := len(t.texts) - 1
	if n == -1 {
		ni, err := w.Write(unsafeString2Bytes(t.template))
		return int64(ni), err
	}

	for i := 0; i < n; i++ {
		ni, err := w.Write(t.texts[i])
		nn += int64(ni)
		if err != nil {
			return nn, err
		}

		ni, err = processTagStd(w, t.tags[i], t.startTag, t.endTag, m)
		nn += int64(ni)
		if err != nil {
			return nn, err
		}
	}
	ni, err := w.Write(t.texts[n])
	nn += int64(ni)
	return nn, err
}

// ExecuteString substitutes template tags (placeholders) with the corresponding
// values from the map m and returns the result.
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// This function is optimized for frozen templates.
// Use ExecuteString for constantly changing templates.
//
// Note: It is advised to call [Validate] before ExecuteString to ensure all
// tags can be resolved or use ExecuteStringStd if you want to keep the unknown
// placeholders.
func (t *Template) ExecuteString(m Map) string {
	bb := t.byteBufferPool.Get()
	t.Execute(bb, m)
	s := bb.String()
	bb.Reset()
	t.byteBufferPool.Put(bb)
	return s
}

// ExecuteStringStd works the same way as ExecuteString, but keeps the unknown placeholders.
// This can be used as a drop-in replacement for strings.Replacer
//
// Substitution map m may contain values with the following types:
//   - []byte - the fastest value type
//   - string - convenient value type
//   - functions - for complex value generation and transformations
//
// This function is optimized for frozen templates.
// Use ExecuteStringStd for constantly changing templates.
//
// Note: It is advised to call [Validate] before ExecuteStringStd if you want to
// ensure all tags can be resolved.
func (t *Template) ExecuteStringStd(m Map) string {
	bb := t.byteBufferPool.Get()
	t.ExecuteStd(bb, m)
	s := bb.String()
	bb.Reset()
	t.byteBufferPool.Put(bb)
	return s
}

// Validate checks if all tags in the template can be resolved by the provided
// [Map].
//
// It returns nil if all tags are resolvable, otherwise it returns an error with
// details about the first unresolved tag found.
func (t *Template) Validate(m Map) error {
	if m == nil {
		// If no map is provided, return error for any tags
		if len(t.tags) > 0 {
			return fmt.Errorf("unresolved tag %q: nil map provided", t.tags[0])
		}
		return nil
	}

	for _, tag := range t.tags {
		if isFunctionCall(tag) {
			funcCall, err := parseFunctionCall(tag)
			if err != nil {
				return fmt.Errorf("invalid function call %q: %w", tag, err)
			}

			funcExists := false
			for k, v := range m {
				if v != nil && reflect.TypeOf(v).Kind() == reflect.Func && k == funcCall.Name {
					funcExists = true
					break
				}
			}

			if !funcExists {
				return fmt.Errorf("unresolved function %q in tag %q", funcCall.Name, tag)
			}

			// We don't validate function args here as they could be vars
			// that will be resolved during execution
			continue
		}

		// check for expressions with operators
		if isExpression(tag) {
			// We don't validate expressions in detail as vars within
			// expressions will be resolved during execution later
			continue
		}

		// check if regular tag exists in map
		if _, ok := m[tag]; !ok {
			return fmt.Errorf("unresolved tag %q", tag)
		}
	}

	return nil
}

// Helper functions to process tags

func processTag(w io.Writer, tag string, m Map) (int, error) {
	if isFunctionCall(tag) {
		funcCall, err := parseFunctionCall(tag)
		if err != nil {
			return 0, fmt.Errorf("error parsing function call %q: %w", tag, err)
		}

		if m != nil {
			tempFuncs := Map{}

			// scan for all funcs in the `m` map
			for k, v := range m {
				if v != nil && reflect.TypeOf(v).Kind() == reflect.Func {
					tempFuncs[k] = v
				}
			}

			// check if we have the func being called
			if fn, ok := tempFuncs[funcCall.Name]; ok {
				fnType := reflect.TypeOf(fn)
				if !isValidArgCount(fnType, len(funcCall.Args)) {
					return 0, fmt.Errorf("invalid argument count for function %q", funcCall.Name)
				}

				// exec the func with access to all funcs for nested calls
				result, err := funcCall.execute(tempFuncs, m)
				if err != nil {
					return 0, err // Propagate the error from the function call
				}
				if result != nil {
					switch v := result.(type) {
					case []byte:
						return w.Write(v)
					case string:
						return w.Write(unsafeString2Bytes(v))
					default:
						return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
					}
				}
				return 0, nil
			}
			// Function not found, return a specific error
			return 0, fmt.Errorf("function not found: %s", funcCall.Name)
		}
		// Missing map, return an error
		return 0, fmt.Errorf("no functions map provided for function call: %s", tag)
	}

	// Check if this is an expr with operators
	if isExpression(tag) {
		result, err := evalExpression(tag, m)
		if err != nil {
			return 0, err
		}
		if result != nil {
			switch v := result.(type) {
			case []byte:
				return w.Write(v)
			case string:
				return w.Write(unsafeString2Bytes(v))
			default:
				return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
			}
		}
		return 0, nil
	}

	v, ok := m[tag]
	if !ok {
		return 0, fmt.Errorf("%w: %s", errVariableNotFound, tag)
	}
	if v == nil {
		return 0, nil
	}
	switch value := v.(type) {
	case []byte:
		return w.Write(value)
	case string:
		return w.Write(unsafeString2Bytes(value))
	case func(io.Writer, string) (int, error):
		// Maintain compatibility with existing code that uses TagFunc
		return value(w, tag)
	default:
		// Convert numeric types and other values to string
		return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
	}
}

func processTagStd(w io.Writer, tag, startTag, endTag string, m Map) (int, error) {
	// First check if this is a function call
	if isFunctionCall(tag) {
		funcCall, err := parseFunctionCall(tag)
		if err != nil {
			// Preserve the original tag if there's a parsing error
			if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
				return 0, err
			}
			return len(startTag) + len(tag) + len(endTag), nil
		}

		if m != nil {
			tempFuncs := Map{}

			// scan for all funcs in the `m` map
			for k, v := range m {
				if v != nil && reflect.TypeOf(v).Kind() == reflect.Func {
					tempFuncs[k] = v
				}
			}

			// check if we have the func being called
			if fn, ok := tempFuncs[funcCall.Name]; ok {
				fnType := reflect.TypeOf(fn)
				if !isValidArgCount(fnType, len(funcCall.Args)) {
					// Preserve the original tag if the argument count is incorrect
					if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
						return 0, err
					}
					return len(startTag) + len(tag) + len(endTag), nil
				}

				// exec the func with access to all funcs for nested calls
				result, err := funcCall.execute(tempFuncs, m)
				if err != nil {
					// for func errors, preserve the original tag
					if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
						return 0, err
					}
					return len(startTag) + len(tag) + len(endTag), nil
				}

				if result != nil {
					switch v := result.(type) {
					case []byte:
						return w.Write(v)
					case string:
						return w.Write(unsafeString2Bytes(v))
					default:
						return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
					}
				}
				return 0, nil
			}
		}

		// If we get here, this is an unknown func call - keep the tag
		if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
			return 0, err
		}
		return len(startTag) + len(tag) + len(endTag), nil
	}

	// Check if this is an expression with operators
	if isExpression(tag) {
		result, err := evalExpression(tag, m)
		if err != nil {
			// for expression errors, preserve the original tag
			if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
				return 0, err
			}
			return len(startTag) + len(tag) + len(endTag), nil
		}
		if result != nil {
			switch v := result.(type) {
			case []byte:
				return w.Write(v)
			case string:
				return w.Write(unsafeString2Bytes(v))
			default:
				return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
			}
		}
		return 0, nil
	}

	// Handle normal tags (not func calls or expressions)
	v, ok := m[tag]
	if !ok {
		if _, err := preserveTag(w, tag, startTag, endTag); err != nil {
			return 0, err
		}
		return len(startTag) + len(tag) + len(endTag), nil
	}
	if v == nil {
		return 0, nil
	}
	switch value := v.(type) {
	case []byte:
		return w.Write(value)
	case string:
		return w.Write(unsafeString2Bytes(value))
	case func(io.Writer, string) (int, error):
		// Maintain compatibility with existing code that uses TagFunc
		return value(w, tag)
	default:
		// Convert numeric types and other values to string
		return w.Write(unsafeString2Bytes(fmt.Sprintf("%v", v)))
	}
}

// Helper function to check if the argument count is valid for a func
func isValidArgCount(fnType reflect.Type, argCount int) bool {
	if fnType.IsVariadic() {
		// For variadic funcs, the number of non-variadic arguments must match
		return argCount >= fnType.NumIn()-1
	}
	// For non-variadic funcs, the argument count must match exactly
	return fnType.NumIn() == argCount
}

func preserveTag(w io.Writer, tag, startTag, endTag string) (int, error) {
	if _, err := w.Write(unsafeString2Bytes(startTag)); err != nil {
		return 0, err
	}
	if _, err := w.Write(unsafeString2Bytes(tag)); err != nil {
		return 0, err
	}
	return w.Write(unsafeString2Bytes(endTag))
}
