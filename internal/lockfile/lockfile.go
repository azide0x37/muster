// Package lockfile turns authoring YAML into a deterministic, immutable object
// graph. Installed inspectors may then apply host observations without
// reinterpreting application-specific structure.
package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/azide0x37/muster/internal/manifest"
	"github.com/azide0x37/muster/internal/model"
)

const CurrentSchema = "muster.lock/v1"

type Source struct {
	ManifestSHA256 string            `json:"manifest_sha256"`
	Repository     string            `json:"pattern_repository,omitempty"`
	Commit         string            `json:"pattern_commit,omitempty"`
	Roots          []string          `json:"pattern_roots,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type Document struct {
	Schema   string                           `json:"schema"`
	Source   Source                           `json:"source"`
	Graph    model.Graph                      `json:"graph"`
	Adapters map[model.ID]manifest.SourceSpec `json:"adapters"`
	Actions  map[model.ID]Action              `json:"actions,omitempty"`
}

type Action struct {
	ComponentID model.ID            `json:"component_id"`
	Spec        manifest.ActionSpec `json:"spec"`
}

func Generate(document manifest.Document, manifestBytes []byte, version string) (*Document, error) {
	if document.Schema != 2 {
		return nil, fmt.Errorf("lockfile: schema 2 manifest required")
	}
	rootID := model.ID(document.Inspection.ID)
	components := []model.Component{{
		ID: rootID, Kind: "implementation", Health: model.Health{Status: model.HealthUnknown},
		Summary: document.Inspection.Summary, Children: toIDs(document.Inspection.RootComponents),
		What: document.Inspection.Literate.What, Why: document.Inspection.Literate.Why,
		Responsibilities: append([]string(nil), document.Inspection.Literate.Responsibilities...),
		FailureModes:     toFailureModes(document.Inspection.Literate.FailureModes),
	}}
	allIDs := []model.ID{rootID}
	adapters := make(map[model.ID]manifest.SourceSpec)
	actions := make(map[model.ID]Action)
	for _, spec := range document.Inspection.Components {
		id := model.ID(spec.ID)
		component := model.Component{
			ID: id, Kind: model.ComponentKind(spec.Kind), Health: model.Health{Status: model.HealthUnknown},
			Summary: spec.Summary, Metadata: toMetadata(spec.Metadata), Children: toIDs(spec.Children),
			What: spec.Literate.What, Why: spec.Literate.Why,
			Responsibilities: append([]string(nil), spec.Literate.Responsibilities...),
			FailureModes:     toFailureModes(spec.Literate.FailureModes),
		}
		for _, actionSpec := range spec.Actions {
			actionID := model.ID(actionSpec.ID)
			component.Actions = append(component.Actions, model.Action{
				ID: actionID, Kind: model.ActionKind(actionSpec.Kind), Label: actionSpec.Label,
				Target: id, RequiresRoot: actionSpec.RequiresRoot, RequiresConfirmation: true,
			})
			actions[actionID] = Action{ComponentID: id, Spec: actionSpec}
		}
		components = append(components, component)
		allIDs = append(allIDs, id)
		adapters[id] = spec.Source
	}
	edges := make([]model.Edge, 0, len(document.Inspection.Edges)+len(document.Inspection.RootComponents))
	for _, spec := range document.Inspection.Edges {
		edges = append(edges, model.Edge{From: model.ID(spec.From), Type: model.EdgeType(spec.Relation), To: model.ID(spec.To), Summary: spec.Summary})
	}
	for _, child := range document.Inspection.RootComponents {
		edges = append(edges, model.Edge{From: rootID, Type: model.EdgeOwns, To: model.ID(child)})
	}
	graph, err := model.NewGraph([]model.Implementation{{
		ID: rootID, Version: version, Summary: document.Inspection.Summary,
		Metadata: model.Metadata{"project": document.Project.Name}, Components: allIDs,
	}}, components, edges, nil)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(manifestBytes)
	result := &Document{
		Schema: CurrentSchema,
		Source: Source{ManifestSHA256: hex.EncodeToString(digest[:]), Metadata: map[string]string{}},
		Graph:  *graph, Adapters: adapters, Actions: actions,
	}
	if value, ok := document.Patterns["source"]; ok {
		result.Source.Repository = fmt.Sprint(value)
	}
	if value, ok := document.Patterns["verified_head"]; ok {
		result.Source.Commit = fmt.Sprint(value)
	}
	if value, ok := document.Patterns["primary"]; ok {
		result.Source.Roots = append(result.Source.Roots, fmt.Sprint(value))
	}
	if values, ok := document.Patterns["integrations"].([]any); ok {
		for _, value := range values {
			result.Source.Roots = append(result.Source.Roots, fmt.Sprint(value))
		}
	}
	sort.Strings(result.Source.Roots)
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode lock %s: %w", path, err)
	}
	if document.Schema != CurrentSchema {
		return nil, fmt.Errorf("unsupported lock schema %q", document.Schema)
	}
	if err := document.Validate(); err != nil {
		return nil, err
	}
	return &document, nil
}

// Validate checks that the frozen graph, adapters, and executable action
// declarations describe the same globally addressed objects.
func (document *Document) Validate() error {
	if document == nil {
		return fmt.Errorf("lockfile: nil document")
	}
	if document.Schema != CurrentSchema {
		return fmt.Errorf("unsupported lock schema %q", document.Schema)
	}
	if len(document.Source.ManifestSHA256) != sha256.Size*2 {
		return fmt.Errorf("lockfile: manifest_sha256 must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(document.Source.ManifestSHA256); err != nil {
		return fmt.Errorf("lockfile: manifest_sha256 is invalid: %w", err)
	}
	if err := document.Graph.Validate(); err != nil {
		return fmt.Errorf("invalid locked graph: %w", err)
	}
	for _, component := range document.Graph.Components {
		if component.DeclaredHealth != nil {
			return fmt.Errorf("lockfile: component %s contains runtime-only declared_health", component.ID)
		}
	}
	for id := range document.Adapters {
		if _, ok := document.Graph.Lookup(id); !ok {
			return fmt.Errorf("lockfile: adapter references unknown component %s", id)
		}
	}
	type advertisedAction struct {
		owner  model.ID
		action model.Action
	}
	graphActions := make(map[model.ID]advertisedAction)
	for _, component := range document.Graph.Components {
		for _, action := range component.Actions {
			graphActions[action.ID] = advertisedAction{owner: component.ID, action: action}
		}
	}
	for id, locked := range document.Actions {
		advertised, ok := graphActions[id]
		if !ok {
			return fmt.Errorf("lockfile: executable action %s is not advertised by the graph", id)
		}
		if locked.ComponentID != advertised.owner {
			return fmt.Errorf("lockfile: action %s owner %s does not match graph owner %s", id, locked.ComponentID, advertised.owner)
		}
		if locked.Spec.ID != string(id) {
			return fmt.Errorf("lockfile: action map key %s does not match spec id %s", id, locked.Spec.ID)
		}
		if string(advertised.action.Kind) != locked.Spec.Kind {
			return fmt.Errorf("lockfile: action %s kind %s does not match executable kind %s", id, advertised.action.Kind, locked.Spec.Kind)
		}
		if advertised.action.Label != locked.Spec.Label {
			return fmt.Errorf("lockfile: action %s label does not match executable declaration", id)
		}
		if advertised.action.Target != locked.ComponentID {
			return fmt.Errorf("lockfile: action %s target %s does not match owner %s", id, advertised.action.Target, locked.ComponentID)
		}
		if advertised.action.RequiresRoot != locked.Spec.RequiresRoot {
			return fmt.Errorf("lockfile: action %s requires_root does not match executable declaration", id)
		}
		if locked.Spec.Kind == "doctor.run" && !advertised.action.RequiresConfirmation {
			return fmt.Errorf("lockfile: doctor action %s must require confirmation", id)
		}
		if locked.Spec.Kind == "doctor.run" && len(locked.Spec.Command) == 0 {
			return fmt.Errorf("lockfile: doctor action %s has no command", id)
		}
		if locked.Spec.Kind == "doctor.run" && !filepath.IsAbs(locked.Spec.Command[0]) {
			return fmt.Errorf("lockfile: doctor action %s command must be an absolute path", id)
		}
		delete(graphActions, id)
	}
	if len(graphActions) > 0 {
		ids := make([]string, 0, len(graphActions))
		for id := range graphActions {
			ids = append(ids, string(id))
		}
		sort.Strings(ids)
		return fmt.Errorf("lockfile: advertised action %s has no executable declaration", ids[0])
	}
	return nil
}

func VerifyManifest(document *Document, manifestBytes []byte) error {
	digest := sha256.Sum256(manifestBytes)
	actual := hex.EncodeToString(digest[:])
	if actual != document.Source.ManifestSHA256 {
		return fmt.Errorf("manifest digest %s does not match lock %s", actual, document.Source.ManifestSHA256)
	}
	return nil
}

func toIDs(values []string) []model.ID {
	result := make([]model.ID, len(values))
	for index, value := range values {
		result[index] = model.ID(value)
	}
	return result
}

func toMetadata(values map[string]any) model.Metadata {
	result := make(model.Metadata, len(values))
	for key, value := range values {
		if text, ok := value.(string); ok {
			result[key] = text
			continue
		}
		encoded, _ := json.Marshal(value)
		result[key] = string(encoded)
	}
	return result
}

func toFailureModes(values []string) []model.FailureMode {
	result := make([]model.FailureMode, 0, len(values))
	for index, value := range values {
		result = append(result, model.FailureMode{ID: fmt.Sprintf("failure-%d", index+1), Summary: value})
	}
	return result
}
