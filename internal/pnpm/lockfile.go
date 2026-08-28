// Package pnpm parses the dependency graph recorded by pnpm lockfiles.
package pnpm

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DependencyRef is a dependency declaration inside a pnpm importer.
type DependencyRef struct {
	Specifier string `yaml:"specifier" json:"specifier"`
	Version   string `yaml:"version" json:"version"`
}

// Importer is the resolved direct-dependency state for one workspace package.
type Importer struct {
	Dependencies         map[string]DependencyRef `yaml:"dependencies"`
	DevDependencies      map[string]DependencyRef `yaml:"devDependencies"`
	OptionalDependencies map[string]DependencyRef `yaml:"optionalDependencies"`
}

// Node is a resolved package node and its outgoing dependency edges.
type Node struct {
	Dependencies         map[string]string `yaml:"dependencies"`
	OptionalDependencies map[string]string `yaml:"optionalDependencies"`
}

// Lockfile contains the pnpm v9 fields needed for review.
type Lockfile struct {
	LockfileVersion any                 `yaml:"lockfileVersion"`
	Importers       map[string]Importer `yaml:"importers"`
	Packages        map[string]Node     `yaml:"packages"`
	Snapshots       map[string]Node     `yaml:"snapshots"`
	canonical       any
}

// Parse decodes a pnpm lockfile and rejects unsupported major versions.
func Parse(data []byte) (*Lockfile, error) {
	var result Lockfile
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse pnpm-lock.yaml: %w", err)
	}

	version := fmt.Sprint(result.LockfileVersion)
	if !strings.HasPrefix(version, "9") {
		return nil, fmt.Errorf("unsupported pnpm lockfile version %q: only v9 is currently supported", version)
	}

	if result.Importers == nil {
		return nil, fmt.Errorf("parse pnpm-lock.yaml: importers section is missing")
	}

	if err := yaml.Unmarshal(data, &result.canonical); err != nil {
		return nil, fmt.Errorf("normalize pnpm-lock.yaml: %w", err)
	}
	return &result, nil
}

// SemanticallyEqual reports whether two lockfiles contain the same YAML data,
// independent of quoting, whitespace, or map ordering.
func (l *Lockfile) SemanticallyEqual(other *Lockfile) bool {
	return reflect.DeepEqual(l.canonical, other.canonical)
}

// Direct returns every resolved dependency in the selected importer.
func (l *Lockfile) Direct(importer string) map[string]DependencyRef {
	entry, ok := l.Importers[importer]
	if !ok {
		return nil
	}

	result := make(map[string]DependencyRef)
	for name, dependency := range entry.Dependencies {
		result[name] = dependency
	}
	for name, dependency := range entry.OptionalDependencies {
		result[name] = dependency
	}
	for name, dependency := range entry.DevDependencies {
		result[name] = dependency
	}
	return result
}

// VersionsByPackage returns the distinct resolved versions for every package.
func (l *Lockfile) VersionsByPackage() map[string][]string {
	source := l.Packages
	if len(source) == 0 {
		source = l.Snapshots
	}

	sets := make(map[string]map[string]struct{})
	for key := range source {
		name, version, ok := SplitPackageKey(key)
		if !ok {
			continue
		}
		if sets[name] == nil {
			sets[name] = make(map[string]struct{})
		}
		sets[name][version] = struct{}{}
	}

	result := make(map[string][]string, len(sets))
	for name, versions := range sets {
		for version := range versions {
			result[name] = append(result[name], version)
		}
		sort.Strings(result[name])
	}
	return result
}

// DependencyClosure returns package names reachable from the named direct
// dependencies in an importer. Both the roots and transitive packages are
// included.
func (l *Lockfile) DependencyClosure(importer string, roots []string) map[string]struct{} {
	direct := l.Direct(importer)
	result := make(map[string]struct{})
	visited := make(map[string]struct{})
	queue := make([]string, 0, len(roots))

	for _, name := range roots {
		result[name] = struct{}{}
		dependency, ok := direct[name]
		if !ok {
			continue
		}
		if key := l.resolveKey(name, dependency.Version); key != "" {
			queue = append(queue, key)
		}
	}

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := visited[key]; ok {
			continue
		}
		visited[key] = struct{}{}

		name, _, ok := SplitPackageKey(key)
		if ok {
			result[name] = struct{}{}
		}

		node, ok := l.Snapshots[key]
		if !ok {
			node, ok = l.Packages[key]
		}
		if !ok {
			continue
		}

		for dependencyName, dependencyVersion := range node.Dependencies {
			if dependencyKey := l.resolveKey(dependencyName, dependencyVersion); dependencyKey != "" {
				queue = append(queue, dependencyKey)
			}
		}
		for dependencyName, dependencyVersion := range node.OptionalDependencies {
			if dependencyKey := l.resolveKey(dependencyName, dependencyVersion); dependencyKey != "" {
				queue = append(queue, dependencyKey)
			}
		}
	}

	return result
}

// PackageClosure returns package names reachable from every resolved node with
// one of the supplied names. It supports lockfile-only security updates where
// the target is not a direct manifest dependency.
func (l *Lockfile) PackageClosure(roots []string) map[string]struct{} {
	wanted := make(map[string]struct{}, len(roots))
	result := make(map[string]struct{}, len(roots))
	for _, name := range roots {
		wanted[name] = struct{}{}
		result[name] = struct{}{}
	}

	queue := make([]string, 0)
	for key := range l.Snapshots {
		name, _, ok := SplitPackageKey(key)
		if _, wantedName := wanted[name]; ok && wantedName {
			queue = append(queue, key)
		}
	}
	if len(l.Snapshots) == 0 {
		for key := range l.Packages {
			name, _, ok := SplitPackageKey(key)
			if _, wantedName := wanted[name]; ok && wantedName {
				queue = append(queue, key)
			}
		}
	}

	visited := make(map[string]struct{})
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := visited[key]; ok {
			continue
		}
		visited[key] = struct{}{}
		name, _, ok := SplitPackageKey(key)
		if ok {
			result[name] = struct{}{}
		}
		node, ok := l.Snapshots[key]
		if !ok {
			node, ok = l.Packages[key]
		}
		if !ok {
			continue
		}
		for dependencyName, dependencyVersion := range node.Dependencies {
			if dependencyKey := l.resolveKey(dependencyName, dependencyVersion); dependencyKey != "" {
				queue = append(queue, dependencyKey)
			}
		}
		for dependencyName, dependencyVersion := range node.OptionalDependencies {
			if dependencyKey := l.resolveKey(dependencyName, dependencyVersion); dependencyKey != "" {
				queue = append(queue, dependencyKey)
			}
		}
	}
	return result
}

func (l *Lockfile) resolveKey(name, dependencyVersion string) string {
	if isLocalReference(dependencyVersion) {
		return ""
	}

	candidate := name + "@" + dependencyVersion
	if _, ok := l.Snapshots[candidate]; ok {
		return candidate
	}
	if _, ok := l.Packages[candidate]; ok {
		return candidate
	}

	version := StripPeerContext(dependencyVersion)
	candidate = name + "@" + version
	if _, ok := l.Snapshots[candidate]; ok {
		return candidate
	}
	if _, ok := l.Packages[candidate]; ok {
		return candidate
	}

	prefix := candidate + "("
	for key := range l.Snapshots {
		if strings.HasPrefix(key, prefix) {
			return key
		}
	}
	for key := range l.Packages {
		if strings.HasPrefix(key, prefix) {
			return key
		}
	}
	return ""
}

func isLocalReference(value string) bool {
	for _, prefix := range []string{"link:", "workspace:", "file:", "portal:", "github:", "git+", "http:", "https:"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

// StripPeerContext removes pnpm's parenthesized peer-dependency suffix.
func StripPeerContext(value string) string {
	if index := strings.IndexByte(value, '('); index >= 0 {
		return value[:index]
	}
	return value
}

// SplitPackageKey separates a pnpm package or snapshot key into package name
// and resolved version.
func SplitPackageKey(key string) (string, string, bool) {
	withoutPeers := StripPeerContext(key)
	separator := strings.LastIndex(withoutPeers, "@")
	if separator <= 0 || separator == len(withoutPeers)-1 {
		return "", "", false
	}
	return withoutPeers[:separator], withoutPeers[separator+1:], true
}
