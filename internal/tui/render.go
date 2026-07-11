package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"

	"github.com/azide0x37/muster/internal/model"
)

const (
	wideBreakpoint = 88
	minimumWidth   = 34
	minimumHeight  = 12
)

func (a app) render() string {
	width, height := a.dimensions()
	s := newStyles(a.dark, a.noColor)
	header := a.renderHeader(width, s)
	footer := a.renderFooter(width, s)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	var body string
	switch {
	case width < minimumWidth || height < minimumHeight:
		body = a.renderTooSmall(width, bodyHeight, s)
	case a.help:
		body = a.renderHelp(width, bodyHeight, s)
	case a.inspect != "":
		body = a.renderInspector(width, bodyHeight, s)
	default:
		body = a.renderBrowser(width, bodyHeight, s)
	}

	content := strings.Join([]string{header, body, footer}, "\n")
	return s.screen.
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
}

func (a app) dimensions() (int, int) {
	width, height := a.width, a.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	return width, height
}

func (a app) renderHeader(width int, s styles) string {
	strap := s.brand.Render("MUSTER") + " " + s.eyebrow.Render("SERVER INSPECTOR")
	title := s.title.Render(truncatePlain("Muster implementations on "+a.hostname, width))
	counts := s.subtitle.Render(truncatePlain(a.implementationCounts(), width))
	return strings.Join([]string{strap, title, counts}, "\n")
}

func (a app) implementationCounts() string {
	counts := map[model.HealthStatus]int{
		model.HealthHealthy:   0,
		model.HealthDegraded:  0,
		model.HealthUnhealthy: 0,
		model.HealthUnknown:   0,
	}
	for _, implementation := range a.graph.Implementations {
		status := model.HealthUnknown
		if component, ok := a.graph.Lookup(implementation.ID); ok {
			status = normalizedStatus(component.Health.Status)
		}
		counts[status]++
	}
	count := len(a.graph.Implementations)
	parts := []string{fmt.Sprintf("%d %s", count, plural(count, "implementation", "implementations"))}
	for _, status := range []model.HealthStatus{
		model.HealthHealthy, model.HealthDegraded, model.HealthUnhealthy, model.HealthUnknown,
	} {
		if counts[status] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[status], status))
		}
	}
	if count == 0 {
		parts = append(parts, "nothing registered yet")
	}
	return strings.Join(parts, " · ")
}

func (a app) renderFooter(width int, s styles) string {
	contextLine := "Read-only view · live state and declared intent share one graph"
	if a.status != "" {
		contextLine = a.status
	}
	if a.busy && a.operation != "" {
		contextLine = "◌ " + a.operation + " · " + contextLine
	}

	keys := "↑↓/jk select · tab pane · enter inspect · r refresh · ? help · q quit"
	if a.confirmAction != "" {
		keys = "y/enter confirm doctor · n/esc cancel · q quit"
		return s.muted.Render(truncatePlain(contextLine, width)) + "\n" +
			s.faint.Render(truncatePlain(keys, width))
	}
	if a.inspect != "" {
		keys = "↑↓/jk scroll · esc/backspace back · r refresh · ? help · q quit"
	}
	if _, ok := a.doctorAction(a.activeID()); ok && a.opts.RunDoctor != nil {
		keys = strings.Replace(keys, "r refresh", "d doctor · r refresh", 1)
	}
	return s.muted.Render(truncatePlain(contextLine, width)) + "\n" +
		s.faint.Render(truncatePlain(keys, width))
}

func (a app) renderTooSmall(width, height int, s styles) string {
	lines := []string{
		s.section.Render("A LITTLE MORE ROOM"),
		s.body.Render("Muster keeps the view legible instead of crushing the graph."),
		s.muted.Render(fmt.Sprintf("Current terminal: %d×%d · minimum: %d×%d", width, a.height, minimumWidth, minimumHeight)),
	}
	return a.panel(width, height, false, "Terminal too small", lines, 0, s)
}

func (a app) renderHelp(width, height int, s styles) string {
	contentWidth := max(12, width-4)
	w := newLineWriter(contentWidth, s)
	w.section("Navigation")
	w.keyValue("↑ / k", "move up or scroll up")
	w.keyValue("↓ / j", "move down or scroll down")
	w.keyValue("tab", "move focus between the implementation tree and its summary")
	w.keyValue("enter / → / l", "open the selected inspectable object")
	w.keyValue("esc / backspace / ← / h", "return to the implementation tree")
	w.section("Operations")
	w.keyValue("r", "refresh the normalized runtime graph")
	w.keyValue("d", "run doctor only when the selected object advertises that action")
	w.keyValue("?", "open or close this help")
	w.keyValue("q / ctrl+c", "leave Muster")
	w.section("Reading health")
	w.text("● HEALTHY  the object is operating within its declared contract", s.good)
	w.text("◐ DEGRADED  useful, but attention is warranted", s.warn)
	w.text("× UNHEALTHY  the declared contract is not being met", s.bad)
	w.text("? UNKNOWN  Muster does not have enough current evidence", s.unknown)
	w.blank()
	w.paragraph("Design rule", "Color reinforces a word and glyph; it never carries status alone.")
	return a.panel(width, height, true, "Keyboard and language", w.lines, 0, s)
}

func (a app) renderBrowser(width, height int, s styles) string {
	if width < wideBreakpoint {
		if a.focus == focusDetail {
			return a.renderSummaryPanel(width, height, true, s)
		}
		return a.renderTreePanel(width, height, true, s)
	}
	leftWidth := clamp(width*38/100, 34, 50)
	rightWidth := width - leftWidth - 1
	left := a.renderTreePanel(leftWidth, height, a.focus == focusTree, s)
	right := a.renderSummaryPanel(rightWidth, height, a.focus == focusDetail, s)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (a app) renderTreePanel(width, height int, focused bool, s styles) string {
	rows := a.treeRows()
	innerWidth := max(8, width-4)
	lines := a.renderTreeLines(rows, innerWidth, s)
	viewportHeight := max(1, height-4)
	offset := selectedOffset(rows, a.selected, viewportHeight)
	title := fmt.Sprintf("Implementations · %d objects", len(rows))
	return a.panel(width, height, focused, title, lines, offset, s)
}

func (a app) renderSummaryPanel(width, height int, focused bool, s styles) string {
	innerWidth := max(8, width-4)
	lines := a.overviewLines(a.selected, innerWidth, s)
	title := "Selected object"
	if component, ok := a.graph.Lookup(a.selected); ok {
		title += " · " + string(component.Kind) + " · " + string(component.ID)
	}
	return a.panel(width, height, focused, title, lines, a.scroll, s)
}

func (a app) renderInspector(width, height int, s styles) string {
	innerWidth := max(8, width-4)
	lines := a.inspectLines(a.inspect, innerWidth, s)
	title := "Inspect"
	if component, ok := a.graph.Lookup(a.inspect); ok {
		title = "Inspect · " + string(component.Kind) + " · " + string(component.ID)
	}
	return a.panel(width, height, true, title, lines, a.scroll, s)
}

func (a app) panel(width, height int, focused bool, title string, lines []string, offset int, s styles) string {
	width = max(4, width)
	height = max(3, height)
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	viewportHeight := max(1, innerHeight-2)
	visible := viewport(lines, offset, viewportHeight, s)
	title = truncatePlain(title, innerWidth)
	content := s.eyebrow.Render(title) + "\n" +
		s.faint.Render(strings.Repeat("─", innerWidth)) + "\n" +
		strings.Join(visible, "\n")
	style := s.panel
	if focused {
		style = s.focusedPanel
	}
	return style.
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
}

func viewport(lines []string, offset, height int, s styles) []string {
	if height <= 0 {
		return nil
	}
	maximum := max(0, len(lines)-height)
	offset = clamp(offset, 0, maximum)
	end := min(len(lines), offset+height)
	visible := append([]string(nil), lines[offset:end]...)
	for len(visible) < height {
		visible = append(visible, "")
	}
	if offset > 0 && len(visible) > 0 {
		visible[0] = s.faint.Render(fmt.Sprintf("↑ %d more line%s", offset, pluralSuffix(offset)))
	}
	if remaining := len(lines) - end; remaining > 0 && len(visible) > 0 {
		visible[len(visible)-1] = s.faint.Render(fmt.Sprintf("↓ %d more line%s", remaining, pluralSuffix(remaining)))
	}
	return visible
}

func (a app) overviewLines(id model.ID, width int, s styles) []string {
	component, ok := a.graph.Lookup(id)
	if !ok {
		w := newLineWriter(width, s)
		w.section("No object selected")
		w.paragraph("What", "Register a Muster implementation to populate this server graph.")
		return w.lines
	}

	w := newLineWriter(width, s)
	w.text(displayName(*component), s.title)
	w.text(string(component.Kind)+" · "+string(component.ID), s.muted)
	w.blank()
	w.health(component.Health)
	if implementation, found := a.implementation(component.ID); found && implementation.Version != "" {
		w.keyValue("Version", implementation.Version)
	}

	what := firstNonEmpty(component.What, component.Summary, "No literate description has been declared yet.")
	w.paragraph("What", what)
	why := firstNonEmpty(component.Why, "No rationale has been declared yet.")
	w.paragraph("Why", why)

	if len(component.Children) > 0 {
		w.section(fmt.Sprintf("Children · %d", len(component.Children)))
		for _, childID := range component.Children {
			if child, found := a.graph.Lookup(childID); found {
				w.healthObject("• "+displayName(*child), child.Health)
			}
		}
	}

	if observation, found := a.graph.LatestObservation(component.ID, model.ObservationDoctor); found {
		w.section("Latest doctor")
		w.health(observation.DerivedHealth())
	}

	relations := a.directRelations(component.ID)
	if len(relations) > 0 {
		w.section("Relations")
		for _, relation := range relations {
			w.bullet(relation)
		}
	}

	if len(component.Actions) > 0 {
		w.section("Actions")
		for _, action := range component.Actions {
			prefix := "•"
			if isDoctorAction(action) {
				prefix = "[d]"
			}
			w.bulletWithPrefix(prefix, actionLabel(action))
		}
	}

	w.blank()
	w.text("Enter opens the complete literate object: intent, failure modes, evidence, and graph paths.", s.faint)
	return w.lines
}

func (a app) inspectLines(id model.ID, width int, s styles) []string {
	component, ok := a.graph.Lookup(id)
	if !ok {
		w := newLineWriter(width, s)
		w.section("Object no longer present")
		w.text(string(id), s.muted)
		return w.lines
	}

	w := newLineWriter(width, s)
	w.text(displayName(*component), s.title)
	w.text(string(component.Kind)+" · "+string(component.ID), s.muted)
	w.blank()
	w.health(component.Health)
	if implementation, found := a.implementation(component.ID); found {
		if implementation.Version != "" {
			w.keyValue("Version", implementation.Version)
		}
		if implementation.Summary != "" && implementation.Summary != component.Summary {
			w.keyValue("Implementation", implementation.Summary)
		}
	}

	w.paragraph("What", firstNonEmpty(component.What, component.Summary, "Not declared."))
	w.paragraph("Why", firstNonEmpty(component.Why, "Not declared."))

	w.section("Responsibilities")
	if len(component.Responsibilities) == 0 {
		w.text("No responsibilities declared.", s.faint)
	} else {
		for _, responsibility := range component.Responsibilities {
			w.bullet(responsibility)
		}
	}

	w.section("Failure modes")
	if len(component.FailureModes) == 0 {
		w.text("No failure modes declared.", s.faint)
	} else {
		for _, failure := range component.FailureModes {
			w.text("× "+firstNonEmpty(failure.Summary, failure.ID), s.bad)
			if failure.ID != "" && failure.ID != failure.Summary {
				w.keyValue("Mode", failure.ID)
			}
			if failure.Effect != "" {
				w.keyValue("Effect", failure.Effect)
			}
			if failure.Recovery != "" {
				w.keyValue("Recovery", failure.Recovery)
			}
		}
	}

	w.section("Health explanation")
	if explanation, err := a.graph.Explain(id); err == nil {
		w.keyValue("Declared", healthWords(explanation.Health.Declared))
		w.keyValue("Effective", healthWords(explanation.Health.Effective))
		for _, cause := range explanation.Health.Causes {
			line := "Cause: " + a.nameFor(cause.ComponentID)
			if len(cause.Path) > 0 {
				line += " via " + healthPath(cause.Path, a)
			} else {
				line += " (declared here)"
			}
			w.bullet(line)
		}
	}

	w.section("Children · hierarchy")
	childRows := a.childRows(component.ID)
	if len(childRows) == 0 {
		w.text("No child components.", s.faint)
	} else {
		for _, row := range childRows {
			child, found := a.graph.Lookup(row.id)
			if !found {
				continue
			}
			w.healthObject(row.prefix+displayName(*child), child.Health)
		}
	}

	w.section("Relations")
	relations := a.directRelations(component.ID)
	if len(relations) == 0 {
		w.text("No direct graph relations.", s.faint)
	} else {
		for _, relation := range relations {
			w.bullet(relation)
		}
	}

	w.section("Dependency explanation")
	if dependencies, err := a.graph.ExplainDependencies(component.ID); err == nil {
		if len(dependencies.DependsOn) == 0 && len(dependencies.RequiredBy) == 0 {
			w.text("No transitive dependency paths.", s.faint)
		}
		for _, dependency := range dependencies.DependsOn {
			w.bullet("Needs " + a.nameFor(dependency.ComponentID) + " via " + a.idPath(dependency.Path))
		}
		for _, dependent := range dependencies.RequiredBy {
			w.bullet("Needed by " + a.nameFor(dependent.ComponentID) + " via " + a.idPath(dependent.Path))
		}
	}

	w.section("Metadata")
	if len(component.Metadata) == 0 {
		w.text("No metadata.", s.faint)
	} else {
		keys := make([]string, 0, len(component.Metadata))
		for key := range component.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.keyValue(key, component.Metadata[key])
		}
	}

	w.section("Actions")
	if len(component.Actions) == 0 {
		w.text("No actions advertised.", s.faint)
	} else {
		for _, action := range component.Actions {
			prefix := "•"
			if isDoctorAction(action) {
				prefix = "[d]"
			}
			w.bulletWithPrefix(prefix, actionLabel(action))
			if action.Summary != "" {
				w.text(action.Summary, s.muted)
			}
			if action.RequiresConfirmation {
				w.text("Confirmation required before execution.", s.faint)
			}
			if action.RequiresRoot {
				w.text("Requires root; rerun Muster with sudo to execute.", s.faint)
			}
		}
	}

	w.section("Observations")
	observations := a.graph.ObservationsFor(component.ID)
	if len(observations) == 0 {
		w.text("No recorded observations.", s.faint)
	} else {
		for _, observation := range observations {
			w.healthObject(strings.ToUpper(string(observation.Kind))+" · "+observation.ObservedAt.Format(time.RFC3339), observation.DerivedHealth())
			if observation.Summary != "" {
				w.text(observation.Summary, s.body)
			}
			w.keyValue("Duration", fmt.Sprintf("%d ms", observation.DurationMS))
			for _, check := range observation.Checks {
				w.check(check)
			}
			for _, artifact := range observation.Artifacts {
				value := artifact.URI
				if artifact.Summary != "" {
					value += " — " + artifact.Summary
				}
				w.keyValue("Artifact", value)
			}
		}
	}

	return w.lines
}

func (a app) directRelations(id model.ID) []string {
	result := make([]string, 0)
	for _, edge := range a.graph.Outgoing(id) {
		line := "→ " + relationWords(edge.Type) + " " + a.nameFor(edge.To)
		if edge.Summary != "" {
			line += " — " + edge.Summary
		}
		result = append(result, line)
	}
	for _, edge := range a.graph.Incoming(id) {
		verb := relationWords(edge.Type) + " from"
		if edge.Type == model.EdgeDependsOn {
			verb = "required by"
		}
		line := "← " + verb + " " + a.nameFor(edge.From)
		if edge.Summary != "" {
			line += " — " + edge.Summary
		}
		result = append(result, line)
	}
	return result
}

func (a app) implementation(id model.ID) (model.Implementation, bool) {
	for _, implementation := range a.graph.Implementations {
		if implementation.ID == id {
			return implementation, true
		}
	}
	return model.Implementation{}, false
}

func (a app) idPath(path []model.ID) string {
	parts := make([]string, 0, len(path))
	for _, id := range path {
		parts = append(parts, a.nameFor(id))
	}
	return strings.Join(parts, " → ")
}

func (a app) nameFor(id model.ID) string {
	if component, ok := a.graph.Lookup(id); ok {
		return displayName(*component)
	}
	if id == "" {
		return "the selected object"
	}
	return string(id)
}

func (a app) clampScroll(scroll int) int {
	return clamp(scroll, 0, a.scrollLimit())
}

func (a app) scrollLimit() int {
	width, height := a.dimensions()
	headerHeight := 3
	footerHeight := 2
	bodyHeight := max(3, height-headerHeight-footerHeight-2)
	viewportHeight := max(1, bodyHeight-4)
	s := newStyles(a.dark, a.noColor)
	var lineCount int
	if a.inspect != "" {
		lineCount = len(a.inspectLines(a.inspect, max(8, width-4), s))
	} else if a.focus == focusDetail {
		panelWidth := width
		if width >= wideBreakpoint {
			leftWidth := clamp(width*38/100, 34, 50)
			panelWidth = width - leftWidth - 1
		}
		lineCount = len(a.overviewLines(a.selected, max(8, panelWidth-4), s))
	}
	return max(0, lineCount-viewportHeight)
}

type treeRow struct {
	id     model.ID
	prefix string
	root   bool
}

func (a app) treeRows() []treeRow {
	if a.graph == nil {
		return nil
	}
	rows := make([]treeRow, 0, len(a.graph.Components))
	for _, implementation := range a.graph.Implementations {
		root, ok := a.graph.Lookup(implementation.ID)
		if !ok {
			continue
		}
		rows = append(rows, treeRow{id: root.ID, root: true})
		seen := map[model.ID]bool{root.ID: true}
		children := append([]model.ID(nil), root.Children...)
		for _, child := range children {
			markDescendants(a.graph, child, seen)
		}
		for _, componentID := range implementation.Components {
			if !seen[componentID] {
				children = append(children, componentID)
				markDescendants(a.graph, componentID, seen)
			}
		}
		path := map[model.ID]bool{root.ID: true}
		rendered := map[model.ID]bool{root.ID: true}
		for index, child := range children {
			a.appendTree(&rows, child, "", index == len(children)-1, path, rendered)
		}
	}
	return rows
}

func markDescendants(graph *model.Graph, id model.ID, seen map[model.ID]bool) {
	if seen[id] {
		return
	}
	seen[id] = true
	if component, ok := graph.Lookup(id); ok {
		for _, child := range component.Children {
			markDescendants(graph, child, seen)
		}
	}
}

func (a app) appendTree(rows *[]treeRow, id model.ID, parentPrefix string, last bool, path, rendered map[model.ID]bool) {
	if path[id] || rendered[id] {
		return
	}
	component, ok := a.graph.Lookup(id)
	if !ok {
		return
	}
	connector := "├─ "
	nextPrefix := parentPrefix + "│  "
	if last {
		connector = "└─ "
		nextPrefix = parentPrefix + "   "
	}
	*rows = append(*rows, treeRow{id: id, prefix: parentPrefix + connector})
	rendered[id] = true
	nextPath := clonePath(path)
	nextPath[id] = true
	for index, child := range component.Children {
		a.appendTree(rows, child, nextPrefix, index == len(component.Children)-1, nextPath, rendered)
	}
}

func (a app) childRows(id model.ID) []treeRow {
	component, ok := a.graph.Lookup(id)
	if !ok {
		return nil
	}
	rows := make([]treeRow, 0)
	path := map[model.ID]bool{id: true}
	rendered := map[model.ID]bool{id: true}
	for index, child := range component.Children {
		a.appendTree(&rows, child, "", index == len(component.Children)-1, path, rendered)
	}
	return rows
}

func clonePath(source map[model.ID]bool) map[model.ID]bool {
	result := make(map[model.ID]bool, len(source)+1)
	for id, present := range source {
		result[id] = present
	}
	return result
}

func (a app) renderTreeLines(rows []treeRow, width int, s styles) []string {
	if len(rows) == 0 {
		return []string{
			s.section.Render("NO IMPLEMENTATIONS REGISTERED"),
			s.body.Render("The first Muster service installed on this server will appear here."),
			s.muted.Render("The console itself is ready."),
		}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		component, ok := a.graph.Lookup(row.id)
		if !ok {
			continue
		}
		indicator := "  "
		if row.id == a.selected {
			indicator = "▸ "
		}
		statusWord := strings.ToUpper(string(normalizedStatus(component.Health.Status)))
		plainPrefix := indicator + row.prefix
		nameWidth := max(1, width-runeWidth(plainPrefix)-runeWidth(statusWord)-5)
		name := truncatePlain(displayName(*component), nameWidth)
		line := plainPrefix + healthStyle(component.Health.Status, s).Render(healthGlyph(component.Health.Status)) +
			" " + name + " · " + statusWord
		if row.id == a.selected {
			line = s.selected.Width(width).MaxWidth(width).Render(line)
		} else if row.root {
			line = s.title.Render(line)
		} else {
			line = s.body.Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func selectedOffset(rows []treeRow, selected model.ID, height int) int {
	index := 0
	for candidate, row := range rows {
		if row.id == selected {
			index = candidate
			break
		}
	}
	if index < height {
		return 0
	}
	return index - height + 1
}

func displayName(component model.Component) string {
	for _, key := range []string{"display_name", "name", "project", "unit", "label"} {
		if value := strings.TrimSpace(component.Metadata[key]); value != "" {
			return value
		}
	}
	value := string(component.ID)
	if index := strings.IndexRune(value, ':'); index >= 0 && index+1 < len(value) {
		value = value[index+1:]
	}
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ", "/", " ", ":", " ")
	words := strings.Fields(replacer.Replace(value))
	for index, word := range words {
		if word == strings.ToUpper(word) && len(word) > 1 {
			continue
		}
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}
	if len(words) == 0 {
		return string(component.ID)
	}
	return strings.Join(words, " ")
}

func normalizedStatus(status model.HealthStatus) model.HealthStatus {
	if status == "" {
		return model.HealthUnknown
	}
	return status
}

func healthGlyph(status model.HealthStatus) string {
	switch normalizedStatus(status) {
	case model.HealthHealthy:
		return "●"
	case model.HealthDegraded:
		return "◐"
	case model.HealthUnhealthy:
		return "×"
	default:
		return "?"
	}
}

func healthStyle(status model.HealthStatus, s styles) lipgloss.Style {
	switch normalizedStatus(status) {
	case model.HealthHealthy:
		return s.good
	case model.HealthDegraded:
		return s.warn
	case model.HealthUnhealthy:
		return s.bad
	default:
		return s.unknown
	}
}

func healthWords(health model.Health) string {
	result := healthGlyph(health.Status) + " " + strings.ToUpper(string(normalizedStatus(health.Status)))
	if health.Summary != "" {
		result += " — " + health.Summary
	}
	return result
}

func healthPath(steps []model.HealthStep, a app) string {
	if len(steps) == 0 {
		return "declared here"
	}
	parts := []string{a.nameFor(steps[0].From)}
	for _, step := range steps {
		parts = append(parts, "—"+relationWords(step.Relationship)+"→", a.nameFor(step.To))
	}
	return strings.Join(parts, " ")
}

func relationWords(relation model.EdgeType) string {
	return strings.ReplaceAll(string(relation), "_", " ")
}

func isDoctorAction(action model.Action) bool {
	kind := strings.ToLower(strings.TrimSpace(string(action.Kind)))
	return kind == "doctor.run" || kind == "doctor" || kind == "run_doctor" || kind == "run-doctor"
}

func actionLabel(action model.Action) string {
	label := firstNonEmpty(action.Label, string(action.ID), "Unnamed action")
	if action.Kind != "" {
		label += " · " + string(action.Kind)
	}
	if action.RequiresRoot {
		label += " · root"
	}
	if action.ID != "" && label != string(action.ID) {
		label += " · " + string(action.ID)
	}
	return label
}

type lineWriter struct {
	width int
	lines []string
	style styles
}

func newLineWriter(width int, s styles) *lineWriter {
	return &lineWriter{width: max(8, width), style: s}
}

func (w *lineWriter) blank() {
	if len(w.lines) == 0 || w.lines[len(w.lines)-1] == "" {
		return
	}
	w.lines = append(w.lines, "")
}

func (w *lineWriter) section(title string) {
	w.blank()
	w.lines = append(w.lines, w.style.section.Render(truncatePlain(strings.ToUpper(title), w.width)))
}

func (w *lineWriter) paragraph(title, text string) {
	w.section(title)
	w.text(text, w.style.body)
}

func (w *lineWriter) text(text string, style lipgloss.Style) {
	for _, line := range wrapText(text, w.width) {
		w.lines = append(w.lines, style.Render(line))
	}
}

func (w *lineWriter) bullet(text string) {
	w.bulletWithPrefix("•", text)
}

func (w *lineWriter) bulletWithPrefix(prefix, text string) {
	prefix += " "
	continuation := strings.Repeat(" ", runeWidth(prefix))
	wrapped := wrapTextWithPrefixes(text, w.width, prefix, continuation)
	for _, line := range wrapped {
		w.lines = append(w.lines, w.style.body.Render(line))
	}
}

func (w *lineWriter) keyValue(key, value string) {
	prefix := key + ": "
	continuation := strings.Repeat(" ", runeWidth(prefix))
	wrapped := wrapTextWithPrefixes(value, w.width, prefix, continuation)
	for _, line := range wrapped {
		w.lines = append(w.lines, w.style.body.Render(line))
	}
}

func (w *lineWriter) health(health model.Health) {
	style := healthStyle(health.Status, w.style)
	w.text(healthWords(health), style)
	if health.ObservedAt != nil {
		w.keyValue("Observed", health.ObservedAt.Format(time.RFC3339))
	}
}

func (w *lineWriter) healthObject(label string, health model.Health) {
	line := healthGlyph(health.Status) + " " + label + " · " + strings.ToUpper(string(normalizedStatus(health.Status)))
	w.text(line, healthStyle(health.Status, w.style))
	if health.Summary != "" {
		w.text("  "+health.Summary, w.style.muted)
	}
}

func (w *lineWriter) check(check model.Check) {
	glyph, word, style := "?", "UNKNOWN", w.style.unknown
	switch check.Status {
	case model.CheckPass:
		glyph, word, style = "●", "PASS", w.style.good
	case model.CheckWarn:
		glyph, word, style = "◐", "WARN", w.style.warn
	case model.CheckFail:
		glyph, word, style = "×", "FAIL", w.style.bad
	}
	line := glyph + " " + word + " · " + check.ID
	if check.Summary != "" {
		line += " — " + check.Summary
	}
	w.text(line, style)
	if len(check.Evidence) > 0 {
		keys := make([]string, 0, len(check.Evidence))
		for key := range check.Evidence {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.keyValue("  "+key, check.Evidence[key])
		}
	}
}

func wrapText(text string, width int) []string {
	return wrapTextWithPrefixes(text, width, "", "")
}

func wrapTextWithPrefixes(text string, width int, firstPrefix, continuationPrefix string) []string {
	width = max(1, width)
	paragraphs := strings.Split(strings.ReplaceAll(text, "\t", "  "), "\n")
	result := make([]string, 0)
	first := true
	for _, paragraph := range paragraphs {
		prefix := continuationPrefix
		if first {
			prefix = firstPrefix
		}
		if strings.TrimSpace(paragraph) == "" {
			result = append(result, truncatePlain(prefix, width))
			first = false
			continue
		}
		words := strings.Fields(paragraph)
		line := prefix
		for len(words) > 0 {
			word := words[0]
			separator := ""
			if strings.TrimSpace(line) != "" && line != prefix {
				separator = " "
			}
			if runeWidth(line)+runeWidth(separator)+runeWidth(word) <= width {
				line += separator + word
				words = words[1:]
				continue
			}
			if strings.TrimSpace(line) != "" && line != prefix {
				result = append(result, line)
				line = continuationPrefix
				first = false
				continue
			}
			available := max(1, width-runeWidth(line))
			chunk, rest := splitRunes(word, available)
			line += chunk
			result = append(result, line)
			line = continuationPrefix
			first = false
			if rest == "" {
				words = words[1:]
			} else {
				words[0] = rest
			}
		}
		if strings.TrimSpace(line) != "" || len(result) == 0 {
			result = append(result, line)
		}
		first = false
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func splitRunes(value string, count int) (string, string) {
	if count <= 0 {
		return "", value
	}
	runes := []rune(value)
	used := 0
	for index, current := range runes {
		currentWidth := lipgloss.Width(string(current))
		if used+currentWidth > count {
			if index == 0 {
				return string(current), string(runes[1:])
			}
			return string(runes[:index]), string(runes[index:])
		}
		used += currentWidth
	}
	return value, ""
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if runeWidth(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	chunk, _ := splitRunes(value, width-1)
	return chunk + "…"
}

func runeWidth(value string) int {
	return lipgloss.Width(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func clamp(value, low, high int) int {
	if high < low {
		high = low
	}
	return min(max(value, low), high)
}
