package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const ellipsisSuffix = "..."

func scan(paths []string, kinds map[kind]bool) ([]pkg, error) {
	dirs, err := markerDirs(paths)
	if err != nil {
		return nil, err
	}

	var pkgs []pkg

	for _, dir := range dirs {
		found, err := scanDir(dir, kinds)
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, found...)
	}

	return pkgs, nil
}

// markerDirs treats a path ending in "..." as every package below the
// directory, as the go command does.
func markerDirs(paths []string) ([]string, error) {
	var dirs []string

	seen := make(map[string]bool)

	for _, path := range paths {
		root, err := rootDir(path)
		if err != nil {
			return nil, err
		}

		found, err := subdirs(root)
		if err != nil {
			return nil, err
		}

		for _, dir := range found {
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}

	return dirs, nil
}

func rootDir(path string) (string, error) {
	dir := filepath.Clean(strings.TrimSuffix(path, ellipsisSuffix))

	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("scan path %s: %w", path, err)
	}

	if !info.IsDir() {
		return filepath.Dir(dir), nil
	}

	return dir, nil
}

// subdirs walks root. The root itself is never pruned: filepath.WalkDir
// reports it under the name ".", which the hidden-directory rule matches.
func subdirs(root string) ([]string, error) {
	dirs := []string{root}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return fmt.Errorf("walk %s: %w", path, err)
		case !d.IsDir() || path == root:
			return nil
		case shouldSkipDir(d.Name()):
			return filepath.SkipDir
		}

		dirs = append(dirs, path)

		return nil
	})

	return dirs, err
}

// shouldSkipDir reports whether a directory tree is excluded from the scan.
// Hidden directories (".git", ".idea"), dependency trees ("vendor",
// "node_modules"), and fixture dirs ("testdata") never carry markers.
func shouldSkipDir(name string) bool {
	if name == "vendor" || name == "node_modules" || name == "testdata" {
		return true
	}

	return strings.HasPrefix(name, ".")
}

func scanDir(dir string, kinds map[kind]bool) ([]pkg, error) {
	files, err := goFiles(dir)
	if err != nil || len(files) == 0 {
		return nil, err
	}

	fset := token.NewFileSet()

	parsed := make([]*ast.File, 0, len(files))

	for _, name := range files {
		path := filepath.Join(dir, name)

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		parsed = append(parsed, file)
	}

	return resolvePackage(dir, parsed, fset, kinds)
}

// goFiles skips files the build excludes (build tags, GOOS/GOARCH suffixes):
// they can never reach the compiler, so their markers can never be
// referenced either.
func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || name == generatedFile {
			continue
		}

		match, err := build.Default.MatchFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("build constraints of %s: %w", filepath.Join(dir, name), err)
		}

		if match {
			names = append(names, name)
		}
	}

	return names, nil
}
