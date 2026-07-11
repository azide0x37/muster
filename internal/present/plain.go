// Package present renders the normalized graph for non-interactive terminals.
package present

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/azide0x37/muster/internal/model"
)

func StatusGlyph(status model.HealthStatus) string {
	switch status {
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

func Implementations(w io.Writer, graph *model.Graph, hostname string) {
	fmt.Fprintf(w, "Muster implementations on %s\n", hostname)
	if len(graph.Implementations) == 0 {
		fmt.Fprintln(w, "No Muster implementations are registered.")
		return
	}
	for _, implementation := range graph.Implementations {
		component, _ := graph.Lookup(implementation.ID)
		health := component.Health
		version := ""
		if implementation.Version != "" {
			version = " · " + implementation.Version
		}
		fmt.Fprintf(w, "%s %-10s %s%s\n", StatusGlyph(health.Status), strings.ToUpper(string(health.Status)), implementation.ID, version)
		if implementation.Summary != "" {
			fmt.Fprintf(w, "  %s\n", implementation.Summary)
		}
	}
}

func Component(w io.Writer, graph *model.Graph, component model.Component) {
	fmt.Fprintf(w, "%s\n%s %s · %s\n", component.ID, StatusGlyph(component.Health.Status), strings.ToUpper(string(component.Health.Status)), component.Kind)
	if component.DeclaredHealth != nil && component.DeclaredHealth.Status != component.Health.Status {
		fmt.Fprintf(w, "  Declared %s · effective %s\n", strings.ToUpper(string(component.DeclaredHealth.Status)), strings.ToUpper(string(component.Health.Status)))
	}
	if component.Summary != "" {
		fmt.Fprintf(w, "\n%s\n", component.Summary)
	}
	if component.What != "" {
		fmt.Fprintf(w, "\nWhat\n  %s\n", component.What)
	}
	if component.Why != "" {
		fmt.Fprintf(w, "\nWhy\n  %s\n", component.Why)
	}
	if len(component.Responsibilities) > 0 {
		fmt.Fprintln(w, "\nResponsibilities")
		for _, responsibility := range component.Responsibilities {
			fmt.Fprintf(w, "  • %s\n", responsibility)
		}
	}
	if len(component.FailureModes) > 0 {
		fmt.Fprintln(w, "\nFailure modes")
		for _, failure := range component.FailureModes {
			fmt.Fprintf(w, "  • %s\n", failure.Summary)
		}
	}
	if len(component.Children) > 0 {
		fmt.Fprintln(w, "\nChildren")
		for _, childID := range component.Children {
			child, ok := graph.Lookup(childID)
			if !ok {
				continue
			}
			fmt.Fprintf(w, "  %s %-10s %s — %s\n", StatusGlyph(child.Health.Status), child.Health.Status, child.ID, child.Summary)
		}
	}
	if len(component.Metadata) > 0 {
		fmt.Fprintln(w, "\nMetadata")
		keys := make([]string, 0, len(component.Metadata))
		for key := range component.Metadata {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(w, "  %-18s %s\n", key, component.Metadata[key])
		}
	}
	if len(component.Actions) > 0 {
		fmt.Fprintln(w, "\nActions")
		for _, action := range component.Actions {
			requirement := ""
			if action.RequiresRoot {
				requirement = " · requires root"
			}
			fmt.Fprintf(w, "  • %s · %s%s\n", action.ID, action.Label, requirement)
		}
	}
	relations := append(graph.Outgoing(component.ID), graph.Incoming(component.ID)...)
	if len(relations) > 0 {
		fmt.Fprintln(w, "\nRelations")
		for _, edge := range relations {
			direction, other := "→", edge.To
			if edge.To == component.ID {
				direction, other = "←", edge.From
			}
			fmt.Fprintf(w, "  %s %-12s %s", direction, edge.Type, other)
			if edge.Summary != "" {
				fmt.Fprintf(w, " — %s", edge.Summary)
			}
			fmt.Fprintln(w)
		}
	}
	observations := graph.ObservationsFor(component.ID)
	if len(observations) > 0 {
		fmt.Fprintln(w, "\nObservations")
		for _, observation := range observations {
			Observation(w, observation)
		}
	}
}

func Action(w io.Writer, action model.Action, componentID model.ID) {
	fmt.Fprintf(w, "%s\n%s\n  Owner: %s\n  Target: %s\n", action.ID, action.Label, componentID, action.Target)
	fmt.Fprintf(w, "  Kind: %s\n", action.Kind)
	if action.Summary != "" {
		fmt.Fprintf(w, "  %s\n", action.Summary)
	}
	if action.RequiresRoot {
		fmt.Fprintln(w, "  Requires root: yes")
	}
	if action.RequiresConfirmation {
		fmt.Fprintln(w, "  Requires confirmation: yes")
	}
}

func Observation(w io.Writer, observation model.Observation) {
	health := observation.DerivedHealth()
	stale := ""
	if observation.Stale {
		stale = " · stale"
	}
	fmt.Fprintf(w, "  %s %s · %s · %s%s\n", StatusGlyph(health.Status), observation.ID, observation.Kind, relativeTime(observation.ObservedAt), stale)
	if observation.Summary != "" {
		fmt.Fprintf(w, "    %s\n", observation.Summary)
	}
	for _, check := range observation.Checks {
		fmt.Fprintf(w, "    %s %-7s %s — %s\n", checkGlyph(check.Status), check.Status, check.ID, check.Summary)
	}
	for _, artifact := range observation.Artifacts {
		fmt.Fprintf(w, "    artifact: %s", artifact.URI)
		if artifact.Summary != "" {
			fmt.Fprintf(w, " — %s", artifact.Summary)
		}
		fmt.Fprintln(w)
	}
}

func Explanation(w io.Writer, graph *model.Graph, explanation model.Explanation) {
	Component(w, graph, explanation.Component)
	fmt.Fprintf(w, "\nHealth\n  Declared: %s\n  Effective: %s\n", strings.ToUpper(string(explanation.Health.Declared.Status)), strings.ToUpper(string(explanation.Health.Effective.Status)))
	if len(explanation.Health.Causes) > 0 {
		fmt.Fprintln(w, "\nHealth causes")
		for _, cause := range explanation.Health.Causes {
			fmt.Fprintf(w, "  %s %s — %s\n", StatusGlyph(cause.Health.Status), cause.ComponentID, cause.Health.Summary)
			if len(cause.Path) > 0 {
				parts := []string{string(cause.Path[0].From)}
				for _, step := range cause.Path {
					parts = append(parts, fmt.Sprintf("-%s-> %s", step.Relationship, step.To))
				}
				fmt.Fprintf(w, "    %s\n", strings.Join(parts, " "))
			}
		}
	}
	if len(explanation.Dependencies.DependsOn) > 0 {
		fmt.Fprintln(w, "\nDepends on")
		for _, dependency := range explanation.Dependencies.DependsOn {
			fmt.Fprintf(w, "  • %s\n", strings.Join(idsToStrings(dependency.Path), " → "))
		}
	}
	if len(explanation.Dependencies.RequiredBy) > 0 {
		fmt.Fprintln(w, "\nRequired by")
		for _, dependency := range explanation.Dependencies.RequiredBy {
			fmt.Fprintf(w, "  • %s\n", strings.Join(idsToStrings(dependency.Path), " → "))
		}
	}
}

func idsToStrings(ids []model.ID) []string {
	result := make([]string, len(ids))
	for index, id := range ids {
		result[index] = string(id)
	}
	return result
}

func checkGlyph(status model.CheckStatus) string {
	switch status {
	case model.CheckPass:
		return "✓"
	case model.CheckWarn:
		return "!"
	case model.CheckFail:
		return "×"
	default:
		return "?"
	}
}

func relativeTime(value time.Time) string {
	delta := time.Since(value)
	if delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	}
	return value.Format(time.RFC3339)
}
