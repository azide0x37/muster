// Package inspector contains bounded host adapters that project installed
// implementations into the generic model. Nothing in this package knows what
// a DVD, Bluetooth speaker, database, or camera is.
package inspector

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/azide0x37/muster/internal/lockfile"
	"github.com/azide0x37/muster/internal/manifest"
	"github.com/azide0x37/muster/internal/model"
	"github.com/azide0x37/muster/internal/registry"
)

const maxStateBytes = 1 << 20

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type EnvRunner interface {
	RunEnv(context.Context, []string, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (ExecRunner) RunEnv(ctx context.Context, environment []string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = environment
	return command.CombinedOutput()
}

type Inspector struct {
	Root   string
	Now    func() time.Time
	Runner Runner
}

type Snapshot struct {
	Graph   *model.Graph
	Sources map[model.ID]manifest.SourceSpec
	Actions map[model.ID]Action
}

type Action struct {
	ComponentID model.ID
	Spec        manifest.ActionSpec
}

func New(root string) *Inspector {
	return &Inspector{Root: root, Now: time.Now, Runner: ExecRunner{}}
}

func (i *Inspector) Load(ctx context.Context) (*Snapshot, error) {
	entries, err := registry.Load(i.Root)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		graph, graphErr := model.NewGraph(nil, nil, nil, nil)
		if graphErr != nil {
			return nil, graphErr
		}
		return &Snapshot{Graph: graph, Sources: map[model.ID]manifest.SourceSpec{}, Actions: map[model.ID]Action{}}, nil
	}

	var implementations []model.Implementation
	var components []model.Component
	var edges []model.Edge
	var observations []model.Observation
	sources := make(map[model.ID]manifest.SourceSpec)
	actions := make(map[model.ID]Action)

	for _, entry := range entries {
		manifestPath, resolveErr := registry.ResolveManifest(i.Root, entry)
		if resolveErr != nil {
			return nil, fmt.Errorf("%s: %w", entry.ID, resolveErr)
		}
		projected, projectErr := i.loadProjection(ctx, entry, manifestPath)
		if projectErr != nil {
			return nil, fmt.Errorf("project %s: %w", entry.ID, projectErr)
		}
		if string(projected.implementation.ID) != entry.ID {
			return nil, fmt.Errorf("registry %s points to manifest for %s", entry.ID, projected.implementation.ID)
		}
		implementations = append(implementations, projected.implementation)
		components = append(components, projected.components...)
		edges = append(edges, projected.edges...)
		observations = append(observations, projected.observations...)
		for id, source := range projected.sources {
			sources[id] = source
		}
		for id, action := range projected.actions {
			if _, exists := actions[id]; exists {
				return nil, fmt.Errorf("duplicate global action ID %s", id)
			}
			actions[id] = action
		}
	}

	graph, err := model.NewGraph(implementations, components, edges, observations)
	if err != nil {
		return nil, err
	}
	// Materialize recursive health into the snapshot so every renderer and API
	// sees the same answer while explanations retain each adapter's direct
	// assertion and the graph path to the actual cause.
	if err := graph.MaterializeDerivedHealth(); err != nil {
		return nil, err
	}
	return &Snapshot{Graph: graph, Sources: sources, Actions: actions}, nil
}

func (i *Inspector) loadProjection(ctx context.Context, entry registry.Entry, manifestPath string) (projection, error) {
	if entry.Lock != "" {
		lockPath, err := registry.HostPath(i.Root, entry.Lock)
		if err != nil {
			return projection{}, err
		}
		if _, err := os.Stat(lockPath); err != nil {
			if os.IsNotExist(err) {
				return projection{}, fmt.Errorf("declared implementation lock is missing: %s", lockPath)
			}
			return projection{}, err
		}
		manifestBytes, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return projection{}, readErr
		}
		locked, loadErr := lockfile.Load(lockPath)
		if loadErr != nil {
			return projection{}, loadErr
		}
		if verifyErr := lockfile.VerifyManifest(locked, manifestBytes); verifyErr != nil {
			return projection{}, fmt.Errorf("%s: %w", lockPath, verifyErr)
		}
		return i.projectLock(ctx, locked)
	}
	document, err := manifest.Load(manifestPath)
	if err != nil {
		return projection{}, err
	}
	return i.project(ctx, document, manifestPath)
}

func (i *Inspector) projectLock(ctx context.Context, locked *lockfile.Document) (projection, error) {
	if len(locked.Graph.Implementations) != 1 {
		return projection{}, fmt.Errorf("implementation lock must contain exactly one implementation")
	}
	implementation := locked.Graph.Implementations[0]
	project := implementation.Metadata["project"]
	result := projection{
		implementation: implementation,
		edges:          append([]model.Edge(nil), locked.Graph.Edges...),
		sources:        make(map[model.ID]manifest.SourceSpec),
		actions:        make(map[model.ID]Action),
	}
	for _, lockedComponent := range locked.Graph.Components {
		component := lockedComponent
		component.DeclaredHealth = nil
		source, hasSource := locked.Adapters[component.ID]
		if hasSource {
			health, observation, err := i.inspectSource(ctx, project, component.ID, source)
			if err != nil {
				health = model.Health{Status: model.HealthUnhealthy, Summary: err.Error()}
			}
			component.Health = health
			result.sources[component.ID] = source
			if observation != nil {
				result.observations = append(result.observations, *observation)
			}
		} else if component.ID == implementation.ID || len(component.Children) > 0 {
			component.Health = model.Health{Status: model.HealthHealthy, Summary: "declared"}
		}
		result.components = append(result.components, component)
	}
	for id, lockedAction := range locked.Actions {
		result.actions[id] = Action{ComponentID: lockedAction.ComponentID, Spec: lockedAction.Spec}
	}
	return result, nil
}

type projection struct {
	implementation model.Implementation
	components     []model.Component
	edges          []model.Edge
	observations   []model.Observation
	sources        map[model.ID]manifest.SourceSpec
	actions        map[model.ID]Action
}

func (i *Inspector) project(ctx context.Context, document manifest.Document, manifestPath string) (projection, error) {
	if document.Schema != 2 {
		return projection{}, fmt.Errorf("unsupported muster.yaml schema %d (want 2)", document.Schema)
	}
	if document.Framework != "Muster" {
		return projection{}, fmt.Errorf("framework must be Muster")
	}
	if document.Project.Name == "" || document.Inspection.ID == "" {
		return projection{}, fmt.Errorf("project.name and inspection.id are required")
	}
	rootID := model.ID(document.Inspection.ID)
	if rootID != model.ID("implementation:"+document.Project.Name) {
		return projection{}, fmt.Errorf("inspection.id must be implementation:%s", document.Project.Name)
	}
	version := readVersion(manifestPath, document.Project.VersionFile)
	allIDs := make([]model.ID, 0, len(document.Inspection.Components)+1)
	allIDs = append(allIDs, rootID)
	components := []model.Component{{
		ID:               rootID,
		Kind:             "implementation",
		Health:           model.Health{Status: model.HealthHealthy, Summary: "declared implementation"},
		Summary:          document.Inspection.Summary,
		Children:         ids(document.Inspection.RootComponents),
		What:             document.Inspection.Literate.What,
		Why:              document.Inspection.Literate.Why,
		Responsibilities: append([]string(nil), document.Inspection.Literate.Responsibilities...),
		FailureModes:     failureModes(document.Inspection.Literate.FailureModes),
		Metadata: model.Metadata{
			"project":  document.Project.Name,
			"manifest": logicalManifestPath(i.Root, manifestPath),
		},
	}}
	result := projection{
		sources: make(map[model.ID]manifest.SourceSpec),
		actions: make(map[model.ID]Action),
	}
	for _, spec := range document.Inspection.Components {
		id := model.ID(spec.ID)
		if id == "" {
			return projection{}, fmt.Errorf("component id is required")
		}
		health, observation, inspectErr := i.inspectSource(ctx, document.Project.Name, id, spec.Source)
		if inspectErr != nil {
			health = model.Health{Status: model.HealthUnhealthy, Summary: inspectErr.Error()}
		}
		component := model.Component{
			ID: id, Kind: model.ComponentKind(spec.Kind), Health: health,
			Summary: spec.Summary, Metadata: metadata(spec.Metadata), Children: ids(spec.Children),
			What: spec.Literate.What, Why: spec.Literate.Why,
			Responsibilities: append([]string(nil), spec.Literate.Responsibilities...),
			FailureModes:     failureModes(spec.Literate.FailureModes),
		}
		for _, actionSpec := range spec.Actions {
			actionID := model.ID(actionSpec.ID)
			component.Actions = append(component.Actions, model.Action{
				ID: actionID, Kind: model.ActionKind(actionSpec.Kind), Label: actionSpec.Label,
				Target: id, RequiresRoot: actionSpec.RequiresRoot, RequiresConfirmation: true,
			})
			result.actions[actionID] = Action{ComponentID: id, Spec: actionSpec}
		}
		components = append(components, component)
		allIDs = append(allIDs, id)
		result.sources[id] = spec.Source
		if observation != nil {
			result.observations = append(result.observations, *observation)
		}
	}

	for _, edge := range document.Inspection.Edges {
		result.edges = append(result.edges, model.Edge{
			From: model.ID(edge.From), Type: model.EdgeType(edge.Relation), To: model.ID(edge.To), Summary: edge.Summary,
		})
	}
	// Ownership is explicit in the graph even when the author only selected
	// top-level children for presentation.
	for _, child := range document.Inspection.RootComponents {
		result.edges = append(result.edges, model.Edge{From: rootID, Type: model.EdgeOwns, To: model.ID(child)})
	}
	result.implementation = model.Implementation{
		ID: rootID, Version: version, Summary: document.Inspection.Summary,
		Metadata: model.Metadata{"project": document.Project.Name}, Components: allIDs,
	}
	result.components = components
	return result, nil
}

func (i *Inspector) inspectSource(ctx context.Context, project string, id model.ID, source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	switch source.Adapter {
	case "", "static", "pattern":
		status := model.HealthHealthy
		if source.Status != "" {
			status = healthStatus(source.Status)
			if status == model.HealthUnknown && !strings.EqualFold(source.Status, "unknown") {
				return model.Health{}, nil, fmt.Errorf("invalid declared status %q", source.Status)
			}
		}
		return model.Health{Status: status, Summary: "declared " + string(status)}, nil, nil
	case "systemd.unit":
		return i.inspectSystemd(ctx, source)
	case "metadata.file":
		return i.inspectFile(source)
	case "release.current":
		return i.inspectRelease(source)
	case "observation.file":
		return i.inspectObservation(project, id, source)
	case "legacy.json":
		return i.inspectLegacy(id, source)
	default:
		return model.Health{}, nil, fmt.Errorf("unknown adapter %q", source.Adapter)
	}
}

func (i *Inspector) inspectSystemd(ctx context.Context, source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	if source.Unit == "" {
		return model.Health{}, nil, errors.New("systemd.unit adapter requires unit")
	}
	if i.Root != "" && i.Root != "/" {
		path, _ := registry.HostPath(i.Root, "/etc/systemd/system/"+source.Unit)
		if _, err := os.Stat(path); err != nil {
			return model.Health{Status: model.HealthUnhealthy, Summary: "unit is not installed"}, nil, nil
		}
		return model.Health{Status: model.HealthHealthy, Summary: "unit installed; staged root has no live systemd"}, nil, nil
	}
	output, err := i.runner().Run(ctx, "systemctl", "show", source.Unit, "--no-pager", "--property=LoadState,ActiveState,SubState,UnitFileState,Result,Type")
	properties := parseProperties(string(output))
	if err != nil && len(properties) == 0 {
		return model.Health{}, nil, fmt.Errorf("systemctl show %s: %w", source.Unit, err)
	}
	active := properties["ActiveState"]
	load := properties["LoadState"]
	result := properties["Result"]
	summary := strings.TrimSpace(active + " (" + properties["SubState"] + ")")
	if load == "not-found" || load == "error" || active == "failed" || (result != "" && result != "success") {
		return model.Health{Status: model.HealthUnhealthy, Summary: summary}, nil, nil
	}
	if active == "active" || active == "reloading" {
		return model.Health{Status: model.HealthHealthy, Summary: summary}, nil, nil
	}
	if active == "activating" || active == "deactivating" {
		return model.Health{Status: model.HealthDegraded, Summary: summary}, nil, nil
	}
	if active == "inactive" && source.AllowInactive {
		return model.Health{Status: model.HealthHealthy, Summary: "inactive as permitted"}, nil, nil
	}
	return model.Health{Status: model.HealthUnknown, Summary: summary}, nil, nil
}

func (i *Inspector) inspectFile(source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	path, err := registry.HostPath(i.Root, source.Path)
	if err != nil {
		return model.Health{}, nil, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return absentHealth(source), nil, nil
	}
	if err != nil {
		return model.Health{}, nil, err
	}
	return model.Health{Status: model.HealthHealthy, Summary: fmt.Sprintf("present (%s, mode %s)", byteCount(info.Size()), info.Mode().Perm())}, nil, nil
}

func (i *Inspector) inspectRelease(source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	path, err := registry.HostPath(i.Root, source.Path)
	if err != nil {
		return model.Health{}, nil, err
	}
	target, err := os.Readlink(path)
	if err != nil {
		if os.IsNotExist(err) {
			return absentHealth(source), nil, nil
		}
		return model.Health{}, nil, err
	}
	return model.Health{Status: model.HealthHealthy, Summary: "current -> " + target}, nil, nil
}

type observationEnvelope struct {
	Schema         string            `json:"schema"`
	Implementation string            `json:"implementation"`
	Component      string            `json:"component"`
	Scope          string            `json:"scope"`
	Health         string            `json:"health"`
	Status         string            `json:"status"`
	Summary        string            `json:"summary"`
	ObservedAt     time.Time         `json:"observed_at"`
	DurationMS     int64             `json:"duration_ms"`
	ValidFor       int64             `json:"valid_for_seconds"`
	Checks         []envelopeCheck   `json:"checks"`
	Artifacts      []model.Artifact  `json:"artifacts"`
	Metadata       map[string]string `json:"metadata"`
}

type envelopeCheck struct {
	ID       string            `json:"id"`
	Health   string            `json:"health"`
	Summary  string            `json:"summary"`
	Evidence map[string]string `json:"evidence"`
}

func (i *Inspector) inspectObservation(project string, id model.ID, source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	logical := source.StateFile
	if logical == "" {
		logical = source.Path
	}
	path, err := registry.HostPath(i.Root, logical)
	if err != nil {
		return model.Health{}, nil, err
	}
	data, err := readBounded(path)
	if os.IsNotExist(err) {
		return absentHealth(source), nil, nil
	}
	if err != nil {
		return model.Health{}, nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.Health{}, nil, err
	}
	var envelope observationEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return model.Health{Status: model.HealthDegraded, Summary: "observation is not valid JSON"}, nil, nil
	}
	if envelope.Schema != "muster.observation/v1" {
		return model.Health{Status: model.HealthDegraded, Summary: "unsupported observation schema"}, nil, nil
	}
	if envelope.Implementation != project {
		return model.Health{}, nil, fmt.Errorf("observation implementation is %q", envelope.Implementation)
	}
	status := healthStatus(envelope.Health)
	observation := model.Observation{
		ID:          model.ID(fmt.Sprintf("observation:%s:%d", strings.TrimPrefix(string(id), "component:"), info.ModTime().UnixNano())),
		ComponentID: id, Kind: model.ObservationDoctor, Status: status, Summary: envelope.Summary,
		ObservedAt: envelope.ObservedAt, DurationMS: envelope.DurationMS, Artifacts: envelope.Artifacts,
		Metadata: model.Metadata(envelope.Metadata),
	}
	if observation.Metadata == nil {
		observation.Metadata = make(model.Metadata)
	}
	if envelope.Scope != "" {
		observation.Metadata["scope"] = envelope.Scope
	}
	if envelope.Status != "" {
		observation.Metadata["status"] = envelope.Status
	}
	for _, check := range envelope.Checks {
		observation.Checks = append(observation.Checks, model.Check{
			ID: check.ID, Status: checkStatus(check.Health), Summary: check.Summary, Evidence: model.Metadata(check.Evidence),
		})
	}
	health := observation.DerivedHealth()
	maxAge := source.MaxAgeSeconds
	if maxAge == 0 {
		maxAge = envelope.ValidFor
	}
	if maxAge > 0 && i.now().Sub(envelope.ObservedAt) > time.Duration(maxAge)*time.Second {
		observation.Stale = true
		health = model.Health{Status: model.HealthUnknown, Summary: "observation is stale", ObservedAt: &envelope.ObservedAt}
	}
	return health, &observation, nil
}

func (i *Inspector) inspectLegacy(id model.ID, source manifest.SourceSpec) (model.Health, *model.Observation, error) {
	path, err := registry.HostPath(i.Root, source.Path)
	if err != nil {
		return model.Health{}, nil, err
	}
	data, err := readBounded(path)
	if os.IsNotExist(err) {
		return absentHealth(source), nil, nil
	}
	if err != nil {
		return model.Health{}, nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return model.Health{Status: model.HealthDegraded, Summary: "legacy state is not valid JSON"}, nil, nil
	}
	status := model.HealthHealthy
	summary := "state observed"
	if source.StatusField != "" {
		value := fmt.Sprint(object[source.StatusField])
		if mapped := source.StatusMap[value]; mapped != "" {
			status = healthStatus(mapped)
		}
		summary = source.StatusField + "=" + value
	}
	info, _ := os.Stat(path)
	observedAt := i.now()
	if info != nil {
		observedAt = info.ModTime()
	}
	observation := model.Observation{
		ID:          model.ID(fmt.Sprintf("observation:%s:%d", strings.TrimPrefix(string(id), "component:"), observedAt.UnixNano())),
		ComponentID: id, Kind: model.ObservationKind("state"), Status: status, Summary: summary, ObservedAt: observedAt,
		Metadata: metadata(object),
	}
	if source.MaxAgeSeconds > 0 && i.now().Sub(observedAt) > time.Duration(source.MaxAgeSeconds)*time.Second {
		return model.Health{Status: model.HealthUnknown, Summary: "state is stale", ObservedAt: &observedAt}, &observation, nil
	}
	return observation.DerivedHealth(), &observation, nil
}

// RunDoctor executes only an explicitly declared doctor.run action. The core
// never interprets shell syntax: each argument is passed directly to exec.
func (i *Inspector) RunDoctor(ctx context.Context, snapshot *Snapshot, actionID model.ID) ([]byte, error) {
	action, ok := snapshot.Actions[actionID]
	if !ok {
		return nil, fmt.Errorf("unknown action %s", actionID)
	}
	if action.Spec.Kind != "doctor.run" {
		return nil, fmt.Errorf("action %s is not a doctor action", actionID)
	}
	if len(action.Spec.Command) == 0 {
		return nil, fmt.Errorf("doctor action %s has no command", actionID)
	}
	if action.Spec.RequiresRoot && (i.Root == "" || i.Root == "/") && os.Geteuid() != 0 {
		return nil, fmt.Errorf("doctor action %s requires root; rerun with sudo", actionID)
	}
	name := action.Spec.Command[0]
	if !filepath.IsAbs(name) {
		return nil, fmt.Errorf("doctor action %s command must be an absolute path", actionID)
	}
	var err error
	name, err = registry.HostPath(i.Root, name)
	if err != nil {
		return nil, err
	}
	timeout := 60 * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if i.Root != "" && i.Root != "/" {
		if runner, ok := i.runner().(EnvRunner); ok {
			environment := append(os.Environ(), "MUSTER_ROOT="+i.Root)
			return runner.RunEnv(runCtx, environment, name, action.Spec.Command[1:]...)
		}
	}
	return i.runner().Run(runCtx, name, action.Spec.Command[1:]...)
}

func (i *Inspector) runner() Runner {
	if i.Runner != nil {
		return i.Runner
	}
	return ExecRunner{}
}

func (i *Inspector) now() time.Time {
	if i.Now != nil {
		return i.Now()
	}
	return time.Now()
}

func parseProperties(text string) map[string]string {
	properties := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), "=")
		if found {
			properties[key] = value
		}
	}
	return properties
}

func absentHealth(source manifest.SourceSpec) model.Health {
	status := healthStatus(source.AbsentStatus)
	if source.AbsentStatus == "" {
		if source.Required {
			status = model.HealthUnhealthy
		} else {
			status = model.HealthUnknown
		}
	}
	return model.Health{Status: status, Summary: "not observed"}
}

func healthStatus(value string) model.HealthStatus {
	switch strings.ToLower(value) {
	case "healthy", "pass", "ok":
		return model.HealthHealthy
	case "degraded", "warn", "warning":
		return model.HealthDegraded
	case "unhealthy", "failed", "fail", "error":
		return model.HealthUnhealthy
	default:
		return model.HealthUnknown
	}
}

func checkStatus(value string) model.CheckStatus {
	switch healthStatus(value) {
	case model.HealthHealthy:
		return model.CheckPass
	case model.HealthDegraded:
		return model.CheckWarn
	case model.HealthUnhealthy:
		return model.CheckFail
	default:
		return model.CheckUnknown
	}
}

func metadata(values map[string]any) model.Metadata {
	result := make(model.Metadata, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		switch value := values[key].(type) {
		case nil:
			result[key] = ""
		case string:
			result[key] = value
		case bool, float64, int, int64:
			result[key] = fmt.Sprint(value)
		default:
			encoded, _ := json.Marshal(value)
			result[key] = string(encoded)
		}
	}
	return result
}

func ids(values []string) []model.ID {
	result := make([]model.ID, len(values))
	for index, value := range values {
		result[index] = model.ID(value)
	}
	return result
}

func failureModes(values []string) []model.FailureMode {
	result := make([]model.FailureMode, 0, len(values))
	for index, value := range values {
		result = append(result, model.FailureMode{ID: "failure-" + strconv.Itoa(index+1), Summary: value})
	}
	return result
}

func readVersion(manifestPath, versionFile string) string {
	if versionFile == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), versionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readBounded(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	limited := make([]byte, maxStateBytes+1)
	n, err := file.Read(limited)
	if err != nil && n == 0 {
		return nil, err
	}
	if n > maxStateBytes {
		return nil, fmt.Errorf("state file %s exceeds %s", path, byteCount(maxStateBytes))
	}
	return limited[:n], nil
}

func logicalManifestPath(root, hostPath string) string {
	if root == "" || root == "/" {
		return hostPath
	}
	relative, err := filepath.Rel(filepath.Clean(root), hostPath)
	if err != nil || strings.HasPrefix(relative, "..") {
		return hostPath
	}
	return "/" + filepath.ToSlash(relative)
}

func byteCount(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f KiB", float64(bytes)/1024)
}
