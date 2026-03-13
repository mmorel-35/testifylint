package main

import (
	"text/template"

	"github.com/Antonboom/testifylint/internal/checkers"
)

type ErrorCompareTestsGenerator struct{}

func (ErrorCompareTestsGenerator) Checker() checkers.Checker {
	return checkers.NewErrorCompare()
}

func (g ErrorCompareTestsGenerator) TemplateData() any {
	checker := g.Checker().Name()

	return struct {
		CheckerName CheckerName
	}{
		CheckerName: CheckerName(checker),
	}
}

func (ErrorCompareTestsGenerator) ErroredTemplate() Executor {
	return template.Must(template.New("ErrorCompareTestsGenerator.ErroredTemplate").
		Funcs(fm).
		Parse(errorCompareTestTmpl))
}

func (ErrorCompareTestsGenerator) GoldenTemplate() Executor {
	return template.Must(template.New("ErrorCompareTestsGenerator.GoldenTemplate").
		Funcs(fm).
		Parse(errorCompareGoldenTmpl))
}

const errorCompareTestTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	var (
		errSentinel = errors.New("user not found")
		err         error
	)

	// Invalid.
	assert.Contains(t, err.Error(), "user not found")                                              // want "error-compare: use assert\\.ErrorContains"
	assert.Containsf(t, err.Error(), "user not found", "msg with args %d %s", 42, "42")           // want "error-compare: use assert\\.ErrorContainsf"
	assert.Contains(t, err.Error(), errSentinel.Error())                                           // want "error-compare: use assert\\.ErrorIs"
	assert.Containsf(t, err.Error(), errSentinel.Error(), "msg with args %d %s", 42, "42")        // want "error-compare: use assert\\.ErrorIsf"
	assert.Equal(t, err.Error(), "user not found")                                                 // want "error-compare: use assert\\.EqualError"
	assert.Equalf(t, err.Error(), "user not found", "msg with args %d %s", 42, "42")              // want "error-compare: use assert\\.EqualErrorf"
	assert.Equal(t, "user not found", err.Error())                                                 // want "error-compare: use assert\\.EqualError"
	assert.Equalf(t, "user not found", err.Error(), "msg with args %d %s", 42, "42")              // want "error-compare: use assert\\.EqualErrorf"
	assert.Equal(t, err, errSentinel)                                                              // want "error-compare: use assert\\.ErrorIs"
	assert.Equal(t, errSentinel, err)                                                              // want "error-compare: use assert\\.ErrorIs"
	assert.NotEqual(t, err, errSentinel)                                                           // want "error-compare: use assert\\.NotErrorIs"
	assert.NotEqual(t, errSentinel, err)                                                           // want "error-compare: use assert\\.NotErrorIs"

	// Valid.
	assert.ErrorContains(t, err, "user not found")
	assert.ErrorIs(t, err, errSentinel)
	assert.NotErrorIs(t, err, errSentinel)
	assert.EqualError(t, err, "user not found")

	// Not errors.
	str1, str2 := "foo", "bar"
	assert.Equal(t, str1, str2)
	assert.NotEqual(t, str1, str2)
	assert.Contains(t, str1, str2)
}
`

// Golden file: contains the expected fixed code.
// Cases with autofix (Contains→ErrorContains/ErrorIs, Equal(err.Error())→EqualError) are fixed.
// Cases without autofix (Equal/NotEqual comparing two errors) remain unchanged.
const errorCompareGoldenTmpl = header + `

package {{ .CheckerName.AsPkgName }}

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func {{ .CheckerName.AsTestName }}(t *testing.T) {
	var (
		errSentinel = errors.New("user not found")
		err         error
	)

	// Invalid.
	assert.ErrorContains(t, err, "user not found")                                              // want "error-compare: use assert\\.ErrorContains"
	assert.ErrorContainsf(t, err, "user not found", "msg with args %d %s", 42, "42")           // want "error-compare: use assert\\.ErrorContainsf"
	assert.ErrorIs(t, err, errSentinel)                                                         // want "error-compare: use assert\\.ErrorIs"
	assert.ErrorIsf(t, err, errSentinel, "msg with args %d %s", 42, "42")                      // want "error-compare: use assert\\.ErrorIsf"
	assert.EqualError(t, err, "user not found")                                                 // want "error-compare: use assert\\.EqualError"
	assert.EqualErrorf(t, err, "user not found", "msg with args %d %s", 42, "42")              // want "error-compare: use assert\\.EqualErrorf"
	assert.EqualError(t, err, "user not found")                                                 // want "error-compare: use assert\\.EqualError"
	assert.EqualErrorf(t, err, "user not found", "msg with args %d %s", 42, "42")              // want "error-compare: use assert\\.EqualErrorf"
	assert.Equal(t, err, errSentinel)                                                           // want "error-compare: use assert\\.ErrorIs"
	assert.Equal(t, errSentinel, err)                                                           // want "error-compare: use assert\\.ErrorIs"
	assert.NotEqual(t, err, errSentinel)                                                        // want "error-compare: use assert\\.NotErrorIs"
	assert.NotEqual(t, errSentinel, err)                                                        // want "error-compare: use assert\\.NotErrorIs"

	// Valid.
	assert.ErrorContains(t, err, "user not found")
	assert.ErrorIs(t, err, errSentinel)
	assert.NotErrorIs(t, err, errSentinel)
	assert.EqualError(t, err, "user not found")

	// Not errors.
	str1, str2 := "foo", "bar"
	assert.Equal(t, str1, str2)
	assert.NotEqual(t, str1, str2)
	assert.Contains(t, str1, str2)
}
`
