package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

const requireLenForIndexReport = "for length assertions guarding index access use require"
const requireLenGuardReport = "for indexed access use require.Len guard"
const defaultIndent = "\t"

// RequireLen checks fail-fast guards for indexed access.
type RequireLen struct{}

// NewRequireLen constructs RequireLen checker.
func NewRequireLen() RequireLen { return RequireLen{} }
func (RequireLen) Name() string { return "require-len" }

func (checker RequireLen) Check(pass *analysis.Pass, insp *inspector.Inspector) []analysis.Diagnostic {
	callsByFunc := make(map[funcID][]*callMeta)

	insp.WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return false
		}
		if len(stack) < 3 {
			return true
		}

		fID := findSurroundingFunc(pass, stack)
		if fID == nil {
			return true
		}

		_, prevIsIfStmt := stack[len(stack)-2].(*ast.IfStmt)
		_, prevIsAssignStmt := stack[len(stack)-2].(*ast.AssignStmt)
		_, prevPrevIsIfStmt := stack[len(stack)-3].(*ast.IfStmt)
		inIfCond := prevIsIfStmt || (prevPrevIsIfStmt && prevIsAssignStmt)

		_, inBoolExpr := stack[len(stack)-2].(*ast.BinaryExpr)

		callExpr := node.(*ast.CallExpr)
		testifyCall := NewCallMeta(pass, callExpr)

		call := &callMeta{
			call:         callExpr,
			testifyCall:  testifyCall,
			rootIf:       findRootIf(stack),
			parentIf:     findNearestNode[*ast.IfStmt](stack),
			parentBlock:  findNearestNode[*ast.BlockStmt](stack),
			inIfCond:     inIfCond,
			inBoolExpr:   inBoolExpr,
			inNoErrorSeq: false,
		}

		callsByFunc[*fID] = append(callsByFunc[*fID], call)
		return testifyCall == nil
	})

	var diagnostics []analysis.Diagnostic

	callsByBlock := map[*ast.BlockStmt][]*callMeta{}
	fileContentCache := make(map[string][]byte)
	for _, calls := range callsByFunc {
		for _, c := range calls {
			if b := c.parentBlock; b != nil {
				callsByBlock[b] = append(callsByBlock[b], c)
			}
		}
	}

	markCallsInNoErrorSequence(callsByBlock)

	for funcInfo, calls := range callsByFunc {
		for i, c := range calls {
			if m := funcInfo.meta; m.isTestCleanup || m.isGoroutine || m.isHTTPHandler {
				continue
			}
			if c.testifyCall == nil || !c.testifyCall.IsAssert {
				continue
			}

			switch c.testifyCall.Fn.NameFTrimmed {
			case "Len", "Lenf":
				if needToSkipBasedOnContext(c, i, calls, callsByBlock) {
					continue
				}
				if shouldRequireLenForIndexedAccess(pass, c, i, calls) {
					diagnostics = append(diagnostics,
						*newDiagnostic(checker.Name(), c.testifyCall, requireLenForIndexReport))
				}
			default:
				if needToSkipForLenGuardContext(c, calls) {
					continue
				}
				if d := newRequireLenGuardDiagnostic(pass, checker.Name(), c, i, calls, fileContentCache); d != nil {
					diagnostics = append(diagnostics, *d)
				}
			}
		}
	}

	return diagnostics
}

func shouldRequireLenForIndexedAccess(
	pass *analysis.Pass,
	currCall *callMeta,
	currCallIndex int,
	otherCalls []*callMeta,
) bool {
	if len(currCall.testifyCall.Args) < 2 {
		return false
	}

	collectionExpr := currCall.testifyCall.Args[0]
	requiredLen, ok := isIntBasicLit(currCall.testifyCall.Args[1])
	if !ok || requiredLen <= 0 {
		return false
	}

	collectionStr := analysisutil.NodeString(pass.Fset, collectionExpr)
	if collectionStr == "" {
		return false
	}

	for i := currCallIndex + 1; i < len(otherCalls); i++ {
		if containsIndexedAccess(pass, otherCalls[i].call, collectionStr, requiredLen) {
			return true
		}
	}
	return false
}

func containsIndexedAccess(pass *analysis.Pass, node ast.Node, collection string, requiredLen int) bool {
	var found bool

	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}

		ie, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}

		idx, ok := isIntBasicLit(ie.Index)
		if !ok || idx < 0 || idx >= requiredLen {
			return true
		}

		if analysisutil.NodeString(pass.Fset, ie.X) == collection {
			found = true
			return false
		}
		return true
	})

	return found
}

func newRequireLenGuardDiagnostic(
	pass *analysis.Pass,
	checkerName string,
	currCall *callMeta,
	currCallIndex int,
	otherCalls []*callMeta,
	fileContentCache map[string][]byte,
) *analysis.Diagnostic {
	if !currCall.testifyCall.IsPkg || (len(currCall.testifyCall.ArgsRaw) < 2) {
		return nil
	}
	tArg := currCall.testifyCall.ArgsRaw[0]
	if !implementsTestingT(pass, tArg) {
		return nil
	}

	for _, target := range indexedAccesses(pass, currCall.call) {
		requiredLen := target.maxIndex + 1
		if hasLenGuard(pass, currCall, currCallIndex, otherCalls, target.collection, requiredLen) {
			continue
		}

		indent := lineIndentAtPos(pass, currCall.call.Pos(), fileContentCache)
		insertText := fmt.Sprintf("require.Len(%s, %s, %d)\n%s",
			analysisutil.NodeString(pass.Fset, tArg), target.collection, requiredLen, indent)
		fixMsg := "Insert `require.Len` guard"
		if requiredLen == 1 {
			insertText = fmt.Sprintf("require.NotEmpty(%s, %s)\n%s",
				analysisutil.NodeString(pass.Fset, tArg), target.collection, indent)
			fixMsg = "Insert `require.NotEmpty` guard"
		}
		return newDiagnostic(checkerName, currCall.testifyCall, requireLenGuardReport, analysis.SuggestedFix{
			Message: fixMsg,
			TextEdits: []analysis.TextEdit{
				{
					Pos:     currCall.call.Pos(),
					End:     currCall.call.Pos(),
					NewText: []byte(insertText),
				},
			},
		})
	}
	return nil
}

type indexedAccess struct {
	collection string
	maxIndex   int
}

func indexedAccesses(pass *analysis.Pass, node ast.Node) []indexedAccess {
	var result []indexedAccess
	indexByCollection := make(map[string]int)

	ast.Inspect(node, func(n ast.Node) bool {
		ie, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}

		idx, ok := isIntBasicLit(ie.Index)
		if !ok || idx < 0 {
			return true
		}

		collection := analysisutil.NodeString(pass.Fset, ie.X)
		if collection == "" {
			return true
		}

		i, exists := indexByCollection[collection]
		if !exists {
			indexByCollection[collection] = len(result)
			result = append(result, indexedAccess{collection: collection, maxIndex: idx})
			return true
		}
		if idx > result[i].maxIndex {
			result[i].maxIndex = idx
		}
		return true
	})

	return result
}

func hasLenGuard(
	pass *analysis.Pass,
	currCall *callMeta,
	currCallIndex int,
	otherCalls []*callMeta,
	collection string,
	requiredLen int,
) bool {
	for i := 0; i < currCallIndex; i++ {
		c := otherCalls[i]
		if c.parentBlock != currCall.parentBlock || c.testifyCall == nil {
			continue
		}
		switch c.testifyCall.Fn.NameFTrimmed {
		case "Len":
			if len(c.testifyCall.Args) < 2 {
				continue
			}
			lenCollection := analysisutil.NodeString(pass.Fset, c.testifyCall.Args[0])
			if lenCollection != collection {
				continue
			}
			return true
		case "NotEmpty":
			if requiredLen != 1 || len(c.testifyCall.Args) < 1 {
				continue
			}
			notEmptyCollection := analysisutil.NodeString(pass.Fset, c.testifyCall.Args[0])
			if notEmptyCollection != collection {
				continue
			}
			return true
		}
	}
	return false
}

func lineIndentAtPos(pass *analysis.Pass, pos token.Pos, fileContentCache map[string][]byte) string {
	p := pass.Fset.PositionFor(pos, false)
	if (p.Filename == "") || (p.Offset < 0) {
		return defaultIndent
	}

	content, ok := fileContentCache[p.Filename]
	if !ok {
		var err error
		content, err = os.ReadFile(p.Filename)
		if err != nil {
			return defaultIndent
		}
		fileContentCache[p.Filename] = content
	}
	if p.Offset > len(content) {
		return defaultIndent
	}

	lineStart := p.Offset
	for lineStart > 0 {
		b := content[lineStart-1]
		if (b == '\n') || (b == '\r') {
			break
		}
		lineStart--
	}

	lineIndentEnd := lineStart
	for lineIndentEnd < len(content) {
		b := content[lineIndentEnd]
		if (b != ' ') && (b != '\t') {
			break
		}
		lineIndentEnd++
	}

	if lineIndentEnd == lineStart {
		return defaultIndent
	}

	return string(content[lineStart:lineIndentEnd])
}
