package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wangxpych/lockfile-review/internal/input"
	"github.com/wangxpych/lockfile-review/internal/report"
	"github.com/wangxpych/lockfile-review/internal/review"
)

var version = "dev"

type options struct {
	baseRef          string
	headRef          string
	baseDir          string
	headDir          string
	repository       string
	root             string
	manifestPath     string
	lockfilePath     string
	expectedPackages string
	format           string
	summaryFile      string
	failOnUnrelated  bool
	failOnDowngrade  bool
	showVersion      bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "lockreview:", err)
		os.Exit(2)
	}
}

func run(arguments []string) error {
	options, err := parseOptions(arguments)
	if err != nil {
		return err
	}
	if options.showVersion {
		fmt.Println(version)
		return nil
	}

	paths := input.Paths{Root: options.root, ManifestPath: options.manifestPath, LockfilePath: options.lockfilePath}
	base, head, err := loadInputs(options, paths)
	if err != nil {
		return err
	}

	result, err := review.RunWithOptions(base, head, review.Options{ExpectedPackages: splitCommaList(options.expectedPackages)})
	if err != nil {
		return err
	}
	policy := review.Policy{FailOnUnrelated: options.failOnUnrelated, FailOnDowngrade: options.failOnDowngrade}
	data, err := report.Render(result, policy, report.Format(options.format))
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	if options.summaryFile != "" {
		markdown, renderErr := report.Render(result, policy, report.FormatMarkdown)
		if renderErr != nil {
			return renderErr
		}
		file, openErr := os.OpenFile(options.summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if openErr != nil {
			return fmt.Errorf("open summary file: %w", openErr)
		}
		if _, writeErr := file.Write(markdown); writeErr != nil {
			_ = file.Close()
			return fmt.Errorf("write summary file: %w", writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("close summary file: %w", closeErr)
		}
	}

	if result.Failed(policy) {
		os.Exit(1)
	}
	return nil
}

func parseOptions(arguments []string) (options, error) {
	result := options{
		headRef:          "HEAD",
		format:           envOrDefault("INPUT_FORMAT", "text"),
		repository:       envOrDefault("GITHUB_WORKSPACE", "."),
		root:             os.Getenv("INPUT_ROOT"),
		manifestPath:     envOrDefault("INPUT_MANIFEST-PATH", "package.json"),
		lockfilePath:     envOrDefault("INPUT_LOCKFILE-PATH", "pnpm-lock.yaml"),
		expectedPackages: os.Getenv("INPUT_EXPECTED-PACKAGES"),
		summaryFile:      os.Getenv("GITHUB_STEP_SUMMARY"),
		failOnUnrelated:  envBool("INPUT_FAIL-ON-UNRELATED", true),
		failOnDowngrade:  envBool("INPUT_FAIL-ON-DOWNGRADE", true),
	}

	flags := flag.NewFlagSet("lockreview", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&result.baseRef, "base-ref", os.Getenv("INPUT_BASE-REF"), "base Git revision")
	flags.StringVar(&result.headRef, "head-ref", envOrDefault("INPUT_HEAD-REF", result.headRef), "head Git revision")
	flags.StringVar(&result.baseDir, "base-dir", os.Getenv("INPUT_BASE-DIR"), "directory containing the base files")
	flags.StringVar(&result.headDir, "head-dir", os.Getenv("INPUT_HEAD-DIR"), "directory containing the head files")
	flags.StringVar(&result.repository, "repository", result.repository, "Git repository directory")
	flags.StringVar(&result.root, "root", result.root, "project root relative to the repository")
	flags.StringVar(&result.manifestPath, "manifest", result.manifestPath, "manifest path relative to root")
	flags.StringVar(&result.lockfilePath, "lockfile", result.lockfilePath, "lockfile path relative to root")
	flags.StringVar(&result.expectedPackages, "expected-packages", result.expectedPackages, "comma-separated packages expected to change")
	flags.StringVar(&result.format, "format", result.format, "output format: text, markdown, or json")
	flags.StringVar(&result.summaryFile, "summary-file", result.summaryFile, "append a Markdown report to this file")
	flags.BoolVar(&result.failOnUnrelated, "fail-on-unrelated", result.failOnUnrelated, "fail when a dependency outside changed roots is modified")
	flags.BoolVar(&result.failOnDowngrade, "fail-on-downgrade", result.failOnDowngrade, "fail when the highest resolved version decreases")
	flags.BoolVar(&result.showVersion, "version", false, "print the version")
	if err := flags.Parse(arguments); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	for _, allowed := range []string{"text", "markdown", "json"} {
		if result.format == allowed {
			return result, nil
		}
	}
	return options{}, fmt.Errorf("unsupported format %q", result.format)
}

func loadInputs(options options, paths input.Paths) (input.Files, input.Files, error) {
	if options.baseDir != "" || options.headDir != "" {
		if options.baseDir == "" || options.headDir == "" {
			return input.Files{}, input.Files{}, errors.New("both --base-dir and --head-dir are required")
		}
		base, err := input.ReadDirectory(options.baseDir, paths)
		if err != nil {
			return input.Files{}, input.Files{}, err
		}
		head, err := input.ReadDirectory(options.headDir, paths)
		return base, head, err
	}

	baseRef := options.baseRef
	headRef := options.headRef
	if baseRef == "" {
		eventBase, eventHead, err := input.GitHubRefs(os.Getenv("GITHUB_EVENT_PATH"))
		if err == nil {
			baseRef, headRef = eventBase, eventHead
		} else {
			baseRef = "HEAD^"
		}
	}
	repository, err := filepath.Abs(options.repository)
	if err != nil {
		return input.Files{}, input.Files{}, fmt.Errorf("resolve repository: %w", err)
	}
	base, err := input.ReadGit(repository, baseRef, paths)
	if err != nil {
		return input.Files{}, input.Files{}, fmt.Errorf("load base %s: %w", baseRef, err)
	}
	head, err := input.ReadGit(repository, headRef, paths)
	if err != nil {
		return input.Files{}, input.Files{}, fmt.Errorf("load head %s: %w", headRef, err)
	}
	return base, head, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCommaList(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
