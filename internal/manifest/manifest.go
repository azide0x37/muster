// Package manifest decodes the human-authored Muster implementation contract.
// It deliberately describes adapters rather than performing inspection: the
// inspector is responsible for projecting these declarations into the shared
// object model.
package manifest

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Document is the on-disk muster.yaml schema.
type Document struct {
	Schema       int            `yaml:"schema" json:"schema"`
	Framework    string         `yaml:"framework" json:"framework"`
	Project      Project        `yaml:"project" json:"project"`
	Architecture map[string]any `yaml:"architecture,omitempty" json:"architecture,omitempty"`
	Lifecycle    map[string]any `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`
	Patterns     map[string]any `yaml:"patterns,omitempty" json:"patterns,omitempty"`
	Python       map[string]any `yaml:"python,omitempty" json:"python,omitempty"`
	Release      map[string]any `yaml:"release,omitempty" json:"release,omitempty"`
	Compliance   map[string]any `yaml:"compliance,omitempty" json:"compliance,omitempty"`
	Inspection   Inspection     `yaml:"inspection" json:"inspection"`
}

type Project struct {
	Name        string `yaml:"name" json:"name"`
	DisplayName string `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	Type        string `yaml:"type" json:"type"`
	VersionFile string `yaml:"version_file" json:"version_file"`
	ConfigDir   string `yaml:"config_dir" json:"config_dir"`
	InstallDir  string `yaml:"install_dir" json:"install_dir"`
	CurrentLink string `yaml:"current_link" json:"current_link"`
	ReleaseDir  string `yaml:"release_dir" json:"release_dir"`
}

// Inspection declares how an implementation projects into Muster's generic
// component graph. IDs are globally addressable and therefore must be stable.
type Inspection struct {
	ID             string          `yaml:"id" json:"id"`
	Summary        string          `yaml:"summary" json:"summary"`
	Literate       Literate        `yaml:"literate,omitempty" json:"literate,omitempty"`
	RootComponents []string        `yaml:"root_components" json:"root_components"`
	Components     []ComponentSpec `yaml:"components" json:"components"`
	Edges          []EdgeSpec      `yaml:"edges,omitempty" json:"edges,omitempty"`
}

type ComponentSpec struct {
	ID       string         `yaml:"id" json:"id"`
	Kind     string         `yaml:"kind" json:"kind"`
	Summary  string         `yaml:"summary" json:"summary"`
	Children []string       `yaml:"children,omitempty" json:"children,omitempty"`
	Metadata map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Source   SourceSpec     `yaml:"source,omitempty" json:"source,omitempty"`
	Actions  []ActionSpec   `yaml:"actions,omitempty" json:"actions,omitempty"`
	Literate Literate       `yaml:"literate,omitempty" json:"literate,omitempty"`
}

// SourceSpec identifies a bounded adapter. It is data, never a shell snippet.
type SourceSpec struct {
	Adapter       string            `yaml:"adapter,omitempty" json:"adapter,omitempty"`
	Unit          string            `yaml:"unit,omitempty" json:"unit,omitempty"`
	Role          string            `yaml:"role,omitempty" json:"role,omitempty"`
	Path          string            `yaml:"path,omitempty" json:"path,omitempty"`
	StateFile     string            `yaml:"state_file,omitempty" json:"state_file,omitempty"`
	VersionFile   string            `yaml:"version_file,omitempty" json:"version_file,omitempty"`
	Required      bool              `yaml:"required,omitempty" json:"required,omitempty"`
	AllowInactive bool              `yaml:"allow_inactive,omitempty" json:"allow_inactive,omitempty"`
	AbsentStatus  string            `yaml:"absent_status,omitempty" json:"absent_status,omitempty"`
	MaxAgeSeconds int64             `yaml:"max_age_seconds,omitempty" json:"max_age_seconds,omitempty"`
	StatusField   string            `yaml:"status_field,omitempty" json:"status_field,omitempty"`
	StatusMap     map[string]string `yaml:"status_map,omitempty" json:"status_map,omitempty"`
	Status        string            `yaml:"status,omitempty" json:"status,omitempty"`
}

type ActionSpec struct {
	ID           string   `yaml:"id" json:"id"`
	Label        string   `yaml:"label" json:"label"`
	Kind         string   `yaml:"kind" json:"kind"`
	Command      []string `yaml:"command" json:"command"`
	RequiresRoot bool     `yaml:"requires_root,omitempty" json:"requires_root,omitempty"`
}

type EdgeSpec struct {
	From     string `yaml:"from" json:"from"`
	Relation string `yaml:"relation" json:"relation"`
	To       string `yaml:"to" json:"to"`
	Summary  string `yaml:"summary,omitempty" json:"summary,omitempty"`
}

type Literate struct {
	What             string   `yaml:"what,omitempty" json:"what,omitempty"`
	Why              string   `yaml:"why,omitempty" json:"why,omitempty"`
	Responsibilities []string `yaml:"responsibilities,omitempty" json:"responsibilities,omitempty"`
	FailureModes     []string `yaml:"failure_modes,omitempty" json:"failure_modes,omitempty"`
}

func Decode(r io.Reader) (Document, error) {
	var document Document
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		return Document{}, fmt.Errorf("decode Muster manifest: %w", err)
	}
	return document, nil
}

func Load(path string) (Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return Document{}, fmt.Errorf("open Muster manifest %s: %w", path, err)
	}
	defer file.Close()

	document, err := Decode(file)
	if err != nil {
		return Document{}, fmt.Errorf("%s: %w", path, err)
	}
	return document, nil
}
