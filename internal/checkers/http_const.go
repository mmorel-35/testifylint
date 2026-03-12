package checkers

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// HTTPConst detects situations like
//
//	assert.HTTPStatusCode(t, handler, "GET", "/index", nil, 200)
//	assert.HTTPBodyContains(t, handler, "GET", "/index", nil, "counter")
//
// and requires
//
//	assert.HTTPStatusCode(t, handler, http.MethodGet, "/index", nil, http.StatusOK)
//	assert.HTTPBodyContains(t, handler, http.MethodGet, "/index", nil, "counter")
type HTTPConst struct{}

// NewHTTPConst constructs HTTPConst checker.
func NewHTTPConst() HTTPConst  { return HTTPConst{} }
func (HTTPConst) Name() string { return "http-const" }

func (checker HTTPConst) Check(_ *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	switch call.Fn.NameFTrimmed {
	case "HTTPBody",
		"HTTPBodyContains",
		"HTTPBodyNotContains",
		"HTTPError",
		"HTTPRedirect",
		"HTTPSuccess":
		if len(call.Args) < 2 {
			return nil
		}
		edit := newHTTPMethodTextEdit(call.Args[1])
		if edit == nil {
			return nil
		}
		return newDiagnostic(checker.Name(), call, "use net/http constants instead of value",
			analysis.SuggestedFix{
				Message:   "Replace with net/http constant",
				TextEdits: []analysis.TextEdit{*edit},
			})

	case "HTTPStatusCode":
		if len(call.Args) < 5 {
			return nil
		}
		var textEdits []analysis.TextEdit
		if edit := newHTTPMethodTextEdit(call.Args[1]); edit != nil {
			textEdits = append(textEdits, *edit)
		}
		if edit := newHTTPStatusCodeTextEdit(call.Args[4]); edit != nil {
			textEdits = append(textEdits, *edit)
		}
		if len(textEdits) == 0 {
			return nil
		}
		return newDiagnostic(checker.Name(), call, "use net/http constants instead of value",
			analysis.SuggestedFix{
				Message:   "Replace with net/http constants",
				TextEdits: textEdits,
			})
	}
	return nil
}

func newHTTPMethodTextEdit(e ast.Expr) *analysis.TextEdit {
	bt, ok := typeSafeBasicLit(e, token.STRING)
	if !ok {
		return nil
	}
	currentVal, ok := unquoteBasicLitValue(bt)
	if !ok {
		return nil
	}
	newVal, ok := httpMethod[strings.ToUpper(currentVal)]
	if !ok {
		return nil
	}
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}
}

func newHTTPStatusCodeTextEdit(e ast.Expr) *analysis.TextEdit {
	bt, ok := typeSafeBasicLit(e, token.INT)
	if !ok {
		return nil
	}
	newVal, ok := httpStatusCode[bt.Value]
	if !ok {
		return nil
	}
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}
}
