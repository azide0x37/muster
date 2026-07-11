package model

import (
	"fmt"
	"sort"
)

// ExplainDependencies finds one deterministic shortest path to every
// transitive dependency and dependent of id.
func (g *Graph) ExplainDependencies(id ID) (DependencyExplanation, error) {
	if _, ok := g.Lookup(id); !ok {
		return DependencyExplanation{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	return DependencyExplanation{
		ComponentID: id,
		DependsOn:   g.dependencyPaths(id, false),
		RequiredBy:  g.dependencyPaths(id, true),
	}, nil
}

// Explain returns the generic object, health propagation, and dependency
// explanation used by literate CLI and TUI views.
func (g *Graph) Explain(id ID) (Explanation, error) {
	component, ok := g.Lookup(id)
	if !ok {
		return Explanation{}, fmt.Errorf("%w: %s", ErrComponentNotFound, id)
	}
	health, err := g.ExplainHealth(id)
	if err != nil {
		return Explanation{}, err
	}
	dependencies, err := g.ExplainDependencies(id)
	if err != nil {
		return Explanation{}, err
	}
	return Explanation{
		Component:    cloneComponents([]Component{*component})[0],
		Health:       health,
		Dependencies: dependencies,
	}, nil
}

func (g *Graph) dependencyPaths(id ID, reverse bool) []DependencyPath {
	type queueItem struct {
		id   ID
		path []ID
	}
	queue := []queueItem{{id: id, path: []ID{id}}}
	visited := map[ID]struct{}{id: {}}
	result := make([]DependencyPath, 0)
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		for _, neighbor := range g.dependencyNeighbors(item.id, reverse) {
			if _, seen := visited[neighbor]; seen {
				continue
			}
			visited[neighbor] = struct{}{}
			path := append(append([]ID(nil), item.path...), neighbor)
			result = append(result, DependencyPath{ComponentID: neighbor, Path: path})
			queue = append(queue, queueItem{id: neighbor, path: path})
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if len(result[i].Path) != len(result[j].Path) {
			return len(result[i].Path) < len(result[j].Path)
		}
		return result[i].ComponentID < result[j].ComponentID
	})
	return result
}

func (g *Graph) dependencyNeighbors(id ID, reverse bool) []ID {
	edges := g.Outgoing(id, EdgeDependsOn)
	if reverse {
		edges = g.Incoming(id, EdgeDependsOn)
	}
	neighbors := make([]ID, 0, len(edges))
	seen := make(map[ID]struct{}, len(edges))
	for _, edge := range edges {
		neighbor := edge.To
		if reverse {
			neighbor = edge.From
		}
		if _, exists := seen[neighbor]; exists {
			continue
		}
		seen[neighbor] = struct{}{}
		neighbors = append(neighbors, neighbor)
	}
	sort.Slice(neighbors, func(i, j int) bool { return neighbors[i] < neighbors[j] })
	return neighbors
}
