package requireerrorskiplogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that require-error reports assert.Error/require.Error as "redundant assertion" when
// immediately followed by EqualError or ErrorContains on the same variable, instead
// of reporting "for error assertions use require".
func Test_RequireError_RedundantError(t *testing.T) {
	var err error

	// assert.Error is reported as "redundant assertion" (not "use require").
	// The following EqualError/ErrorContains are still flagged as "use require".
	assert.Error(t, err) // want "require-error: redundant assertion"
	assert.EqualError(t, err, "end of file") // want "require-error: for error assertions use require"

	assert.Error(t, err) // want "require-error: redundant assertion"
	assert.ErrorContains(t, err, "end of file") // want "require-error: for error assertions use require"

	// require.Error is also reported as "redundant assertion".
	require.Error(t, err) // want "require-error: redundant assertion"
	assert.EqualError(t, err, "end of file") // want "require-error: for error assertions use require"

	require.Error(t, err) // want "require-error: redundant assertion"
	assert.ErrorContains(t, err, "end of file") // want "require-error: for error assertions use require"

	// assert.Error IS still flagged as "use require" when the next assertion is on a different variable.
	var err2 error
	assert.Error(t, err)                        // want "require-error: for error assertions use require"
	assert.EqualError(t, err2, "end of file")   // want "require-error: for error assertions use require"

	// assert.Errorf (formatted version) is NOT treated as redundant.
	assert.Errorf(t, err, "checking error") // want "require-error: for error assertions use require"
	assert.EqualError(t, err, "end of file") // want "require-error: for error assertions use require"

	// assert.Error followed by ErrorIs/ErrorAs is NOT redundant (legitimate defensive pattern).
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError) // want "require-error: for error assertions use require"

	nopRedundant2()
}

func nopRedundant2() {}
