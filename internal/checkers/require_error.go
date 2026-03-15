package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/Antonboom/testifylint/internal/analysisutil"
	"github.com/Antonboom/testifylint/internal/testify"
)

const requireErrorReport = "for error assertions use require"

// RequireError detects error assertions like
//
//	assert.Error(t, err) // s.Error(err), s.Assert().Error(err)
//	assert.ErrorIs(t, err, io.EOF)
//	assert.ErrorAs(t, err, &target)
//	assert.EqualError(t, err, "end of file")
//	assert.ErrorContains(t, err, "end of file")
//	assert.NoError(t, err)
//	assert.NotErrorIs(t, err, io.EOF)
//
// and requires
//
//	require.Error(t, err) // s.Require().Error(err), s.Require().Error(err)
//	require.ErrorIs(t, err, io.EOF)
//	require.ErrorAs(t, err, &target)
//	...
//
// RequireError ignores:
// - non-negated assertions in the `if` condition;
// - assertions in the bool expression;
// - the entire `if-else[-if]` block, if there is an assertion in any `if` condition;
// - the last assertion in the block, if there are no methods/functions calls after it;
// - assertions in an explicit goroutine (including `http.Handler`);
// - assertions in an explicit testing cleanup function or suite teardown methods;
// - sequence of NoError assertions.
//
// RequireError reports and provides a fix for negated assertions in an if
// condition when the if body consists solely of a return or continue statement
// and there is no else clause, e.g.:
//
//	if !assert.NoError(t, err) {
//	    return
//	}
//
// and requires
//
//	require.NoError(t, err)
type RequireError struct {
	fnPattern *regexp.Regexp
}

// NewRequireError constructs RequireError checker.
func NewRequireError() *RequireError { return new(RequireError) }
func (RequireError) Name() string    { return "require-error" }

func (checker *RequireError) SetFnPattern(p *regexp.Regexp) *RequireError {
	if p != nil {
		checker.fnPattern = p
	}
	return checker
}

func (checker RequireError) Check(pass *analysis.Pass, insp *inspector.Inspector) []analysis.Diagnostic {
	callsByFunc := make(map[funcID][]*callMeta)

	// Stage 1. Collect meta information about any calls inside functions.

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
		_, prevIsUnaryExpr := stack[len(stack)-2].(*ast.UnaryExpr)
		_, prevPrevIsIfStmt := stack[len(stack)-3].(*ast.IfStmt)
		inIfCond := prevIsIfStmt || (prevPrevIsIfStmt && prevIsAssignStmt)

		// Fix: also detect !assert.xxx() in if condition.
		// Stack pattern for `if !assert.xxx(...)`: [..., IfStmt, UnaryExpr, CallExpr]
		negatedInIfCond := prevPrevIsIfStmt && prevIsUnaryExpr
		if negatedInIfCond {
			inIfCond = true
		}

		_, inBoolExpr := stack[len(stack)-2].(*ast.BinaryExpr)

		// Fix: also detect !assert.xxx() in BinaryExpr inside if condition.
		// Stack pattern for `if !assert.xxx() || ...`: [..., IfStmt, BinaryExpr, UnaryExpr, CallExpr]
		if !negatedInIfCond && prevIsUnaryExpr && len(stack) >= 5 {
			_, prevPrevIsBinaryExpr := stack[len(stack)-3].(*ast.BinaryExpr)
			_, p4IsIfStmt := stack[len(stack)-4].(*ast.IfStmt)
			if prevPrevIsBinaryExpr && p4IsIfStmt {
				negatedInIfCond = true
				inIfCond = true
				inBoolExpr = true
			}
		}

		callExpr := node.(*ast.CallExpr)
		testifyCall := NewCallMeta(pass, callExpr)

		call := &callMeta{
			call:            callExpr,
			testifyCall:     testifyCall,
			rootIf:          findRootIf(stack),
			parentIf:        findNearestNode[*ast.IfStmt](stack),
			parentBlock:     findNearestNode[*ast.BlockStmt](stack),
			inIfCond:        inIfCond,
			inBoolExpr:      inBoolExpr,
			inNoErrorSeq:    false, // Will be filled in below.
			negatedInIfCond: negatedInIfCond,
		}

		callsByFunc[*fID] = append(callsByFunc[*fID], call)
		return testifyCall == nil // Do not support asserts in asserts.
	})

	// Stage 2. Analyze calls and block context.

	var diagnostics []analysis.Diagnostic

	callsByBlock := map[*ast.BlockStmt][]*callMeta{}
	for _, calls := range callsByFunc {
		for _, c := range calls {
			if b := c.parentBlock; b != nil {
				callsByBlock[b] = append(callsByBlock[b], c)
			}
		}
	}

	markCallsInNoErrorSequence(callsByBlock)

	// Stage 2a. Identify fixable negated-if patterns.
	// A negated-if pattern is fixable when:
	//   - All assertions in the if condition are negated error assertions (via ||)
	//   - The if body is a single return or continue statement
	//   - There is no else clause
	type fixableIfInfo struct {
		calls []*callMeta // Ordered error-assertion calls in the if condition.
	}
	fixableIfs := make(map[*ast.IfStmt]*fixableIfInfo)

	for _, calls := range callsByFunc {
		// Group negated-if error-assertion calls by their parent if statement.
		negatedIfGroups := make(map[*ast.IfStmt][]*callMeta)
		for _, c := range calls {
			if !c.negatedInIfCond || c.parentIf == nil || c.testifyCall == nil {
				continue
			}
			// Skip if the if is part of an else-if chain: rootIf != parentIf means
			// the if is directly nested inside another if's Else field, making an
			// isolated fix incorrect (it would leave a dangling "else").
			if c.rootIf != c.parentIf {
				continue
			}
			if !c.testifyCall.IsAssert {
				continue
			}
			switch c.testifyCall.Fn.NameFTrimmed {
			case "Error", "ErrorIs", "ErrorAs", "EqualError", "ErrorContains", "NoError", "NotErrorIs":
				negatedIfGroups[c.parentIf] = append(negatedIfGroups[c.parentIf], c)
			}
		}

		for ifStmt, group := range negatedIfGroups {
			if !isSimpleReturnBody(ifStmt) || ifStmt.Else != nil {
				continue
			}
			// Check that ALL top-level calls in the if condition are negated
			// error assertions (i.e. our group covers the entire condition).
			condCalls, allNegatedOr := collectNegatedOrCalls(ifStmt.Cond)
			if !allNegatedOr || len(condCalls) != len(group) {
				continue
			}
			groupSet := make(map[*ast.CallExpr]struct{}, len(group))
			for _, c := range group {
				groupSet[c.call] = struct{}{}
			}
			allMatch := true
			for _, cc := range condCalls {
				if _, ok := groupSet[cc]; !ok {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
			// Sort by source position to keep a stable replacement order.
			sorted := make([]*callMeta, len(group))
			copy(sorted, group)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].call.Pos() < sorted[j].call.Pos()
			})
			fixableIfs[ifStmt] = &fixableIfInfo{calls: sorted}
		}
	}

	// Tracks if statements that have already been reported (compound || case).
	reportedIfs := make(map[*ast.IfStmt]bool)

	for funcInfo, calls := range callsByFunc {
		for i, c := range calls {
			if m := funcInfo.meta; m.isTestCleanup || m.isGoroutine || m.isHTTPHandler {
				continue
			}

			if c.testifyCall == nil {
				continue
			}
			if !c.testifyCall.IsAssert {
				continue
			}
			switch c.testifyCall.Fn.NameFTrimmed {
			default:
				continue
			case "Error", "ErrorIs", "ErrorAs", "EqualError", "ErrorContains", "NoError", "NotErrorIs":
			}

			// Handle negated-if patterns before the normal skip logic.
			if c.negatedInIfCond && c.parentIf != nil {
				if info := fixableIfs[c.parentIf]; info != nil {
					// Fixable pattern: report with a suggested fix (first call wins).
					if reportedIfs[c.parentIf] {
						continue // Already reported for this if statement.
					}
					if p := checker.fnPattern; p != nil && !p.MatchString(c.testifyCall.Fn.Name) {
						continue
					}
					reportedIfs[c.parentIf] = true
					if fix, ok := buildNegatedIfFix(pass, c.parentIf, info.calls); ok {
						diagnostics = append(diagnostics,
							*newDiagnostic(checker.Name(), c.testifyCall, requireErrorReport, fix))
					} else {
						diagnostics = append(diagnostics,
							*newDiagnostic(checker.Name(), c.testifyCall, requireErrorReport))
					}
					continue
				}
				// Non-fixable negated-if: skip (treat as inIfCond).
				continue
			}

			if needToSkipBasedOnContext(c, i, calls, callsByBlock) {
				continue
			}
			if p := checker.fnPattern; p != nil && !p.MatchString(c.testifyCall.Fn.Name) {
				continue
			}

			diagnostics = append(diagnostics,
				*newDiagnostic(checker.Name(), c.testifyCall, requireErrorReport))
		}
	}

	return diagnostics
}

// isSimpleReturnBody reports whether the body of ifStmt consists of exactly one
// statement that is a return or continue (no other side effects).
func isSimpleReturnBody(ifStmt *ast.IfStmt) bool {
	if len(ifStmt.Body.List) != 1 {
		return false
	}
	switch ifStmt.Body.List[0].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		b := ifStmt.Body.List[0].(*ast.BranchStmt)
		return b.Tok == token.CONTINUE
	}
	return false
}

// collectNegatedOrCalls recursively collects all top-level CallExpr nodes from
// a condition that consists exclusively of negated calls (UnaryExpr with !) joined
// by logical-or (||). Returns the calls in left-to-right order and true if the
// entire expression matches that pattern; returns nil and false otherwise.
func collectNegatedOrCalls(expr ast.Expr) ([]*ast.CallExpr, bool) {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			if ce, ok := e.X.(*ast.CallExpr); ok {
				return []*ast.CallExpr{ce}, true
			}
		}
		return nil, false
	case *ast.BinaryExpr:
		if e.Op != token.LOR {
			return nil, false
		}
		left, leftOk := collectNegatedOrCalls(e.X)
		right, rightOk := collectNegatedOrCalls(e.Y)
		if !leftOk || !rightOk {
			return nil, false
		}
		return append(left, right...), true
	default:
		return nil, false
	}
}

// buildNegatedIfFix constructs a SuggestedFix that replaces the entire ifStmt
// with one require.XXX call per assertion in calls (in source order).
// The require import is added to the file if it is not already present.
func buildNegatedIfFix(pass *analysis.Pass, ifStmt *ast.IfStmt, calls []*callMeta) (analysis.SuggestedFix, bool) {
	if len(calls) == 0 {
		return analysis.SuggestedFix{}, false
	}

	// Determine the local qualifier for the require package.
	qualName, importEdit, ok := addImportFix(pass.Files, calls[0].call.Pos(), testify.RequirePkgPath)
	if !ok {
		return analysis.SuggestedFix{}, false
	}

	// Build the replacement text: one require call per assertion.
	indent := getLineIndent(pass, ifStmt.Pos())
	var requireCalls []string
	for _, c := range calls {
		callText := analysisutil.NodeString(pass.Fset, c.testifyCall.Call)
		oldPkg := c.testifyCall.SelectorXStr
		newCallText := qualName + callText[len(oldPkg):]
		requireCalls = append(requireCalls, newCallText)
	}
	newText := strings.Join(requireCalls, "\n"+indent)

	textEdits := []analysis.TextEdit{
		{
			Pos:     ifStmt.Pos(),
			End:     ifStmt.End(),
			NewText: []byte(newText),
		},
	}
	if importEdit != nil {
		textEdits = append(textEdits, *importEdit)
	}

	msg := fmt.Sprintf("Replace with %s.%s", qualName, calls[0].testifyCall.Fn.Name)
	if len(calls) > 1 {
		msg = fmt.Sprintf("Replace with %s calls", qualName)
	}

	return analysis.SuggestedFix{
		Message:   msg,
		TextEdits: textEdits,
	}, true
}

// getLineIndent returns the whitespace prefix (indentation) of the source line
// that contains pos. Falls back to tab-based indentation using the token column.
func getLineIndent(pass *analysis.Pass, pos token.Pos) string {
	tokenFile := pass.Fset.File(pos)
	if tokenFile == nil {
		return "\t"
	}

	content, err := pass.ReadFile(tokenFile.Name())
	if err != nil {
		// Fallback: assume tab indentation based on column.
		col := pass.Fset.Position(pos).Column - 1
		if col < 0 {
			col = 0
		}
		return strings.Repeat("\t", col)
	}

	offset := tokenFile.Offset(pos)

	// Find the start of the line.
	lineStart := offset
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}

	// Extract only the leading whitespace up to pos.
	end := lineStart
	for end < offset && (content[end] == ' ' || content[end] == '\t') {
		end++
	}

	return string(content[lineStart:end])
}

func needToSkipBasedOnContext(
	currCall *callMeta,
	currCallIndex int,
	otherCalls []*callMeta,
	callsByBlock map[*ast.BlockStmt][]*callMeta,
) bool {
	if currCall.inIfCond || currCall.inBoolExpr || currCall.inNoErrorSeq {
		return true
	}

	if currCall.rootIf != nil {
		for _, rootCall := range otherCalls {
			if (rootCall.rootIf == currCall.rootIf) && rootCall.inIfCond {
				// Skip assertions in the entire if-else[-if] block, if some of "if condition" contains assertion.
				return true
			}
		}
	}

	block := currCall.parentBlock
	blockCalls := callsByBlock[block]
	isLastCallInBlock := blockCalls[len(blockCalls)-1] == currCall

	noCallsAfter := true

	_, blockEndWithReturn := block.List[len(block.List)-1].(*ast.ReturnStmt)
	if !blockEndWithReturn {
		for i := currCallIndex + 1; i < len(otherCalls); i++ {
			nextCall := otherCalls[i]
			nextCallInElseBlock := false

			if pIf := currCall.parentIf; pIf != nil && pIf.Else != nil {
				ast.Inspect(pIf.Else, func(n ast.Node) bool {
					if n == nextCall.call {
						nextCallInElseBlock = true
						return false
					}
					return true
				})
			}

			if !nextCallInElseBlock {
				noCallsAfter = false
				break
			}
		}
	}

	// Skip assertion if this is the last operation in the test.
	return isLastCallInBlock && noCallsAfter
}

func findRootIf(stack []ast.Node) *ast.IfStmt {
	nearestIf, i := findNearestNodeWithIdx[*ast.IfStmt](stack)
	for ; i > 0; i-- {
		parent, ok := stack[i-1].(*ast.IfStmt)
		if !ok {
			break
		}
		nearestIf = parent
	}
	return nearestIf
}

func markCallsInNoErrorSequence(callsByBlock map[*ast.BlockStmt][]*callMeta) {
	for _, calls := range callsByBlock {
		for i, c := range calls {
			if c.testifyCall == nil {
				continue
			}

			var prevIsNoError bool
			if i > 0 {
				if prev := calls[i-1].testifyCall; prev != nil {
					prevIsNoError = isNoErrorAssertion(prev.Fn.Name)
				}
			}

			var nextIsNoError bool
			if i < len(calls)-1 {
				if next := calls[i+1].testifyCall; next != nil {
					nextIsNoError = isNoErrorAssertion(next.Fn.Name)
				}
			}

			if isNoErrorAssertion(c.testifyCall.Fn.Name) && (prevIsNoError || nextIsNoError) {
				calls[i].inNoErrorSeq = true
			}
		}
	}
}

type callMeta struct {
	call            *ast.CallExpr
	testifyCall     *CallMeta
	rootIf          *ast.IfStmt // The root `if` in if-else[-if] chain.
	parentIf        *ast.IfStmt // The nearest `if`, can be equal with rootIf.
	parentBlock     *ast.BlockStmt
	inIfCond        bool // True for code like `if assert.ErrorAs(t, err, &target) {`.
	inBoolExpr      bool // True for code like `assert.Error(t, err) && assert.ErrorContains(t, err, "value")`
	inNoErrorSeq    bool // True for sequence of `assert.NoError` assertions.
	negatedInIfCond bool // True for code like `if !assert.NoError(t, err) {`.
}

func isNoErrorAssertion(fnName string) bool {
	return (fnName == "NoError") || (fnName == "NoErrorf")
}
