package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

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

// callInStmt pairs a call with its parent-statement index in the enclosing block.
type callInStmt struct {
	call    *CallMeta
	stmtIdx int
}

func (checker HTTPMultiple) checkBody(pass *analysis.Pass, body *ast.BlockStmt) []analysis.Diagnostic {
	groups := make(map[httpCallKey][]callInStmt)

	// Iterate over each top-level statement so we can track statement indices.
	for i, stmt := range body.List {
		ast.Inspect(stmt, func(node ast.Node) bool {
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
			groups[key] = append(groups[key], callInStmt{call: call, stmtIdx: i})
			return true
		})
	}

	var diagnostics []analysis.Diagnostic
	for key, calls := range groups {
		if len(calls) < 2 {
			continue
		}
		sort.Slice(calls, func(i, j int) bool {
			return calls[i].call.Pos() < calls[j].call.Pos()
		})

		// A fix is only generated when all calls sit in consecutive statements.
		consecutive := true
		for i := 1; i < len(calls); i++ {
			if calls[i].stmtIdx != calls[i-1].stmtIdx+1 {
				consecutive = false
				break
			}
		}

		var fix *analysis.SuggestedFix
		if consecutive {
			fix = checker.generateFix(pass, body, key, calls)
		}

		for i, cis := range calls {
			if i == 0 && fix != nil {
				d := newDiagnostic(checker.Name(), cis.call, httpMultipleReport, *fix)
				diagnostics = append(diagnostics, *d)
			} else {
				d := newDiagnostic(checker.Name(), cis.call, httpMultipleReport)
				diagnostics = append(diagnostics, *d)
			}
		}
	}
	return diagnostics
}

func (checker HTTPMultiple) generateFix(
	pass *analysis.Pass,
	body *ast.BlockStmt,
	key httpCallKey,
	calls []callInStmt,
) *analysis.SuggestedFix {
	// Use the first call's package (assert or require).
	pkg := calls[0].call.SelectorXStr

	// Derive indentation from the column of the first statement.
	// Column is 1-indexed and counts bytes, so col-1 = number of leading tab/space bytes.
	// Go source files always use tabs (enforced by gofmt), so this is a safe assumption.
	firstStmt := body.List[calls[0].stmtIdx]
	col := pass.Fset.Position(firstStmt.Pos()).Column
	indent := strings.Repeat("\t", col-1)
	innerIndent := indent + "\t"

	// Collect replacement assertion lines for every call in the group.
	var assertLines []string
	for _, cis := range calls {
		lines := httpAssertionReplacement(pass, pkg, cis.call)
		if lines == nil {
			return nil // Cannot auto-fix this particular assertion type.
		}
		assertLines = append(assertLines, lines...)
	}

	// Build the replacement block: a scoped { } avoids variable re-declaration
	// when multiple groups are fixed in the same function at once.
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(innerIndent)
	// httptest.NewRequest panics on invalid method/URL and is idiomatic for test code.
	sb.WriteString(fmt.Sprintf("req := httptest.NewRequest(%s, %s, nil)\n", key.method, key.url))
	if key.values != "nil" {
		sb.WriteString(innerIndent)
		sb.WriteString(fmt.Sprintf("req.Form = %s\n", key.values))
	}
	sb.WriteString(innerIndent)
	sb.WriteString("rr := httptest.NewRecorder()\n")
	sb.WriteString(innerIndent)
	sb.WriteString(fmt.Sprintf("%s(rr, req)\n", key.handler))
	for _, line := range assertLines {
		sb.WriteString(innerIndent)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("}")

	// Extend the end of the edit range to the start of the next line so that any
	// trailing inline comment (e.g. // want "...") on the last statement is also removed.
	lastStmt := body.List[calls[len(calls)-1].stmtIdx]
	f := pass.Fset.File(lastStmt.End())
	lastLine := f.Line(lastStmt.End())
	var endPos token.Pos
	if lastLine < f.LineCount() {
		endPos = f.LineStart(lastLine + 1)
		sb.WriteString("\n") // replace the consumed line-ending
	} else {
		endPos = token.Pos(f.Base() + f.Size())
	}

	return &analysis.SuggestedFix{
		Message: "Use httptest.NewRecorder()",
		TextEdits: []analysis.TextEdit{{
			Pos:     firstStmt.Pos(),
			End:     endPos,
			NewText: []byte(sb.String()),
		}},
	}
}

// httpAssertionReplacement maps one HTTP testify assertion to its httptest equivalent line(s).
// call.Args layout: [handler, method, url, values, <assertion-specific args...>]
// Returns nil when the assertion cannot be automatically fixed.
func httpAssertionReplacement(pass *analysis.Pass, pkg string, call *CallMeta) []string {
	extra := call.Args[4:] // args after handler/method/url/values
	fSuffix := ""
	if call.Fn.IsFmt {
		fSuffix = "f"
	}

	argsStr := func(args []ast.Expr) string {
		if len(args) == 0 {
			return ""
		}
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = analysisutil.NodeString(pass.Fset, a)
		}
		return ", " + strings.Join(parts, ", ")
	}

	switch call.Fn.NameFTrimmed {
	case "HTTPStatusCode":
		if len(extra) < 1 {
			return nil
		}
		code := analysisutil.NodeString(pass.Fset, extra[0])
		msg := argsStr(extra[1:])
		return []string{fmt.Sprintf("%s.Equal%s(t, %s, rr.Code%s)", pkg, fSuffix, code, msg)}

	case "HTTPBodyContains":
		if len(extra) < 1 {
			return nil
		}
		str := analysisutil.NodeString(pass.Fset, extra[0])
		msg := argsStr(extra[1:])
		return []string{fmt.Sprintf("%s.Contains%s(t, rr.Body.String(), %s%s)", pkg, fSuffix, str, msg)}

	case "HTTPBodyNotContains":
		if len(extra) < 1 {
			return nil
		}
		str := analysisutil.NodeString(pass.Fset, extra[0])
		msg := argsStr(extra[1:])
		return []string{fmt.Sprintf("%s.NotContains%s(t, rr.Body.String(), %s%s)", pkg, fSuffix, str, msg)}

	case "HTTPError":
		msg := argsStr(extra)
		return []string{fmt.Sprintf("%s.GreaterOrEqual%s(t, rr.Code, 400%s)", pkg, fSuffix, msg)}

	case "HTTPSuccess":
		msg := argsStr(extra)
		return []string{
			fmt.Sprintf("%s.GreaterOrEqual%s(t, rr.Code, 200%s)", pkg, fSuffix, msg),
			fmt.Sprintf("%s.Less%s(t, rr.Code, 300%s)", pkg, fSuffix, msg),
		}

	case "HTTPRedirect":
		msg := argsStr(extra)
		return []string{
			fmt.Sprintf("%s.GreaterOrEqual%s(t, rr.Code, 300%s)", pkg, fSuffix, msg),
			fmt.Sprintf("%s.Less%s(t, rr.Code, 400%s)", pkg, fSuffix, msg),
		}
	}

	return nil // HTTPBody or unknown — skip fix.
}

func isHTTPAssertion(call *CallMeta) bool {
	switch call.Fn.NameFTrimmed {
	case "HTTPBody", "HTTPBodyContains", "HTTPBodyNotContains",
		"HTTPError", "HTTPRedirect", "HTTPStatusCode", "HTTPSuccess":
		return true
	}
	return false
}
