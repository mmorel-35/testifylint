package checkers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

const gracefulTeardownReport = "do not use require in cleanup code"

// GracefulTeardown detects usage of require package's functions in t.Cleanup functions and suite teardown methods.
//
//	func (s *ServiceIntegrationSuite) TearDownTest() {
//		if p := s.verdictsProducer; p != nil {
//			s.Require().NoError(p.Close()) // ← not ok, use s.Assert() instead
//		}
//		if c := s.dlqVerdictsConsumer; c != nil {
//			s.NoError(c.Close()) // ← ok
//		}
//	}
type GracefulTeardown struct{}

// NewGracefulTeardown constructs GracefulTeardown checker.
func NewGracefulTeardown() GracefulTeardown { return GracefulTeardown{} }
func (GracefulTeardown) Name() string       { return "graceful-teardown" }

func (checker GracefulTeardown) Check(pass *analysis.Pass, insp *inspector.Inspector) (diagnostics []analysis.Diagnostic) {
	insp.WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return false
		}
		if len(stack) < 3 {
			return true
		}

		fID := findSurroundingFunc(pass, stack)
		if fID == nil || !fID.meta.isTestCleanup {
			return true
		}

		call := NewCallMeta(pass, node.(*ast.CallExpr))
		if call == nil {
			return true
		}

		if !call.IsAssert {
			d := newDiagnostic(checker.Name(), call, gracefulTeardownReport)
			diagnostics = append(diagnostics, *d)
		}
		return false
	})
	return diagnostics
}
