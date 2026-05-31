package requirelen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLenGuard(t *testing.T) {
	arr := []int{0, 1}

	assert.Len(t, arr, 2) // want "require-len: for length assertions guarding index access use require"
	assert.Positive(t, arr[1])
}

func TestLenGuardf(t *testing.T) {
	arr := []int{0, 1}

	assert.Lenf(t, arr, 2, "msg") // want "require-len: for length assertions guarding index access use require"
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

func TestLenGuardInsertedFromIndexAccess(t *testing.T) {
	arr := []int{0, 1}

	assert.Positive(t, arr[1]) // want "require-len: for indexed access use require\\.Len guard"
}

func TestLenGuardInsertedUsesGreatestIndex(t *testing.T) {
	arr := []int{0, 1, 2}

	assert.Equal(t, arr[0]+arr[2], 2) // want "require-len: for indexed access use require\\.Len guard"
}

func TestLenGuardInsertedSingleIndexUsesNotEmpty(t *testing.T) {
	arr := []int{0}

	assert.Positive(t, arr[0]) // want "require-len: for indexed access use require\\.Len guard"
}
