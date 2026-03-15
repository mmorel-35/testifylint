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
		CheckerName       CheckerName
		InvalidAssertions []Assertion
	}{
		CheckerName: CheckerName(checker),
		InvalidAssertions: []Assertion{
			{Fn: "Fail", Argsf: `"failure"`, ReportMsgf: checker + ": use t.Error or t.Errorf instead"},
			{Fn: "FailNow", Argsf: `"failure"`, ReportMsgf: checker + ": use t.Fatal or t.Fatalf instead"},
		},
	}
}

func (FailNowTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("FailNowTestsGenerator.ErroredTemplate").
		Funcs(fm).
		Parse(failNowTestTmpl))
}

func (FailNowTestsGenerator) GoldenTemplate() Executor {
	// NOTE: Only the developer understands the correct picture.
	return nil
}

const failNowTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	// Invalid.
	{
		{{- range $ai, $assrn := $.InvalidAssertions }}
			{{ NewAssertionExpander.FullMode.Expand $assrn "assert" "t" nil }}
			{{ NewAssertionExpander.FullMode.Expand $assrn "require" "t" nil }}
		{{- end }}
	}
}
`
