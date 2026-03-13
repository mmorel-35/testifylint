package checkers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Regexp detects situations like
//
//	assert.Regexp(t, regexp.MustCompile(`\[.*\] DEBUG \(.*TestNew.*\): message`), out)
//	assert.NotRegexp(t, regexp.MustCompile(`\[.*\] TRACE message`), out)
//
// and requires
//
//	assert.Regexp(t, `\[.*\] DEBUG \(.*TestNew.*\): message`, out)
//	assert.NotRegexp(t, `\[.*\] TRACE message`, out)
//
// Also detects situations like
//
//	assert.Regexp(t, []byte(`\w+`), str)
//
// and requires
//
//	assert.Regexp(t, `\w+`, str) // or *regexp.Regexp
type Regexp struct{}

// NewRegexp constructs Regexp checker.
func NewRegexp() Regexp     { return Regexp{} }
func (Regexp) Name() string { return "regexp" }

func (checker Regexp) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	switch call.Fn.NameFTrimmed {
	default:
		return nil
	case "Regexp", "NotRegexp":
	}

	if len(call.Args) < 1 {
		return nil
	}

	arg := call.Args[0]

	ce, ok := arg.(*ast.CallExpr)
	if ok && len(ce.Args) == 1 && isRegexpMustCompileCall(pass, ce) {
		return newRemoveMustCompileDiagnostic(pass, checker.Name(), call, ce, ce.Args[0])
	}

	if !isStringOrRegexpType(pass, arg) {
		return newDiagnostic(checker.Name(), call,
			"use string or *regexp.Regexp as the first argument")
	}
	return nil
}

// isStringOrRegexpType returns true if the expression is of a type whose underlying type is string
// (including untyped strings, defined string types, and string-based type aliases), *regexp.Regexp
// type (including type aliases), or a type parameter (accepted conservatively to avoid false
// positives in generic code).
func isStringOrRegexpType(pass *analysis.Pass, e ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(e)
	if t == nil {
		return false
	}

	// Conservatively accept type parameters to avoid false positives in generic code.
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return true
	}

	// Check for string types (includes string, untyped string, and string-underlying type aliases).
	if isUnderlying(pass, e, types.IsString) {
		return true
	}

	// Check for *regexp.Regexp (using Unalias on both the type and pointer element to handle
	// type aliases such as `type MyRx = *regexp.Regexp` or `type MyRegexp = regexp.Regexp`).
	ptr, ok := types.Unalias(t).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "regexp" && obj.Name() == "Regexp"
}
