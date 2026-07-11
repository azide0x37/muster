// Package model defines Muster's normalized runtime object graph.
//
// The package deliberately contains no host integration. Adapters may inspect
// systemd, parse manifests, or run doctors, but they all project their findings
// into the same graph before a CLI, TUI, or API consumes them.
package model

import "time"

// CurrentSchemaVersion identifies the JSON representation emitted by Graph.
const CurrentSchemaVersion = "muster.runtime/v1"

// ID is a globally addressable object identifier, such as
// "implementation:dvd-ingester" or "component:systemd.publisher".
type ID string

// Metadata carries adapter-specific facts without teaching graph consumers
// about every possible component kind. JSON encoding sorts string map keys, so
// this remains deterministic while retaining a convenient object shape.
type Metadata map[string]string

// ComponentKind describes the kind of a component without constraining Muster
// to a closed list of infrastructure technologies.
type ComponentKind string

// ActionKind describes an operation a presentation layer may expose.
type ActionKind string

// Action is an addressable operation advertised by a component. Execution and
// authorization remain responsibilities of the adapter or application layer.
type Action struct {
	ID                   ID         `json:"id"`
	Kind                 ActionKind `json:"kind"`
	Label                string     `json:"label"`
	Summary              string     `json:"summary,omitempty"`
	Target               ID         `json:"target,omitempty"`
	RequiresRoot         bool       `json:"requires_root,omitempty"`
	RequiresConfirmation bool       `json:"requires_confirmation,omitempty"`
}

// FailureMode documents how a component can fail and how an operator can
// recognize and recover from the failure.
type FailureMode struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Effect   string `json:"effect,omitempty"`
	Recovery string `json:"recovery,omitempty"`
}

// Component is the common inspectable object rendered by every Muster view.
// Children are globally addressable component IDs rather than nested values;
// the same component can therefore also participate in the typed graph.
type Component struct {
	ID     ID            `json:"id"`
	Kind   ComponentKind `json:"kind"`
	Health Health        `json:"health"`
	// DeclaredHealth preserves the adapter's direct assertion after Health is
	// materialized to the recursively derived value for renderers and exports.
	DeclaredHealth *Health  `json:"declared_health,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
	Actions        []Action `json:"actions,omitempty"`
	Children       []ID     `json:"children,omitempty"`

	What             string        `json:"what,omitempty"`
	Why              string        `json:"why,omitempty"`
	Responsibilities []string      `json:"responsibilities,omitempty"`
	FailureModes     []FailureMode `json:"failure_modes,omitempty"`
}

// Implementation identifies one installed Muster implementation. ID resolves
// to its root Component, and Components contains the complete ordered
// projection for that implementation, including ID itself.
type Implementation struct {
	ID         ID       `json:"id"`
	Version    string   `json:"version,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Components []ID     `json:"components"`
}

// EdgeType gives semantic meaning to graph relationships.
type EdgeType string

const (
	EdgeDependsOn  EdgeType = "depends_on"
	EdgeImplements EdgeType = "implements"
	EdgeOwns       EdgeType = "owns"
	EdgeProduces   EdgeType = "produces"
	EdgeConsumes   EdgeType = "consumes"
	EdgeObserves   EdgeType = "observes"
	EdgeConfigures EdgeType = "configures"
)

// Valid reports whether t is one of Muster's defined relationship types.
func (t EdgeType) Valid() bool {
	switch t {
	case EdgeDependsOn, EdgeImplements, EdgeOwns, EdgeProduces, EdgeConsumes, EdgeObserves, EdgeConfigures:
		return true
	default:
		return false
	}
}

// Edge connects two globally addressable components.
type Edge struct {
	From     ID       `json:"from"`
	Type     EdgeType `json:"type"`
	To       ID       `json:"to"`
	Summary  string   `json:"summary,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// HealthStatus is a component or observation health classification.
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "unknown"
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// Valid reports whether s is a defined health status. The empty value is
// accepted as an uninitialized spelling of unknown for ergonomic struct use.
func (s HealthStatus) Valid() bool {
	switch s {
	case "", HealthUnknown, HealthHealthy, HealthDegraded, HealthUnhealthy:
		return true
	default:
		return false
	}
}

// Health is a health assertion. Derived health retains this simple shape so it
// can be used anywhere a directly observed health value can be used.
type Health struct {
	Status     HealthStatus `json:"status"`
	Summary    string       `json:"summary,omitempty"`
	ObservedAt *time.Time   `json:"observed_at,omitempty"`
}

// ObservationKind identifies the producer or purpose of an observation.
type ObservationKind string

const (
	// ObservationDoctor is evidence produced by a Muster doctor run.
	ObservationDoctor ObservationKind = "doctor"
)

// CheckStatus is the result of one diagnostic check.
type CheckStatus string

const (
	CheckUnknown CheckStatus = "unknown"
	CheckPass    CheckStatus = "pass"
	CheckWarn    CheckStatus = "warn"
	CheckFail    CheckStatus = "fail"
)

// Valid reports whether s is a defined check status. Empty means unknown.
func (s CheckStatus) Valid() bool {
	switch s {
	case "", CheckUnknown, CheckPass, CheckWarn, CheckFail:
		return true
	default:
		return false
	}
}

// Check is one assertion made by an observation such as a doctor run.
type Check struct {
	ID       string      `json:"id"`
	Status   CheckStatus `json:"status"`
	Summary  string      `json:"summary,omitempty"`
	Evidence Metadata    `json:"evidence,omitempty"`
}

// Artifact points to durable evidence produced by an observation. URI may be
// a local path or another stable URI understood by the presentation layer.
type Artifact struct {
	URI       string `json:"uri"`
	Summary   string `json:"summary,omitempty"`
	MediaType string `json:"media_type,omitempty"`
}

// Observation is time-bound evidence about a component. Doctor is represented
// by Kind=ObservationDoctor rather than by a special top-level model.
type Observation struct {
	ID          ID              `json:"id"`
	ComponentID ID              `json:"component_id"`
	Kind        ObservationKind `json:"kind"`
	Status      HealthStatus    `json:"status,omitempty"`
	Summary     string          `json:"summary,omitempty"`
	ObservedAt  time.Time       `json:"observed_at"`
	DurationMS  int64           `json:"duration_ms"`
	Stale       bool            `json:"stale,omitempty"`
	Checks      []Check         `json:"checks,omitempty"`
	Artifacts   []Artifact      `json:"artifacts,omitempty"`
	Metadata    Metadata        `json:"metadata,omitempty"`
}

// Graph is Muster's normalized, JSON-ready runtime object graph.
type Graph struct {
	SchemaVersion   string           `json:"schema_version"`
	Implementations []Implementation `json:"implementations,omitempty"`
	Components      []Component      `json:"components"`
	Edges           []Edge           `json:"edges,omitempty"`
	Observations    []Observation    `json:"observations,omitempty"`
}

// HealthStep records one relationship through which health propagates.
type HealthStep struct {
	From         ID       `json:"from"`
	Relationship EdgeType `json:"relationship"`
	To           ID       `json:"to"`
}

// HealthCause identifies a direct assertion responsible for effective health.
type HealthCause struct {
	ComponentID ID           `json:"component_id"`
	Health      Health       `json:"health"`
	Path        []HealthStep `json:"path,omitempty"`
}

// HealthExplanation shows both a component's assertion and recursively
// derived health, together with deterministic propagation paths to causes.
type HealthExplanation struct {
	ComponentID ID            `json:"component_id"`
	Declared    Health        `json:"declared"`
	Effective   Health        `json:"effective"`
	Causes      []HealthCause `json:"causes,omitempty"`
}

// DependencyPath is one shortest path from the explained component to another
// component. For RequiredBy paths, dependency edges are traversed in reverse.
type DependencyPath struct {
	ComponentID ID   `json:"component_id"`
	Path        []ID `json:"path"`
}

// DependencyExplanation shows what a component needs and what needs it.
type DependencyExplanation struct {
	ComponentID ID               `json:"component_id"`
	DependsOn   []DependencyPath `json:"depends_on,omitempty"`
	RequiredBy  []DependencyPath `json:"required_by,omitempty"`
}

// Explanation is the complete generic explanation for one component.
type Explanation struct {
	Component    Component             `json:"component"`
	Health       HealthExplanation     `json:"health"`
	Dependencies DependencyExplanation `json:"dependencies"`
}
