package checkers

import "golang.org/x/tools/go/analysis"

// FailNow detects situations like
//
//	assert.Fail(t, "msg")
//	assert.Fail(t, "msg", args...)
//	assert.Failf(t, "failure", "format %s", arg)
//	assert.FailNow(t, "msg")
//	assert.FailNow(t, "msg", args...)
//	assert.FailNowf(t, "failure", "format %s", arg)
//
// and requires
//
//	t.Error("msg") / t.Fatal("msg")
//	t.Errorf("format %s", arg)
//	t.Fatal("msg") / t.Fatalf("msg")
type FailNow struct{}

// NewFailNow constructs FailNow checker.
func NewFailNow() FailNow { return FailNow{} }
func (FailNow) Name() string { return "fail-now" }

func (checker FailNow) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	switch call.Fn.NameFTrimmed {
	case "Fail":
		return newDiagnostic(checker.Name(), call, "use t.Error or t.Errorf instead")

	case "FailNow":
		return newDiagnostic(checker.Name(), call, "use t.Fatal or t.Fatalf instead")
	}
	return nil
}
