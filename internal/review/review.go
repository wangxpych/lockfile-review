// Package review classifies the dependency changes between two project states.
package review

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wangxpych/lockfile-review/internal/input"
	"github.com/wangxpych/lockfile-review/internal/manifest"
	"github.com/wangxpych/lockfile-review/internal/pnpm"
)

// Level is the review severity of a finding.
type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

// FindingCode is a stable machine-readable finding identifier.
type FindingCode string

const (
	CodeRequestedChange  FindingCode = "requested-change"
	CodeTransitiveChange FindingCode = "transitive-change"
	CodeUnrelatedChange  FindingCode = "unrelated-lockfile-change"
	CodeDowngrade        FindingCode = "unexpected-downgrade"
	CodeManifestMismatch FindingCode = "manifest-lockfile-mismatch"
	CodeCosmeticChurn    FindingCode = "cosmetic-lockfile-churn"
)

// Finding describes one classified change.
type Finding struct {
	Code    FindingCode `json:"code"`
	Level   Level       `json:"level"`
	Package string      `json:"package,omitempty"`
	Before  []string    `json:"before,omitempty"`
	After   []string    `json:"after,omitempty"`
	Message string      `json:"message"`
}

// Result is the complete deterministic review report.
type Result struct {
	LockfileVersion string    `json:"lockfileVersion"`
	ChangedRoots    []string  `json:"changedRoots"`
	Findings        []Finding `json:"findings"`
}

// Policy controls which warning classes fail the command.
type Policy struct {
	FailOnUnrelated bool
	FailOnDowngrade bool
}

// Options supplies review context that cannot always be inferred from
// package.json. ExpectedPackages is especially useful for lockfile-only updates.
type Options struct {
	ExpectedPackages []string
}

// Failed reports whether the review violates the selected policy. Structural
// manifest/lockfile mismatches always fail.
func (r Result) Failed(policy Policy) bool {
	for _, finding := range r.Findings {
		if finding.Level == LevelError {
			return true
		}
		if finding.Code == CodeUnrelatedChange && policy.FailOnUnrelated {
			return true
		}
		if finding.Code == CodeDowngrade && policy.FailOnDowngrade {
			return true
		}
	}
	return false
}

// Run reviews two project states at the root pnpm importer.
func Run(beforeFiles, afterFiles input.Files) (Result, error) {
	return RunWithOptions(beforeFiles, afterFiles, Options{})
}

// RunWithOptions reviews two project states at the root pnpm importer.
func RunWithOptions(beforeFiles, afterFiles input.Files, options Options) (Result, error) {
	beforeManifest, err := manifest.Parse(beforeFiles.Manifest)
	if err != nil {
		return Result{}, fmt.Errorf("parse base state: %w", err)
	}
	afterManifest, err := manifest.Parse(afterFiles.Manifest)
	if err != nil {
		return Result{}, fmt.Errorf("parse head state: %w", err)
	}
	beforeLockfile, err := pnpm.Parse(beforeFiles.Lockfile)
	if err != nil {
		return Result{}, fmt.Errorf("parse base state: %w", err)
	}
	afterLockfile, err := pnpm.Parse(afterFiles.Lockfile)
	if err != nil {
		return Result{}, fmt.Errorf("parse head state: %w", err)
	}

	result := Result{LockfileVersion: fmt.Sprint(afterLockfile.LockfileVersion)}
	manifestRoots := directChanges(beforeManifest.Direct(), afterManifest.Direct())
	changedRoots := sortedUnique(append(manifestRoots, options.ExpectedPackages...))
	result.ChangedRoots = changedRoots

	if string(beforeFiles.Lockfile) != string(afterFiles.Lockfile) && beforeLockfile.SemanticallyEqual(afterLockfile) {
		result.Findings = append(result.Findings, Finding{
			Code:    CodeCosmeticChurn,
			Level:   LevelWarning,
			Message: "pnpm-lock.yaml changed textually but contains the same YAML data",
		})
	}

	result.Findings = append(result.Findings, validateManifest(afterManifest, afterLockfile)...)

	expected := beforeLockfile.DependencyClosure(".", manifestRoots)
	for name := range afterLockfile.DependencyClosure(".", manifestRoots) {
		expected[name] = struct{}{}
	}
	for name := range beforeLockfile.PackageClosure(options.ExpectedPackages) {
		expected[name] = struct{}{}
	}
	for name := range afterLockfile.PackageClosure(options.ExpectedPackages) {
		expected[name] = struct{}{}
	}

	beforeVersions := beforeLockfile.VersionsByPackage()
	afterVersions := afterLockfile.VersionsByPackage()
	for _, name := range changedPackageNames(beforeVersions, afterVersions) {
		before := beforeVersions[name]
		after := afterVersions[name]
		_, isExpected := expected[name]
		_, isRoot := contains(changedRoots, name)

		code := CodeTransitiveChange
		level := LevelInfo
		message := "transitive dependency changed within an updated dependency graph"
		if isRoot {
			code = CodeRequestedChange
			message = "direct dependency changed in package.json and pnpm-lock.yaml"
		} else if !isExpected {
			code = CodeUnrelatedChange
			level = LevelWarning
			message = "dependency changed outside every updated direct dependency graph"
		}

		result.Findings = append(result.Findings, Finding{
			Code: code, Level: level, Package: name, Before: before, After: after, Message: message,
		})

		if isDowngrade(before, after) {
			result.Findings = append(result.Findings, Finding{
				Code: CodeDowngrade, Level: LevelWarning, Package: name, Before: before, After: after,
				Message: "highest resolved semantic version decreased",
			})
		}
	}

	sort.SliceStable(result.Findings, func(i, j int) bool {
		left, right := result.Findings[i], result.Findings[j]
		if severityRank(left.Level) != severityRank(right.Level) {
			return severityRank(left.Level) > severityRank(right.Level)
		}
		if left.Package != right.Package {
			return left.Package < right.Package
		}
		return left.Code < right.Code
	})
	return result, nil
}

func validateManifest(head *manifest.Manifest, lockfile *pnpm.Lockfile) []Finding {
	direct := lockfile.Direct(".")
	if direct == nil {
		return []Finding{{
			Code: CodeManifestMismatch, Level: LevelError,
			Message: "pnpm-lock.yaml does not contain the root importer '.'",
		}}
	}

	var findings []Finding
	for name, declaration := range head.Direct() {
		if declaration.Scope == manifest.ScopePeer {
			continue
		}
		locked, ok := direct[name]
		if !ok {
			findings = append(findings, Finding{
				Code: CodeManifestMismatch, Level: LevelError, Package: name,
				After: []string{declaration.Range}, Message: "dependency is declared in package.json but missing from the root lockfile importer",
			})
			continue
		}
		if locked.Specifier != declaration.Range {
			findings = append(findings, Finding{
				Code: CodeManifestMismatch, Level: LevelError, Package: name,
				Before: []string{declaration.Range}, After: []string{locked.Specifier},
				Message: "package.json range does not match the root lockfile importer specifier",
			})
		}
	}
	manifestDirect := head.Direct()
	for name := range direct {
		declaration, ok := manifestDirect[name]
		if ok && declaration.Scope != manifest.ScopePeer {
			continue
		}
		findings = append(findings, Finding{
			Code: CodeManifestMismatch, Level: LevelError, Package: name,
			Message: "dependency exists in the root lockfile importer but not in package.json",
		})
	}
	return findings
}

func directChanges(before, after map[string]manifest.Dependency) []string {
	names := make(map[string]struct{})
	for name := range before {
		names[name] = struct{}{}
	}
	for name := range after {
		names[name] = struct{}{}
	}

	var changed []string
	for name := range names {
		left, leftOK := before[name]
		right, rightOK := after[name]
		if leftOK != rightOK || left.Range != right.Range || left.Scope != right.Scope {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	return changed
}

func changedPackageNames(before, after map[string][]string) []string {
	names := make(map[string]struct{})
	for name := range before {
		names[name] = struct{}{}
	}
	for name := range after {
		names[name] = struct{}{}
	}

	var result []string
	for name := range names {
		if strings.Join(before[name], "\x00") != strings.Join(after[name], "\x00") {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, wanted string) (int, bool) {
	index := sort.SearchStrings(values, wanted)
	return index, index < len(values) && values[index] == wanted
}

func severityRank(level Level) int {
	switch level {
	case LevelError:
		return 3
	case LevelWarning:
		return 2
	default:
		return 1
	}
}

func isDowngrade(before, after []string) bool {
	beforeMax, beforeOK := maximumSemanticVersion(before)
	afterMax, afterOK := maximumSemanticVersion(after)
	return beforeOK && afterOK && compareSemanticVersions(afterMax, beforeMax) < 0
}

func maximumSemanticVersion(versions []string) (string, bool) {
	var maximum string
	found := false
	for _, version := range versions {
		if _, ok := parseSemanticVersion(version); !ok {
			continue
		}
		if !found || compareSemanticVersions(version, maximum) > 0 {
			maximum = version
			found = true
		}
	}
	return maximum, found
}

func compareSemanticVersions(left, right string) int {
	l, leftOK := parseSemanticVersion(left)
	r, rightOK := parseSemanticVersion(right)
	if !leftOK || !rightOK {
		return strings.Compare(left, right)
	}
	for index := 0; index < 3; index++ {
		if l[index] < r[index] {
			return -1
		}
		if l[index] > r[index] {
			return 1
		}
	}
	return 0
}

func parseSemanticVersion(value string) ([3]int, bool) {
	var result [3]int
	value = strings.TrimPrefix(value, "v")
	if index := strings.IndexAny(value, "-+"); index >= 0 {
		value = value[:index]
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, false
	}
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return result, false
		}
		result[index] = parsed
	}
	return result, true
}
