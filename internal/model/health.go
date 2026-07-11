package model

import (
	"fmt"
	"sort"
)

// DerivedHealth recursively combines a component's declared health with the
// health of its children, owned components, and dependencies. The finite
// fixed-point calculation is deterministic and remains safe for graph cycles.
func (g *Graph) DerivedHealth(id ID) (Health, error) {
	component, ok := g.Lookup(id)
	if !ok {
		return Health{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	statuses := g.derivedStatuses()
	effective := statuses[id]
	declared := componentDeclaredHealth(component)
	if declared.Status == effective {
		return declared, nil
	}

	explanation, err := g.explainHealth(id, statuses)
	if err != nil {
		return Health{}, err
	}
	summary := string(effective)
	if len(explanation.Causes) == 1 {
		cause := explanation.Causes[0]
		summary = fmt.Sprintf("%s because %s is %s", effective, cause.ComponentID, cause.Health.Status)
	} else if len(explanation.Causes) > 1 {
		summary = fmt.Sprintf("%s because %d related components are %s", effective, len(explanation.Causes), effective)
	}
	return Health{Status: effective, Summary: summary}, nil
}

// ExplainHealth describes recursive health propagation with shortest paths to
// the direct health assertions responsible for the effective result.
func (g *Graph) ExplainHealth(id ID) (HealthExplanation, error) {
	if _, ok := g.Lookup(id); !ok {
		return HealthExplanation{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	return g.explainHealth(id, g.derivedStatuses())
}

func (g *Graph) explainHealth(id ID, statuses map[ID]HealthStatus) (HealthExplanation, error) {
	component, ok := g.Lookup(id)
	if !ok {
		return HealthExplanation{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	effective, err := g.derivedHealthFromStatuses(id, statuses)
	if err != nil {
		return HealthExplanation{}, err
	}

	explanation := HealthExplanation{
		ComponentID: id,
		Declared:    componentDeclaredHealth(component),
		Effective:   effective,
	}
	if effective.Status == HealthHealthy {
		explanation.Causes = []HealthCause{{ComponentID: id, Health: componentDeclaredHealth(component)}}
		return explanation, nil
	}

	type queueItem struct {
		id   ID
		path []HealthStep
	}
	queue := []queueItem{{id: id}}
	visitedDepth := map[ID]int{id: 0}
	bestDepth := make(map[ID]int)
	causes := make(map[ID]HealthCause)
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		current, exists := g.Lookup(item.id)
		if !exists || statuses[item.id] != effective.Status {
			continue
		}
		declared := componentDeclaredHealth(current)
		if declared.Status == effective.Status {
			if depth, seen := bestDepth[item.id]; !seen || len(item.path) < depth {
				bestDepth[item.id] = len(item.path)
				causes[item.id] = HealthCause{
					ComponentID: item.id,
					Health:      declared,
					Path:        append([]HealthStep(nil), item.path...),
				}
			}
		}
		for _, link := range g.healthLinks(item.id) {
			if statuses[link.To] != effective.Status {
				continue
			}
			path := append(append([]HealthStep(nil), item.path...), link)
			if depth, seen := visitedDepth[link.To]; seen && depth <= len(path) {
				continue
			}
			visitedDepth[link.To] = len(path)
			queue = append(queue, queueItem{id: link.To, path: path})
		}
	}

	explanation.Causes = make([]HealthCause, 0, len(causes))
	for _, cause := range causes {
		explanation.Causes = append(explanation.Causes, cause)
	}
	sort.Slice(explanation.Causes, func(i, j int) bool {
		a, b := explanation.Causes[i], explanation.Causes[j]
		if len(a.Path) != len(b.Path) {
			return len(a.Path) < len(b.Path)
		}
		return a.ComponentID < b.ComponentID
	})
	return explanation, nil
}

func (g *Graph) derivedHealthFromStatuses(id ID, statuses map[ID]HealthStatus) (Health, error) {
	component, ok := g.Lookup(id)
	if !ok {
		return Health{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	effective, exists := statuses[id]
	if !exists {
		effective = componentDeclaredHealth(component).Status
	}
	declared := componentDeclaredHealth(component)
	if declared.Status == effective {
		return declared, nil
	}
	return Health{Status: effective}, nil
}

func (g *Graph) derivedStatuses() map[ID]HealthStatus {
	statuses := make(map[ID]HealthStatus, len(g.Components))
	for _, component := range g.Components {
		statuses[component.ID] = componentDeclaredHealth(&component).Status
	}
	// Every update moves one value monotonically through a four-level lattice,
	// so this converges even when depends_on or owns relationships contain cycles.
	changed := true
	for changed {
		changed = false
		for _, component := range g.Components {
			status := statuses[component.ID]
			for _, link := range g.healthLinks(component.ID) {
				if related, exists := statuses[link.To]; exists {
					status = worseHealthStatus(status, related)
				}
			}
			if status != statuses[component.ID] {
				statuses[component.ID] = status
				changed = true
			}
		}
	}
	return statuses
}

func (g *Graph) healthLinks(id ID) []HealthStep {
	links := make([]HealthStep, 0)
	seen := make(map[string]struct{})
	if component, ok := g.Lookup(id); ok {
		for _, child := range component.Children {
			key := string(EdgeOwns) + "\x00" + string(child)
			seen[key] = struct{}{}
			links = append(links, HealthStep{From: id, Relationship: EdgeOwns, To: child})
		}
	}
	for _, edge := range g.Outgoing(id, EdgeOwns, EdgeDependsOn) {
		key := string(edge.Type) + "\x00" + string(edge.To)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		links = append(links, HealthStep{From: id, Relationship: edge.Type, To: edge.To})
	}
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].Relationship != links[j].Relationship {
			return links[i].Relationship < links[j].Relationship
		}
		return links[i].To < links[j].To
	})
	return links
}

func normalizedHealth(health Health) Health {
	health.Status = normalizeHealthStatus(health.Status)
	return health
}

func componentDeclaredHealth(component *Component) Health {
	if component.DeclaredHealth != nil {
		return normalizedHealth(*component.DeclaredHealth)
	}
	return normalizedHealth(component.Health)
}

// MaterializeDerivedHealth preserves every adapter's direct assertion and
// replaces Health with the recursively effective value. It is idempotent, so
// refresh and export paths can safely call it without losing causal evidence.
func (g *Graph) MaterializeDerivedHealth() error {
	if g == nil {
		return fmt.Errorf("model: nil graph")
	}
	for index := range g.Components {
		if g.Components[index].DeclaredHealth == nil {
			declared := normalizedHealth(g.Components[index].Health)
			g.Components[index].DeclaredHealth = &declared
		}
	}
	for index := range g.Components {
		health, err := g.DerivedHealth(g.Components[index].ID)
		if err != nil {
			return err
		}
		g.Components[index].Health = health
	}
	return nil
}

func normalizeHealthStatus(status HealthStatus) HealthStatus {
	if status == "" {
		return HealthUnknown
	}
	return status
}

func worseHealthStatus(left, right HealthStatus) HealthStatus {
	left = normalizeHealthStatus(left)
	right = normalizeHealthStatus(right)
	if healthSeverity(right) > healthSeverity(left) {
		return right
	}
	return left
}

func healthSeverity(status HealthStatus) int {
	switch normalizeHealthStatus(status) {
	case HealthHealthy:
		return 0
	case HealthUnknown:
		return 1
	case HealthDegraded:
		return 2
	case HealthUnhealthy:
		return 3
	default:
		return 1
	}
}
