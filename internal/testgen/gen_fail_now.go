package main

import (
	"text/template"

	"github.com/Antonboom/testifylint/internal/checkers"
)

type FailNowTestsGenerator struct{}

func (FailNowTestsGenerator) Checker() checkers.Checker {
	return checkers.NewFailNow()
}

func (g FailNowTestsGenerator) TemplateData() any {
	checker := g.Checker().Name()
	return struct {
		CheckerName CheckerName
		FailReport  string
		NowReport   string
	}{
		CheckerName: CheckerName(checker),
		FailReport:  checker + ": use t.Error or t.Errorf instead",
		NowReport:   checker + ": use t.Fatal or t.Fatalf instead",
	}
}

func (FailNowTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("FailNowTestsGenerator.ErroredTemplate").
		Funcs(fm).
		Parse(failNowTestTmpl))
}

func (FailNowTestsGenerator) GoldenTemplate() Executor {
	return template.Must(template.New("FailNowTestsGenerator.GoldenTemplate").
		Funcs(fm).
		Parse(failNowGoldenTmpl))
}

const failNowTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/stretchr/testify/suite"
)

// {{ .CheckerName.AsSuiteName }} is a suite to test suite method call fixes.
type {{ .CheckerName.AsSuiteName }} struct {
suite.Suite
}

func Test{{ .CheckerName.AsSuiteName }}(t *testing.T) {
suite.Run(t, new({{ .CheckerName.AsSuiteName }}))
}

func (s *{{ .CheckerName.AsSuiteName }}) TestFixSuiteMethod() {
// Suite method calls: fixed via s.T().Error/Fatal.
s.Fail("failure") // want {{ QuoteReport .FailReport }}
s.Fail("failure", "extra msg") // want {{ QuoteReport .FailReport }}
s.Fail("failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}
s.Failf("failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}
s.FailNow("failure") // want {{ QuoteReport .NowReport }}
s.FailNow("failure", "extra msg") // want {{ QuoteReport .NowReport }}
s.FailNow("failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
s.FailNowf("failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
}

func {{ .CheckerName.AsTestName }}(t *testing.T) {
// Invalid.
{
// Fail: single arg → t.Error(failureMessage).
assert.Fail(t, "failure") // want {{ QuoteReport .FailReport }}
require.Fail(t, "failure") // want {{ QuoteReport .FailReport }}
// Fail: two args → t.Error(msg), drop failureMessage.
assert.Fail(t, "failure", "extra msg") // want {{ QuoteReport .FailReport }}
require.Fail(t, "failure", "extra msg") // want {{ QuoteReport .FailReport }}
// Fail: three+ args → t.Errorf(format, args...), drop failureMessage.
assert.Fail(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}
require.Fail(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}
// Failf: drop failureMessage, keep format + args.
assert.Failf(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}
require.Failf(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .FailReport }}

// FailNow: single arg → t.Fatal(failureMessage).
assert.FailNow(t, "failure") // want {{ QuoteReport .NowReport }}
require.FailNow(t, "failure") // want {{ QuoteReport .NowReport }}
// FailNow: two args → t.Fatal(msg), drop failureMessage.
assert.FailNow(t, "failure", "extra msg") // want {{ QuoteReport .NowReport }}
require.FailNow(t, "failure", "extra msg") // want {{ QuoteReport .NowReport }}
// FailNow: three+ args → t.Fatalf(format, args...), drop failureMessage.
assert.FailNow(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
require.FailNow(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
// FailNowf: drop failureMessage, keep format + args.
assert.FailNowf(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
require.FailNowf(t, "failure", "fmt %s", "arg") // want {{ QuoteReport .NowReport }}
}
}
`

const failNowGoldenTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/stretchr/testify/suite"
)

// {{ .CheckerName.AsSuiteName }} is a suite to test suite method call fixes.
type {{ .CheckerName.AsSuiteName }} struct {
suite.Suite
}

func Test{{ .CheckerName.AsSuiteName }}(t *testing.T) {
suite.Run(t, new({{ .CheckerName.AsSuiteName }}))
}

func (s *{{ .CheckerName.AsSuiteName }}) TestFixSuiteMethod() {
// Suite method calls: fixed via s.T().Error/Fatal.
s.T().Error("failure") // want {{ QuoteReport .FailReport }}
s.T().Error("extra msg") // want {{ QuoteReport .FailReport }}
s.T().Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}
s.T().Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}
s.T().Fatal("failure") // want {{ QuoteReport .NowReport }}
s.T().Fatal("extra msg") // want {{ QuoteReport .NowReport }}
s.T().Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
s.T().Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
}

func {{ .CheckerName.AsTestName }}(t *testing.T) {
// Invalid.
{
// Fail: single arg → t.Error(failureMessage).
t.Error("failure") // want {{ QuoteReport .FailReport }}
t.Error("failure") // want {{ QuoteReport .FailReport }}
// Fail: two args → t.Error(msg), drop failureMessage.
t.Error("extra msg") // want {{ QuoteReport .FailReport }}
t.Error("extra msg") // want {{ QuoteReport .FailReport }}
// Fail: three+ args → t.Errorf(format, args...), drop failureMessage.
t.Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}
t.Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}
// Failf: drop failureMessage, keep format + args.
t.Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}
t.Errorf("fmt %s", "arg") // want {{ QuoteReport .FailReport }}

// FailNow: single arg → t.Fatal(failureMessage).
t.Fatal("failure") // want {{ QuoteReport .NowReport }}
t.Fatal("failure") // want {{ QuoteReport .NowReport }}
// FailNow: two args → t.Fatal(msg), drop failureMessage.
t.Fatal("extra msg") // want {{ QuoteReport .NowReport }}
t.Fatal("extra msg") // want {{ QuoteReport .NowReport }}
// FailNow: three+ args → t.Fatalf(format, args...), drop failureMessage.
t.Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
t.Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
// FailNowf: drop failureMessage, keep format + args.
t.Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
t.Fatalf("fmt %s", "arg") // want {{ QuoteReport .NowReport }}
}
}
`
