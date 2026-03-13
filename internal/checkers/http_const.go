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
		valueEdit, importEdit := newHTTPMethodTextEdit(pass, call.Args[1])
		if valueEdit == nil {
			return nil
		}
		textEdits := []analysis.TextEdit{*valueEdit}
		if importEdit != nil {
			textEdits = append(textEdits, *importEdit)
		}
		return newDiagnostic(checker.Name(), call, "use net/http constants instead of value",
			analysis.SuggestedFix{
				Message:   "Replace with net/http constant",
				TextEdits: textEdits,
			})

	case "HTTPStatusCode":
		if len(call.Args) < 5 {
			return nil
		}
		methodEdit, methodImport := newHTTPMethodTextEdit(pass, call.Args[1])
		statusEdit, statusImport := newHTTPStatusCodeTextEdit(pass, call.Args[4])
		if methodEdit == nil && statusEdit == nil {
			return nil
		}
		var textEdits []analysis.TextEdit
		if methodEdit != nil {
			textEdits = append(textEdits, *methodEdit)
		}
		if statusEdit != nil {
			textEdits = append(textEdits, *statusEdit)
		}
		// Include the import edit once (both edits target the same file/import).
		importEdit := methodImport
		if importEdit == nil {
			importEdit = statusImport
		}
		if importEdit != nil {
			textEdits = append(textEdits, *importEdit)
		}
		return newDiagnostic(checker.Name(), call, "use net/http constants instead of value",
			analysis.SuggestedFix{
				Message:   "Replace with net/http constants",
				TextEdits: textEdits,
			})
	}
	return nil
}

func newHTTPMethodTextEdit(pass *analysis.Pass, e ast.Expr) (*analysis.TextEdit, *analysis.TextEdit) {
	bt, ok := typeSafeBasicLit(e, token.STRING)
	if !ok {
		return nil, nil
	}
	currentVal, ok := unquoteBasicLitValue(bt)
	if !ok {
		return nil, nil
	}
	constName, ok := httpMethod[strings.ToUpper(currentVal)]
	if !ok {
		return nil, nil
	}
	newVal, importEdit, ok := httpQualifiedName(pass, bt.Pos(), constName)
	if !ok {
		return nil, nil
	}
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}, importEdit
}

func newHTTPStatusCodeTextEdit(pass *analysis.Pass, e ast.Expr) (*analysis.TextEdit, *analysis.TextEdit) {
	bt, ok := typeSafeBasicLit(e, token.INT)
	if !ok {
		return nil, nil
	}
	// Use go/constant to parse the literal, correctly handling all Go integer literal forms
	// (decimal, hex 0xC8, octal 0o310, binary 0b11001000, underscore separators 2_00, etc.).
	v := constant.MakeFromLiteral(bt.Value, token.INT, 0)
	if v.Kind() != constant.Int {
		return nil, nil
	}
	intVal, exact := constant.Int64Val(v)
	if !exact {
		return nil, nil
	}
	// Guard against overflow when converting int64 to int on 32-bit platforms.
	if int64(int(intVal)) != intVal {
		return nil, nil
	}
	constName, ok := httpStatusCode[int(intVal)]
	if !ok {
		return nil, nil
	}
	newVal, importEdit, ok := httpQualifiedName(pass, bt.Pos(), constName)
	if !ok {
		return nil, nil
	}
	return &analysis.TextEdit{
		Pos:     bt.Pos(),
		End:     bt.End(),
		NewText: []byte(newVal),
	}, importEdit
}

// httpQualifiedName returns ("qualifier.constName", importEdit, true) using the local import name of net/http,
// or ("constName", nil, true) if net/http is dot-imported, or ("", nil, false) if net/http
// cannot be provided (blank-imported or all candidate names are taken).
// importEdit is non-nil when net/http is absent and needs to be added.
func httpQualifiedName(pass *analysis.Pass, pos token.Pos, constName string) (string, *analysis.TextEdit, bool) {
	qualifier, importEdit, ok := httpNetPkgName(pass, pos)
	if !ok {
		return "", nil, false
	}
	if qualifier == "" {
		return constName, nil, true // dot-import: no qualifier needed
	}
	return qualifier + "." + constName, importEdit, true
}
