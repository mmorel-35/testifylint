package requireerrorskiplogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLenGuard(t *testing.T) {
	arr := []int{0, 1}

	assert.Len(t, arr, 2) // want "require-error: for length assertions guarding index access use require"
	assert.Positive(t, arr[1])
}

func TestLenGuardf(t *testing.T) {
	arr := []int{0, 1}

	assert.Lenf(t, arr, 2, "msg") // want "require-error: for length assertions guarding index access use require"
	assert.Positive(t, arr[1])
}

func TestLenGuardRequire(t *testing.T) {
	arr := []int{0, 1}

	require.Len(t, arr, 2)
	assert.Positive(t, arr[1])
}

func TestLenGuardIndexOutOfCheckedRange(t *testing.T) {
	arr := []int{0, 1}

	assert.Len(t, arr, 2)
	assert.Positive(t, arr[2])
}
