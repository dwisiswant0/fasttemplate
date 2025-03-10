package fasttemplate

import "errors"

var (
	errVariableNotFound = errors.New("variable not found")
	errFunctionNotFound = errors.New("function not found")
)
