// Package tui presents Muster's normalized runtime graph as a full-screen,
// keyboard-driven operator console.
package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/azide0x37/muster/internal/model"
)

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
	_, err := tea.NewProgram(newApp(graph, opts)).Run()
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
	}
	result.ensureSelection()
	return result
}

func (a app) Init() tea.Cmd {
	return tea.RequestBackgroundColor
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
		return a, nil
	case tea.BackgroundColorMsg:
		a.dark = message.IsDark()
		return a, nil
	case tea.ColorProfileMsg:
		a.noColor = a.forceNoColor || message.Profile <= colorprofile.ASCII
		return a, nil
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
		a.ensureSelection()
		a.scroll = a.clampScroll(a.scroll)
		if afterDoctor {
			a.status = previousStatus + " · runtime graph refreshed"
		} else {
			a.status = "Runtime graph refreshed"
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
		return a.handleKey(message.String())
	}
	return a, nil
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
			return a, doctorCommand(a.opts.RunDoctor, actionID, targetID)
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
		}
		return a, nil
	}

	switch key {
	case "?":
		a.help = true
		a.scroll = 0
		return a, nil
	case "esc", "escape", "backspace", "left", "h":
		if a.inspect != "" {
			a.inspect = ""
			a.scroll = 0
			a.focus = focusTree
		} else if a.focus == focusDetail {
			a.focus = focusTree
			a.scroll = 0
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
		}
		return a, nil
	case "up", "k":
		if a.inspect != "" || a.focus == focusDetail {
			a.scroll = a.clampScroll(a.scroll - 1)
		} else {
			a.moveSelection(-1)
		}
		return a, nil
	case "down", "j":
		if a.inspect != "" || a.focus == focusDetail {
			a.scroll = a.clampScroll(a.scroll + 1)
		} else {
			a.moveSelection(1)
		}
		return a, nil
	case "enter", "right", "l":
		if a.selected != "" {
			a.inspect = a.selected
			a.scroll = 0
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
		return a, refreshCommand(a.opts.Refresh)
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
		return a, doctorCommand(a.opts.RunDoctor, action.ID, id)
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

func (a *app) ensureSelection() {
	rows := a.treeRows()
	if len(rows) == 0 {
		a.selected = ""
		a.inspect = ""
		return
	}
	for _, row := range rows {
		if row.id == a.selected {
			if a.inspect != "" {
				if _, exists := a.graph.Lookup(a.inspect); !exists {
					a.inspect = ""
				}
			}
			return
		}
	}
	a.selected = rows[0].id
	if a.inspect != "" {
		if _, exists := a.graph.Lookup(a.inspect); !exists {
			a.inspect = ""
		}
	}
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
