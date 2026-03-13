package main

import (
	"text/template"

	"github.com/Antonboom/testifylint/internal/checkers"
)

type HTTPMultipleTestsGenerator struct{}

func (HTTPMultipleTestsGenerator) Checker() checkers.Checker {
	return checkers.NewHTTPMultiple()
}

func (g HTTPMultipleTestsGenerator) TemplateData() any {
	checker := g.Checker().Name()
	report := checker + ": use httptest.NewRecorder() instead of multiple HTTP assertions for the same handler call"

	return struct {
		CheckerName CheckerName
		Report      string
	}{
		CheckerName: CheckerName(checker),
		Report:      report,
	}
}

func (HTTPMultipleTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("HTTPMultipleTestsGenerator.ErroredTemplate").
		Funcs(fm).
		Parse(httpMultipleTestTmpl))
}

func (HTTPMultipleTestsGenerator) GoldenTemplate() Executor {
	// NOTE: Usually this warning leads to full refactoring of test architecture.
	return nil
}

const httpMultipleTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handler(w http.ResponseWriter, r *http.Request) {}

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	// Invalid: multiple HTTP assertions with the same handler and args.
	assert.HTTPStatusCode(t, handler, "GET", "/index", nil, http.StatusOK)    // want {{ QuoteReport .Report }}
	assert.HTTPBodyContains(t, handler, "GET", "/index", nil, "hello")        // want {{ QuoteReport .Report }}

	assert.HTTPError(t, handler, "GET", "/error", nil)               // want {{ QuoteReport .Report }}
	assert.HTTPBodyContains(t, handler, "GET", "/error", nil, "oops") // want {{ QuoteReport .Report }}

	require.HTTPRedirect(t, handler, "GET", "/redirect", nil)                        // want {{ QuoteReport .Report }}
	require.HTTPStatusCode(t, handler, "GET", "/redirect", nil, http.StatusFound)    // want {{ QuoteReport .Report }}

	// Invalid: formatted (*f) variants are also detected.
	assert.HTTPStatusCodef(t, handler, "GET", "/fmt", nil, http.StatusOK, "msg")   // want {{ QuoteReport .Report }}
	assert.HTTPBodyContainsf(t, handler, "GET", "/fmt", nil, "hello", "msg")       // want {{ QuoteReport .Report }}

	// Valid: single HTTP assertion.
	assert.HTTPStatusCode(t, handler, "GET", "/single", nil, http.StatusOK)

	// Valid: different handlers.
	assert.HTTPSuccess(t, handler, "GET", "/a", nil)
	assert.HTTPSuccess(t, http.NotFound, "GET", "/a", nil)

	// Valid: different methods.
	assert.HTTPSuccess(t, handler, "GET", "/b", nil)
	assert.HTTPSuccess(t, handler, "POST", "/b", nil)

	// Valid: different URLs.
	assert.HTTPSuccess(t, handler, "GET", "/c", nil)
	assert.HTTPSuccess(t, handler, "GET", "/d", nil)

	// Valid: same-key assertions in separate closures are independent scopes.
	t.Run("sub1", func(t *testing.T) {
		assert.HTTPStatusCode(t, handler, "GET", "/subtest", nil, http.StatusOK)
	})
	t.Run("sub2", func(t *testing.T) {
		assert.HTTPBodyContains(t, handler, "GET", "/subtest", nil, "hello")
	})

	// Valid: goroutine closure is an independent scope.
	go func() {
		assert.HTTPStatusCode(t, handler, "GET", "/goroutine", nil, http.StatusOK)
	}()
	go func() {
		assert.HTTPBodyContains(t, handler, "GET", "/goroutine", nil, "hello")
	}()
}
`
