package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

var (
	// ErrComponentNotFound is returned when a global ID cannot be resolved.
	ErrComponentNotFound = errors.New("component not found")
	// ErrImplementationNotFound is returned when an implementation ID cannot be resolved.
	ErrImplementationNotFound = errors.New("implementation not found")
)

// NewGraph copies, canonicalizes, and validates a normalized runtime graph.
func NewGraph(implementations []Implementation, components []Component, edges []Edge, observations []Observation) (*Graph, error) {
	g := Graph{
		SchemaVersion:   CurrentSchemaVersion,
		Implementations: cloneImplementations(implementations),
		Components:      cloneComponents(components),
		Edges:           cloneEdges(edges),
		Observations:    cloneObservations(observations),
	}
	g.canonicalize()
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// MarshalJSON makes Graph output canonical even when a caller constructed or
// mutated it as a struct literal.
func (g Graph) MarshalJSON() ([]byte, error) {
	canonical := g.Clone()
	canonical.canonicalize()
	type graphJSON Graph
	return json.Marshal(graphJSON(canonical))
}

// Clone returns a deep-enough copy for safe sorting and presentation. All
// mutable slices and metadata maps in the graph are copied.
func (g Graph) Clone() Graph {
	return Graph{
		SchemaVersion:   g.SchemaVersion,
		Implementations: cloneImplementations(g.Implementations),
		Components:      cloneComponents(g.Components),
		Edges:           cloneEdges(g.Edges),
		Observations:    cloneObservations(g.Observations),
	}
}

func (g *Graph) canonicalize() {
	if g.SchemaVersion == "" {
		g.SchemaVersion = CurrentSchemaVersion
	}
	sort.SliceStable(g.Implementations, func(i, j int) bool {
		return g.Implementations[i].ID < g.Implementations[j].ID
	})
	sort.SliceStable(g.Components, func(i, j int) bool {
		return g.Components[i].ID < g.Components[j].ID
	})
	sort.SliceStable(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.To < b.To
	})
	sort.SliceStable(g.Observations, func(i, j int) bool {
		a, b := g.Observations[i], g.Observations[j]
		if a.ComponentID != b.ComponentID {
			return a.ComponentID < b.ComponentID
		}
		if !a.ObservedAt.Equal(b.ObservedAt) {
			return a.ObservedAt.Before(b.ObservedAt)
		}
		return a.ID < b.ID
	})
}

// Validate checks global identity, references, relationship types, and
// observation invariants. Errors are returned in canonical object order.
func (g *Graph) Validate() error {
	if g == nil {
		return errors.New("model: nil graph")
	}
	if g.SchemaVersion == "" {
		return errors.New("model: schema_version is required")
	}

	componentIDs := make(map[ID]struct{}, len(g.Components))
	objectIDs := make(map[ID]string, len(g.Components))
	for i := range g.Components {
		id := g.Components[i].ID
		if id == "" {
			return fmt.Errorf("model: components[%d].id is required", i)
		}
		if _, exists := componentIDs[id]; exists {
			return fmt.Errorf("model: duplicate component id %q", id)
		}
		componentIDs[id] = struct{}{}
		objectIDs[id] = "component"
	}
	for i := range g.Components {
		component := &g.Components[i]
		if component.Kind == "" {
			return fmt.Errorf("model: component %q kind is required", component.ID)
		}
		if !component.Health.Status.Valid() {
			return fmt.Errorf("model: component %q has invalid health %q", component.ID, component.Health.Status)
		}
		if component.DeclaredHealth != nil && !component.DeclaredHealth.Status.Valid() {
			return fmt.Errorf("model: component %q has invalid declared health %q", component.ID, component.DeclaredHealth.Status)
		}
		for _, action := range component.Actions {
			if action.ID == "" {
				return fmt.Errorf("model: component %q has an action without an id", component.ID)
			}
			if kind, exists := objectIDs[action.ID]; exists {
				return fmt.Errorf("model: duplicate global object id %q (action conflicts with %s)", action.ID, kind)
			}
			objectIDs[action.ID] = "action"
			if action.Target != "" {
				if _, exists := componentIDs[action.Target]; !exists {
					return fmt.Errorf("model: action %q references unknown target %q", action.ID, action.Target)
				}
			}
		}
	}

	implementationIDs := make(map[ID]struct{}, len(g.Implementations))
	componentOwners := make(map[ID]ID, len(componentIDs))
	for _, implementation := range g.Implementations {
		if implementation.ID == "" {
			return errors.New("model: implementation id is required")
		}
		if _, exists := implementationIDs[implementation.ID]; exists {
			return fmt.Errorf("model: duplicate implementation id %q", implementation.ID)
		}
		implementationIDs[implementation.ID] = struct{}{}
		if _, exists := componentIDs[implementation.ID]; !exists {
			return fmt.Errorf("model: implementation %q does not resolve to a component", implementation.ID)
		}
		seen := make(map[ID]struct{}, len(implementation.Components))
		includesRoot := false
		for _, id := range implementation.Components {
			if _, exists := componentIDs[id]; !exists {
				return fmt.Errorf("model: implementation %q references unknown component %q", implementation.ID, id)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("model: implementation %q repeats component %q", implementation.ID, id)
			}
			seen[id] = struct{}{}
			if owner, exists := componentOwners[id]; exists && owner != implementation.ID {
				return fmt.Errorf("model: component %q belongs to both %q and %q", id, owner, implementation.ID)
			}
			componentOwners[id] = implementation.ID
			includesRoot = includesRoot || id == implementation.ID
		}
		if !includesRoot {
			return fmt.Errorf("model: implementation %q components do not include its root", implementation.ID)
		}
	}
	for id := range componentIDs {
		if _, exists := componentOwners[id]; !exists {
			return fmt.Errorf("model: component %q does not belong to an implementation", id)
		}
	}

	for _, component := range g.Components {
		seen := make(map[ID]struct{}, len(component.Children))
		for _, child := range component.Children {
			if _, exists := componentIDs[child]; !exists {
				return fmt.Errorf("model: component %q references unknown child %q", component.ID, child)
			}
			if child == component.ID {
				return fmt.Errorf("model: component %q cannot be its own child", component.ID)
			}
			if _, exists := seen[child]; exists {
				return fmt.Errorf("model: component %q repeats child %q", component.ID, child)
			}
			seen[child] = struct{}{}
		}
	}

	edges := make(map[string]struct{}, len(g.Edges))
	for _, edge := range g.Edges {
		if !edge.Type.Valid() {
			return fmt.Errorf("model: edge %q -> %q has invalid type %q", edge.From, edge.To, edge.Type)
		}
		if _, exists := componentIDs[edge.From]; !exists {
			return fmt.Errorf("model: edge references unknown source %q", edge.From)
		}
		if _, exists := componentIDs[edge.To]; !exists {
			return fmt.Errorf("model: edge references unknown target %q", edge.To)
		}
		key := string(edge.From) + "\x00" + string(edge.Type) + "\x00" + string(edge.To)
		if _, exists := edges[key]; exists {
			return fmt.Errorf("model: duplicate %s edge %q -> %q", edge.Type, edge.From, edge.To)
		}
		edges[key] = struct{}{}
	}

	observationIDs := make(map[ID]struct{}, len(g.Observations))
	for _, observation := range g.Observations {
		if observation.ID == "" {
			return errors.New("model: observation id is required")
		}
		if kind, exists := objectIDs[observation.ID]; exists {
			return fmt.Errorf("model: duplicate global object id %q (observation conflicts with %s)", observation.ID, kind)
		}
		if _, exists := observationIDs[observation.ID]; exists {
			return fmt.Errorf("model: duplicate observation id %q", observation.ID)
		}
		observationIDs[observation.ID] = struct{}{}
		objectIDs[observation.ID] = "observation"
		if _, exists := componentIDs[observation.ComponentID]; !exists {
			return fmt.Errorf("model: observation %q references unknown component %q", observation.ID, observation.ComponentID)
		}
		if observation.Kind == "" {
			return fmt.Errorf("model: observation %q kind is required", observation.ID)
		}
		if !observation.Status.Valid() {
			return fmt.Errorf("model: observation %q has invalid health %q", observation.ID, observation.Status)
		}
		if observation.ObservedAt.IsZero() {
			return fmt.Errorf("model: observation %q observed_at is required", observation.ID)
		}
		if observation.DurationMS < 0 {
			return fmt.Errorf("model: observation %q duration_ms cannot be negative", observation.ID)
		}
		checks := make(map[string]struct{}, len(observation.Checks))
		for _, check := range observation.Checks {
			if check.ID == "" {
				return fmt.Errorf("model: observation %q has a check without an id", observation.ID)
			}
			if _, exists := checks[check.ID]; exists {
				return fmt.Errorf("model: observation %q repeats check %q", observation.ID, check.ID)
			}
			checks[check.ID] = struct{}{}
			if !check.Status.Valid() {
				return fmt.Errorf("model: observation %q check %q has invalid status %q", observation.ID, check.ID, check.Status)
			}
		}
		for _, artifact := range observation.Artifacts {
			if artifact.URI == "" {
				return fmt.Errorf("model: observation %q has an artifact without a uri", observation.ID)
			}
		}
	}
	return nil
}

// Lookup resolves a component by globally unique ID.
func (g *Graph) Lookup(id ID) (*Component, bool) {
	if g == nil {
		return nil, false
	}
	for i := range g.Components {
		if g.Components[i].ID == id {
			return &g.Components[i], true
		}
	}
	return nil, false
}

// LookupImplementation resolves an installed implementation by ID.
func (g *Graph) LookupImplementation(id ID) (*Implementation, bool) {
	if g == nil {
		return nil, false
	}
	for i := range g.Implementations {
		if g.Implementations[i].ID == id {
			return &g.Implementations[i], true
		}
	}
	return nil, false
}

// LookupAction resolves an advertised action by its globally unique ID and
// returns the component that owns it.
func (g *Graph) LookupAction(id ID) (*Action, ID, bool) {
	if g == nil {
		return nil, "", false
	}
	for componentIndex := range g.Components {
		for actionIndex := range g.Components[componentIndex].Actions {
			if g.Components[componentIndex].Actions[actionIndex].ID == id {
				return &g.Components[componentIndex].Actions[actionIndex], g.Components[componentIndex].ID, true
			}
		}
	}
	return nil, "", false
}

// LookupObservation resolves time-bound evidence by its globally unique ID.
func (g *Graph) LookupObservation(id ID) (*Observation, bool) {
	if g == nil {
		return nil, false
	}
	for index := range g.Observations {
		if g.Observations[index].ID == id {
			return &g.Observations[index], true
		}
	}
	return nil, false
}

// Outgoing returns canonically ordered outgoing edges, optionally limited to
// the supplied relationship types.
func (g *Graph) Outgoing(id ID, types ...EdgeType) []Edge {
	return g.edgesFor(id, false, types...)
}

// Incoming returns canonically ordered incoming edges, optionally limited to
// the supplied relationship types.
func (g *Graph) Incoming(id ID, types ...EdgeType) []Edge {
	return g.edgesFor(id, true, types...)
}

func (g *Graph) edgesFor(id ID, incoming bool, types ...EdgeType) []Edge {
	if g == nil {
		return nil
	}
	filter := make(map[EdgeType]struct{}, len(types))
	for _, edgeType := range types {
		filter[edgeType] = struct{}{}
	}
	result := make([]Edge, 0)
	for _, edge := range g.Edges {
		matchID := (!incoming && edge.From == id) || (incoming && edge.To == id)
		if !matchID {
			continue
		}
		if len(filter) > 0 {
			if _, wanted := filter[edge.Type]; !wanted {
				continue
			}
		}
		result = append(result, cloneEdge(edge))
	}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
	return result
}

func cloneImplementations(source []Implementation) []Implementation {
	if source == nil {
		return nil
	}
	result := make([]Implementation, len(source))
	for i, implementation := range source {
		result[i] = implementation
		result[i].Metadata = cloneMetadata(implementation.Metadata)
		result[i].Components = append([]ID(nil), implementation.Components...)
	}
	return result
}

func cloneComponents(source []Component) []Component {
	if source == nil {
		return nil
	}
	result := make([]Component, len(source))
	for i, component := range source {
		result[i] = component
		if component.DeclaredHealth != nil {
			declared := *component.DeclaredHealth
			result[i].DeclaredHealth = &declared
		}
		result[i].Metadata = cloneMetadata(component.Metadata)
		result[i].Actions = append([]Action(nil), component.Actions...)
		result[i].Children = append([]ID(nil), component.Children...)
		result[i].Responsibilities = append([]string(nil), component.Responsibilities...)
		result[i].FailureModes = append([]FailureMode(nil), component.FailureModes...)
	}
	return result
}

func cloneEdges(source []Edge) []Edge {
	if source == nil {
		return nil
	}
	result := make([]Edge, len(source))
	for i, edge := range source {
		result[i] = cloneEdge(edge)
	}
	return result
}

func cloneEdge(edge Edge) Edge {
	edge.Metadata = cloneMetadata(edge.Metadata)
	return edge
}

func cloneObservations(source []Observation) []Observation {
	if source == nil {
		return nil
	}
	result := make([]Observation, len(source))
	for i, observation := range source {
		result[i] = observation
		result[i].Metadata = cloneMetadata(observation.Metadata)
		result[i].Artifacts = append([]Artifact(nil), observation.Artifacts...)
		result[i].Checks = make([]Check, len(observation.Checks))
		for j, check := range observation.Checks {
			result[i].Checks[j] = check
			result[i].Checks[j].Evidence = cloneMetadata(check.Evidence)
		}
	}
	return result
}

func cloneMetadata(source Metadata) Metadata {
	if source == nil {
		return nil
	}
	result := make(Metadata, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
