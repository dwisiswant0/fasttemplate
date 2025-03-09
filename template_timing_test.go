package fasttemplate

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/template"
)

var (
	source = "http://{{uid}}.foo.bar.com/?cb={{cb}}{{width}}&width={{width}}&height={{height}}&timeout={{timeout}}&uid={{uid}}&subid={{subid}}&ref={{ref}}&empty={{empty}}"
	result = "http://aaasdf.foo.bar.com/?cb=12341232&width=1232&height=123&timeout=123123&uid=aaasdf&subid=asdfds&ref=http://google.com/aaa/bbb/ccc&empty="
	// resultEscaped      = "http://aaasdf.foo.bar.com/?cb=12341232&width=1232&height=123&timeout=123123&uid=aaasdf&subid=asdfds&ref=http%3A%2F%2Fgoogle.com%2Faaa%2Fbbb%2Fccc&empty="
	resultStd          = "http://aaasdf.foo.bar.com/?cb=12341232&width=1232&height=123&timeout=123123&uid=aaasdf&subid=asdfds&ref=http://google.com/aaa/bbb/ccc&empty={{empty}}"
	resultTextTemplate = "http://aaasdf.foo.bar.com/?cb=12341232&width=1232&height=123&timeout=123123&uid=aaasdf&subid=asdfds&ref=http://google.com/aaa/bbb/ccc&empty=<no value>"

	resultBytes = []byte(result)
	// resultEscapedBytes      = []byte(resultEscaped)
	resultStdBytes          = []byte(resultStd)
	resultTextTemplateBytes = []byte(resultTextTemplate)

	m = Map{
		"cb":      []byte("1234"),
		"width":   []byte("1232"),
		"height":  []byte("123"),
		"timeout": []byte("123123"),
		"uid":     []byte("aaasdf"),
		"subid":   []byte("asdfds"),
		"ref":     []byte("http://google.com/aaa/bbb/ccc"),
	}
)

// Additional variables for function benchmarks
var (
	functionTemplate       = "Hello, {{upper(\"john\")}}! Today is {{formatDate(\"2023-10-15\")}}."
	nestedFunctionTemplate = "Result: {{multiply(add(5, 10), 2)}}"
	variableArgTemplate    = "List: {{join(\", \", \"a\", \"b\", \"c\", \"d\", \"e\")}}"
	functionVarTemplate    = "{{upper(name)}}'s total: {{formatNumber(total)}}"

	functionResult       = "Hello, JOHN! Today is Monday, October 15."
	nestedFunctionResult = "Result: 30"
	variableArgResult    = "List: a, b, c, d, e"
	functionVarResult    = "ALICE's total: 42.75"

	functionData = Map{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatDate": func(date string) string {
			return "Monday, October 15"
		},
	}

	nestedFunctionData = Map{
		"add": func(a, b int) int {
			return a + b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
	}

	variableArgData = Map{
		"join": func(sep string, items ...string) string {
			return strings.Join(items, sep)
		},
	}

	functionVarData = Map{
		"name":  "alice",
		"total": 42.75,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
	}
)

func map2slice(m Map) []string {
	var a []string
	for k, v := range m {
		a = append(a, "{{"+k+"}}", string(v.([]byte)))
	}
	return a
}

func BenchmarkFmtFprintf(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var w bytes.Buffer
		for pb.Next() {
			fmt.Fprintf(&w,
				"http://%[5]s.foo.bar.com/?cb=%[1]s%[2]s&width=%[2]s&height=%[3]s&timeout=%[4]s&uid=%[5]s&subid=%[6]s&ref=%[7]s&empty=",
				m["cb"], m["width"], m["height"], m["timeout"], m["uid"], m["subid"], m["ref"])
			x := w.Bytes()
			if !bytes.Equal(x, resultBytes) {
				b.Fatalf("Unexpected result\n%q\nExpected\n%q\n", x, result)
			}
			w.Reset()
		}
	})
}

func BenchmarkStringsReplace(b *testing.B) {
	mSlice := map2slice(m)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			x := source
			for i := 0; i < len(mSlice); i += 2 {
				x = strings.Replace(x, mSlice[i], mSlice[i+1], -1)
			}
			if x != resultStd {
				b.Fatalf("Unexpected result\n%q\nExpected\n%q\n", x, resultStd)
			}
		}
	})
}

func BenchmarkStringsReplacer(b *testing.B) {
	mSlice := map2slice(m)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r := strings.NewReplacer(mSlice...)
			x := r.Replace(source)
			if x != resultStd {
				b.Fatalf("Unexpected result\n%q\nExpected\n%q\n", x, resultStd)
			}
		}
	})
}

func BenchmarkTextTemplate(b *testing.B) {
	s := strings.Replace(source, "{{", "{{.", -1)
	t, err := template.New("test").Parse(s)
	if err != nil {
		b.Fatalf("Error when parsing template: %s", err)
	}

	mm := make(map[string]string)
	for k, v := range m {
		mm[k] = string(v.([]byte))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var w bytes.Buffer
		for pb.Next() {
			if err := t.Execute(&w, mm); err != nil {
				b.Fatalf("error when executing template: %s", err)
			}
			x := w.Bytes()
			if !bytes.Equal(x, resultTextTemplateBytes) {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, resultTextTemplateBytes)
			}
			w.Reset()
		}
	})
}

func BenchmarkFastTemplateExecuteFunc(b *testing.B) {
	t, err := NewTemplate(source, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	// Redefine testTagFunc as a map
	tagHandlers := make(Map)
	for k, v := range m {
		value := v // Capture the value
		tagHandlers[k] = func(w io.Writer, tag string) (int, error) {
			return w.Write(value.([]byte))
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var w bytes.Buffer
		for pb.Next() {
			if _, err := t.Execute(&w, tagHandlers); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			x := w.Bytes()
			if !bytes.Equal(x, resultBytes) {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, resultBytes)
			}
			w.Reset()
		}
	})
}

func BenchmarkFastTemplateExecute(b *testing.B) {
	t, err := NewTemplate(source, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var w bytes.Buffer
		for pb.Next() {
			if _, err := t.Execute(&w, m); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			x := w.Bytes()
			if !bytes.Equal(x, resultBytes) {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, resultBytes)
			}
			w.Reset()
		}
	})
}

func BenchmarkFastTemplateExecuteStd(b *testing.B) {
	t, err := NewTemplate(source, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var w bytes.Buffer
		for pb.Next() {
			if _, err := t.ExecuteStd(&w, m); err != nil {
				b.Fatalf("unexpected error: %s", err)
			}
			x := w.Bytes()
			if !bytes.Equal(x, resultStdBytes) {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, resultStdBytes)
			}
			w.Reset()
		}
	})
}

func BenchmarkFastTemplateExecuteString(b *testing.B) {
	t, err := NewTemplate(source, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			x := t.ExecuteString(m)
			if x != result {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, result)
			}
		}
	})
}

func BenchmarkFastTemplateExecuteStringStd(b *testing.B) {
	t, err := NewTemplate(source, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			x := t.ExecuteStringStd(m)
			if x != resultStd {
				b.Fatalf("unexpected result\n%q\nExpected\n%q\n", x, resultStd)
			}
		}
	})
}

func BenchmarkNewTemplate(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New(source, "{{", "}}")
		}
	})
}

func BenchmarkTemplateReset(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		t := New(source, "{{", "}}")
		for pb.Next() {
			t.Reset(source, "{{", "}}")
		}
	})
}

// func BenchmarkTemplateResetExecuteFunc(b *testing.B) {
// 	b.RunParallel(func(pb *testing.PB) {
// 		t := New(source, "{{", "}}")
// 		var w bytes.Buffer
// 		for pb.Next() {
// 			t.Reset(source, "{{", "}}")
// 			t.ExecuteFunc(&w, testTagFunc)
// 			w.Reset()
// 		}
// 	})
// }

func BenchmarkExecuteFunc(b *testing.B) {
	// Redefine testTagFunc as a map
	tagHandlers := make(Map)
	for k, v := range m {
		value := v // Capture the value
		tagHandlers[k] = func(w io.Writer, tag string) (int, error) {
			return w.Write(value.([]byte))
		}
	}

	b.RunParallel(func(pb *testing.PB) {
		var bb bytes.Buffer
		for pb.Next() {
			Execute(source, "{{", "}}", &bb, tagHandlers)
			bb.Reset()
		}
	})
}

func BenchmarkFastTemplateFunctionCall(b *testing.B) {
	t, err := NewTemplate(functionTemplate, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := t.ExecuteString(functionData)
			if result != functionResult {
				b.Fatalf("unexpected result: %q", result)
			}
		}
	})
}

func BenchmarkFastTemplateNestedFunctions(b *testing.B) {
	t, err := NewTemplate(nestedFunctionTemplate, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := t.ExecuteString(nestedFunctionData)
			if result != nestedFunctionResult {
				b.Fatalf("unexpected result: %q", result)
			}
		}
	})
}

func BenchmarkFastTemplateVariableArguments(b *testing.B) {
	t, err := NewTemplate(variableArgTemplate, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := t.ExecuteString(variableArgData)
			if result != variableArgResult {
				b.Fatalf("unexpected result: %q", result)
			}
		}
	})
}

func BenchmarkFastTemplateFunctionWithVariables(b *testing.B) {
	t, err := NewTemplate(functionVarTemplate, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := t.ExecuteString(functionVarData)
			if result != functionVarResult {
				b.Fatalf("unexpected result: %q", result)
			}
		}
	})
}

func BenchmarkFastTemplateComplexFunctions(b *testing.B) {
	// A more complex template with multiple function calls
	complexTemplate := `{{join(" | ", upper(name), 
                              formatNumber(total), 
                              choose(premium, "VIP", "Regular"),
                              concat("ID:", id))}}
                       Has access: {{formatBool(hasAccess)}}`

	expectedResult := "ALICE | 42.75 | VIP | ID:XYZ123 Has access: yes"

	t, err := NewTemplate(complexTemplate, "{{", "}}")
	if err != nil {
		b.Fatalf("error in template: %s", err)
	}

	complexData := Map{
		"name":      "alice",
		"total":     42.75,
		"premium":   true,
		"id":        "XYZ123",
		"hasAccess": true,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"formatNumber": func(n float64) string {
			return fmt.Sprintf("%.2f", n)
		},
		"choose": func(condition bool, trueVal, falseVal string) string {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"concat": func(parts ...string) string {
			return strings.Join(parts, "")
		},
		"join": func(sep string, parts ...string) string {
			return strings.Join(parts, sep)
		},
		"formatBool": func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := t.ExecuteString(complexData)
			// Clean up whitespace for comparison
			result = strings.Join(strings.Fields(result), " ")
			if result != expectedResult {
				b.Fatalf("unexpected result: %q, expected: %q", result, expectedResult)
			}
		}
	})
}

func testTagFunc(w io.Writer, tag string) (int, error) {
	if t, ok := m[tag]; ok {
		return w.Write(t.([]byte))
	}
	return 0, nil
}
