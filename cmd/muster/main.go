package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	charmterm "github.com/charmbracelet/x/term"

	"github.com/azide0x37/muster/internal/cli"
	"github.com/azide0x37/muster/internal/inspector"
	"github.com/azide0x37/muster/internal/model"
	"github.com/azide0x37/muster/internal/tui"
)

func main() {
	application := cli.App{
		Out: os.Stdout, Err: os.Stderr, IsTTY: isTerminal(os.Stdin) && isTerminal(os.Stdout),
		LaunchTUI: launchTUI,
	}
	if err := application.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "muster: %v\n", err)
		os.Exit(1)
	}
}

func launchTUI(_ context.Context, host *inspector.Inspector, initial *inspector.Snapshot, hostname string) error {
	current := initial
	refresh := func(refreshCtx context.Context) (*model.Graph, error) {
		next, err := host.Load(refreshCtx)
		if err != nil {
			return nil, err
		}
		current = next
		return next.Graph, nil
	}
	runDoctor := func(doctorCtx context.Context, actionID model.ID) (*model.Graph, error) {
		action, ok := current.Actions[actionID]
		if !ok {
			return nil, fmt.Errorf("doctor action %s is no longer registered", actionID)
		}
		previous, hadPrevious := current.Graph.LatestObservation(action.ComponentID, model.ObservationDoctor)
		_, runErr := host.RunDoctor(doctorCtx, current, actionID)
		next, reloadErr := host.Load(doctorCtx)
		if reloadErr != nil {
			if runErr != nil {
				return nil, fmt.Errorf("execution failed (%v) and evidence reload failed: %w", runErr, reloadErr)
			}
			return nil, fmt.Errorf("evidence reload failed: %w", reloadErr)
		}
		current = next
		latest, hasLatest := next.Graph.LatestObservation(action.ComponentID, model.ObservationDoctor)
		produced := hasLatest && (!hadPrevious || latest.ID != previous.ID)
		if runErr != nil {
			if produced {
				return next.Graph, fmt.Errorf("completed with unhealthy evidence: %w", runErr)
			}
			return next.Graph, fmt.Errorf("did not run or produce new evidence: %w", runErr)
		}
		if !produced {
			return next.Graph, fmt.Errorf("completed without producing new evidence")
		}
		return next.Graph, nil
	}
	return tui.Run(initial.Graph, tui.Options{
		Hostname: hostname, Refresh: refresh, RunDoctor: runDoctor,
		NoColor: noColorRequested(),
	})
}

func isTerminal(file *os.File) bool {
	if file == nil || !terminalTypeAllowsTUI(os.Getenv("TERM")) {
		return false
	}
	return charmterm.IsTerminal(file.Fd())
}

func terminalTypeAllowsTUI(value string) bool {
	return !strings.EqualFold(strings.TrimSpace(value), "dumb")
}

func noColorRequested() bool {
	value, present := os.LookupEnv("NO_COLOR")
	return present && value != ""
}
