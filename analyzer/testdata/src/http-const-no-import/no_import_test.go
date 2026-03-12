package httpconstnoimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHTTPConstNoImport verifies that the autofix adds the "net/http" import
// when it is absent from the file.
func TestHTTPConstNoImport(t *testing.T) {
	assert.HTTPStatusCode(t, handleOK, "GET", "/", nil, 200) // want "http-const: use net/http constants instead of value"
}
