package analysisutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

func TestPkgBaseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
<<<<<<< HEAD
		importPath string
		want       string
	}{
		{"net/http", "http"},
		{"fmt", "fmt"},
		{"github.com/stretchr/testify/assert", "assert"},
		{"example.com/pkg/v2", "pkg"},
		{"example.com/pkg/v10", "pkg"},
		{"example.com/v2", "example.com"},        // version at top level: falls back to domain
		{"example.com/mypkg/v2alpha", "v2alpha"}, // not a pure integer suffix
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			t.Parallel()

			got := analysisutil.PkgBaseName(tt.importPath)
			if got != tt.want {
				t.Errorf("PkgBaseName(%q) = %q, want %q", tt.importPath, got, tt.want)
=======
		name       string
		importPath string
		want       string
	}{
		{name: "plain", importPath: "github.com/stretchr/testify/require", want: "require"},
		{name: "module v2 suffix", importPath: "example.com/pkg/v2", want: "pkg"},
		{name: "module v1 path segment", importPath: "example.com/v1", want: "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := analysisutil.PkgBaseName(tt.importPath)
			if got != tt.want {
				t.Fatalf("PkgBaseName(%q) = %q, want %q", tt.importPath, got, tt.want)
>>>>>>> origin/master
			}
		})
	}
}

func TestImports(t *testing.T) {
	fset := token.NewFileSet()

	const src = `package simple

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimple(t *testing.T) {
	assert.Equal(t, 4, 2*2)
}`

	f, err := parser.ParseFile(fset, "", src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}

	notImported := make([]string, 0, 7)
	notImported = append(notImported,
		"",
		"net/http",
		"net/http/httptest",
		"github.com/stretchr/testify/suite",
		"github.com/stretchr/testify/require",
		"vendor/github.com/stretchr/testify/require",
	)
	if analysisutil.Imports(f, notImported...) {
		t.FailNow()
	}
	if !analysisutil.Imports(f, slices.Concat(notImported, []string{"testing"})...) {
		t.FailNow()
	}
	if !analysisutil.Imports(f, "github.com/stretchr/testify/assert") {
		t.FailNow()
	}
}

func TestLocalPkgName(t *testing.T) {
	t.Parallel()

<<<<<<< HEAD
	tests := []struct {
		name     string
		src      string
		pkgPath  string
		wantName string
		wantOK   bool
	}{
		{
			name: "regular import uses last path element",
			src: `package p
import "net/http"
var _ = http.Get`,
			pkgPath:  "net/http",
			wantName: "http",
			wantOK:   true,
		},
		{
			name: "aliased import returns alias",
			src: `package p
import stdhttp "net/http"
var _ = stdhttp.Get`,
			pkgPath:  "net/http",
			wantName: "stdhttp",
			wantOK:   true,
		},
		{
			name: "dot-import returns empty string with ok=true",
			src: `package p
import . "net/http"
var _ = Get`,
			pkgPath:  "net/http",
			wantName: "",
			wantOK:   true,
		},
		{
			name: "blank import returns empty string with ok=false",
			src: `package p
import _ "net/http"`,
			pkgPath:  "net/http",
			wantName: "",
			wantOK:   false,
		},
		{
			name: "package not imported returns empty string with ok=false",
			src: `package p
import "fmt"`,
			pkgPath:  "net/http",
			wantName: "",
			wantOK:   false,
		},
		{
			name: "versioned module path returns non-version element",
			src: `package p
import "example.com/pkg/v2"
`,
			pkgPath:  "example.com/pkg/v2",
			wantName: "pkg",
			wantOK:   true,
		},
		{
			name: "versioned module path with alias returns alias",
			src: `package p
import mypkg "example.com/pkg/v2"
`,
			pkgPath:  "example.com/pkg/v2",
			wantName: "mypkg",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", tt.src, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			// Use the file's own start position so LocalPkgName finds the file.
			pos := f.Pos()
			name, ok := analysisutil.LocalPkgName([]*ast.File{f}, pos, tt.pkgPath)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
=======
	const src = `package localpkgname

import (
	"github.com/stretchr/testify/assert"
	r "github.com/stretchr/testify/require"
	. "net/http"
	_ "math"
)

func TestLocalPkgName() {
	assert.Equal(nil, 1, 1)
	r.Len(nil, []int{1}, 1)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}

	pos := token.Pos(1)
	assertName, ok := analysisutil.LocalPkgName([]*ast.File{file}, pos, "github.com/stretchr/testify/assert")
	if !ok || assertName != "assert" {
		t.Fatalf("LocalPkgName(assert) = (%q, %t), want (assert, true)", assertName, ok)
	}

	requireName, ok := analysisutil.LocalPkgName([]*ast.File{file}, pos, "github.com/stretchr/testify/require")
	if !ok || requireName != "r" {
		t.Fatalf("LocalPkgName(require) = (%q, %t), want (r, true)", requireName, ok)
	}

	dotName, ok := analysisutil.LocalPkgName([]*ast.File{file}, pos, "net/http")
	if !ok || dotName != "" {
		t.Fatalf("LocalPkgName(dot) = (%q, %t), want (\"\", true)", dotName, ok)
	}

	blankName, ok := analysisutil.LocalPkgName([]*ast.File{file}, pos, "math")
	if ok || blankName != "" {
		t.Fatalf("LocalPkgName(blank) = (%q, %t), want (\"\", false)", blankName, ok)
>>>>>>> origin/master
	}
}
