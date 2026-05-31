package analysisutil

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
)

// PkgBaseName returns the default package name for the given import path.
<<<<<<< HEAD
// It handles versioned module paths (e.g., "example.com/pkg/v2" → "pkg").
func PkgBaseName(importPath string) string {
	base := path.Base(importPath)
	// If the last element looks like a major version tag (v2, v3, ...), use the preceding element.
=======
// It handles versioned module paths (e.g. "example.com/pkg/v2" -> "pkg").
func PkgBaseName(importPath string) string {
	base := path.Base(importPath)
>>>>>>> origin/master
	after, found := strings.CutPrefix(base, "v")
	if !found {
		return base
	}
<<<<<<< HEAD
	_, err := strconv.Atoi(after)
	if err != nil {
=======
	v, err := strconv.Atoi(after)
	if err != nil || v < 2 {
>>>>>>> origin/master
		return base
	}
	return path.Base(path.Dir(importPath))
}

// Imports tells if the file imports at least one of the packages.
// If no packages provided then function returns false.
func Imports(file *ast.File, pkgs ...string) bool {
	for _, i := range file.Imports {
		if i.Path == nil {
			continue
		}

		importPath, err := strconv.Unquote(i.Path.Value)
		if err != nil {
			continue
		}
		if slices.Contains(pkgs, importPath) { // Small O(n).
			return true
		}
	}
	return false
}

// LocalPkgName returns the local name used for an import with the given path in the
// source file that contains pos.
//
<<<<<<< HEAD
// It returns ("", true) for dot-imports (the package's exported identifiers are
// accessible without a qualifier), ("name", true) for regular or aliased imports
// (where name is either the alias or the last element of the import path), and
// ("", false) when the import is not found in the file or uses a blank name "_".
=======
// It returns ("", true) for dot-imports, ("name", true) for regular or aliased
// imports and ("", false) when the import isn't found in the file or uses "_".
>>>>>>> origin/master
func LocalPkgName(files []*ast.File, pos token.Pos, pkgPath string) (string, bool) {
	for _, file := range files {
		if file.Pos() > pos || pos > file.End() {
			continue
		}
		for _, imp := range file.Imports {
<<<<<<< HEAD
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil || p != pkgPath {
=======
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil || importPath != pkgPath {
>>>>>>> origin/master
				continue
			}
			if imp.Name != nil {
				if imp.Name.Name == "." {
<<<<<<< HEAD
					return "", true // dot-import: symbols accessible unqualified
				}
				if imp.Name.Name == "_" {
					return "", false // blank import: package not accessible
=======
					return "", true
				}
				if imp.Name.Name == "_" {
					return "", false
>>>>>>> origin/master
				}
				return imp.Name.Name, true
			}
			return PkgBaseName(pkgPath), true
		}
		break
	}
	return "", false
}
