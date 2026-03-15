package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

// FailNow detects situations like
//
// assert.Fail(t, "msg")
// assert.Fail(t, "msg", args...)
// assert.Failf(t, "failure", "format %s", arg)
// assert.FailNow(t, "msg")
// assert.FailNow(t, "msg", args...)
// assert.FailNowf(t, "failure", "format %s", arg)
//
// and requires
//
//	t.Error("msg") / t.Errorf("format %s", arg)
//	t.Fatal("msg") / t.Fatalf("format %s", arg)
type FailNow struct{}

// NewFailNow constructs FailNow checker.
func NewFailNow() FailNow    { return FailNow{} }
func (FailNow) Name() string { return "fail-now" }

func (checker FailNow) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	switch call.Fn.NameFTrimmed {
	case "Fail":
		return newDiagnostic(checker.Name(), call, "use t.Error or t.Errorf instead",
			checker.fix(pass, call, "Error")...)

	case "FailNow":
		return newDiagnostic(checker.Name(), call, "use t.Fatal or t.Fatalf instead",
			checker.fix(pass, call, "Fatal")...)
	}
	return nil
}

// fix builds a SuggestedFix replacing the testify call with a standard testing.T method call.
//
// Caller variable is resolved as:
//   - for package-level calls (IsPkg == true): the first raw argument (the TestingT)
//   - for method calls on a testify suite (IsPkg == false): "receiver.T()"
//
// Argument mapping:
//
// Fmt variants (Failf/FailNowf):   drop failureMessage, keep format + args
//
//	assert.Failf(t, "failure", "fmt %s", arg) → t.Errorf("fmt %s", arg)
//
// Non-fmt, 1 arg (failureMessage only):
//
//	assert.Fail(t, "msg")           → t.Error("msg")
//
// Non-fmt, 2 args (failureMessage + one msgAndArgs element):
//
//	assert.Fail(t, "failure", "msg") → t.Error("msg")
//
// Non-fmt, 3+ args (failureMessage + format + args):
//
//	assert.Fail(t, "failure", "fmt %s", arg) → t.Errorf("fmt %s", arg)
func (checker FailNow) fix(pass *analysis.Pass, call *CallMeta, proposedFn string) []analysis.SuggestedFix {
	callerVar, ok := checker.callerVar(pass, call)
	if !ok {
		return nil
	}

	var newArgs []ast.Expr
	fn := proposedFn

	if call.Fn.IsFmt {
		// Failf(t, failureMessage, format, args...) → callerVar.Errorf(format, args...)
		if len(call.Args) < 2 {
			return nil
		}
		fn += "f"
		newArgs = call.Args[1:]
	} else {
		switch len(call.Args) {
		case 0:
			return nil
		case 1:
			// Fail(t, failureMessage) → callerVar.Error(failureMessage)
			newArgs = call.Args
		case 2:
			// Fail(t, failureMessage, msg) → callerVar.Error(msg)
			newArgs = call.Args[1:]
		default:
			// Fail(t, failureMessage, format, args...) → callerVar.Errorf(format, args...)
			fn += "f"
			newArgs = call.Args[1:]
		}
	}

	newText := []byte(fmt.Sprintf("%s.%s(%s)", callerVar, fn, formatAsCallArgs(pass, newArgs...)))

	return []analysis.SuggestedFix{{
		Message: fmt.Sprintf("Replace `%s` with `%s.%s`", call.Fn.Name, callerVar, fn),
		TextEdits: []analysis.TextEdit{{
			Pos:     call.Pos(),
			End:     call.End(),
			NewText: newText,
		}},
	}}
}

// callerVar returns the string to use as the testing.T variable in the replacement call.
//   - Package-level calls (assert.Fail(t, ...)): returns the string of the t argument.
//   - Suite method calls (s.Fail(...)): returns "s.T()" so the fix becomes s.T().Error(...).
func (checker FailNow) callerVar(pass *analysis.Pass, call *CallMeta) (string, bool) {
	if call.IsPkg {
		if len(call.ArgsRaw) == 0 {
			return "", false
		}
		return analysisutil.NodeString(pass.Fset, call.ArgsRaw[0]), true
	}

	// For method calls, the receiver must be a testify suite so we can use .T() to get *testing.T.
	if implementsTestifySuite(pass, call.Selector.X) {
		return analysisutil.NodeString(pass.Fset, call.Selector.X) + ".T()", true
	}
	return "", false
}
