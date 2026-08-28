package review

import (
	"path/filepath"
	"testing"

	"github.com/wangxpych/lockfile-review/internal/input"
)

func TestCleanUpgradeStaysInsideChangedGraph(t *testing.T) {
	t.Parallel()
	result := runFixture(t, "clean")
	if len(result.ChangedRoots) != 1 || result.ChangedRoots[0] != "alpha" {
		t.Fatalf("ChangedRoots = %#v", result.ChangedRoots)
	}
	assertFinding(t, result, CodeRequestedChange, "alpha")
	assertFinding(t, result, CodeTransitiveChange, "transitive-a")
	if result.Failed(Policy{FailOnUnrelated: true, FailOnDowngrade: true}) {
		t.Fatalf("clean result unexpectedly failed: %#v", result.Findings)
	}
}

func TestUnrelatedDowngradesFailStrictPolicy(t *testing.T) {
	t.Parallel()
	result := runFixture(t, "unrelated")
	for _, name := range []string{"@types/node", "rolldown"} {
		assertFinding(t, result, CodeUnrelatedChange, name)
		assertFinding(t, result, CodeDowngrade, name)
	}
	assertFinding(t, result, CodeRequestedChange, "typescript")
	if !result.Failed(Policy{FailOnUnrelated: true, FailOnDowngrade: true}) {
		t.Fatal("strict policy passed, want failure")
	}
	if result.Failed(Policy{}) {
		t.Fatal("advisory policy failed")
	}
}

func TestManifestMismatchAlwaysFails(t *testing.T) {
	t.Parallel()
	result := runFixture(t, "mismatch")
	assertFinding(t, result, CodeManifestMismatch, "example")
	if !result.Failed(Policy{}) {
		t.Fatal("manifest mismatch passed")
	}
}

func TestCosmeticChurnIsReported(t *testing.T) {
	t.Parallel()
	manifestData := []byte(`{"dependencies":{}}`)
	baseLock := []byte("lockfileVersion: '9.0'\nimporters: { '.': {} }\npackages: {}\nsnapshots: {}\n")
	headLock := []byte("lockfileVersion: \"9.0\"\nimporters:\n  .: {}\npackages: {}\nsnapshots: {}\n")
	result, err := Run(
		input.Files{Manifest: manifestData, Lockfile: baseLock},
		input.Files{Manifest: manifestData, Lockfile: headLock},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, result, CodeCosmeticChurn, "")
	if result.Failed(Policy{FailOnUnrelated: true, FailOnDowngrade: true}) {
		t.Fatal("cosmetic churn should not fail")
	}
}

func TestExpectedPackageSupportsLockfileOnlyUpdate(t *testing.T) {
	t.Parallel()
	manifestData := []byte(`{"dependencies":{"app":"^1.0.0"}}`)
	baseLock := []byte(`lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      app: {specifier: ^1.0.0, version: 1.0.0}
packages:
  app@1.0.0: {}
  target@1.0.0: {}
  target-child@1.0.0: {}
snapshots:
  app@1.0.0:
    dependencies: {target: 1.0.0}
  target@1.0.0:
    dependencies: {target-child: 1.0.0}
  target-child@1.0.0: {}
`)
	headLock := []byte(`lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      app: {specifier: ^1.0.0, version: 1.0.0}
packages:
  app@1.0.0: {}
  target@2.0.0: {}
  target-child@2.0.0: {}
snapshots:
  app@1.0.0:
    dependencies: {target: 2.0.0}
  target@2.0.0:
    dependencies: {target-child: 2.0.0}
  target-child@2.0.0: {}
`)
	result, err := RunWithOptions(
		input.Files{Manifest: manifestData, Lockfile: baseLock},
		input.Files{Manifest: manifestData, Lockfile: headLock},
		Options{ExpectedPackages: []string{"target"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, result, CodeRequestedChange, "target")
	assertFinding(t, result, CodeTransitiveChange, "target-child")
	if result.Failed(Policy{FailOnUnrelated: true, FailOnDowngrade: true}) {
		t.Fatalf("lockfile-only expected update failed: %#v", result.Findings)
	}
}

func runFixture(t *testing.T, name string) Result {
	t.Helper()
	paths := input.Paths{}
	base, err := input.ReadDirectory(filepath.Join("testdata", name, "base"), paths)
	if err != nil {
		t.Fatal(err)
	}
	head, err := input.ReadDirectory(filepath.Join("testdata", name, "head"), paths)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(base, head)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertFinding(t *testing.T, result Result, code FindingCode, packageName string) {
	t.Helper()
	for _, finding := range result.Findings {
		if finding.Code == code && finding.Package == packageName {
			return
		}
	}
	t.Fatalf("missing finding %s for %q: %#v", code, packageName, result.Findings)
}
