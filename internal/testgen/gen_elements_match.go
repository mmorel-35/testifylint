package main

import (
	"text/template"

	"github.com/Antonboom/testifylint/internal/checkers"
)

type ElementsMatchTestsGenerator struct{}

func (ElementsMatchTestsGenerator) Checker() checkers.Checker {
	return checkers.NewElementsMatch()
}

func (g ElementsMatchTestsGenerator) TemplateData() any {
	checker := g.Checker().Name()
	report := QuoteReport(checker + ": " + "use assert.ElementsMatch")

	return struct {
		CheckerName CheckerName
		Report      string
	}{
		CheckerName: CheckerName(checker),
		Report:      report,
	}
}

func (ElementsMatchTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("ElementsMatchTestsGenerator.ErroredTemplate").
		Parse(elementsMatchTestTmpl))
}

func (ElementsMatchTestsGenerator) GoldenTemplate() Executor {
	return template.Must(template.New("ElementsMatchTestsGenerator.GoldenTemplate").
		Parse(elementsMatchGoldenTmpl))
}

const elementsMatchTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	var a, b []int

	// Invalid.
	{
		slices.Sort(a)
		slices.Sort(b)
		assert.True(t, slices.Equal(a, b)) // want {{ $.Report }}

		slices.Sort(b)
		slices.Sort(a)
		assert.True(t, slices.Equal(a, b)) // want {{ $.Report }}

		slices.Sort(a)
		slices.Sort(b)
		require.True(t, slices.Equal(a, b)) // want {{ $.Report }}
	}

	// Valid.
	{
		assert.ElementsMatch(t, a, b)

		// Only one sort call preceding assert.
		slices.Sort(a)
		assert.True(t, slices.Equal(a, b))

		// Sort args don't match Equal args.
		slices.Sort(a)
		slices.Sort(b)
		assert.True(t, slices.Equal(a, a))

		// Not a slices.Equal call.
		slices.Sort(a)
		slices.Sort(b)
		assert.True(t, a[0] == b[0])

		// Not assert.True.
		slices.Sort(a)
		slices.Sort(b)
		assert.Equal(t, a, b)
	}
}
`

const elementsMatchGoldenTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	var a, b []int

	// Invalid.
	{
		assert.ElementsMatch(t, a, b) // want {{ $.Report }}

		assert.ElementsMatch(t, a, b) // want {{ $.Report }}

		require.ElementsMatch(t, a, b) // want {{ $.Report }}
	}

	// Valid.
	{
		assert.ElementsMatch(t, a, b)

		// Only one sort call preceding assert.
		slices.Sort(a)
		assert.True(t, slices.Equal(a, b))

		// Sort args don't match Equal args.
		slices.Sort(a)
		slices.Sort(b)
		assert.True(t, slices.Equal(a, a))

		// Not a slices.Equal call.
		slices.Sort(a)
		slices.Sort(b)
		assert.True(t, a[0] == b[0])

		// Not assert.True.
		slices.Sort(a)
		slices.Sort(b)
		assert.Equal(t, a, b)
	}
}
`
