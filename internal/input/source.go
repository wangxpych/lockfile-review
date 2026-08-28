// Package input loads review inputs from directories, Git revisions, or a
// GitHub pull request event.
package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Files is one side of a lockfile review.
type Files struct {
	Manifest []byte
	Lockfile []byte
}

// Paths identifies project files relative to a directory or Git repository.
type Paths struct {
	Root         string
	ManifestPath string
	LockfilePath string
}

// Normalize fills default paths and validates that paths remain relative.
func (p Paths) Normalize() (Paths, error) {
	if p.ManifestPath == "" {
		p.ManifestPath = "package.json"
	}
	if p.LockfilePath == "" {
		p.LockfilePath = "pnpm-lock.yaml"
	}
	for _, value := range []string{p.Root, p.ManifestPath, p.LockfilePath} {
		if filepath.IsAbs(value) || strings.HasPrefix(filepath.Clean(value), "..") {
			return Paths{}, fmt.Errorf("project paths must remain relative: %q", value)
		}
	}
	return p, nil
}

func (p Paths) manifest() string {
	return filepath.Join(p.Root, p.ManifestPath)
}

func (p Paths) lockfile() string {
	return filepath.Join(p.Root, p.LockfilePath)
}

// ReadDirectory loads package.json and pnpm-lock.yaml from a directory.
func ReadDirectory(directory string, paths Paths) (Files, error) {
	paths, err := paths.Normalize()
	if err != nil {
		return Files{}, err
	}
	manifestData, err := os.ReadFile(filepath.Join(directory, paths.manifest()))
	if err != nil {
		return Files{}, fmt.Errorf("read manifest from %s: %w", directory, err)
	}
	lockfileData, err := os.ReadFile(filepath.Join(directory, paths.lockfile()))
	if err != nil {
		return Files{}, fmt.Errorf("read lockfile from %s: %w", directory, err)
	}
	return Files{Manifest: manifestData, Lockfile: lockfileData}, nil
}

// ReadGit loads package.json and pnpm-lock.yaml from a Git revision. If the
// revision is not available locally, it fetches that single revision from
// origin before retrying.
func ReadGit(repository, revision string, paths Paths) (Files, error) {
	paths, err := paths.Normalize()
	if err != nil {
		return Files{}, err
	}
	if revision == "" {
		return Files{}, errors.New("git revision is required")
	}

	read := func(path string) ([]byte, error) {
		command := exec.Command("git", "-c", "safe.directory="+repository, "show", revision+":"+filepath.ToSlash(path))
		command.Dir = repository
		data, commandErr := command.Output()
		if commandErr == nil {
			return data, nil
		}

		fetch := exec.Command("git", "-c", "safe.directory="+repository, "fetch", "--no-tags", "--depth=1", "origin", revision)
		fetch.Dir = repository
		if output, fetchErr := fetch.CombinedOutput(); fetchErr != nil {
			return nil, fmt.Errorf("read %s from %s and fetch missing revision: %w (%s)", path, revision, fetchErr, strings.TrimSpace(string(output)))
		}

		command = exec.Command("git", "-c", "safe.directory="+repository, "show", revision+":"+filepath.ToSlash(path))
		command.Dir = repository
		data, commandErr = command.Output()
		if commandErr != nil {
			return nil, fmt.Errorf("read %s from %s: %w", path, revision, commandErr)
		}
		return data, nil
	}

	manifestData, err := read(paths.manifest())
	if err != nil {
		return Files{}, err
	}
	lockfileData, err := read(paths.lockfile())
	if err != nil {
		return Files{}, err
	}
	return Files{Manifest: manifestData, Lockfile: lockfileData}, nil
}

// GitHubRefs extracts immutable base and head revisions from a pull request
// event payload.
func GitHubRefs(eventPath string) (base, head string, err error) {
	if eventPath == "" {
		return "", "", errors.New("GitHub event path is empty")
	}
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return "", "", fmt.Errorf("read GitHub event: %w", err)
	}
	var event struct {
		PullRequest struct {
			Base struct {
				SHA string `json:"sha"`
			} `json:"base"`
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return "", "", fmt.Errorf("parse GitHub event: %w", err)
	}
	if event.PullRequest.Base.SHA == "" || event.PullRequest.Head.SHA == "" {
		return "", "", errors.New("GitHub event does not contain pull_request base and head revisions")
	}
	return event.PullRequest.Base.SHA, event.PullRequest.Head.SHA, nil
}
