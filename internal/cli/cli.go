// Package cli exposes Muster's object model as stable, scriptable commands.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/azide0x37/muster/internal/inspector"
	"github.com/azide0x37/muster/internal/lockfile"
	"github.com/azide0x37/muster/internal/manifest"
	"github.com/azide0x37/muster/internal/model"
	"github.com/azide0x37/muster/internal/present"
)

var Version = "dev"

type App struct {
	Out          io.Writer
	Err          io.Writer
	Hostname     string
	IsTTY        bool
	NewInspector func(root string) *inspector.Inspector
	LaunchTUI    func(context.Context, *inspector.Inspector, *inspector.Snapshot, string) error
}

type options struct {
	root string
	json bool
	help bool
	args []string
}

func (app App) Run(ctx context.Context, args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.help {
		usage(app.output())
		return nil
	}
	if len(opts.args) > 0 && opts.args[0] == "version" {
		if opts.json {
			return writeJSON(app.output(), map[string]string{"version": Version})
		}
		fmt.Fprintln(app.output(), Version)
		return nil
	}
	if len(opts.args) > 0 && opts.args[0] == "compile" {
		return app.compile(opts.args[1:], opts.json)
	}

	hostInspector := inspector.New(opts.root)
	if app.NewInspector != nil {
		hostInspector = app.NewInspector(opts.root)
	}
	snapshot, err := hostInspector.Load(ctx)
	if err != nil {
		return err
	}

	command := ""
	commandArgs := []string(nil)
	if len(opts.args) > 0 {
		command, commandArgs = opts.args[0], opts.args[1:]
	}
	if command == "" && app.IsTTY && app.LaunchTUI != nil && !opts.json {
		return app.LaunchTUI(ctx, hostInspector, snapshot, app.hostname())
	}
	if command == "" {
		command = "status"
	}

	switch command {
	case "list":
		return app.list(snapshot.Graph, opts.json)
	case "status":
		return app.status(snapshot.Graph, commandArgs, opts.json)
	case "inspect":
		return app.inspect(snapshot.Graph, commandArgs, opts.json)
	case "explain":
		return app.explain(snapshot.Graph, commandArgs, opts.json)
	case "export":
		return app.export(snapshot.Graph, commandArgs)
	case "doctor":
		return app.doctor(ctx, hostInspector, snapshot, commandArgs, opts.json)
	case "validate":
		if len(commandArgs) > 0 {
			return errors.New("validate takes no positional arguments")
		}
		if opts.json {
			return writeJSON(app.output(), map[string]any{"valid": true, "implementations": len(snapshot.Graph.Implementations)})
		}
		fmt.Fprintf(app.output(), "PASS: %d registered Muster implementation(s) form a valid object graph\n", len(snapshot.Graph.Implementations))
		return nil
	case "help":
		usage(app.output())
		return nil
	default:
		return fmt.Errorf("unknown command %q (try muster help)", command)
	}
}

func (app App) compile(args []string, asJSON bool) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("usage: muster compile MANIFEST [LOCK]")
	}
	manifestPath := args[0]
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	document, err := manifest.Decode(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	version := ""
	if document.Project.VersionFile != "" {
		versionData, readErr := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), document.Project.VersionFile))
		if readErr == nil {
			version = strings.TrimSpace(string(versionData))
		}
	}
	locked, err := lockfile.Generate(document, data, version)
	if err != nil {
		return err
	}
	lockPath := filepath.Join(filepath.Dir(manifestPath), "muster.lock.json")
	if len(args) == 2 {
		lockPath = args[1]
	}
	temporary := lockPath + ".new"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(locked); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, lockPath); err != nil {
		return err
	}
	if asJSON {
		return writeJSON(app.output(), map[string]any{"lock": lockPath, "manifest_sha256": locked.Source.ManifestSHA256})
	}
	fmt.Fprintf(app.output(), "Wrote %s (%d components, %d edges)\n", lockPath, len(locked.Graph.Components), len(locked.Graph.Edges))
	return nil
}

func (app App) list(graph *model.Graph, asJSON bool) error {
	type row struct {
		ID      model.ID           `json:"id"`
		Version string             `json:"version,omitempty"`
		Health  model.HealthStatus `json:"health"`
		Summary string             `json:"summary,omitempty"`
	}
	rows := make([]row, 0, len(graph.Implementations))
	for _, implementation := range graph.Implementations {
		component, _ := graph.Lookup(implementation.ID)
		rows = append(rows, row{ID: implementation.ID, Version: implementation.Version, Health: component.Health.Status, Summary: implementation.Summary})
	}
	if asJSON {
		return writeJSON(app.output(), rows)
	}
	for _, row := range rows {
		fmt.Fprintf(app.output(), "%s\t%s\t%s\t%s\n", row.ID, row.Health, row.Version, row.Summary)
	}
	return nil
}

func (app App) status(graph *model.Graph, args []string, asJSON bool) error {
	if len(args) > 1 {
		return errors.New("status accepts at most one implementation ID")
	}
	if len(args) == 0 {
		if asJSON {
			return app.list(graph, true)
		}
		present.Implementations(app.output(), graph, app.hostname())
		return nil
	}
	component, err := resolveComponent(graph, args[0])
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(app.output(), component)
	}
	present.Component(app.output(), graph, component)
	return nil
}

func (app App) inspect(graph *model.Graph, args []string, asJSON bool) error {
	if len(args) != 1 {
		return errors.New("usage: muster inspect ID")
	}
	if component, err := resolveComponent(graph, args[0]); err == nil {
		if asJSON {
			payload := struct {
				Component    model.Component     `json:"component"`
				Outgoing     []model.Edge        `json:"outgoing,omitempty"`
				Incoming     []model.Edge        `json:"incoming,omitempty"`
				Observations []model.Observation `json:"observations,omitempty"`
			}{component, graph.Outgoing(component.ID), graph.Incoming(component.ID), graph.ObservationsFor(component.ID)}
			return writeJSON(app.output(), payload)
		}
		present.Component(app.output(), graph, component)
		return nil
	}
	id := model.ID(args[0])
	if action, owner, ok := graph.LookupAction(id); ok {
		if asJSON {
			return writeJSON(app.output(), struct {
				Action model.Action `json:"action"`
				Owner  model.ID     `json:"owner"`
			}{*action, owner})
		}
		present.Action(app.output(), *action, owner)
		return nil
	}
	if observation, ok := graph.LookupObservation(id); ok {
		if asJSON {
			return writeJSON(app.output(), observation)
		}
		present.Observation(app.output(), *observation)
		return nil
	}
	return fmt.Errorf("object %q not found", args[0])
}

func (app App) explain(graph *model.Graph, args []string, asJSON bool) error {
	if len(args) != 1 {
		return errors.New("usage: muster explain ID")
	}
	component, componentErr := resolveComponent(graph, args[0])
	var prefix any
	if componentErr != nil {
		id := model.ID(args[0])
		if action, owner, ok := graph.LookupAction(id); ok {
			target := action.Target
			if target == "" {
				target = owner
			}
			resolved, ok := graph.Lookup(target)
			if !ok {
				return fmt.Errorf("action %s target %s not found", id, target)
			}
			component = *resolved
			prefix = struct {
				Action model.Action `json:"action"`
				Owner  model.ID     `json:"owner"`
			}{*action, owner}
		} else if observation, ok := graph.LookupObservation(id); ok {
			resolved, found := graph.Lookup(observation.ComponentID)
			if !found {
				return fmt.Errorf("observation %s component %s not found", id, observation.ComponentID)
			}
			component = *resolved
			prefix = *observation
		} else {
			return fmt.Errorf("object %q not found", args[0])
		}
	}
	explanation, err := graph.Explain(component.ID)
	if err != nil {
		return err
	}
	if asJSON {
		if prefix != nil {
			return writeJSON(app.output(), map[string]any{"object": prefix, "target_explanation": explanation})
		}
		return writeJSON(app.output(), explanation)
	}
	if prefix != nil {
		fmt.Fprintf(app.output(), "Object %s resolves through %s\n\n", args[0], component.ID)
	}
	present.Explanation(app.output(), graph, explanation)
	return nil
}

func (app App) export(graph *model.Graph, args []string) error {
	if len(args) > 1 {
		return errors.New("export accepts at most one component ID")
	}
	if len(args) == 0 {
		return writeJSON(app.output(), graph)
	}
	return app.inspect(graph, args, true)
}

func (app App) doctor(ctx context.Context, hostInspector *inspector.Inspector, snapshot *inspector.Snapshot, args []string, asJSON bool) error {
	if len(args) > 1 {
		return errors.New("doctor accepts at most one implementation, component, or action ID")
	}
	target := ""
	if len(args) == 1 {
		target = args[0]
	}
	actionID, err := findDoctorAction(snapshot, target)
	if err != nil {
		return err
	}
	action := snapshot.Actions[actionID]
	previous, hadPrevious := snapshot.Graph.LatestObservation(action.ComponentID, model.ObservationDoctor)
	output, err := hostInspector.RunDoctor(ctx, snapshot, actionID)
	runErr := err
	reloaded, reloadErr := hostInspector.Load(ctx)
	observation, found := model.Observation{}, false
	if reloadErr == nil {
		observation, found = reloaded.Graph.LatestObservation(action.ComponentID, model.ObservationDoctor)
		if found && hadPrevious && observation.ID == previous.ID {
			found = false
		}
	}
	if asJSON {
		var writeErr error
		if !found {
			payload := map[string]any{"action": actionID, "output": string(output)}
			if runErr != nil {
				payload["error"] = runErr.Error()
			}
			writeErr = writeJSON(app.output(), payload)
		} else {
			writeErr = writeJSON(app.output(), observation)
		}
		if writeErr != nil {
			return writeErr
		}
	} else if _, writeErr := app.output().Write(output); writeErr != nil {
		return writeErr
	}
	if reloadErr != nil {
		if runErr != nil {
			return fmt.Errorf("doctor %s: %w; evidence reload failed: %v", actionID, runErr, reloadErr)
		}
		return fmt.Errorf("reload doctor evidence: %w", reloadErr)
	}
	if runErr != nil {
		if found {
			return fmt.Errorf("doctor %s completed with unhealthy evidence: %w", actionID, runErr)
		}
		return fmt.Errorf("doctor %s did not produce new evidence: %w", actionID, runErr)
	}
	if !found {
		return fmt.Errorf("doctor %s completed without producing new evidence", actionID)
	}
	return nil
}

func findDoctorAction(snapshot *inspector.Snapshot, target string) (model.ID, error) {
	if target != "" {
		if action, ok := snapshot.Actions[model.ID(target)]; ok && action.Spec.Kind == "doctor.run" {
			return model.ID(target), nil
		}
		component, err := resolveComponent(snapshot.Graph, target)
		if err != nil {
			return "", err
		}
		ids := make(map[model.ID]struct{})
		collect := func(candidate model.Component) {
			for _, action := range candidate.Actions {
				if executable, ok := snapshot.Actions[action.ID]; ok && action.Kind == "doctor.run" && executable.Spec.Kind == "doctor.run" {
					ids[action.ID] = struct{}{}
				}
			}
		}
		collect(component)
		if implementation, ok := snapshot.Graph.LookupImplementation(component.ID); ok {
			for _, componentID := range implementation.Components {
				candidate, _ := snapshot.Graph.Lookup(componentID)
				collect(*candidate)
			}
		}
		ordered := make([]model.ID, 0, len(ids))
		for id := range ids {
			ordered = append(ordered, id)
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
		switch len(ordered) {
		case 0:
			return "", fmt.Errorf("%s has no doctor action", target)
		case 1:
			return ordered[0], nil
		default:
			return "", fmt.Errorf("%s has multiple doctor actions; specify an action ID", target)
		}
	}
	ids := make([]model.ID, 0)
	for id, action := range snapshot.Actions {
		if action.Spec.Kind == "doctor.run" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) == 1 {
		return ids[0], nil
	}
	if len(ids) == 0 {
		return "", errors.New("no doctor actions are registered")
	}
	return "", errors.New("multiple doctors are registered; specify an implementation or component ID")
}

func resolveComponent(graph *model.Graph, value string) (model.Component, error) {
	candidates := []model.ID{model.ID(value)}
	if !strings.Contains(value, ":") {
		candidates = append(candidates, model.ID("implementation:"+value))
	}
	for _, candidate := range candidates {
		if component, ok := graph.Lookup(candidate); ok {
			return *component, nil
		}
	}
	return model.Component{}, fmt.Errorf("object %q not found", value)
}

func parseOptions(args []string) (options, error) {
	var result options
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--json":
			result.json = true
		case argument == "--help" || argument == "-h":
			result.help = true
		case argument == "--root":
			index++
			if index >= len(args) {
				return options{}, errors.New("--root requires a path")
			}
			result.root = args[index]
		case strings.HasPrefix(argument, "--root="):
			result.root = strings.TrimPrefix(argument, "--root=")
		case strings.HasPrefix(argument, "-") && len(result.args) == 0:
			return options{}, fmt.Errorf("unknown option %s", argument)
		default:
			result.args = append(result.args, argument)
		}
	}
	return result, nil
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (app App) output() io.Writer {
	if app.Out != nil {
		return app.Out
	}
	return os.Stdout
}

func (app App) hostname() string {
	if app.Hostname != "" {
		return app.Hostname
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "this server"
	}
	return hostname
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Muster — inspect the implementations on this server

Usage:
  muster                         open the TUI (or print status when piped)
  muster list [--json]           list registered implementations
  muster status [ID] [--json]    summarize health
  muster inspect ID [--json]     inspect one globally addressed object
  muster explain ID [--json]     explain health and graph dependencies
  muster doctor [ID] [--json]    run declared doctor evidence (may require root)
  muster export [ID]             export canonical JSON
  muster compile MANIFEST [LOCK] compile a deterministic release lock
  muster validate [--json]       validate the registered object graph
  muster version [--json]        print the shared core version

Global options:
  --root PATH                    inspect a staged root instead of /
  --json                         emit machine-readable JSON`)
}
