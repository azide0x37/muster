// Package registry discovers installed Muster implementations without owning
// their lifecycle. Registry entries are stable locators to each project's
// active, versioned manifest.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const LogicalDirectory = "/etc/muster/implementations.d"

type Entry struct {
	Schema   int    `json:"schema"`
	ID       string `json:"id"`
	Manifest string `json:"manifest"`
	Lock     string `json:"lock,omitempty"`
	Source   string `json:"-"`
}

// HostPath resolves a logical absolute host path beneath root. This is what
// lets the real inspector exercise staged installations without chrooting.
func HostPath(root, logical string) (string, error) {
	if !filepath.IsAbs(logical) {
		return "", fmt.Errorf("logical path must be absolute: %q", logical)
	}
	clean := filepath.Clean(logical)
	if clean == "/" {
		return filepath.Clean(root), nil
	}
	if root == "" || root == "/" {
		return clean, nil
	}
	return filepath.Join(filepath.Clean(root), strings.TrimPrefix(clean, "/")), nil
}

func Load(root string) ([]Entry, error) {
	directory, err := HostPath(root, LogicalDirectory)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Muster registry %s: %w", directory, err)
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
			continue
		}
		path := filepath.Join(directory, item.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read registry entry %s: %w", path, readErr)
		}
		var entry Entry
		if decodeErr := json.Unmarshal(data, &entry); decodeErr != nil {
			return nil, fmt.Errorf("decode registry entry %s: %w", path, decodeErr)
		}
		entry.Source = path
		if entry.Schema != 1 {
			return nil, fmt.Errorf("registry entry %s has unsupported schema %d", path, entry.Schema)
		}
		if entry.ID == "" || entry.Manifest == "" {
			return nil, fmt.Errorf("registry entry %s requires id and manifest", path)
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	for index := 1; index < len(entries); index++ {
		if entries[index-1].ID == entries[index].ID {
			return nil, fmt.Errorf("duplicate registry implementation ID %s", entries[index].ID)
		}
	}
	return entries, nil
}

func ResolveManifest(root string, entry Entry) (string, error) {
	return HostPath(root, entry.Manifest)
}
