package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FieldError is what a section validator returns for an invalid value.
// Path is relative to the section, e.g. "[2].principal_cents" for the third
// element's field, or "marginal_tax_rate_pct" for a mapping key. Message is
// French and is shown to the user as is.
type FieldError struct {
	Path    string
	Message string
}

func (e *FieldError) Error() string { return e.Message }

// LoadError is an invalid data file. Its message is French and names the file
// and, when known, the line.
type LoadError struct {
	File    string
	Line    int // 0 when unknown
	Message string
}

func (e *LoadError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("Le fichier %s est invalide (ligne %d) : %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("Le fichier %s est invalide : %s", e.File, e.Message)
}

var yamlLine = regexp.MustCompile(`line (\d+)`)

// syntaxError turns a YAML parser error into a LoadError, keeping its line.
func syntaxError(file string, err error) *LoadError {
	line := 0
	if m := yamlLine.FindStringSubmatch(err.Error()); m != nil {
		line, _ = strconv.Atoi(m[1])
	}
	detail := strings.TrimPrefix(err.Error(), "yaml: ")
	return &LoadError{File: file, Line: line, Message: "YAML mal formé (" + detail + ")"}
}

// lineOf finds the line of path inside node, falling back to the deepest
// node it could reach.
func lineOf(node *yaml.Node, path string) int {
	line := node.Line
	for _, seg := range splitPath(path) {
		next := child(node, seg)
		if next == nil {
			break
		}
		node, line = next, next.Line
	}
	return line
}

// splitPath turns "[2].principal_cents" into ["[2]", "principal_cents"].
func splitPath(path string) []string {
	var segs []string
	for _, part := range strings.Split(path, ".") {
		for part != "" {
			if strings.HasPrefix(part, "[") {
				end := strings.Index(part, "]")
				if end < 0 {
					return segs
				}
				segs = append(segs, part[:end+1])
				part = part[end+1:]
				continue
			}
			end := strings.Index(part, "[")
			if end < 0 {
				end = len(part)
			}
			segs = append(segs, part[:end])
			part = part[end:]
		}
	}
	return segs
}

func child(node *yaml.Node, seg string) *yaml.Node {
	if strings.HasPrefix(seg, "[") {
		i, err := strconv.Atoi(strings.Trim(seg, "[]"))
		if err != nil || node.Kind != yaml.SequenceNode || i < 0 || i >= len(node.Content) {
			return nil
		}
		return node.Content[i]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == seg {
			return node.Content[i+1]
		}
	}
	return nil
}
