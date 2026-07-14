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
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type Chunk struct {
	Content    string
	SourceLine int
}

// readFileUTF8 reads a file and converts from GBK if not valid UTF-8.
func readFileUTF8(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Try UTF-8 first
	if utf8.Valid(data) {
		return string(data), nil
	}

	// Try GBK -> UTF-8
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Data, err := decoder.Bytes(data)
	if err != nil {
		// Fallback: force valid UTF-8, dropping invalid bytes
		return strings.ToValidUTF8(string(data), ""), nil
	}
	return string(utf8Data), nil
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
	content, err := readFileUTF8(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
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
		content, err := readFileUTF8(filePath)
		if err != nil {
			return nil, err
		}
		return ChunkMarkdown(content), nil
	}
	if strings.HasSuffix(filePath, ".go") {
		return ChunkGoSource(filePath)
	}
	return nil, nil
}