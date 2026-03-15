package requireerrorskiplogic

import (
"testing"

"github.com/stretchr/testify/assert"
)

// TestNegatedIfSkippedByRequireError verifies that require-error correctly
// skips all `if !assert.xxx { return/continue }` patterns — they are handled
// by the dedicated negated-assert checker instead.
func TestNegatedIfSkippedByRequireError(t *testing.T) {
var err error

// All of these are handled by negated-assert, not require-error.
if !assert.NoError(t, err) {
return
}
if !assert.NoErrorf(t, err, "msg") {
return
}
if !assert.Error(t, err) {
return
}
if !assert.NoError(t, err) || !assert.Error(t, err) {
return
}
if !assert.Contains(t, "str", "s") {
return
}
for i := 0; i < 10; i++ {
if !assert.NoError(t, err) {
continue
}
}

// These patterns are correctly skipped by both checkers.

// Complex body — skipped (not a simple return/continue).
if !assert.NoError(t, err) {
_ = "something"
return
}
// Positive condition — skipped (not negated).
if assert.NoError(t, err) {
_ = err
}
// Has else — skipped.
if !assert.NoError(t, err) {
return
} else {
_ = err
}
// Else-if chain — outer has else, skipped.
if !assert.NoError(t, err) {
return
} else if !assert.Error(t, err) {
return
}
}
