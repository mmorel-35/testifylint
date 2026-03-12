package checkers

import (
	"go/ast"
	"go/constant"
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

func (checker HTTPConst) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
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
		edit := newHTTPMethodTextEdit(pass, call.Args[1])
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
		if edit := newHTTPMethodTextEdit(pass, call.Args[1]); edit != nil {
			textEdits = append(textEdits, *edit)
		}
		if edit := newHTTPStatusCodeTextEdit(pass, call.Args[4]); edit != nil {
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

func newHTTPMethodTextEdit(pass *analysis.Pass, e ast.Expr) *analysis.TextEdit {
	bt, ok := typeSafeBasicLit(e, token.STRING)
	if !ok {
		return nil
	}
	currentVal, ok := unquoteBasicLitValue(bt)
	if !ok {
		return nil
	}
	constName, ok := httpMethod[strings.ToUpper(currentVal)]
	if !ok {
		return nil
	}
	newVal := httpQualifiedName(pass, bt.Pos(), constName)
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}
}

func newHTTPStatusCodeTextEdit(pass *analysis.Pass, e ast.Expr) *analysis.TextEdit {
	bt, ok := typeSafeBasicLit(e, token.INT)
	if !ok {
		return nil
	}
	// Use go/constant to parse the literal, correctly handling all Go integer literal forms
	// (decimal, hex 0xC8, octal 0o310, binary 0b11001000, underscore separators 2_00, etc.).
	v := constant.MakeFromLiteral(bt.Value, token.INT, 0)
	if v.Kind() != constant.Int {
		return nil
	}
	intVal, exact := constant.Int64Val(v)
	if !exact {
		return nil
	}
	constName, ok := httpStatusCode[int(intVal)]
	if !ok {
		return nil
	}
	newVal := httpQualifiedName(pass, bt.Pos(), constName)
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}
}

// httpQualifiedName returns "qualifier.constName" using the local import name of net/http,
// or just "constName" if net/http is dot-imported.
func httpQualifiedName(pass *analysis.Pass, pos token.Pos, constName string) string {
	qualifier := httpNetPkgName(pass, pos)
	if qualifier == "" {
		return constName
	}
	return qualifier + "." + constName
}
