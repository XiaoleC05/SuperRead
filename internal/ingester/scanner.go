package ingester

import (
	"os"
	"path/filepath"
	"strings"
)

var toolRepos = []string{
	"AIHelper", "SuperRead", "SecretStore", "CS2Lab",
	"DormGuard", "MusicBox", "AgentCanvas", "XiaoleC05.github.io",
}

func getCodeRoot() string {
	root := os.Getenv("SMARTKB_CODE_ROOT")
	if root == "" {
		root = `D:\07_Projects\code`
	}
	return root
}

// platformPaths returns candidate Oxelia51 source dirs to scan for docs.
func platformPaths() []string {
	return []string{
		filepath.Join(getCodeRoot(), "Oxelia51"),
		"/opt/Oxelia51-src",
		"/opt/Oxelia51",
	}
}

// findFirstDir returns the first directory in candidates that exists.
func findFirstDir(candidates ...string) string {
	for _, d := range candidates {
		if _, err := os.Stat(d); err == nil {
			return d
		}
	}
	return ""
}

// ScanFiles returns all .md and .go files to ingest.
func ScanFiles() ([]string, error) {
	codeRoot := getCodeRoot()
	var files []string

	// 1. Oxelia51 docs/**/*.md (try multiple possible paths)
	oxRoot := findFirstDir(platformPaths()...)
	if oxRoot != "" {
		docsDir := filepath.Join(oxRoot, "docs")
		filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") {
				files = append(files, path)
			}
			return nil
		})

		// 2. Oxelia51 root *.md
		for _, name := range []string{"AGENTS.md", "CLAUDE.md", "README.md", "CHANGELOG.md"} {
			p := filepath.Join(oxRoot, name)
			if _, err := os.Stat(p); err == nil {
				files = append(files, p)
			}
		}
	}

	// 3. Tool repos: root *.md, internal/**/*.go, cmd/**/*.go
	for _, repo := range toolRepos {
		repoPath := filepath.Join(codeRoot, repo)

		// Root *.md files
		entries, _ := os.ReadDir(repoPath)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				files = append(files, filepath.Join(repoPath, e.Name()))
			}
		}

		// internal/**/*.go
		internalDir := filepath.Join(repoPath, "internal")
		filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			return nil
		})

		// cmd/**/*.go
		cmdDir := filepath.Join(repoPath, "cmd")
		filepath.Walk(cmdDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			return nil
		})
	}

	return files, nil
}

// FileModifiedSince checks if file mtime is after the given unix timestamp.
func FileModifiedSince(path string, since int64) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return info.ModTime().Unix() > since
}