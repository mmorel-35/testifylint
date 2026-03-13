package requireerror

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRequireErrorChecker_RedundantError(t *testing.T) {
	var err error

	assObj, reqObj := assert.New(t), require.New(t)

	// Invalid: assert.Error/require.Error is redundant when immediately followed
	// by EqualError or ErrorContains on the same variable.
	// Both assert.Error and require.Error are flagged as "redundant assertion".
	assert.Error(t, err)   // want "require-error: redundant assertion"
	require.EqualError(t, err, "end of file")

	assert.Error(t, err)   // want "require-error: redundant assertion"
	require.ErrorContains(t, err, "end of file")

	require.Error(t, err)   // want "require-error: redundant assertion"
	require.EqualError(t, err, "end of file")

	require.Error(t, err)   // want "require-error: redundant assertion"
	require.ErrorContains(t, err, "end of file")

	// Object form.
	assObj.Error(err)   // want "require-error: redundant assertion"
	reqObj.EqualError(err, "end of file")

	assObj.Error(err)   // want "require-error: redundant assertion"
	reqObj.ErrorContains(err, "end of file")

	reqObj.Error(err)   // want "require-error: redundant assertion"
	reqObj.EqualError(err, "end of file")

	reqObj.Error(err)   // want "require-error: redundant assertion"
	reqObj.ErrorContains(err, "end of file")

	// Valid: Error followed by ErrorIs/ErrorAs is NOT redundant.
	require.Error(t, err)
	require.ErrorIs(t, err, assert.AnError)

	// Valid: last call in block.
	nopRequireRedundant()
	require.Error(t, err)
}

type RequireErrorCheckerRedundantSuite struct {
	suite.Suite
}

func TestRequireErrorCheckerRedundantSuite(t *testing.T) {
	suite.Run(t, new(RequireErrorCheckerRedundantSuite))
}

func (s *RequireErrorCheckerRedundantSuite) TestAll() {
	var err error

	// Invalid.
	s.Error(err)   // want "require-error: redundant assertion"
	s.Require().EqualError(err, "end of file")

	s.Error(err)   // want "require-error: redundant assertion"
	s.Require().ErrorContains(err, "end of file")

	s.Assert().Error(err)   // want "require-error: redundant assertion"
	s.Require().EqualError(err, "end of file")

	s.Require().Error(err)   // want "require-error: redundant assertion"
	s.Require().EqualError(err, "end of file")
}

func nopRequireRedundant() {}
