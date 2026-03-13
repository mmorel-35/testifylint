package checkers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

const httpMultipleReport = "use httptest.NewRecorder() instead of multiple HTTP assertions for the same handler call"

// HTTPMultiple detects situations like
//
//	assert.HTTPStatusCode(t, handler, "GET", "/path", nil, 200)
//	assert.HTTPBodyContains(t, handler, "GET", "/path", nil, "hello")
//
// and requires
//
//	r := httptest.NewRequest("GET", "/path", nil)
//	w := httptest.NewRecorder()
//	handler(w, r)
//	assert.Equal(t, 200, w.Code)
//	assert.Contains(t, w.Body.String(), "hello")
type HTTPMultiple struct{}

// NewHTTPMultiple constructs HTTPMultiple checker.
func NewHTTPMultiple() HTTPMultiple { return HTTPMultiple{} }
func (HTTPMultiple) Name() string   { return "http-multiple" }

func (checker HTTPMultiple) Check(pass *analysis.Pass, insp *inspector.Inspector) []analysis.Diagnostic {
	var diagnostics []analysis.Diagnostic

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}, func(node ast.Node) {
		var body *ast.BlockStmt
		switch n := node.(type) {
		case *ast.FuncDecl:
			body = n.Body
		case *ast.FuncLit:
			body = n.Body
		}
		if body == nil {
			return
		}

		diagnostics = append(diagnostics, checker.checkBody(pass, body)...)
	})

	return diagnostics
}

type httpCallKey struct {
	handler, method, url, values string
}

func (checker HTTPMultiple) checkBody(pass *analysis.Pass, body *ast.BlockStmt) []analysis.Diagnostic {
	groups := make(map[httpCallKey][]*CallMeta)

	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		// Don't cross function literal boundaries; they form independent scopes.
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}

		ce, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		call := NewCallMeta(pass, ce)
		if call == nil {
			return true
		}

		if !isHTTPAssertion(call) {
			return true
		}

		// HTTP assertions have at least 4 args after t: handler, method, url, values.
		if len(call.Args) < 4 {
			return true
		}

		key := httpCallKey{
			handler: analysisutil.NodeString(pass.Fset, call.Args[0]),
			method:  analysisutil.NodeString(pass.Fset, call.Args[1]),
			url:     analysisutil.NodeString(pass.Fset, call.Args[2]),
			values:  analysisutil.NodeString(pass.Fset, call.Args[3]),
		}
		groups[key] = append(groups[key], call)
		return true
	})

	var diagnostics []analysis.Diagnostic
	for _, calls := range groups {
		if len(calls) < 2 {
			continue
		}
		for _, call := range calls {
			d := newDiagnostic(checker.Name(), call, httpMultipleReport)
			diagnostics = append(diagnostics, *d)
		}
	}
	return diagnostics
}

func isHTTPAssertion(call *CallMeta) bool {
	switch call.Fn.NameFTrimmed {
	case "HTTPBody", "HTTPBodyContains", "HTTPBodyNotContains",
		"HTTPError", "HTTPRedirect", "HTTPStatusCode", "HTTPSuccess":
		return true
	}
	return false
}
