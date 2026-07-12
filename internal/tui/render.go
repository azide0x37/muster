package tui

import (
	"fmt"
	"image/color"
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
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(footer)
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
	screen := s.screen.
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
	if a.confirmAction != "" && width >= minimumWidth && height >= minimumHeight {
		dialog := a.renderConfirmDialog(width, s)
		x := max(0, (width-lipgloss.Width(dialog))/2)
		y := max(0, (height-lipgloss.Height(dialog))/2)
		screen = lipgloss.NewCompositor(
			lipgloss.NewLayer(screen),
			lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
		).Render()
	}
	return screen
}

// renderConfirmDialog is the modal that guards doctor execution. It floats
// over the graph so the question is unmistakable, while the ribbon repeats
// the same keys for anyone who tuned the modal out.
func (a app) renderConfirmDialog(width int, s styles) string {
	name := a.nameFor(a.confirmTarget)
	requiresRoot := false
	for _, action := range a.doctorActions(a.confirmTarget) {
		if action.ID == a.confirmAction {
			requiresRoot = action.RequiresRoot
		}
	}
	innerWidth := clamp(runeWidth(name)+6, 22, max(22, width-8))
	w := newLineWriter(innerWidth, s)
	w.lines = append(w.lines, s.dialogTitle.Render(truncatePlain("CONFIRM DOCTOR", innerWidth)))
	w.blank()
	w.text(name, s.title)
	w.text("Run doctor for this object now?", s.body)
	if requiresRoot {
		w.text("Requires root · rerun Muster with sudo.", s.warn)
	}
	w.blank()
	w.lines = append(w.lines,
		s.ribbonKey.Render("y/enter")+" "+s.ribbonDesc.Render("run")+
			s.ribbonSep.Render(" · ")+
			s.ribbonKey.Render("n/esc")+" "+s.ribbonDesc.Render("cancel"))
	return s.dialog.Render(strings.Join(w.lines, "\n"))
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
	strap := s.gradient("MUSTER", true) + s.faint.Render(" · server inspector")
	title := s.title.Render(truncatePlain("Muster implementations on "+a.hostname, width))
	counts := a.renderImplementationCounts(width, s)
	return strings.Join([]string{strap, title, counts}, "\n")
}

// countPart pairs one clause of the implementation summary with the health
// style it deserves; the words alone still tell the whole story.
type countPart struct {
	text  string
	style lipgloss.Style
}

func (a app) implementationCountParts(s styles) []countPart {
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
	parts := []countPart{{
		text:  fmt.Sprintf("%d %s", count, plural(count, "implementation", "implementations")),
		style: s.subtitle,
	}}
	for _, status := range []model.HealthStatus{
		model.HealthHealthy, model.HealthDegraded, model.HealthUnhealthy, model.HealthUnknown,
	} {
		if counts[status] > 0 {
			parts = append(parts, countPart{
				text:  fmt.Sprintf("%d %s", counts[status], status),
				style: healthStyle(status, s),
			})
		}
	}
	if count == 0 {
		parts = append(parts, countPart{text: "nothing registered yet", style: s.faint})
	}
	return parts
}

func (a app) implementationCounts() string {
	parts := a.implementationCountParts(newStyles(a.dark, true))
	texts := make([]string, len(parts))
	for index, part := range parts {
		texts[index] = part.text
	}
	return strings.Join(texts, " · ")
}

func (a app) renderImplementationCounts(width int, s styles) string {
	parts := a.implementationCountParts(s)
	if runeWidth(a.implementationCounts()) > width {
		return s.subtitle.Render(truncatePlain(a.implementationCounts(), width))
	}
	rendered := make([]string, len(parts))
	for index, part := range parts {
		rendered[index] = part.style.Render(part.text)
	}
	return strings.Join(rendered, s.faint.Render(" · "))
}

// renderFooter is the bottom chrome: a segmented status bar in the language
// of the Muster website's fixed frame, and a contextual key ribbon beneath it.
func (a app) renderFooter(width int, s styles) string {
	return a.renderStatusBar(width, s) + "\n" + a.renderRibbon(width, s)
}

// sectionLabel names where the operator is, the way the site's bottom bar
// tracks the section in view.
func (a app) sectionLabel() string {
	switch {
	case a.confirmAction != "":
		return "CONFIRM"
	case a.help:
		return "HELP"
	case a.inspect != "":
		return "INSPECT"
	default:
		return "BROWSE"
	}
}

// fleetHealth is the single LED verdict for everything registered here.
func (a app) fleetHealth() model.HealthStatus {
	if len(a.graph.Implementations) == 0 {
		return model.HealthUnknown
	}
	worst := model.HealthHealthy
	for _, implementation := range a.graph.Implementations {
		status := model.HealthUnknown
		if component, ok := a.graph.Lookup(implementation.ID); ok {
			status = normalizedStatus(component.Health.Status)
		}
		switch status {
		case model.HealthUnhealthy:
			return model.HealthUnhealthy
		case model.HealthDegraded:
			worst = model.HealthDegraded
		case model.HealthUnknown:
			if worst == model.HealthHealthy {
				worst = model.HealthUnknown
			}
		}
	}
	return worst
}

func healthColor(status model.HealthStatus, c palette) color.Color {
	switch normalizedStatus(status) {
	case model.HealthHealthy:
		return c.good
	case model.HealthDegraded:
		return c.warn
	case model.HealthUnhealthy:
		return c.bad
	default:
		return c.muted
	}
}

// renderStatusBar lays out the bar's units: LED + host, a flexible message
// cell, the section cell, and object counters. Narrow terminals shed the
// outer units first, exactly like the site's bar hides its small cells.
func (a app) renderStatusBar(width int, s styles) string {
	host := a.renderHostCell(s)
	section := a.renderSectionCell(s)
	counts := a.renderCountsCell(s)

	used := lipgloss.Width(host) + lipgloss.Width(section) + lipgloss.Width(counts)
	if width < wideBreakpoint || width-used < 14 {
		counts = ""
		used = lipgloss.Width(host) + lipgloss.Width(section)
	}
	if width < 56 || width-used < 14 {
		host = ""
		used = lipgloss.Width(section)
	}
	message := a.renderMessageCell(max(1, width-used), s)
	bar := host + message + section + counts
	if lipgloss.Width(bar) > width {
		bar = host + message + section
	}
	return bar
}

func (a app) renderHostCell(s styles) string {
	c := s.colors
	led := lipgloss.NewStyle().Background(c.panelHigh)
	if s.noColor {
		return "● " + a.hostname + " "
	}
	ledColor := healthColor(a.fleetHealth(), c)
	if a.ledDim {
		ledColor = c.faint
	}
	return led.Foreground(ledColor).Render(" ●") + s.barHost.Render(" "+a.hostname+" ")
}

func (a app) renderSectionCell(s styles) string {
	label := " § " + a.sectionLabel() + " "
	if a.confirmAction != "" {
		return s.barAlert.Render(label)
	}
	return s.barSection.Render(label)
}

func (a app) renderCountsCell(s styles) string {
	c := s.colors
	counts := map[model.HealthStatus]int{}
	total := 0
	for _, row := range a.allTreeRows() {
		if component, ok := a.graph.Lookup(row.id); ok {
			counts[normalizedStatus(component.Health.Status)]++
			total++
		}
	}
	cell := s.barCounts.Render(fmt.Sprintf(" %d obj ", total))
	for _, status := range []model.HealthStatus{
		model.HealthHealthy, model.HealthDegraded, model.HealthUnhealthy, model.HealthUnknown,
	} {
		if counts[status] == 0 {
			continue
		}
		glyph := healthGlyph(status) + fmt.Sprintf("%d ", counts[status])
		if s.noColor {
			cell += glyph
			continue
		}
		cell += lipgloss.NewStyle().
			Foreground(healthColor(status, c)).
			Background(c.panelHigh).
			Render(glyph)
	}
	return cell
}

func (a app) renderMessageCell(width int, s styles) string {
	message := "Read-only"
	if !a.refreshedAt.IsZero() {
		message += " · state as of " + a.refreshedAt.Format("15:04:05")
	}
	if a.status != "" {
		message = a.status
	}
	cell := ""
	reserved := 1
	if a.busy {
		cell += s.barMsg.Render(" ") + a.spin.View()
		reserved += 2
		if a.status == "" && a.operation != "" {
			message = a.operation
		}
	}
	cell += s.barMsg.Render(" " + truncatePlain(message, max(1, width-reserved-1)))
	if padding := width - lipgloss.Width(cell); padding > 0 {
		cell += s.barMsg.Render(strings.Repeat(" ", padding))
	}
	return cell
}

// renderRibbon lists the keys that matter right now, dropping trailing hints
// when the terminal cannot hold them all.
func (a app) renderRibbon(width int, s styles) string {
	bindings := a.ribbonBindings()
	fits := len(bindings)
	for ; fits > 1; fits-- {
		plain := 0
		for index := 0; index < fits; index++ {
			help := bindings[index].Help()
			plain += runeWidth(help.Key) + 1 + runeWidth(help.Desc)
		}
		plain += (fits-1)*3 + 1
		if plain <= width {
			break
		}
	}
	parts := make([]string, 0, fits)
	for index := 0; index < fits; index++ {
		help := bindings[index].Help()
		parts = append(parts, s.ribbonKey.Render(help.Key)+" "+s.ribbonDesc.Render(help.Desc))
	}
	return " " + strings.Join(parts, s.ribbonSep.Render(" · "))
}

func (a app) renderTooSmall(width, height int, s styles) string {
	lines := []string{
		s.body.Render(fmt.Sprintf("Current terminal: %d×%d", width, a.height)),
		s.muted.Render(fmt.Sprintf("Muster needs at least %d×%d.", minimumWidth, minimumHeight)),
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
	w.keyValue("enter", "open the selected inspectable object")
	w.keyValue("→ / l", "unfold the selected subtree, or open the object")
	w.keyValue("← / h", "fold the selected subtree, or jump to its parent")
	w.keyValue("space", "fold or unfold the selected subtree")
	w.keyValue("esc / backspace", "return to the implementation tree")
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
	w.paragraph("Reading the tree", "Healthy rows show only the glyph; a spelled-out status word marks something needing attention. Fully healthy subtrees start folded — ▸ counts the objects beneath a folded row.")
	w.paragraph("Motion", "Scrolling glides and the status LED breathes. Set MUSTER_REDUCE_MOTION=1 to keep the console perfectly still.")
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
	viewportHeight := max(1, height-2)
	offset := selectedOffset(rows, a.selected, viewportHeight)
	title := fmt.Sprintf("Implementations · %d objects", len(a.allTreeRows()))
	return a.panel(width, height, focused, title, lines, offset, s)
}

func (a app) renderSummaryPanel(width, height int, focused bool, s styles) string {
	innerWidth := max(8, width-4)
	lines := a.overviewLines(a.selected, innerWidth, s)
	return a.panel(width, height, focused, "Selected object", lines, a.displayScroll(), s)
}

func (a app) renderInspector(width, height int, s styles) string {
	innerWidth := max(8, width-4)
	lines := a.inspectLines(a.inspect, innerWidth, s)
	return a.panel(width, height, true, "Inspect", lines, a.displayScroll(), s)
}

// panel draws a framed pane whose title lives inside the top border, the way
// charm-family tools label their windows. Focus is carried by border weight
// or color and, in color terminals, by an accent→ember rule.
func (a app) panel(width, height int, focused bool, title string, lines []string, offset int, s styles) string {
	width = max(4, width)
	height = max(3, height)
	viewportHeight := max(1, height-2)
	visible := viewport(lines, offset, viewportHeight, s)
	top := panelTopLine(width, focused, truncatePlain(title, max(0, width-8)), s)
	style := s.panelBody
	if focused {
		style = s.focusedPanelBody
	}
	body := style.
		Width(width).
		Height(height - 1).
		MaxWidth(width).
		MaxHeight(height - 1).
		Render(strings.Join(visible, "\n"))
	return top + "\n" + body
}

// panelTopLine draws the top border with the pane's title set into it:
// ╭─ Title ────────╮. In no-color terminals a focused pane keeps the double
// border so focus never depends on color.
func panelTopLine(width int, focused bool, title string, s styles) string {
	if width < 2 {
		return ""
	}
	cornerL, horizontal, cornerR := "╭", "─", "╮"
	if s.noColor && focused {
		cornerL, horizontal, cornerR = "╔", "═", "╗"
	}
	lineStyle, titleStyle := s.panelLine, s.panelTitle
	if focused {
		lineStyle, titleStyle = s.focusedPanelLine, s.focusedPanelTitle
	}
	inner := width - 2
	if title == "" || runeWidth(title)+4 > inner {
		return lineStyle.Render(cornerL + strings.Repeat(horizontal, max(0, inner)) + cornerR)
	}
	trail := inner - runeWidth(title) - 3
	head := lineStyle.Render(cornerL+horizontal+" ") + titleStyle.Render(title) + lineStyle.Render(" ")
	if focused && !s.noColor {
		ember := lipgloss.NewStyle().Foreground(s.colors.ember)
		return head + s.gradientRule(trail, horizontal) + ember.Render(cornerR)
	}
	return head + lineStyle.Render(strings.Repeat(horizontal, trail)+cornerR)
}

// viewport windows lines at offset. A negative offset pads the top with
// blank rows — the inspector uses this to glide new content into place.
func viewport(lines []string, offset, height int, s styles) []string {
	if height <= 0 {
		return nil
	}
	pad := 0
	if offset < 0 {
		pad = min(-offset, height)
		offset = 0
	}
	bodyHeight := height - pad
	maximum := max(0, len(lines)-bodyHeight)
	offset = clamp(offset, 0, maximum)
	end := min(len(lines), offset+bodyHeight)
	visible := make([]string, 0, height)
	for index := 0; index < pad; index++ {
		visible = append(visible, "")
	}
	visible = append(visible, lines[offset:end]...)
	for len(visible) < height {
		visible = append(visible, "")
	}
	if offset > 0 && bodyHeight > 0 {
		visible[pad] = s.faint.Render(fmt.Sprintf("↑ %d more line%s", offset, pluralSuffix(offset)))
	}
	if remaining := len(lines) - end; remaining > 0 && bodyHeight > 0 {
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

	if explanation, err := a.graph.Explain(component.ID); err == nil &&
		normalizedStatus(explanation.Health.Effective.Status) != model.HealthHealthy {
		// The object restating itself as its own cause adds nothing over the
		// verdict line above; only causes elsewhere in the graph earn a section.
		causes := make([]string, 0, len(explanation.Health.Causes))
		for _, cause := range explanation.Health.Causes {
			if cause.ComponentID == component.ID && len(cause.Path) == 0 {
				continue
			}
			line := a.nameFor(cause.ComponentID)
			if len(cause.Path) > 0 {
				line += " via " + healthPath(cause.Path, a)
			}
			causes = append(causes, line)
		}
		if len(causes) > 0 {
			w.section("Health causes")
			for _, cause := range causes {
				w.bullet(cause)
			}
		}
	}

	w.section("Latest evidence")
	if observation, found := a.graph.LatestObservation(component.ID, ""); found {
		w.healthObject(strings.ToUpper(string(observation.Kind))+" · "+observation.ObservedAt.Format(time.RFC3339), observation.DerivedHealth())
		for _, check := range observation.Checks {
			w.check(check)
		}
	} else {
		w.text("No observations recorded for this object.", s.faint)
	}

	if len(component.Children) > 0 {
		w.section(fmt.Sprintf("Children · %d", len(component.Children)))
		parentName := displayName(*component)
		for _, childID := range component.Children {
			if child, found := a.graph.Lookup(childID); found {
				w.healthObject("• "+contextualName(displayName(*child), []string{parentName}), child.Health)
			}
		}
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

	if what := firstNonEmpty(component.What, component.Summary); what != "" {
		w.paragraph("What", what)
	}
	if why := strings.TrimSpace(component.Why); why != "" {
		w.paragraph("Why", why)
	}
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
			w.healthObject(row.prefix+row.label, child.Health)
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
	bodyHeight := max(3, height-headerHeight-footerHeight)
	viewportHeight := max(1, bodyHeight-2)
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
	label  string
	parent model.ID
	hidden int // objects folded away beneath this row
}

// treeRows is the visible tree: fully healthy subtrees stay folded unless the
// operator opened them, and every path to a non-healthy object is open.
func (a app) treeRows() []treeRow {
	return a.buildTreeRows(true)
}

// allTreeRows ignores folding; counters and ancestry lookups need every object.
func (a app) allTreeRows() []treeRow {
	return a.buildTreeRows(false)
}

func (a app) buildTreeRows(fold bool) []treeRow {
	if a.graph == nil {
		return nil
	}
	rows := make([]treeRow, 0, len(a.graph.Components))
	for _, implementation := range a.graph.Implementations {
		root, ok := a.graph.Lookup(implementation.ID)
		if !ok {
			continue
		}
		children := a.implementationChildren(implementation, root)
		rootName := displayName(*root)
		row := treeRow{id: root.ID, root: true, label: rootName}
		if fold && !a.expandedState(root.ID, children) {
			row.hidden = countDescendants(a.graph, children)
			rows = append(rows, row)
			continue
		}
		rows = append(rows, row)
		path := map[model.ID]bool{root.ID: true}
		rendered := map[model.ID]bool{root.ID: true}
		ancestors := []string{rootName}
		for index, child := range children {
			a.appendTree(&rows, child, root.ID, "", index == len(children)-1, path, rendered, ancestors, fold)
		}
	}
	return rows
}

// implementationChildren is the root's effective child list: declared children
// plus any registered components not reachable through them.
func (a app) implementationChildren(implementation model.Implementation, root *model.Component) []model.ID {
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
	return children
}

// nodeChildren returns the child list folding operates on, which for an
// implementation root includes components outside its declared children.
func (a app) nodeChildren(id model.ID) []model.ID {
	component, ok := a.graph.Lookup(id)
	if !ok {
		return nil
	}
	if implementation, isImplementation := a.graph.LookupImplementation(id); isImplementation {
		return a.implementationChildren(*implementation, component)
	}
	return append([]model.ID(nil), component.Children...)
}

// expandedState resolves one node: an explicit operator toggle wins, otherwise
// a subtree is open exactly when something inside it is not healthy.
func (a app) expandedState(id model.ID, children []model.ID) bool {
	if state, ok := a.expanded[id]; ok {
		return state
	}
	seen := map[model.ID]bool{}
	for _, child := range children {
		if a.needsAttention(child, seen) {
			return true
		}
	}
	return false
}

func (a app) needsAttention(id model.ID, seen map[model.ID]bool) bool {
	if seen[id] {
		return false
	}
	seen[id] = true
	component, ok := a.graph.Lookup(id)
	if !ok {
		return false
	}
	if normalizedStatus(component.Health.Status) != model.HealthHealthy {
		return true
	}
	for _, child := range component.Children {
		if a.needsAttention(child, seen) {
			return true
		}
	}
	return false
}

func countDescendants(graph *model.Graph, children []model.ID) int {
	seen := map[model.ID]bool{}
	for _, child := range children {
		markDescendants(graph, child, seen)
	}
	return len(seen)
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

func (a app) appendTree(rows *[]treeRow, id, parent model.ID, parentPrefix string, last bool, path, rendered map[model.ID]bool, ancestors []string, fold bool) {
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
	name := displayName(*component)
	row := treeRow{id: id, prefix: parentPrefix + connector, label: contextualName(name, ancestors), parent: parent}
	if fold && len(component.Children) > 0 && !a.expandedState(id, component.Children) {
		row.hidden = countDescendants(a.graph, component.Children)
		*rows = append(*rows, row)
		rendered[id] = true
		return
	}
	*rows = append(*rows, row)
	rendered[id] = true
	nextPath := clonePath(path)
	nextPath[id] = true
	nextAncestors := append(append([]string(nil), ancestors...), name)
	for index, child := range component.Children {
		a.appendTree(rows, child, id, nextPrefix, index == len(component.Children)-1, nextPath, rendered, nextAncestors, fold)
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
	ancestors := []string{displayName(*component)}
	for index, child := range component.Children {
		a.appendTree(&rows, child, id, "", index == len(component.Children)-1, path, rendered, ancestors, false)
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
		}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		component, ok := a.graph.Lookup(row.id)
		if !ok {
			continue
		}
		lines = append(lines, a.renderTreeRow(row, *component, width, s))
	}
	return lines
}

// renderTreeRow lays out one object: selection bar, tree lineage, health
// glyph, contextual name, fold count, and a right-aligned status word. The
// selected row reads as a charm-style list item — accent bar on a raised
// background. Healthy rows keep only the glyph so a spelled-out status word
// always marks something needing attention.
func (a app) renderTreeRow(row treeRow, component model.Component, width int, s styles) string {
	selected := row.id == a.selected
	indicator := "  "
	if selected {
		indicator = "▌ "
	}
	glyph := healthGlyph(component.Health.Status)
	statusWord := ""
	if status := normalizedStatus(component.Health.Status); status != model.HealthHealthy {
		statusWord = strings.ToUpper(string(status))
	}
	foldMark := ""
	if row.hidden > 0 {
		foldMark = fmt.Sprintf(" ▸ %d", row.hidden)
	}
	nameWidth := max(1, width-runeWidth(indicator+row.prefix)-runeWidth(glyph)-runeWidth(foldMark)-runeWidth(statusWord)-3)
	name := truncatePlain(row.label, nameWidth)
	padding := max(1, width-runeWidth(indicator+row.prefix)-runeWidth(glyph)-1-runeWidth(name)-runeWidth(foldMark)-runeWidth(statusWord))

	if s.noColor {
		return indicator + row.prefix + glyph + " " + name + foldMark + strings.Repeat(" ", padding) + statusWord
	}
	if selected {
		raised := lipgloss.NewStyle().Background(s.colors.panelHigh)
		return s.selectedBar.Render("▌ ") +
			raised.Foreground(s.colors.faint).Render(row.prefix) +
			raised.Foreground(healthColor(component.Health.Status, s.colors)).Render(glyph) +
			s.selected.Render(" "+name) +
			raised.Foreground(s.colors.faint).Render(foldMark) +
			raised.Render(strings.Repeat(" ", padding)) +
			raised.Foreground(healthColor(component.Health.Status, s.colors)).Render(statusWord)
	}
	nameStyle := s.body
	if row.root {
		nameStyle = s.title
	}
	return indicator + s.faint.Render(row.prefix) +
		healthStyle(component.Health.Status, s).Render(glyph) +
		nameStyle.Render(" "+name) +
		s.faint.Render(foldMark) +
		strings.Repeat(" ", padding) +
		healthStyle(component.Health.Status, s).Render(statusWord)
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

// systemdUnitSuffixes mark ID tails that are operational identifiers — the
// strings an operator pastes into systemctl or journalctl. They stay verbatim;
// prettifying them destroys case, templates (@), and the unit type.
var systemdUnitSuffixes = []string{
	".service", ".socket", ".device", ".mount", ".automount",
	".swap", ".target", ".path", ".timer", ".slice", ".scope",
}

func machineIdentifier(value string) bool {
	for _, suffix := range systemdUnitSuffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

// contextualName removes ancestor names repeated at the front of a child's
// name, so each tree row leads with what distinguishes it instead of
// re-stating the lineage the tree already draws.
func contextualName(name string, ancestors []string) string {
	words := strings.Fields(name)
	for stripped := true; stripped; {
		stripped = false
		for _, ancestor := range ancestors {
			prefix := strings.Fields(ancestor)
			if len(prefix) == 0 || len(words) <= len(prefix) || !equalWordsFold(words[:len(prefix)], prefix) {
				continue
			}
			words = words[len(prefix):]
			stripped = true
		}
	}
	if len(words) == 0 {
		return name
	}
	return strings.Join(words, " ")
}

func equalWordsFold(a, b []string) bool {
	for index := range a {
		if !strings.EqualFold(a[index], b[index]) {
			return false
		}
	}
	return true
}

func displayName(component model.Component) string {
	for _, key := range []string{"display_name", "name", "project", "unit", "label"} {
		if value := strings.TrimSpace(component.Metadata[key]); value != "" {
			return value
		}
	}
	value := string(component.ID)
	if index := strings.LastIndexByte(value, ':'); index >= 0 && index+1 < len(value) {
		if tail := value[index+1:]; machineIdentifier(tail) {
			return tail
		}
	}
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
	line := healthGlyph(health.Status) + " " + label
	if status := normalizedStatus(health.Status); status != model.HealthHealthy {
		line += " · " + strings.ToUpper(string(status))
	}
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
