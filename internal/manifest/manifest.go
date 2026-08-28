// Package manifest parses JavaScript package manifests without depending on a
// package manager implementation.
package manifest

import (
	"encoding/json"
	"fmt"
)

// Scope identifies the manifest section that owns a dependency.
type Scope string

const (
	ScopeProduction  Scope = "dependencies"
	ScopeDevelopment Scope = "devDependencies"
	ScopeOptional    Scope = "optionalDependencies"
	ScopePeer        Scope = "peerDependencies"
)

// Dependency is a direct dependency declaration from package.json.
type Dependency struct {
	Name  string `json:"name"`
	Range string `json:"range"`
	Scope Scope  `json:"scope"`
}

// Manifest is the dependency-oriented subset of package.json.
type Manifest struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

// Parse decodes a package.json document.
func Parse(data []byte) (*Manifest, error) {
	var result Manifest
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	return &result, nil
}

// Direct returns all direct dependencies keyed by package name. Runtime,
// optional, development, and peer declarations are considered in that order.
func (m *Manifest) Direct() map[string]Dependency {
	result := make(map[string]Dependency)
	add := func(values map[string]string, scope Scope) {
		for name, dependencyRange := range values {
			result[name] = Dependency{Name: name, Range: dependencyRange, Scope: scope}
		}
	}

	add(m.Dependencies, ScopeProduction)
	add(m.OptionalDependencies, ScopeOptional)
	add(m.DevDependencies, ScopeDevelopment)
	add(m.PeerDependencies, ScopePeer)
	return result
}
