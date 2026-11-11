package formats

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlTopLevelFields lists the only keys allowed at the root of a system document.
// Unknown nested keys remain tolerated (and ignored) so YAML stays compatible with
// the JSON loader, which ignores fields absent from the model structs.
var yamlTopLevelFields = map[string]bool{
	"systemName":    true,
	"description":   true,
	"discoveryYear": true,
	"distance":      true,
	"galaxy":        true,
	"bodies":        true,
}

// yamlLineRE extracts the 1-based line number embedded in yaml.v3 error messages.
var yamlLineRE = regexp.MustCompile(`line (\d+)`)

// yamlPositionError is a YAML parsing error that carries a 1-based row and column.
type yamlPositionError struct {
	Line   int
	Column int
	Msg    string
}

func (e *yamlPositionError) Error() string {
	if e.Column > 0 {
		return fmt.Sprintf("YAML line %d, column %d: %s", e.Line, e.Column, e.Msg)
	}

	return fmt.Sprintf("YAML line %d: %s", e.Line, e.Msg)
}

func nodeError(node *yaml.Node, format string, args ...interface{}) error {
	line, column := 1, 1
	if node != nil {
		if node.Line > 0 {
			line = node.Line
		}
		if node.Column > 0 {
			column = node.Column
		}
	}

	return &yamlPositionError{Line: line, Column: column, Msg: fmt.Sprintf(format, args...)}
}

// YAMLFormat implements the FileFormat interface for YAML files
type YAMLFormat struct{}

// NewYAMLFormat creates a new YAML format handler
func NewYAMLFormat() *YAMLFormat {
	return &YAMLFormat{}
}

// GetSupportedExtensions returns the file extensions this handler supports
func (yf *YAMLFormat) GetSupportedExtensions() []string {
	return []string{".yaml", ".yml"}
}

// GetFormatName returns a human-readable name for this format
func (yf *YAMLFormat) GetFormatName() string {
	return "YAML"
}

// ParseSystemData parses the complete system data from YAML content
func (yf *YAMLFormat) ParseSystemData(data []byte) (*SystemData, error) {
	root, err := yf.parseDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML system data: %w", err)
	}

	var system SystemData
	if err := root.Decode(&system); err != nil {
		return nil, fmt.Errorf("failed to parse YAML system data: %w", enrichYAMLError(err, root))
	}

	if err := validateSystemData(&system); err != nil {
		return nil, fmt.Errorf("invalid system data: %w", err)
	}

	return &system, nil
}

// ParseSystemMetadata parses only the metadata (for performance) from YAML content
func (yf *YAMLFormat) ParseSystemMetadata(data []byte) (*SystemMetadata, error) {
	root, err := yf.parseDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML system metadata: %w", err)
	}

	var metadata SystemMetadata
	if err := root.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse YAML system metadata: %w", enrichYAMLError(err, root))
	}

	if err := validateSystemMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("invalid system metadata: %w", err)
	}

	return &metadata, nil
}

// ValidateFormat performs a real parse of the YAML content instead of guessing
// from punctuation. It enforces the same required fields as JSON validation.
func (yf *YAMLFormat) ValidateFormat(data []byte) error {
	root, err := yf.parseDocument(data)
	if err != nil {
		return err
	}

	requiredFields := []string{"systemName", "bodies"}
	for _, field := range requiredFields {
		if _, err := yamlMappingValue(root, field); err != nil {
			return nodeError(root, "missing required field: %s", field)
		}
	}

	return nil
}

// GetMimeType returns the MIME type for YAML
func (yf *YAMLFormat) GetMimeType() string {
	return "application/x-yaml"
}

// parseDocument runs the size check and full YAML parse, returning the document
// root mapping node together with position-aware structural errors.
func (yf *YAMLFormat) parseDocument(data []byte) (*yaml.Node, error) {
	if err := checkSize(data); err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nodeError(nil, "empty YAML document")
		}

		return nil, enrichYAMLError(err, nil)
	}

	// A second, non-empty document means the file uses multi-document markers.
	var extra yaml.Node
	if err := decoder.Decode(&extra); err == nil {
		return nil, nodeError(&extra, "multiple YAML documents are not allowed; expected a single document")
	} else if !errors.Is(err, io.EOF) {
		return nil, enrichYAMLError(err, &document)
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil, nodeError(&document, "empty YAML document")
	}

	root := document.Content[0]
	if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
		return nil, nodeError(root, "empty YAML document")
	}

	if err := validateYAMLStructure(root); err != nil {
		return nil, err
	}

	return root, nil
}

// validateYAMLStructure walks the parsed tree and rejects structural problems
// that the typed decode cannot express with precise positions: non-mapping roots,
// duplicate keys, unknown top-level fields and wrong node kinds.
func validateYAMLStructure(root *yaml.Node) error {
	if root.Kind != yaml.MappingNode {
		return nodeError(root, "YAML document root must be a mapping of system fields")
	}

	seen := make(map[string]bool)

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]

		if key.Kind != yaml.ScalarNode {
			return nodeError(key, "YAML mapping keys must be scalars")
		}

		name := key.Value
		if seen[name] {
			return nodeError(key, "duplicate mapping key %q", name)
		}
		seen[name] = true

		if !yamlTopLevelFields[name] {
			return nodeError(key, "unknown top-level field %q", name)
		}

		switch name {
		case "bodies":
			if value.Kind != yaml.SequenceNode {
				return nodeError(value, "field %q must be a sequence of celestial bodies", name)
			}

			for _, body := range value.Content {
				if body.Kind != yaml.MappingNode {
					return nodeError(body, "each entry in %q must be a mapping", name)
				}
			}
		default:
			if value.Kind != yaml.ScalarNode {
				return nodeError(value, "field %q must be a scalar value", name)
			}
		}
	}

	return validateNestedMappings(root)
}

// validateNestedMappings recursively rejects duplicate keys inside every nested
// mapping (bodies, mass, moons, aroundPlanet, ...).
func validateNestedMappings(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		seen := make(map[string]bool)

		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			if key.Kind == yaml.ScalarNode {
				if seen[key.Value] {
					return nodeError(key, "duplicate mapping key %q", key.Value)
				}
				seen[key.Value] = true
			}

			if err := validateNestedMappings(value); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := validateNestedMappings(child); err != nil {
				return err
			}
		}
	}

	return nil
}

// yamlMappingValue returns the value node for a top-level key, or an error
// when the key is absent.
func yamlMappingValue(root *yaml.Node, key string) (*yaml.Node, error) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Kind == yaml.ScalarNode && root.Content[i].Value == key {
			return root.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("missing key %q", key)
}

// lineColumns indexes the column of the first node found on each line so that
// yaml.v3 type errors can be annotated with column positions.
func lineColumns(node *yaml.Node) map[int]int {
	columns := make(map[int]int)

	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil {
			return
		}
		if n.Line > 0 {
			if _, exists := columns[n.Line]; !exists {
				columns[n.Line] = n.Column
			}
		}
		for _, child := range n.Content {
			walk(child)
		}
		if n.Alias != nil {
			walk(n.Alias)
		}
	}
	walk(node)

	return columns
}

// annotateYAMLMessage rewrites a single yaml.v3 message, e.g.
// "yaml: line 4: cannot unmarshal !!str `abc` into float64", adding the column
// recorded for that line in the parsed node tree.
func annotateYAMLMessage(msg string, columns map[int]int) string {
	msg = strings.TrimPrefix(msg, "yaml: ")

	loc := yamlLineRE.FindStringSubmatchIndex(msg)
	if loc == nil {
		return msg
	}

	line, _ := strconv.Atoi(msg[loc[2]:loc[3]])
	detail := strings.TrimSpace(strings.TrimPrefix(msg[loc[1]:], ":"))

	if column, ok := columns[line]; ok && column > 0 {
		return fmt.Sprintf("line %d, column %d: %s", line, column, detail)
	}

	return fmt.Sprintf("line %d: %s", line, detail)
}

// enrichYAMLError converts yaml.v3 syntax and type errors into position-aware
// errors expressed in terms of the parsed document.
func enrichYAMLError(err error, root *yaml.Node) error {
	columns := lineColumns(root)

	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		messages := make([]string, 0, len(typeErr.Errors))
		for _, msg := range typeErr.Errors {
			messages = append(messages, annotateYAMLMessage(msg, columns))
		}

		return fmt.Errorf("YAML %s", strings.Join(messages, "; "))
	}

	return fmt.Errorf("YAML %s", annotateYAMLMessage(err.Error(), columns))
}
