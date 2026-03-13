package main

import (
	"strings"
	"text/template"

	"github.com/Antonboom/testifylint/internal/checkers"
)

type RegexpTestsGenerator struct{}

func (RegexpTestsGenerator) Checker() checkers.Checker {
	return checkers.NewRegexp()
}

func (g RegexpTestsGenerator) TemplateData() any {
	var (
		checker           = g.Checker().Name()
		reportMustCompile = checker + ": remove unnecessary regexp.MustCompile"
		reportInvalidArg  = checker + ": use string or *regexp.Regexp as the first argument"
	)

	return struct {
		CheckerName           CheckerName
		InvalidMustCompile    []Assertion
		InvalidTypeAssertions []Assertion
		ValidAssertions       []Assertion
	}{
		CheckerName: CheckerName(checker),
		InvalidMustCompile: []Assertion{
			{
				Fn: "Regexp", Argsf: "regexp.MustCompile(`\\[.*\\] DEBUG \\(.*TestNew.*\\): message`), out",
				ReportMsgf: reportMustCompile, ProposedArgsf: "`\\[.*\\] DEBUG \\(.*TestNew.*\\): message`, out",
			},
			{
				Fn: "NotRegexp", Argsf: "regexp.MustCompile(`\\[.*\\] TRACE message`), out",
				ReportMsgf: reportMustCompile, ProposedArgsf: "`\\[.*\\] TRACE message`, out",
			},
		},
		InvalidTypeAssertions: []Assertion{
			{
				Fn:         "Regexp",
				Argsf:      "[]byte(`\\w+`), out",
				ReportMsgf: reportInvalidArg,
			},
			{
				Fn:         "NotRegexp",
				Argsf:      "[]byte(`\\w+`), out",
				ReportMsgf: reportInvalidArg,
			},
		},
		ValidAssertions: []Assertion{
			{Fn: "Regexp", Argsf: "`\\[.*\\] DEBUG \\(.*TestNew.*\\): message`, out"},
			{Fn: "NotRegexp", Argsf: "`\\[.*\\] TRACE message`, out"},
			{Fn: "Regexp", Argsf: "compiledRegexp, out"},
			{Fn: "NotRegexp", Argsf: "compiledRegexp, out"},
		},
	}
}

func (RegexpTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("RegexpTestsGenerator.ErroredTemplate").
		Funcs(fm).
		Parse(regexpTestTmpl))
}

func (RegexpTestsGenerator) GoldenTemplate() Executor {
	return template.Must(template.New("RegexpTestsGenerator.GoldenTemplate").
		Funcs(fm).
		Parse(strings.ReplaceAll(regexpTestTmpl, "NewAssertionExpander", "NewAssertionExpander.AsGolden")))
}

const regexpTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	var out string
	compiledRegexp := regexp.MustCompile(` + "`" + `\w+` + "`" + `)

	// Invalid: regexp.MustCompile usage.
	{
		{{- range $ai, $assrn := $.InvalidMustCompile }}
			{{ NewAssertionExpander.Expand $assrn "assert" "t" nil }}
		{{- end }}
	}

	// Invalid: non-string, non-*regexp.Regexp first argument.
	{
		{{- range $ai, $assrn := $.InvalidTypeAssertions }}
			{{ NewAssertionExpander.Expand $assrn "assert" "t" nil }}
		{{- end }}
	}

	// Valid.
	{
		{{- range $ai, $assrn := $.ValidAssertions }}
			{{ NewAssertionExpander.Expand $assrn "assert" "t" nil }}
		{{- end }}
	}
}
`
