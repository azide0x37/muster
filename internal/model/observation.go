package model

import "sort"

// DerivedHealth combines an observation's explicit status with all of its
// checks. A failing check cannot be hidden by an optimistic summary status.
func (o Observation) DerivedHealth() Health {
	if o.Stale {
		observedAt := o.ObservedAt
		return Health{Status: HealthUnknown, Summary: "observation is stale", ObservedAt: &observedAt}
	}
	status := o.Status
	if status == "" && len(o.Checks) > 0 {
		status = HealthHealthy
	} else {
		status = normalizeHealthStatus(status)
	}
	for _, check := range o.Checks {
		status = worseHealthStatus(status, checkHealthStatus(check.Status))
	}
	observedAt := o.ObservedAt
	return Health{Status: status, Summary: o.Summary, ObservedAt: &observedAt}
}

// ObservationsFor returns observations for a component newest-first, with IDs
// as a deterministic tie breaker.
func (g *Graph) ObservationsFor(componentID ID) []Observation {
	if g == nil {
		return nil
	}
	result := make([]Observation, 0)
	for _, observation := range g.Observations {
		if observation.ComponentID == componentID {
			result = append(result, cloneObservations([]Observation{observation})[0])
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].ObservedAt.Equal(result[j].ObservedAt) {
			return result[i].ObservedAt.After(result[j].ObservedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// LatestObservation returns the newest observation of kind for a component.
// Passing an empty kind considers every observation kind.
func (g *Graph) LatestObservation(componentID ID, kind ObservationKind) (Observation, bool) {
	for _, observation := range g.ObservationsFor(componentID) {
		if kind == "" || observation.Kind == kind {
			return observation, true
		}
	}
	return Observation{}, false
}

func checkHealthStatus(status CheckStatus) HealthStatus {
	switch status {
	case CheckPass:
		return HealthHealthy
	case CheckWarn:
		return HealthDegraded
	case CheckFail:
		return HealthUnhealthy
	case "", CheckUnknown:
		return HealthUnknown
	default:
		return HealthUnknown
	}
}
