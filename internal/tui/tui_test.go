package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
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
		"/ filter",
		"7 objects",
		"Media Gateway",
		"1.2.3",
		"Device-triggered Conveyor",
		"Weather Station · 2 objects · 0.4.0",
		"Selected object",
		"HEALTH CAUSES",
		"LATEST EVIDENCE",
		"enter inspect",
	)
	if strings.Contains(first, "Sample Timer") {
		t.Fatal("fully healthy subtree was not folded by default")
	}
	if strings.Contains(first, "HEALTHY") {
		t.Fatal("healthy objects spelled out a status word; healthy rows must stay quiet")
	}
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
		"pattern · component:pattern.conveyor",
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
		"T2R4.device-triggered-conveyor",
		"ACTIONS",
		"[d] Run doctor · doctor.run",
		"OBSERVATIONS",
		"DOCTOR · 2026-07-10T12:30:00Z",
		"STATUS",
		"CHECK",
		"EVIDENCE",
		"● PASS",
		"hot queue writable",
		"path: /mnt/archive",
		"Artifact: /run/muster/media-gateway/doctor.json",
	)
}

func TestInspectChecksTableFocusAndCursor(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01", NoColor: true})
	a.width, a.height = 110, 80
	a.selected = "component:pattern.conveyor"
	a.inspect = a.selected

	if count := a.inspectChecksCount(); count != 3 {
		t.Fatalf("checks row count = %d, want 3 (two checks plus one evidence row)", count)
	}
	updatedModel, _ := a.handleKey("tab")
	a = updatedModel.(app)
	if !a.tableFocused {
		t.Fatal("tab did not focus the checks table")
	}
	updatedModel, _ = a.handleKey("j")
	a = updatedModel.(app)
	if a.tableCursor != 1 || a.scroll != 0 {
		t.Fatalf("j moved cursor to %d (scroll %d), want row 1 with no pane scroll", a.tableCursor, a.scroll)
	}
	cursorRow := ""
	for _, line := range strings.Split(a.render(), "\n") {
		if strings.Contains(line, "▌") {
			cursorRow = line
		}
	}
	if !strings.Contains(cursorRow, "◐ WARN") || !strings.Contains(cursorRow, "archive") {
		t.Fatalf("cursor bar is not on the archive check row: %q", cursorRow)
	}
	updatedModel, _ = a.handleKey("esc")
	a = updatedModel.(app)
	if a.tableFocused || a.inspect == "" {
		t.Fatal("esc should leave the table but stay in the inspect view")
	}
}

func TestSidebarRendersCardsAndStrips(t *testing.T) {
	graph := fixtureGraph(t)
	base := RenderOptions{Hostname: "shed-pi-01", Width: 118, Height: 34, NoColor: true}

	strip := base
	strip.Selected = "implementation:weather-station"
	output := Render(graph, strip)
	assertContains(t, output,
		"▌▸ ● Weather Station · 2 objects · 0.4.0",
		"─ ◐ Media Gateway",
	)

	header := base
	header.Selected = "implementation:media-gateway"
	output = Render(graph, header)
	assertContains(t, output,
		"╔▌ ◐ Media Gateway",
		" ▸ ● Weather Station · 2 objects · 0.4.0",
	)
}

func TestFilterNarrowsCardsAndKeepsLineage(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01", NoColor: true})
	a.width, a.height = 118, 34

	updatedModel, _ := a.handleKey("/")
	a = updatedModel.(app)
	if !a.filterEditing {
		t.Fatal("/ did not focus the filter input")
	}
	a.filter.SetValue("sample")
	a.selectFirstMatch()

	if a.selected != "component:weather.timer" {
		t.Fatalf("selection = %q, want the first matching object", a.selected)
	}
	if count := a.filterMatchCount(); count != 1 {
		t.Fatalf("match count = %d, want 1", count)
	}
	visible := map[model.ID]bool{}
	for _, row := range a.treeRows() {
		visible[row.id] = true
	}
	if !visible["implementation:weather-station"] || !visible["component:weather.timer"] {
		t.Fatal("matching object or its lineage is missing from the filtered view")
	}
	if visible["component:pattern.conveyor"] {
		t.Fatal("non-matching subtree survived the filter")
	}

	a.filterEditing = false
	output := a.render()
	assertContains(t, output,
		"/ sample",
		"1 match",
		"Sample Timer",
		"0.4.0 · 1 of 2 shown",
		"§ FILTER",
		"1 of 7 objects match “sample”",
	)
	if strings.Contains(output, "Device-triggered Conveyor") {
		t.Fatal("filtered-out card still shows its rows")
	}

	updatedModel, _ = a.handleKey("esc")
	a = updatedModel.(app)
	if a.filterQuery() != "" {
		t.Fatal("esc did not clear the applied filter")
	}
}

func TestInspectOmitsUndeclaredSections(t *testing.T) {
	graph := fixtureGraph(t)
	output := Render(graph, RenderOptions{
		Hostname: "shed-pi-01", Width: 110, Height: 60, NoColor: true,
		Selected: "component:weather.publisher",
		Inspect:  "component:weather.publisher",
	})

	for _, placeholder := range []string{
		"Not declared.",
		"No responsibilities declared.",
		"No failure modes declared.",
		"No child components.",
		"No direct graph relations.",
		"No transitive dependency paths.",
		"No metadata.",
		"No actions advertised.",
	} {
		if strings.Contains(output, placeholder) {
			t.Errorf("inspect rendered placeholder %q for undeclared content", placeholder)
		}
	}
	for _, section := range []string{"WHAT", "WHY", "RESPONSIBILITIES", "FAILURE MODES", "RELATIONS", "ACTIONS"} {
		if strings.Contains(output, section) {
			t.Errorf("inspect rendered section %q for undeclared content", section)
		}
	}
	assertContains(t, output,
		"Weather Publisher",
		"HEALTH EXPLANATION",
		"OBSERVATIONS",
		"No recorded observations.",
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
		"7 objects",
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

func TestConfirmationDialogFitsMinimumTerminal(t *testing.T) {
	graph := fixtureGraph(t)
	a := newApp(graph, Options{Hostname: "edge-box"})
	a.width = minimumWidth
	a.height = minimumHeight
	a.confirmAction = "action:media-gateway.doctor"
	a.confirmTarget = "implementation:media-gateway"

	output := a.render()
	assertContains(t, output, "CONFIRM DOCTOR", "y/enter", "n/esc")
	for lineNumber, line := range strings.Split(output, "\n") {
		if width := lipgloss.Width(line); width > minimumWidth {
			t.Fatalf("line %d has width %d, want <= %d: %q", lineNumber+1, width, minimumWidth, line)
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
	messages := drainCommand(command)
	if doctorID != "action:media-gateway.doctor" {
		t.Fatalf("doctor target = %q, want doctor action", doctorID)
	}
	for _, message := range messages {
		updatedModel, _ = updated.Update(message)
		updated = updatedModel.(app)
	}
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
	var refreshCommand tea.Cmd
	for _, message := range drainCommand(doctorCommand) {
		var followUp tea.Cmd
		updatedModel, followUp = updatedModel.(app).Update(message)
		if _, isDone := message.(doctorDoneMsg); isDone {
			refreshCommand = followUp
		}
	}
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
	// The shared node is degraded so every path to it unfolds by default and
	// the whole DAG is visible without operator toggles.
	shared := model.Health{Status: model.HealthDegraded, Summary: "flaky"}
	components := []model.Component{
		{ID: "implementation:dag", Kind: "implementation", Health: health, Children: []model.ID{"component:dag:a", "component:dag:b"}},
		{ID: "component:dag:a", Kind: "component.group", Health: health, Children: []model.ID{"component:dag:shared"}},
		{ID: "component:dag:b", Kind: "component.group", Health: health, Children: []model.ID{"component:dag:shared"}},
		{ID: "component:dag:shared", Kind: "systemd.service", Health: shared},
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

func TestOverviewLeadsWithEvidenceAndDemotesLiterateProse(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01", NoColor: true})
	lines := strings.Join(a.overviewLines("component:pattern.conveyor", 80, newStyles(true, true)), "\n")
	causes := strings.Index(lines, "HEALTH CAUSES")
	evidence := strings.Index(lines, "LATEST EVIDENCE")
	what := strings.Index(lines, "WHAT")
	why := strings.Index(lines, "WHY")
	if causes < 0 || evidence < 0 || what < 0 || why < 0 {
		t.Fatalf("overview is missing a section: causes=%d evidence=%d what=%d why=%d\n%s", causes, evidence, what, why, lines)
	}
	if !(causes < evidence && evidence < what && what < why) {
		t.Fatalf("overview order is not evidence-first: causes=%d evidence=%d what=%d why=%d", causes, evidence, what, why)
	}
	if strings.Contains(lines, "literate") {
		t.Fatalf("overview still speaks internal vocabulary:\n%s", lines)
	}
}

func TestHealthySubtreesFoldAndToggleOpen(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01"})

	visible := func() map[model.ID]bool {
		ids := map[model.ID]bool{}
		for _, row := range a.treeRows() {
			ids[row.id] = true
		}
		return ids
	}

	rows := visible()
	if !rows["component:pattern.conveyor"] || !rows["component:archive"] {
		t.Fatal("path to the degraded archive is not open by default")
	}
	if rows["component:weather.timer"] {
		t.Fatal("fully healthy weather-station subtree is not folded by default")
	}
	for _, row := range a.treeRows() {
		if row.id == "implementation:weather-station" && row.hidden != 2 {
			t.Fatalf("folded weather-station hides %d objects, want 2", row.hidden)
		}
	}

	a.selected = "implementation:weather-station"
	updatedModel, _ := a.handleKey("space")
	a = updatedModel.(app)
	if !visible()["component:weather.timer"] {
		t.Fatal("space did not unfold the selected subtree")
	}
	updatedModel, _ = a.handleKey("h")
	a = updatedModel.(app)
	if visible()["component:weather.timer"] {
		t.Fatal("h did not fold the open subtree")
	}

	a.selected = "component:capture"
	updatedModel, _ = a.handleKey("h")
	a = updatedModel.(app)
	if a.selected != "component:pattern.conveyor" {
		t.Fatalf("h on a leaf selected %q, want its parent", a.selected)
	}

	updatedModel, _ = a.handleKey("h")
	a = updatedModel.(app)
	if a.expandedState("component:pattern.conveyor", a.nodeChildren("component:pattern.conveyor")) {
		t.Fatal("h did not fold the open conveyor subtree")
	}
	updatedModel, command := a.handleKey("l")
	a = updatedModel.(app)
	if command != nil || a.inspect != "" || !a.expandedState("component:pattern.conveyor", a.nodeChildren("component:pattern.conveyor")) {
		t.Fatal("l on a folded node did not unfold it")
	}
	updatedModel, command = a.handleKey("l")
	a = updatedModel.(app)
	if a.inspect != "component:pattern.conveyor" || command == nil {
		t.Fatal("l on an open node did not inspect it")
	}
}

func TestFoldedSelectionFallsBackToVisibleAncestor(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01"})
	a.selected = "implementation:weather-station"
	updatedModel, _ := a.handleKey("space")
	a = updatedModel.(app)
	a.selected = "component:weather.timer"

	a.expanded = map[model.ID]bool{}
	a.ensureSelection()
	if a.selected != "implementation:weather-station" {
		t.Fatalf("selection after folding = %q, want nearest visible ancestor", a.selected)
	}
}

func TestContextualNamesDropLineageAndKeepUnitIdentifiers(t *testing.T) {
	health := model.Health{Status: model.HealthHealthy}
	broken := model.Health{Status: model.HealthUnhealthy, Summary: "activation failed"}
	components := []model.Component{
		{
			ID: "implementation:bt-audio-gateway", Kind: "implementation", Health: broken,
			Metadata: model.Metadata{"project": "Bt Audio Gateway"},
			Children: []model.ID{"component:bt-audio-gateway:services"},
		},
		{
			ID: "component:bt-audio-gateway:services", Kind: "component.group", Health: broken,
			Children: []model.ID{"component:bt-audio-gateway:unit:snapclient-bt@.service"},
		},
		{ID: "component:bt-audio-gateway:unit:snapclient-bt@.service", Kind: "systemd.service", Health: broken},
	}
	graph, err := model.NewGraph([]model.Implementation{{
		ID: "implementation:bt-audio-gateway",
		Components: []model.ID{
			"implementation:bt-audio-gateway",
			"component:bt-audio-gateway:services",
			"component:bt-audio-gateway:unit:snapclient-bt@.service",
		},
	}}, components, nil, nil)
	if err != nil {
		t.Fatalf("NewGraph: %v", err)
	}
	_ = health

	a := newApp(graph, Options{Hostname: "euterpe"})
	labels := map[model.ID]string{}
	for _, row := range a.treeRows() {
		labels[row.id] = row.label
	}
	want := map[model.ID]string{
		"implementation:bt-audio-gateway":                        "Bt Audio Gateway",
		"component:bt-audio-gateway:services":                    "Services",
		"component:bt-audio-gateway:unit:snapclient-bt@.service": "snapclient-bt@.service",
	}
	for id, expected := range want {
		if labels[id] != expected {
			t.Errorf("label for %s = %q, want %q", id, labels[id], expected)
		}
	}
}

func TestDetailPanesScrollWithViewportAndScrollbar(t *testing.T) {
	graph := fixtureGraph(t)
	output := Render(graph, RenderOptions{
		Hostname: "shed-pi-01", Width: 110, Height: 24, NoColor: true,
		Selected: "component:pattern.conveyor",
		Inspect:  "component:pattern.conveyor",
	})
	if !strings.Contains(output, "┃") || !strings.Contains(output, "╎") {
		t.Fatal("overflowing inspect pane did not render a scrollbar")
	}
	if strings.Contains(output, "more line") {
		t.Fatal("scrollbar should replace the more-lines hint in detail panes")
	}

	a := newApp(graph, Options{Hostname: "shed-pi-01", NoColor: true})
	a.width, a.height = 110, 24
	a.selected = "component:pattern.conveyor"
	a.inspect = a.selected
	page := a.detailViewportHeight()

	updatedModel, _ := a.handleKey("pgdown")
	a = updatedModel.(app)
	if a.scroll != page {
		t.Fatalf("pgdown scrolled to %d, want one page (%d)", a.scroll, page)
	}
	updatedModel, _ = a.handleKey("G")
	a = updatedModel.(app)
	if a.scroll != a.scrollLimit() {
		t.Fatalf("G scrolled to %d, want bottom (%d)", a.scroll, a.scrollLimit())
	}
	updatedModel, _ = a.handleKey("g")
	a = updatedModel.(app)
	if a.scroll != 0 {
		t.Fatalf("g scrolled to %d, want top", a.scroll)
	}
}

func TestMotionSpringSettlesAndHonorsReducedMotion(t *testing.T) {
	a := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01"})
	a.width, a.height = 118, 34
	a.selected = "component:pattern.conveyor"

	updatedModel, command := a.handleKey("enter")
	a = updatedModel.(app)
	if a.inspect == "" || command == nil {
		t.Fatal("opening the inspector did not start the entrance glide")
	}
	if a.displayScroll() >= 0 {
		t.Fatalf("entrance display offset = %d, want negative", a.displayScroll())
	}
	for frame := 0; frame < 600 && a.animating; frame++ {
		updatedModel, _ = a.Update(frameMsg(time.Now()))
		a = updatedModel.(app)
	}
	if a.animating || a.displayScroll() != 0 {
		t.Fatalf("entrance did not settle: animating = %v, offset = %d", a.animating, a.displayScroll())
	}

	updatedModel, command = a.handleKey("j")
	a = updatedModel.(app)
	if a.scroll != 1 || command == nil || !a.animating {
		t.Fatalf("scroll glide did not start: target = %d, animating = %v", a.scroll, a.animating)
	}

	updatedModel, command = a.Update(ledTickMsg(time.Now()))
	a = updatedModel.(app)
	if !a.ledDim || command == nil {
		t.Fatal("LED tick did not toggle and reschedule")
	}

	if _, command = a.Update(spinner.TickMsg{}); command != nil {
		t.Fatal("idle spinner tick was rescheduled")
	}

	t.Setenv(reduceMotionEnv, "1")
	calm := newApp(fixtureGraph(t), Options{Hostname: "shed-pi-01"})
	calm.selected = "component:pattern.conveyor"
	updatedModel, command = calm.handleKey("enter")
	calm = updatedModel.(app)
	if command != nil || calm.animating || calm.displayScroll() != 0 {
		t.Fatal("reduced motion still animated the inspector entrance")
	}
}

// drainCommand executes a command, flattening tea.Batch trees into the
// messages they produce, so tests can dispatch them the way a program would.
func drainCommand(command tea.Cmd) []tea.Msg {
	if command == nil {
		return nil
	}
	message := command()
	if batch, ok := message.(tea.BatchMsg); ok {
		var messages []tea.Msg
		for _, sub := range batch {
			messages = append(messages, drainCommand(sub)...)
		}
		return messages
	}
	return []tea.Msg{message}
}

func assertContains(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Errorf("render did not contain %q\n\n%s", needle, haystack)
		}
	}
}
