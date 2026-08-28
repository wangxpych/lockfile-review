package pnpm

import (
	"reflect"
	"testing"
)

const graphFixture = `lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      app:
        specifier: ^2.0.0
        version: 2.0.0(peer@1.0.0)
packages:
  app@2.0.0:
    resolution: {integrity: example}
  child@3.0.0: {}
  optional-child@4.0.0: {}
snapshots:
  app@2.0.0(peer@1.0.0):
    dependencies:
      child: 3.0.0
    optionalDependencies:
      optional-child: 4.0.0
  child@3.0.0: {}
  optional-child@4.0.0: {}
`

func TestParseAndDependencyClosure(t *testing.T) {
	t.Parallel()
	lockfile, err := Parse([]byte(graphFixture))
	if err != nil {
		t.Fatal(err)
	}
	closure := lockfile.DependencyClosure(".", []string{"app"})
	for _, name := range []string{"app", "child", "optional-child"} {
		if _, ok := closure[name]; !ok {
			t.Fatalf("closure missing %q: %#v", name, closure)
		}
	}
	if got := lockfile.Direct(".")["app"].Version; got != "2.0.0(peer@1.0.0)" {
		t.Fatalf("direct version = %q", got)
	}
}

func TestVersionsByPackageAndSplitPackageKey(t *testing.T) {
	t.Parallel()
	lockfile, err := Parse([]byte(graphFixture))
	if err != nil {
		t.Fatal(err)
	}
	if got := lockfile.VersionsByPackage()["app"]; !reflect.DeepEqual(got, []string{"2.0.0"}) {
		t.Fatalf("app versions = %#v", got)
	}
	name, version, ok := SplitPackageKey("@scope/tool@1.2.3(peer@4.0.0)")
	if !ok || name != "@scope/tool" || version != "1.2.3" {
		t.Fatalf("SplitPackageKey() = %q, %q, %v", name, version, ok)
	}
	if _, _, ok := SplitPackageKey("invalid"); ok {
		t.Fatal("SplitPackageKey(invalid) succeeded")
	}
}

func TestSemanticEqualityIgnoresFormatting(t *testing.T) {
	t.Parallel()
	left, err := Parse([]byte("lockfileVersion: '9.0'\nimporters: { '.': {} }\npackages: {}\nsnapshots: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	right, err := Parse([]byte("lockfileVersion: \"9.0\"\nimporters:\n  .: {}\npackages: {}\nsnapshots: {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !left.SemanticallyEqual(right) {
		t.Fatal("SemanticallyEqual() = false")
	}
}

func TestParseRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte("lockfileVersion: '8.0'\nimporters: {}\n")); err == nil {
		t.Fatal("Parse() error = nil, want unsupported version")
	}
}

func TestPackageClosureStartsAtTransitivePackage(t *testing.T) {
	t.Parallel()
	lockfile, err := Parse([]byte(graphFixture))
	if err != nil {
		t.Fatal(err)
	}
	closure := lockfile.PackageClosure([]string{"child"})
	if _, ok := closure["child"]; !ok {
		t.Fatalf("closure = %#v", closure)
	}
}
