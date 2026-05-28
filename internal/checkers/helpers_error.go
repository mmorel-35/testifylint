package checkers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var (
	errorObj   = types.Universe.Lookup("error")
	errorType  = errorObj.Type()
	errorIface = errorType.Underlying().(*types.Interface)
)

func isError(pass *analysis.Pass, expr ast.Expr) bool {
	return pass.TypesInfo.TypeOf(expr) == errorType
}

func isErrorsIsCall(pass *analysis.Pass, ce *ast.CallExpr) bool {
	return isPkgFnCall(pass, ce, "errors", "Is")
}

func isErrorsAsCall(pass *analysis.Pass, ce *ast.CallExpr) bool {
	return isPkgFnCall(pass, ce, "errors", "As")
}

// isErrErrorCall returns the receiver expression if e is a method call of the form
// `receiver.Error()` where receiver implements the error interface.
func isErrErrorCall(pass *analysis.Pass, e ast.Expr) (ast.Expr, bool) {
	ce, ok := e.(*ast.CallExpr)
	if !ok || len(ce.Args) != 0 {
		return nil, false
	}

	se, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || !isIdentWithName("Error", se.Sel) {
		return nil, false
	}

	if !implementsErrorIface(pass, se.X) {
		return nil, false
	}
	return se.X, true
}

// implementsErrorIface returns true if the expression's type implements the error interface.
func implementsErrorIface(pass *analysis.Pass, e ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(e)
	if t == nil {
		return false
	}
	return types.Implements(t, errorIface) || types.Implements(types.NewPointer(t), errorIface)
}
