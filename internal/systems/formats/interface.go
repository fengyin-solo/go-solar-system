package formats

import (
	"fmt"
	"strings"

	"github.com/furan917/go-solar-system/internal/models"
)

// MaxSystemFileSize bounds the size of an external system definition. It mirrors
// the 10 MiB response cap used for the live Solar System API (api.MaxResponseSize)
// so file-based and network-based systems share the same upper limit.
const MaxSystemFileSize = 10 * 1024 * 1024

// SystemData represents an external star system with metadata
type SystemData struct {
	SystemName    string                 `json:"systemName" yaml:"systemName"`
	Description   string                 `json:"description" yaml:"description"`
	DiscoveryYear string                 `json:"discoveryYear" yaml:"discoveryYear"`
	Distance      string                 `json:"distance" yaml:"distance"`
	Galaxy        string                 `json:"galaxy" yaml:"galaxy"`
	Bodies        []models.CelestialBody `json:"bodies" yaml:"bodies"`
}

// SystemMetadata represents just the metadata portion (without celestial bodies)
type SystemMetadata struct {
	SystemName    string `json:"systemName" yaml:"systemName"`
	Description   string `json:"description" yaml:"description"`
	DiscoveryYear string `json:"discoveryYear" yaml:"discoveryYear"`
	Distance      string `json:"distance" yaml:"distance"`
	Galaxy        string `json:"galaxy" yaml:"galaxy"`
}

// validateSystemData validates the complete system data structure.
// Shared by every registered format so JSON and YAML enforce the same rules.
func validateSystemData(system *SystemData) error {
	if strings.TrimSpace(system.SystemName) == "" {
		return fmt.Errorf("systemName cannot be empty")
	}

	if len(system.Bodies) == 0 {
		return fmt.Errorf("system must contain at least one celestial body")
	}

	// Validate each celestial body has required fields
	for i, body := range system.Bodies {
		if strings.TrimSpace(body.EnglishName) == "" {
			return fmt.Errorf("celestial body at index %d missing englishName", i)
		}
	}

	return nil
}

// validateSystemMetadata validates the system metadata structure.
// Shared by every registered format so JSON and YAML enforce the same rules.
func validateSystemMetadata(metadata *SystemMetadata) error {
	if strings.TrimSpace(metadata.SystemName) == "" {
		return fmt.Errorf("systemName cannot be empty")
	}

	return nil
}

// checkSize rejects payloads larger than the shared system definition limit.
func checkSize(data []byte) error {
	if len(data) > MaxSystemFileSize {
		return fmt.Errorf("system file is too large: %d bytes (max: %d)", len(data), MaxSystemFileSize)
	}

	return nil
}

// FileFormat defines the interface that all file format handlers must implement
// This allows for easy extension to support additional formats beyond JSON
type FileFormat interface {
	// GetSupportedExtensions returns the file extensions this handler supports (e.g., [".json", ".yaml"])
	GetSupportedExtensions() []string

	// GetFormatName returns a human-readable name for this format (e.g., "JSON", "YAML")
	GetFormatName() string

	// ParseSystemData parses the complete system data from file content
	ParseSystemData(data []byte) (*SystemData, error)

	// ParseSystemMetadata parses only the metadata (for performance) from file content
	ParseSystemMetadata(data []byte) (*SystemMetadata, error)

	// ValidateFormat performs basic validation to ensure the data is in the expected format
	ValidateFormat(data []byte) error

	// GetMimeType returns the MIME type for this format (optional, can return empty string)
	GetMimeType() string
}

// FormatRegistry manages all available file format handlers
type FormatRegistry struct {
	handlers map[string]FileFormat // extension -> handler mapping
	formats  []FileFormat          // list of all handlers
}

// NewFormatRegistry creates a new format registry with all available formats
func NewFormatRegistry() *FormatRegistry {
	registry := &FormatRegistry{
		handlers: make(map[string]FileFormat),
		formats:  make([]FileFormat, 0),
	}

	// Register built-in formats. JSON is registered first so content-based
	// detection claims JSON payloads before YAML (JSON is valid YAML syntax).
	registry.RegisterFormat(NewJSONFormat())
	registry.RegisterFormat(NewYAMLFormat())

	return registry
}

// RegisterFormat registers a new file format handler
func (fr *FormatRegistry) RegisterFormat(format FileFormat) {
	fr.formats = append(fr.formats, format)

	for _, ext := range format.GetSupportedExtensions() {
		fr.handlers[ext] = format
	}
}

// GetHandlerForExtension returns the file format handler for a given extension
func (fr *FormatRegistry) GetHandlerForExtension(extension string) (FileFormat, bool) {
	handler, exists := fr.handlers[extension]
	return handler, exists
}

// GetAllFormats returns all registered format handlers
func (fr *FormatRegistry) GetAllFormats() []FileFormat {
	return fr.formats
}

// GetSupportedExtensions returns all supported file extensions
func (fr *FormatRegistry) GetSupportedExtensions() []string {
	var extensions []string
	for ext := range fr.handlers {
		extensions = append(extensions, ext)
	}
	return extensions
}

// DetectFormat attempts to detect the format of file content by trying all registered handlers
func (fr *FormatRegistry) DetectFormat(data []byte) (FileFormat, error) {
	var lastError error

	for _, format := range fr.formats {
		if err := format.ValidateFormat(data); err == nil {
			return format, nil
		} else {
			lastError = err
		}
	}

	if lastError != nil {
		return nil, lastError
	}

	return nil, fmt.Errorf("no supported format detected")
}
