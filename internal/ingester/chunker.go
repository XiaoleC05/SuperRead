package ingester

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

type Chunk struct {
	Content    string
	SourceLine int
}

// ChunkMarkdown splits markdown by ## headers, <=1000 chars, 100 char overlap.
func ChunkMarkdown(content string) []Chunk {
	var chunks []Chunk

	sections := strings.Split(content, "\n## ")
	for i, section := range sections {
		if i > 0 {
			section = "## " + section
		}
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		if len(section) <= 1000 {
			chunks = append(chunks, Chunk{Content: section})
		} else {
			for start := 0; start < len(section); {
				end := start + 1000
				if end > len(section) {
					end = len(section)
				}
				chunks = append(chunks, Chunk{Content: section[start:end]})
				if end >= len(section) {
					break
				}
				start = end - 100
				if start < 0 {
					start = 0
				}
			}
		}
	}
	return chunks
}

// ChunkGoSource parses Go file with go/parser, each func/method becomes a chunk.
func ChunkGoSource(filePath string) ([]Chunk, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	pkgName := f.Name.Name
	var chunks []Chunk

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fn); err != nil {
			continue
		}

		content := buf.String()
		content = fmt.Sprintf("package %s\n\n%s", pkgName, content)
		if len(content) > 1000 {
			content = content[:1000]
		}

		line := fset.Position(fn.Pos()).Line
		chunks = append(chunks, Chunk{
			Content:    content,
			SourceLine: line,
		})
	}
	return chunks, nil
}

// ChunkFile dispatches to the right chunker based on file extension.
func ChunkFile(filePath string) ([]Chunk, error) {
	if strings.HasSuffix(filePath, ".md") {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		return ChunkMarkdown(string(data)), nil
	}
	if strings.HasSuffix(filePath, ".go") {
		return ChunkGoSource(filePath)
	}
	return nil, nil
}