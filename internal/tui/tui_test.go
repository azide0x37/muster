package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/azide0x37/muster/internal/model"
)

func TestRenderWideBrowserIsDeterministicAndLiterate(t *testing.T) {
	graph := fixtureGraph(t)
	opts := RenderOptions{
		Hostname:       "shed-pi-01",
		Width:          118,
		Height:         34,
		DarkBackground: true,
		NoColor:        true,
		Selected:       "component:pattern.conveyor",
	}

	first := Render(graph, opts)
	second := Render(graph, opts)
	if first != second {
		t.Fatal("Render produced different output for the same graph and options")
	}
	assertContains(t, first,
		"Muster implementations on shed-pi-01",
		"2 implementations · 1 healthy · 1 degraded",
		"Implementations · 7 objects",
		"Media Gateway",
		"Device-triggered Conveyor",
		"Selected object",
		"WHAT",
		"WHY",
		"enter inspect",
	)
	if strings.Contains(first, "\x1b[") {
		t.Fatal("NoColor render unexpectedly contains ANSI escapes")
	}
	if got := lipgloss.Width(first); got != opts.Width {
		t.Fatalf("render width = %d, want %d", got, opts.Width)
	}
	if got := lipgloss.Height(first); got != opts.Height {
		t.Fatalf("render height = %d, want %d", got, opts.Height)
	}
}

func TestRenderInspectorShowsCompleteGenericObject(t *testing.T) {
	graph := fixtureGraph(t)
	output := Render(graph, RenderOptions{
		Hostname:       "shed-pi-01",
		Width:          110,
		Height:         120,
		DarkBackground: true,
		NoColor:        true,
		Selected:       "component:pattern.conveyor",
		Inspect:        "component:pattern.conveyor",
	})

	assertContains(t, output,
		"Inspect · pattern · component:pattern.conveyor",
		"WHAT",
		"Turns device readiness into bounded work and an atomic publication.",
		"WHY",
		"Keeps unreliable media away from durable storage until work is complete.",
		"RESPONSIBILITIES",
		"Bound every capture attempt",
		"FAILURE MODES",
		"Destination disappears",
		"Recovery: Restore the mount and rerun publication.",
		"HEALTH EXPLANATION",
		"Effective: ◐ DEGRADED",
		"CHILDREN · HIERARCHY",
		"Capture Worker",
		"RELATIONS",
		"depends on Archive Mount",
		"DEPENDENCY EXPLANATION",
		"Needs Archive Mount via Device-triggered Conveyor → Archive Mount",
		"METADATA",
		"pattern: T2R4.device-triggered-conveyor",
		"ACTIONS",
		"[d] Run doctor · doctor.run",
		"OBSERVATIONS",
		"DOCTOR · 2026-07-10T12:30:00Z",
		"● PASS · queue",
		"Artifact: /run/muster/media-gateway/doctor.json",
	)
}

func TestNarrowLayoutUsesOnePaneAndNeverOverflows(t *testing.T) {
	graph := fixtureGraph(t)
	opts := RenderOptions{
		Hostname: "edge-box", Width: 52, Height: 26, NoColor: true,
		Selected: "component:pattern.conveyor",
	}
	output := Render(graph, opts)

	assertContains(t, output,
		"Muster implementations on edge-box",
		"Implementations · 7 objects",
		"Device-triggered Conveyor",
	)
	if strings.Contains(output, "Selected object") {
		t.Fatal("narrow tree view rendered a second pane")
	}
	for lineNumber, line := range strings.Split(output, "\n") {
		if width := lipgloss.Width(line); width > opts.Width {
			t.Fatalf("line %d has width %d, want <= %d: %q", lineNumber+1, width, opts.Width, line)
		}
	}
}

func TestKeyboardNavigationDrillBackAndDoctor(t *testing.T) {
	graph := fixtureGraph(t)
	var doctorID model.ID
	a := newApp(graph, Options{
		Hostname: "shed-pi-01",
		RunDoctor: func(_ context.Context, id model.ID) (*model.Graph, error) {
			doctorID = id
			return graph, nil
		},
	})
	if action, ok := a.doctorAction("implementation:media-gateway"); !ok || action.ID != "action:media-gateway.doctor" {
		t.Fatalf("implementation doctor action = %q, %v", action.ID, ok)
	}
	a.selected = "component:pattern.conveyor"

	updatedModel, command := a.handleKey("d")
	updated := updatedModel.(app)
	if command != nil || updated.confirmAction != "action:media-gateway.doctor" || doctorID != "" {
		t.Fatalf("doctor ran before confirmation: command nil = %v, pending = %q, ran = %q", command == nil, updated.confirmAction, doctorID)
	}
	if footer := updated.renderFooter(100, newStyles(true, true)); !strings.Contains(footer, "y/enter confirm doctor") {
		t.Fatalf("confirmation footer = %q", footer)
	}

	cancelledModel, _ := updated.handleKey("n")
	cancelled := cancelledModel.(app)
	if cancelled.confirmAction != "" || doctorID != "" || !strings.Contains(cancelled.status, "cancelled") {
		t.Fatalf("cancelled confirmation = %#v, doctor = %q", cancelled, doctorID)
	}

	updatedModel, _ = cancelled.handleKey("d")
	updatedModel, command = updatedModel.(app).handleKey("y")
	updated = updatedModel.(app)
	if command == nil {
		t.Fatal("confirmed doctor returned no command")
	}
	message := command()
	if doctorID != "action:media-gateway.doctor" {
		t.Fatalf("doctor target = %q, want doctor action", doctorID)
	}
	updatedModel, _ = updated.Update(message)
	updated = updatedModel.(app)
	if !strings.Contains(updated.status, "Doctor completed") {
		t.Fatalf("doctor completion status = %q", updated.status)
	}

	updatedModel, _ = updated.handleKey("enter")
	updated = updatedModel.(app)
	if updated.inspect != "component:pattern.conveyor" {
		t.Fatalf("inspect = %q, want selected object", updated.inspect)
	}
	updatedModel, _ = updated.handleKey("backspace")
	updated = updatedModel.(app)
	if updated.inspect != "" {
		t.Fatalf("inspect remained %q after back", updated.inspect)
	}

	updated.selected = "implementation:media-gateway"
	updatedModel, _ = updated.handleKey("j")
	updated = updatedModel.(app)
	if updated.selected != "component:pattern.conveyor" {
		t.Fatalf("j selected %q, want first generic child", updated.selected)
	}
}

func TestRefreshPreservesSelectionAndRejectsInvalidGraph(t *testing.T) {
	graph := fixtureGraph(t)
	a := newApp(graph, Options{Hostname: "shed-pi-01"})
	a.selected = "component:capture"

	updatedModel, _ := a.Update(refreshDoneMsg{graph: graph})
	updated := updatedModel.(app)
	if updated.selected != "component:capture" {
		t.Fatalf("selection after refresh = %q, want component:capture", updated.selected)
	}

	invalid := &model.Graph{SchemaVersion: model.CurrentSchemaVersion, Components: []model.Component{{ID: "broken"}}}
	updatedModel, _ = updated.Update(refreshDoneMsg{graph: invalid})
	updated = updatedModel.(app)
	if updated.graph != graph {
		t.Fatal("invalid refresh replaced the last valid graph")
	}
	if !strings.Contains(updated.status, "Refresh failed") {
		t.Fatalf("invalid refresh status = %q", updated.status)
	}
}

func TestFailedDoctorStillRefreshesWrittenEvidence(t *testing.T) {
	graph := fixtureGraph(t)
	refreshes := 0
	a := newApp(graph, Options{
		Hostname: "shed-pi-01",
		RunDoctor: func(context.Context, model.ID) (*model.Graph, error) {
			return graph, errors.New("completed with unhealthy evidence: exit status 1")
		},
		Refresh: func(context.Context) (*model.Graph, error) {
			refreshes++
			return graph, nil
		},
	})
	a.selected = "component:pattern.conveyor"

	updatedModel, doctorCommand := a.handleKey("d")
	if doctorCommand != nil {
		t.Fatal("doctor ran before confirmation")
	}
	updatedModel, doctorCommand = updatedModel.(app).handleKey("enter")
	if doctorCommand == nil {
		t.Fatal("confirmed doctor returned no command")
	}
	updatedModel, refreshCommand := updatedModel.(app).Update(doctorCommand())
	updated := updatedModel.(app)
	if refreshCommand != nil || !strings.Contains(updated.status, "Doctor for") || !strings.Contains(updated.status, "runtime graph refreshed") {
		t.Fatalf("failed doctor status = %q, unexpected follow-up refresh = %v", updated.status, refreshCommand != nil)
	}
	if refreshes != 0 {
		t.Fatalf("refreshes = %d, status = %q", refreshes, updated.status)
	}
}

func TestAmbiguousDoctorActionsAreNotChosenImplicitly(t *testing.T) {
	graph := fixtureGraph(t)
	component, _ := graph.Lookup("component:pattern.conveyor")
	component.Actions = append(component.Actions, model.Action{
		ID: "action:media-gateway.doctor.deep", Kind: "doctor.run", Label: "Run deep doctor",
		Target: component.ID,
	})
	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
	a := newApp(graph, Options{RunDoctor: func(context.Context, model.ID) (*model.Graph, error) { return graph, nil }})
	a.selected = component.ID
	updatedModel, command := a.handleKey("d")
	updated := updatedModel.(app)
	if command != nil || !strings.Contains(updated.status, "Multiple doctor actions") {
		t.Fatalf("ambiguous doctor command nil = %v, status = %q", command == nil, updated.status)
	}
}

func TestSharedChildDAGIsRenderedOnceAndNavigationRemainsLinear(t *testing.T) {
	graph := dagFixture(t)
	a := newApp(graph, Options{Hostname: "dag-box"})
	rows := a.treeRows()
	want := []model.ID{
		"implementation:dag",
		"component:dag:a",
		"component:dag:shared",
		"component:dag:b",
	}
	if len(rows) != len(want) {
		t.Fatalf("tree rows = %#v, want %d unique objects", rows, len(want))
	}
	for index, id := range want {
		if rows[index].id != id {
			t.Fatalf("tree row %d = %q, want %q", index, rows[index].id, id)
		}
	}

	a.selected = want[0]
	for _, expected := range append(want[1:], want[0]) {
		updatedModel, _ := a.handleKey("j")
		a = updatedModel.(app)
		if a.selected != expected {
			t.Fatalf("selection = %q, want %q", a.selected, expected)
		}
	}
}

func TestBackReturnsSummaryFocusToTreeInNarrowLayout(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "edge-box"})
	a.width = 52
	a.height = 26

	updatedModel, _ := a.handleKey("tab")
	a = updatedModel.(app)
	if a.focus != focusDetail || !strings.Contains(a.render(), "Selected object") {
		t.Fatal("tab did not open the narrow summary pane")
	}

	updatedModel, _ = a.handleKey("h")
	a = updatedModel.(app)
	if a.focus != focusTree || strings.Contains(a.render(), "Selected object") {
		t.Fatal("h did not return the narrow summary to the tree")
	}
}

func TestNoColorUsesVisibleFocusBorderAndHonorsDetectedProfile(t *testing.T) {
	s := newStyles(true, true)
	plain := s.panel.Width(20).Height(4).Render("plain")
	focused := s.focusedPanel.Width(20).Height(4).Render("focused")
	if !strings.Contains(plain, "╭") || !strings.Contains(focused, "╔") || plain == focused {
		t.Fatalf("no-color focus is not visibly distinct\nplain:\n%s\nfocused:\n%s", plain, focused)
	}
	if strings.Contains(plain+focused, "\x1b[") {
		t.Fatal("no-color focus border emitted ANSI escapes")
	}

	a := newApp(fixtureGraph(t), Options{})
	updatedModel, _ := a.Update(tea.ColorProfileMsg{Profile: colorprofile.ASCII})
	a = updatedModel.(app)
	if !a.noColor {
		t.Fatal("ASCII terminal profile did not enable no-color rendering")
	}
	updatedModel, _ = a.Update(tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	a = updatedModel.(app)
	if a.noColor {
		t.Fatal("ANSI terminal profile did not restore color rendering")
	}

	forced := newApp(fixtureGraph(t), Options{NoColor: true})
	updatedModel, _ = forced.Update(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	if !updatedModel.(app).noColor {
		t.Fatal("explicit no-color option was overridden by terminal profile")
	}
}

func dagFixture(t *testing.T) *model.Graph {
	t.Helper()
	health := model.Health{Status: model.HealthHealthy, Summary: "ready"}
	components := []model.Component{
		{ID: "implementation:dag", Kind: "implementation", Health: health, Children: []model.ID{"component:dag:a", "component:dag:b"}},
		{ID: "component:dag:a", Kind: "component.group", Health: health, Children: []model.ID{"component:dag:shared"}},
		{ID: "component:dag:b", Kind: "component.group", Health: health, Children: []model.ID{"component:dag:shared"}},
		{ID: "component:dag:shared", Kind: "systemd.service", Health: health},
	}
	graph, err := model.NewGraph([]model.Implementation{{
		ID:         "implementation:dag",
		Components: []model.ID{"implementation:dag", "component:dag:a", "component:dag:b", "component:dag:shared"},
	}}, components, nil, nil)
	if err != nil {
		t.Fatalf("NewGraph: %v", err)
	}
	return graph
}

func fixtureGraph(t *testing.T) *model.Graph {
	t.Helper()
	observedAt := time.Date(2026, time.July, 10, 12, 30, 0, 0, time.UTC)
	components := []model.Component{
		{
			ID: "implementation:media-gateway", Kind: "implementation",
			Health:   model.Health{Status: model.HealthDegraded, Summary: "publication is waiting for storage"},
			Metadata: model.Metadata{"project": "Media Gateway"},
			Children: []model.ID{"component:pattern.conveyor"},
			What:     "A bounded removable-media pipeline.",
			Why:      "One calm operational surface for capture and publication.",
		},
		{
			ID: "component:pattern.conveyor", Kind: "pattern",
			Health:  model.Health{Status: model.HealthDegraded, Summary: "archive dependency is unavailable"},
			Summary: "Bounded capture, hot handoff, and atomic cold publication.",
			Metadata: model.Metadata{
				"display_name": "Device-triggered Conveyor",
				"pattern":      "T2R4.device-triggered-conveyor",
			},
			Actions: []model.Action{{
				ID: "action:media-gateway.doctor", Kind: "doctor.run", Label: "Run doctor",
				Target: "component:pattern.conveyor", RequiresConfirmation: true,
			}},
			Children: []model.ID{"component:capture", "component:archive"},
			What:     "Turns device readiness into bounded work and an atomic publication.",
			Why:      "Keeps unreliable media away from durable storage until work is complete.",
			Responsibilities: []string{
				"Bound every capture attempt",
				"Publish only completed handoffs",
			},
			FailureModes: []model.FailureMode{{
				ID: "destination-missing", Summary: "Destination disappears",
				Effect:   "Completed work remains hot and unpublished.",
				Recovery: "Restore the mount and rerun publication.",
			}},
		},
		{
			ID: "component:capture", Kind: "systemd.service",
			Health:   model.Health{Status: model.HealthHealthy, Summary: "active (running)"},
			Metadata: model.Metadata{"display_name": "Capture Worker", "unit": "media-capture@.service"},
			What:     "Runs one bounded capture.", Why: "Leaves retries to systemd.",
		},
		{
			ID: "component:archive", Kind: "mount",
			Health:   model.Health{Status: model.HealthDegraded, Summary: "mount is temporarily absent"},
			Metadata: model.Metadata{"display_name": "Archive Mount", "path": "/mnt/archive"},
			What:     "Provides durable cold storage.", Why: "Publication must be atomic.",
		},
		{
			ID: "implementation:weather-station", Kind: "implementation",
			Health:   model.Health{Status: model.HealthHealthy, Summary: "all components healthy"},
			Metadata: model.Metadata{"project": "Weather Station"},
			Children: []model.ID{"component:weather.timer"},
			What:     "Samples weather sensors.", Why: "Makes local conditions inspectable.",
		},
		{
			ID: "component:weather.timer", Kind: "systemd.timer",
			Health:   model.Health{Status: model.HealthHealthy, Summary: "waiting"},
			Metadata: model.Metadata{"display_name": "Sample Timer"},
		},
		{
			ID: "component:weather.publisher", Kind: "systemd.service",
			Health:   model.Health{Status: model.HealthHealthy, Summary: "inactive as permitted"},
			Metadata: model.Metadata{"display_name": "Weather Publisher"},
		},
	}
	implementations := []model.Implementation{
		{
			ID: "implementation:media-gateway", Version: "1.2.3", Summary: "Removable media gateway",
			Components: []model.ID{"implementation:media-gateway", "component:pattern.conveyor", "component:capture", "component:archive"},
		},
		{
			ID: "implementation:weather-station", Version: "0.4.0", Summary: "Weather telemetry",
			Components: []model.ID{"implementation:weather-station", "component:weather.timer", "component:weather.publisher"},
		},
	}
	edges := []model.Edge{
		{From: "implementation:media-gateway", Type: model.EdgeOwns, To: "component:pattern.conveyor"},
		{From: "component:pattern.conveyor", Type: model.EdgeOwns, To: "component:capture"},
		{From: "component:pattern.conveyor", Type: model.EdgeOwns, To: "component:archive"},
		{From: "component:pattern.conveyor", Type: model.EdgeDependsOn, To: "component:archive", Summary: "cold publication target"},
		{From: "implementation:weather-station", Type: model.EdgeOwns, To: "component:weather.timer"},
	}
	observations := []model.Observation{{
		ID: "observation:media-gateway:doctor", ComponentID: "component:pattern.conveyor",
		Kind: model.ObservationDoctor, Status: model.HealthDegraded,
		Summary: "Queue is ready; archive mount is absent.", ObservedAt: observedAt, DurationMS: 842,
		Checks: []model.Check{
			{ID: "queue", Status: model.CheckPass, Summary: "hot queue writable"},
			{ID: "archive", Status: model.CheckWarn, Summary: "mount absent", Evidence: model.Metadata{"path": "/mnt/archive"}},
		},
		Artifacts: []model.Artifact{{URI: "/run/muster/media-gateway/doctor.json", Summary: "machine-readable doctor evidence", MediaType: "application/json"}},
	}}
	graph, err := model.NewGraph(implementations, components, edges, observations)
	if err != nil {
		t.Fatalf("NewGraph: %v", err)
	}
	return graph
}

func assertContains(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Errorf("render did not contain %q\n\n%s", needle, haystack)
		}
	}
}
