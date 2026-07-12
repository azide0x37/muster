// Package tui presents Muster's normalized runtime graph as a full-screen,
// keyboard-driven operator console.
package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/harmonica"

	"github.com/azide0x37/muster/internal/model"
)

// Motion tuning. One spring serves every scroll glide; a critically damped
// ratio means content settles without overshooting past a line of text.
const (
	animationFPS     = 30
	entranceOffset   = 3.0
	ledPulseInterval = 900 * time.Millisecond
	reduceMotionEnv  = "MUSTER_REDUCE_MOTION"
)

var scrollSpring = harmonica.NewSpring(harmonica.FPS(animationFPS), 8.0, 1.0)

// Options supplies host-specific behavior without coupling the TUI to an
// inspector, registry, privilege boundary, or process runner.
type Options struct {
	Hostname  string
	Refresh   func(context.Context) (*model.Graph, error)
	RunDoctor func(context.Context, model.ID) (*model.Graph, error)
	NoColor   bool
}

// RenderOptions configures a deterministic, non-interactive rendering. It is
// useful for snapshots, documentation, and tests. NoColor keeps every word,
// glyph, border, and layout decision while omitting ANSI color sequences.
type RenderOptions struct {
	Hostname       string
	Width          int
	Height         int
	DarkBackground bool
	NoColor        bool
	Selected       model.ID
	Inspect        model.ID
}

// Run opens the Muster operator console in the terminal's alternate screen.
func Run(graph *model.Graph, opts Options) error {
	if graph == nil {
		return errors.New("tui: graph is required")
	}
	if err := graph.Validate(); err != nil {
		return fmt.Errorf("tui: invalid graph: %w", err)
	}
	app := newApp(graph, opts)
	app.refreshedAt = time.Now()
	_, err := tea.NewProgram(app).Run()
	return err
}

// Render returns a stable screen without starting a terminal program.
func Render(graph *model.Graph, opts RenderOptions) string {
	if graph == nil {
		graph = &model.Graph{SchemaVersion: model.CurrentSchemaVersion}
	}
	app := newApp(graph, Options{Hostname: opts.Hostname, NoColor: opts.NoColor})
	app.width = opts.Width
	app.height = opts.Height
	app.dark = opts.DarkBackground
	app.noColor = opts.NoColor
	app.selected = opts.Selected
	app.inspect = opts.Inspect
	app.ensureSelection()
	return app.render()
}

type focusPane uint8

const (
	focusTree focusPane = iota
	focusDetail
)

type app struct {
	graph         *model.Graph
	opts          Options
	hostname      string
	width         int
	height        int
	dark          bool
	noColor       bool
	forceNoColor  bool
	selected      model.ID
	inspect       model.ID
	focus         focusPane
	scroll        int
	help          bool
	busy          bool
	operation     string
	status        string
	confirmAction model.ID
	confirmTarget model.ID

	// expanded holds explicit fold toggles by object ID. Nodes without an
	// entry fold by default when their whole subtree is healthy.
	expanded map[model.ID]bool
	// filter narrows the sidebar to matching objects and their lineage.
	// filterEditing routes keys into the input instead of navigation.
	filter        textinput.Model
	filterEditing bool
	// refreshedAt is when the graph now on screen was produced. It stays zero
	// in deterministic renders so snapshots never embed a wall clock.
	refreshedAt time.Time

	// Motion state. scroll is always the destination; scrollPos is where the
	// eye currently is. The deterministic Render path never animates, so the
	// two only diverge inside a live program.
	keys         keymap
	spin         spinner.Model
	ledDim       bool
	scrollPos    float64
	scrollVel    float64
	animating    bool
	reduceMotion bool
}

func newApp(graph *model.Graph, opts Options) app {
	hostname := strings.TrimSpace(opts.Hostname)
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	if hostname == "" {
		hostname = "this server"
	}
	result := app{
		graph:        graph,
		opts:         opts,
		hostname:     hostname,
		width:        100,
		height:       30,
		dark:         true,
		noColor:      opts.NoColor,
		forceNoColor: opts.NoColor,
		expanded:     map[model.ID]bool{},
		keys:         newKeymap(),
		spin:         spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(spinnerStyle(true, opts.NoColor))),
		reduceMotion: os.Getenv(reduceMotionEnv) != "",
	}
	result.filter = textinput.New()
	result.filter.Placeholder = "filter"
	result.filter.Prompt = " / "
	result.syncFilterStyles()
	result.ensureSelection()
	return result
}

// filterQuery is the live filter text; blank means no filtering.
func (a app) filterQuery() string {
	return strings.TrimSpace(a.filter.Value())
}

func (a *app) syncFilterStyles() {
	styles := textinput.Styles{}
	if !a.noColor {
		colors := newPalette(a.dark, false)
		state := textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(colors.fg),
			Placeholder: lipgloss.NewStyle().Foreground(colors.faint),
			Prompt:      lipgloss.NewStyle().Bold(true).Foreground(colors.accent),
		}
		styles.Focused = state
		styles.Blurred = state
		styles.Cursor.Color = colors.accent
	}
	styles.Cursor.Blink = !a.reduceMotion && !a.noColor
	a.filter.SetStyles(styles)
}

// keymap names every key the console understands; the ribbon renders the
// subset that applies to the moment.
type keymap struct {
	move        key.Binding
	scroll      key.Binding
	inspect     key.Binding
	fold        key.Binding
	unfold      key.Binding
	back        key.Binding
	pane        key.Binding
	doctor      key.Binding
	refresh     key.Binding
	help        key.Binding
	closeHelp   key.Binding
	quit        key.Binding
	yes         key.Binding
	no          key.Binding
	filter      key.Binding
	applyFilter key.Binding
	clearFilter key.Binding
	matchMove   key.Binding
}

func newKeymap() keymap {
	return keymap{
		move:      key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑↓/jk", "move")),
		scroll:    key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑↓/jk", "scroll")),
		inspect:   key.NewBinding(key.WithKeys("enter", "right", "l"), key.WithHelp("enter", "inspect")),
		fold:      key.NewBinding(key.WithKeys("space", "h", "left"), key.WithHelp("space", "fold")),
		unfold:    key.NewBinding(key.WithKeys("space", "l", "right"), key.WithHelp("space", "unfold")),
		back:      key.NewBinding(key.WithKeys("esc", "backspace", "left", "h"), key.WithHelp("esc", "back")),
		pane:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "pane")),
		doctor:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "doctor")),
		refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		closeHelp: key.NewBinding(key.WithKeys("?", "esc"), key.WithHelp("?/esc", "close help")),
		quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		yes:         key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y/enter", "confirm doctor")),
		no:          key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "cancel")),
		filter:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		applyFilter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
		clearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		matchMove:   key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "move")),
	}
}

func (a app) ribbonBindings() []key.Binding {
	k := a.keys
	if a.confirmAction != "" {
		return []key.Binding{k.yes, k.no, k.quit}
	}
	if a.help {
		return []key.Binding{k.closeHelp, k.quit}
	}
	if a.filterEditing {
		return []key.Binding{k.matchMove, k.applyFilter, k.clearFilter}
	}
	var bindings []key.Binding
	if a.inspect != "" {
		bindings = []key.Binding{k.scroll, k.back}
	} else {
		bindings = []key.Binding{k.move, k.inspect}
		if a.filterQuery() != "" {
			bindings = append(bindings, k.clearFilter)
		} else {
			bindings = append(bindings, k.filter)
		}
		if a.focus == focusTree && a.selected != "" && a.filterQuery() == "" {
			if children := a.nodeChildren(a.selected); len(children) > 0 {
				if a.expandedState(a.selected, children) {
					bindings = append(bindings, k.fold)
				} else {
					bindings = append(bindings, k.unfold)
				}
			}
		}
		bindings = append(bindings, k.pane)
	}
	if _, ok := a.doctorAction(a.activeID()); ok && a.opts.RunDoctor != nil {
		bindings = append(bindings, k.doctor)
	}
	return append(bindings, k.refresh, k.help, k.quit)
}

func (a app) Init() tea.Cmd {
	if a.reduceMotion {
		return tea.RequestBackgroundColor
	}
	return tea.Batch(tea.RequestBackgroundColor, ledTick())
}

type ledTickMsg time.Time

func ledTick() tea.Cmd {
	return tea.Tick(ledPulseInterval, func(t time.Time) tea.Msg { return ledTickMsg(t) })
}

type frameMsg time.Time

func frameTick() tea.Cmd {
	return tea.Tick(time.Second/animationFPS, func(t time.Time) tea.Msg { return frameMsg(t) })
}

// displayScroll is the offset the viewer actually sees: the spring position
// mid-glide, the true offset at rest.
func (a app) displayScroll() int {
	if a.animating {
		return int(math.Round(a.scrollPos))
	}
	return a.scroll
}

func (a *app) snapScroll() {
	a.scrollPos = float64(a.scroll)
	a.scrollVel = 0
	a.animating = false
}

// animateScrollTo retargets the scroll spring and starts the frame loop if
// it is not already running.
func (a *app) animateScrollTo(target int) tea.Cmd {
	target = a.clampScroll(target)
	if target == a.scroll && !a.animating {
		return nil
	}
	a.scroll = target
	if a.reduceMotion {
		a.snapScroll()
		return nil
	}
	if a.animating {
		return nil
	}
	a.animating = true
	return frameTick()
}

// beginEntrance glides freshly opened content up into place.
func (a *app) beginEntrance() tea.Cmd {
	a.scroll = 0
	if a.reduceMotion {
		a.snapScroll()
		return nil
	}
	a.scrollPos = -entranceOffset
	a.scrollVel = 0
	if a.animating {
		return nil
	}
	a.animating = true
	return frameTick()
}

type refreshDoneMsg struct {
	graph *model.Graph
	err   error
}

type doctorDoneMsg struct {
	id    model.ID
	graph *model.Graph
	err   error
}

func (a app) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		a.width = message.Width
		a.height = message.Height
		a.scroll = a.clampScroll(a.scroll)
		a.snapScroll()
		return a, nil
	case tea.BackgroundColorMsg:
		a.dark = message.IsDark()
		a.spin.Style = spinnerStyle(a.dark, a.noColor)
		a.syncFilterStyles()
		return a, nil
	case tea.ColorProfileMsg:
		a.noColor = a.forceNoColor || message.Profile <= colorprofile.ASCII
		a.spin.Style = spinnerStyle(a.dark, a.noColor)
		a.syncFilterStyles()
		return a, nil
	case ledTickMsg:
		a.ledDim = !a.ledDim
		return a, ledTick()
	case frameMsg:
		if !a.animating {
			return a, nil
		}
		a.scrollPos, a.scrollVel = scrollSpring.Update(a.scrollPos, a.scrollVel, float64(a.scroll))
		if math.Abs(a.scrollPos-float64(a.scroll)) < 0.05 && math.Abs(a.scrollVel) < 0.05 {
			a.snapScroll()
			return a, nil
		}
		return a, frameTick()
	case spinner.TickMsg:
		if !a.busy {
			return a, nil
		}
		var command tea.Cmd
		a.spin, command = a.spin.Update(message)
		return a, command
	case refreshDoneMsg:
		afterDoctor := a.operation == "refreshing after doctor"
		previousStatus := a.status
		a.busy = false
		a.operation = ""
		if message.err != nil {
			if afterDoctor {
				a.status = previousStatus + " · runtime graph refresh failed: " + message.err.Error()
			} else {
				a.status = "Refresh failed: " + message.err.Error()
			}
			return a, nil
		}
		if message.graph == nil {
			a.status = "Refresh failed: inspector returned no graph"
			return a, nil
		}
		if err := message.graph.Validate(); err != nil {
			a.status = "Refresh failed: " + err.Error()
			return a, nil
		}
		a.graph = message.graph
		a.refreshedAt = time.Now()
		a.ensureSelection()
		a.scroll = a.clampScroll(a.scroll)
		if afterDoctor {
			a.status = previousStatus + " · runtime graph refreshed"
		} else {
			// The status bar's default line shows the new state-as-of time,
			// which is the whole story of a successful refresh.
			a.status = ""
		}
		return a, nil
	case doctorDoneMsg:
		a.busy = false
		a.operation = ""
		if message.err != nil {
			a.status = "Doctor for " + a.nameFor(message.id) + ": " + message.err.Error()
		} else {
			a.status = "Doctor completed for " + a.nameFor(message.id)
		}
		if message.graph != nil {
			if graphErr := message.graph.Validate(); graphErr != nil {
				a.status += " · returned graph invalid: " + graphErr.Error()
				return a, nil
			}
			a.graph = message.graph
			a.refreshedAt = time.Now()
			a.ensureSelection()
			a.scroll = a.clampScroll(a.scroll)
			a.status += " · runtime graph refreshed"
			return a, nil
		}
		if a.opts.Refresh != nil {
			a.busy = true
			a.operation = "refreshing after doctor"
			return a, refreshCommand(a.opts.Refresh)
		}
		return a, nil
	case tea.KeyPressMsg:
		if a.filterEditing {
			return a.handleFilterKey(message)
		}
		return a.handleKey(message.String())
	}
	return a, nil
}

// handleFilterKey routes keys while the filter input is focused: navigation
// and mode keys are handled here, everything else edits the query.
func (a app) handleFilterKey(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "esc", "escape":
		a.filter.SetValue("")
		a.filter.Blur()
		a.filterEditing = false
		a.ensureSelection()
		return a, nil
	case "enter", "tab":
		a.filter.Blur()
		a.filterEditing = false
		return a, nil
	case "up":
		a.moveSelection(-1)
		return a, nil
	case "down":
		a.moveSelection(1)
		return a, nil
	default:
		var command tea.Cmd
		a.filter, command = a.filter.Update(message)
		a.selectFirstMatch()
		return a, command
	}
}

// selectFirstMatch lands the selection on the first object that itself
// matches the query, not merely a lineage row kept for context.
func (a *app) selectFirstMatch() {
	query := a.filterQuery()
	if query == "" {
		a.ensureSelection()
		return
	}
	for _, row := range a.treeRows() {
		if component, ok := a.graph.Lookup(row.id); ok && matchesQuery(*component, row.label, query) {
			a.selected = row.id
			return
		}
	}
	a.ensureSelection()
}

func (a app) handleKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		return a, tea.Quit
	}

	if a.confirmAction != "" {
		switch key {
		case "y", "Y", "enter":
			actionID, targetID := a.confirmAction, a.confirmTarget
			a.confirmAction = ""
			a.confirmTarget = ""
			a.busy = true
			a.operation = "running doctor"
			a.status = "Running doctor for " + a.nameFor(targetID) + "…"
			return a, tea.Batch(doctorCommand(a.opts.RunDoctor, actionID, targetID), a.spin.Tick)
		case "n", "N", "esc", "escape", "backspace", "left", "h":
			targetID := a.confirmTarget
			a.confirmAction = ""
			a.confirmTarget = ""
			a.status = "Doctor cancelled for " + a.nameFor(targetID)
			return a, nil
		default:
			return a, nil
		}
	}

	if a.help {
		switch key {
		case "?", "esc", "escape", "backspace", "left", "h":
			a.help = false
			a.scroll = 0
			a.snapScroll()
		}
		return a, nil
	}

	switch key {
	case "?":
		a.help = true
		a.scroll = 0
		a.snapScroll()
		return a, nil
	case "esc", "escape", "backspace":
		if a.inspect != "" {
			a.inspect = ""
			a.scroll = 0
			a.snapScroll()
			a.focus = focusTree
		} else if a.focus == focusDetail {
			a.focus = focusTree
			a.scroll = 0
			a.snapScroll()
		} else if a.filterQuery() != "" {
			a.filter.SetValue("")
			a.ensureSelection()
		}
		return a, nil
	case "/":
		if a.inspect == "" && a.focus == focusTree {
			a.filterEditing = true
			return a, a.filter.Focus()
		}
		return a, nil
	case "left", "h":
		if a.inspect != "" {
			a.inspect = ""
			a.scroll = 0
			a.snapScroll()
			a.focus = focusTree
		} else if a.focus == focusDetail {
			a.focus = focusTree
			a.scroll = 0
			a.snapScroll()
		} else {
			a.foldOrJumpToParent()
		}
		return a, nil
	case "tab":
		if a.inspect == "" {
			if a.focus == focusTree {
				a.focus = focusDetail
			} else {
				a.focus = focusTree
			}
			a.scroll = 0
			a.snapScroll()
		}
		return a, nil
	case "up", "k":
		if a.inspect != "" || a.focus == focusDetail {
			return a, a.animateScrollTo(a.scroll - 1)
		}
		a.moveSelection(-1)
		return a, nil
	case "down", "j":
		if a.inspect != "" || a.focus == focusDetail {
			return a, a.animateScrollTo(a.scroll + 1)
		}
		a.moveSelection(1)
		return a, nil
	case "enter":
		if a.selected != "" {
			a.inspect = a.selected
			return a, a.beginEntrance()
		}
		return a, nil
	case "right", "l":
		if a.inspect == "" && a.focus == focusTree && a.selected != "" {
			if children := a.nodeChildren(a.selected); len(children) > 0 && !a.expandedState(a.selected, children) {
				a.expanded[a.selected] = true
				return a, nil
			}
		}
		if a.selected != "" {
			a.inspect = a.selected
			return a, a.beginEntrance()
		}
		return a, nil
	case "space", " ":
		if a.inspect == "" && a.focus == focusTree {
			a.toggleFold()
		}
		return a, nil
	case "r":
		if a.busy {
			a.status = "Already " + a.operation
			return a, nil
		}
		if a.opts.Refresh == nil {
			a.status = "Refresh is not available in this session"
			return a, nil
		}
		a.busy = true
		a.operation = "refreshing runtime graph"
		a.status = "Refreshing runtime graph…"
		return a, tea.Batch(refreshCommand(a.opts.Refresh), a.spin.Tick)
	case "d":
		if a.busy {
			a.status = "Already " + a.operation
			return a, nil
		}
		id := a.activeID()
		actions := a.doctorActions(id)
		if a.opts.RunDoctor == nil || len(actions) == 0 {
			a.status = "No doctor action is available for " + a.nameFor(id)
			return a, nil
		}
		if len(actions) > 1 {
			ids := make([]string, len(actions))
			for index, candidate := range actions {
				ids[index] = string(candidate.ID)
			}
			a.status = "Multiple doctor actions are available: " + strings.Join(ids, ", ")
			return a, nil
		}
		action := actions[0]
		if action.RequiresConfirmation {
			a.confirmAction = action.ID
			a.confirmTarget = id
			a.status = "Confirm doctor for " + a.nameFor(id)
			if action.RequiresRoot {
				a.status += " (requires root)"
			}
			return a, nil
		}
		a.busy = true
		a.operation = "running doctor"
		a.status = "Running doctor for " + a.nameFor(id) + "…"
		return a, tea.Batch(doctorCommand(a.opts.RunDoctor, action.ID, id), a.spin.Tick)
	}
	return a, nil
}

func refreshCommand(refresh func(context.Context) (*model.Graph, error)) tea.Cmd {
	return func() tea.Msg {
		graph, err := refresh(context.Background())
		return refreshDoneMsg{graph: graph, err: err}
	}
}

func doctorCommand(run func(context.Context, model.ID) (*model.Graph, error), actionID, componentID model.ID) tea.Cmd {
	return func() tea.Msg {
		graph, err := run(context.Background(), actionID)
		return doctorDoneMsg{id: componentID, graph: graph, err: err}
	}
}

func (a app) View() tea.View {
	content := a.render()
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "Muster · " + a.hostname
	colors := newPalette(a.dark, a.noColor)
	view.BackgroundColor = colors.bg
	view.ForegroundColor = colors.fg
	return view
}

// toggleFold flips the selected subtree between open and folded.
func (a *app) toggleFold() {
	if a.selected == "" {
		return
	}
	children := a.nodeChildren(a.selected)
	if len(children) == 0 {
		return
	}
	a.expanded[a.selected] = !a.expandedState(a.selected, children)
}

// foldOrJumpToParent folds an open subtree; on a leaf or an already folded
// node it climbs to the parent instead, the way file-tree UIs treat ←.
func (a *app) foldOrJumpToParent() {
	if a.selected == "" {
		return
	}
	children := a.nodeChildren(a.selected)
	if len(children) > 0 && a.expandedState(a.selected, children) {
		a.expanded[a.selected] = false
		return
	}
	for _, row := range a.treeRows() {
		if row.id == a.selected {
			if row.parent != "" {
				a.selected = row.parent
			}
			return
		}
	}
}

func (a *app) ensureSelection() {
	rows := a.treeRows()
	if len(rows) == 0 {
		a.selected = ""
		a.inspect = ""
		return
	}
	validateInspect := func() {
		if a.inspect != "" {
			if _, exists := a.graph.Lookup(a.inspect); !exists {
				a.inspect = ""
			}
		}
	}
	for _, row := range rows {
		if row.id == a.selected {
			validateInspect()
			return
		}
	}
	// The selection may have been folded away (for example after a refresh
	// turned its subtree healthy); stay as close to it as possible.
	if ancestor := a.nearestVisibleAncestor(a.selected, rows); ancestor != "" {
		a.selected = ancestor
		validateInspect()
		return
	}
	a.selected = rows[0].id
	validateInspect()
}

func (a app) nearestVisibleAncestor(id model.ID, visible []treeRow) model.ID {
	if id == "" {
		return ""
	}
	parents := make(map[model.ID]model.ID)
	for _, row := range a.allTreeRows() {
		parents[row.id] = row.parent
	}
	shown := make(map[model.ID]bool, len(visible))
	for _, row := range visible {
		shown[row.id] = true
	}
	for current := parents[id]; current != ""; current = parents[current] {
		if shown[current] {
			return current
		}
	}
	return ""
}

func (a *app) moveSelection(delta int) {
	rows := a.treeRows()
	if len(rows) == 0 {
		return
	}
	index := 0
	for candidate := range rows {
		if rows[candidate].id == a.selected {
			index = candidate
			break
		}
	}
	index += delta
	if index < 0 {
		index = len(rows) - 1
	}
	if index >= len(rows) {
		index = 0
	}
	a.selected = rows[index].id
	a.scroll = 0
	a.snapScroll()
}

func (a app) activeID() model.ID {
	if a.inspect != "" {
		return a.inspect
	}
	return a.selected
}

func (a app) doctorAction(id model.ID) (model.Action, bool) {
	actions := a.doctorActions(id)
	if len(actions) != 1 {
		return model.Action{}, false
	}
	return actions[0], true
}

func (a app) doctorActions(id model.ID) []model.Action {
	component, ok := a.graph.Lookup(id)
	if !ok {
		return nil
	}
	byID := make(map[model.ID]model.Action)
	collect := func(candidate *model.Component) {
		for _, action := range candidate.Actions {
			if isDoctorAction(action) {
				byID[action.ID] = action
			}
		}
	}
	collect(component)
	implementation, isImplementation := a.graph.LookupImplementation(id)
	if isImplementation {
		for _, componentID := range implementation.Components {
			candidate, exists := a.graph.Lookup(componentID)
			if exists {
				collect(candidate)
			}
		}
	}
	actions := make([]model.Action, 0, len(byID))
	for _, action := range byID {
		actions = append(actions, action)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].ID < actions[j].ID })
	return actions
}
