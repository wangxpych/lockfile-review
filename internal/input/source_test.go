package input

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPathsNormalize(t *testing.T) {
	t.Parallel()
	paths, err := (Paths{}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if paths.ManifestPath != "package.json" || paths.LockfilePath != "pnpm-lock.yaml" {
		t.Fatalf("paths = %#v", paths)
	}
	if _, err := (Paths{Root: "../outside"}).Normalize(); err == nil {
		t.Fatal("parent traversal accepted")
	}
	if _, err := (Paths{ManifestPath: "/tmp/package.json"}).Normalize(); err == nil {
		t.Fatal("absolute path accepted")
	}
}

func TestReadDirectory(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "package.json"), []byte(`{"dependencies":{}}`))
	writeTestFile(t, filepath.Join(directory, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"))
	files, err := ReadDirectory(directory, Paths{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files.Manifest) == 0 || len(files.Lockfile) == 0 {
		t.Fatalf("files = %#v", files)
	}
}

func TestGitHubRefs(t *testing.T) {
	t.Parallel()
	event := filepath.Join(t.TempDir(), "event.json")
	writeTestFile(t, event, []byte(`{"pull_request":{"base":{"sha":"base-sha"},"head":{"sha":"head-sha"}}}`))
	base, head, err := GitHubRefs(event)
	if err != nil {
		t.Fatal(err)
	}
	if base != "base-sha" || head != "head-sha" {
		t.Fatalf("refs = %q, %q", base, head)
	}
	writeTestFile(t, event, []byte(`{}`))
	if _, _, err := GitHubRefs(event); err == nil {
		t.Fatal("event without pull request refs succeeded")
	}
}

func TestReadGit(t *testing.T) {
	t.Parallel()
	repository := t.TempDir()
	runGit(t, repository, "init", "-b", "main")
	runGit(t, repository, "config", "user.name", "Test")
	runGit(t, repository, "config", "user.email", "test@example.com")
	writeTestFile(t, filepath.Join(repository, "package.json"), []byte(`{"dependencies":{}}`))
	writeTestFile(t, filepath.Join(repository, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\nimporters: {'.': {}}\n"))
	runGit(t, repository, "add", "package.json", "pnpm-lock.yaml")
	runGit(t, repository, "commit", "-m", "fixture")
	files, err := ReadGit(repository, "HEAD", Paths{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files.Manifest) == 0 || len(files.Lockfile) == 0 {
		t.Fatalf("files = %#v", files)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", arguments, err, output)
	}
}
