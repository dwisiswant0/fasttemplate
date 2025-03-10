package fasttemplate

import (
	"strings"
	"testing"
)

func TestEval_Variables(t *testing.T) {
	data := Map{
		"strVal":   "hello",
		"intVal":   42,
		"floatVal": 3.14,
		"boolVal":  true,
	}

	// Test string variable
	strResult, err := Eval[string]("strVal", data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if strResult != "hello" {
		t.Errorf("Expected 'hello', got %v", strResult)
	}

	// Test int variable
	intResult, err := Eval[int]("intVal", data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if intResult != 42 {
		t.Errorf("Expected 42, got %v", intResult)
	}

	// Test float variable
	floatResult, err := Eval[float64]("floatVal", data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if floatResult != 3.14 {
		t.Errorf("Expected 3.14, got %v", floatResult)
	}

	// Test bool variable
	boolResult, err := Eval[bool]("boolVal", data)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !boolResult {
		t.Errorf("Expected true, got %v", boolResult)
	}

	// Test non-existent variable
	_, err = Eval[string]("nonExistent", data)
	if err == nil {
		t.Error("Expected error for non-existent variable, but got nil")
	}
}

func TestEval_Expressions(t *testing.T) {
	data := Map{
		"a": 10,
		"b": 5,
		"c": "hello",
		"d": true,
	}

	// Test addition
	sumResult, err := Eval[int]("a + b", data)
	if err != nil {
		t.Errorf("Addition test failed: %v", err)
	}
	if sumResult != 15 {
		t.Errorf("Expected 15, got %v", sumResult)
	}

	// Test subtraction
	subResult, err := Eval[int]("a - b", data)
	if err != nil {
		t.Errorf("Subtraction test failed: %v", err)
	}
	if subResult != 5 {
		t.Errorf("Expected 5, got %v", subResult)
	}

	// Test multiplication
	mulResult, err := Eval[int]("a * b", data)
	if err != nil {
		t.Errorf("Multiplication test failed: %v", err)
	}
	if mulResult != 50 {
		t.Errorf("Expected 50, got %v", mulResult)
	}

	// Test division
	divResult, err := Eval[float64]("a / b", data)
	if err != nil {
		t.Errorf("Division test failed: %v", err)
	}
	if divResult != 2 {
		t.Errorf("Expected 2, got %v", divResult)
	}

	// Test comparison
	compResult, err := Eval[bool]("a > b", data)
	if err != nil {
		t.Errorf("Comparison test failed: %v", err)
	}
	if !compResult {
		t.Errorf("Expected true, got %v", compResult)
	}

	// Test logical operators
	logicResult, err := Eval[bool]("a > b && d", data)
	if err != nil {
		t.Errorf("Logical operator test failed: %v", err)
	}
	if !logicResult {
		t.Errorf("Expected true, got %v", logicResult)
	}

	// Test complex expression
	complexResult, err := Eval[float64]("(a + b) * 2 / 5", data)
	if err != nil {
		t.Errorf("Complex expression test failed: %v", err)
	}
	if complexResult != 6 {
		t.Errorf("Expected 6, got %v", complexResult)
	}

	// Test string concatenation
	concatResult, err := Eval[string]("c + ' world'", data)
	if err != nil {
		t.Errorf("String concatenation test failed: %v", err)
	}
	if concatResult != "hello world" {
		t.Errorf("Expected 'hello world', got %v", concatResult)
	}
}

func TestEval_Functions(t *testing.T) {
	data := Map{
		"name": "john",
		"num1": 10,
		"num2": 5,
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"mult": func(a, b int) int {
			return a * b
		},
	}

	// Test simple function call
	upperResult, err := Eval[string]("upper(name)", data)
	if err != nil {
		t.Errorf("Function call test failed: %v", err)
	}
	if upperResult != "JOHN" {
		t.Errorf("Expected 'JOHN', got %v", upperResult)
	}

	// Test function with variable args
	addResult, err := Eval[int]("add(num1, num2)", data)
	if err != nil {
		t.Errorf("Function with variable args test failed: %v", err)
	}
	if addResult != 15 {
		t.Errorf("Expected 15, got %v", addResult)
	}

	// Test function with literal args
	multResult, err := Eval[int]("mult(6, 7)", data)
	if err != nil {
		t.Errorf("Function with literal args test failed: %v", err)
	}
	if multResult != 42 {
		t.Errorf("Expected 42, got %v", multResult)
	}

	// Test nested function calls
	nestedResult, err := Eval[int]("add(mult(2, 3), num2)", data)
	if err != nil {
		t.Errorf("Nested function call test failed: %v", err)
	}
	if nestedResult != 11 {
		t.Errorf("Expected 11, got %v", nestedResult)
	}

	// Test non-existent function
	_, err = Eval[string]("nonExistentFunc(name)", data)
	if err == nil {
		t.Error("Expected error for non-existent function, but got nil")
	}
}

func TestEval_TypeConversion(t *testing.T) {
	data := Map{
		"intVal":    42,
		"floatVal":  3.14,
		"strVal":    "123",
		"boolVal":   true,
		"emptyStr":  "",
		"zeroInt":   0,
		"falseVal":  false,
		"trueAsStr": "true",
	}

	// Test int to float
	floatFromInt, err := Eval[float64]("intVal", data)
	if err != nil {
		t.Errorf("Int to float conversion failed: %v", err)
	}
	if floatFromInt != 42.0 {
		t.Errorf("Expected 42.0, got %v", floatFromInt)
	}

	// Test float to int (should truncate)
	intFromFloat, err := Eval[int]("floatVal", data)
	if err != nil {
		t.Errorf("Float to int conversion failed: %v", err)
	}
	if intFromFloat != 3 {
		t.Errorf("Expected 3, got %v", intFromFloat)
	}

	// Test string to int
	intFromStr, err := Eval[int]("strVal", data)
	if err != nil {
		t.Errorf("String to int conversion failed: %v", err)
	}
	if intFromStr != 123 {
		t.Errorf("Expected 123, got %v", intFromStr)
	}

	// Test bool to int
	intFromBool, err := Eval[int]("boolVal", data)
	if err != nil {
		t.Errorf("Bool to int conversion failed: %v", err)
	}
	if intFromBool != 1 {
		t.Errorf("Expected 1, got %v", intFromBool)
	}

	// Test int to bool
	boolFromInt, err := Eval[bool]("intVal", data)
	if err != nil {
		t.Errorf("Int to bool conversion failed: %v", err)
	}
	if !boolFromInt {
		t.Errorf("Expected true, got %v", boolFromInt)
	}

	// Test zero int to bool
	boolFromZero, err := Eval[bool]("zeroInt", data)
	if err != nil {
		t.Errorf("Zero int to bool conversion failed: %v", err)
	}
	if boolFromZero {
		t.Errorf("Expected false, got %v", boolFromZero)
	}

	// Test string to bool
	boolFromStr, err := Eval[bool]("trueAsStr", data)
	if err != nil {
		t.Errorf("String to bool conversion failed: %v", err)
	}
	if !boolFromStr {
		t.Errorf("Expected true, got %v", boolFromStr)
	}
}
