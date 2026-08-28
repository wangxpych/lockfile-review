// Package report renders review results for humans and automation.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wangxpych/lockfile-review/internal/review"
)

// Format selects an output representation.
type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
)

// Render serializes a review result.
func Render(result review.Result, policy review.Policy, format Format) ([]byte, error) {
	switch format {
	case FormatText:
		return renderText(result, policy), nil
	case FormatMarkdown:
		return renderMarkdown(result, policy), nil
	case FormatJSON:
		data, err := json.MarshalIndent(struct {
			Status string        `json:"status"`
			Policy review.Policy `json:"policy"`
			review.Result
		}{Status: status(result, policy), Policy: policy, Result: result}, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("render JSON report: %w", err)
		}
		return append(data, '\n'), nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}

func status(result review.Result, policy review.Policy) string {
	if result.Failed(policy) {
		return "failed"
	}
	return "passed"
}

func renderText(result review.Result, policy review.Policy) []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "lockfile-review: %s\n", status(result, policy))
	fmt.Fprintf(&buffer, "pnpm lockfile version: %s\n", result.LockfileVersion)
	if len(result.ChangedRoots) == 0 {
		buffer.WriteString("changed direct dependencies: none\n")
	} else {
		fmt.Fprintf(&buffer, "changed direct dependencies: %s\n", strings.Join(result.ChangedRoots, ", "))
	}

	if len(result.Findings) == 0 {
		buffer.WriteString("findings: none\n")
		return buffer.Bytes()
	}
	buffer.WriteString("findings:\n")
	for _, finding := range result.Findings {
		packageName := ""
		if finding.Package != "" {
			packageName = " " + finding.Package
		}
		fmt.Fprintf(&buffer, "- [%s] %s%s: %s", finding.Level, finding.Code, packageName, finding.Message)
		if len(finding.Before) > 0 || len(finding.After) > 0 {
			fmt.Fprintf(&buffer, " (%s -> %s)", versions(finding.Before), versions(finding.After))
		}
		buffer.WriteByte('\n')
	}
	return buffer.Bytes()
}

func renderMarkdown(result review.Result, policy review.Policy) []byte {
	var buffer bytes.Buffer
	icon := "✅"
	if result.Failed(policy) {
		icon = "❌"
	}
	reviewStatus := status(result, policy)
	fmt.Fprintf(&buffer, "## %s Lockfile Review: %s%s\n\n", icon, strings.ToUpper(reviewStatus[:1]), reviewStatus[1:])
	fmt.Fprintf(&buffer, "- pnpm lockfile version: `%s`\n", result.LockfileVersion)
	if len(result.ChangedRoots) == 0 {
		buffer.WriteString("- Changed direct dependencies: none\n")
	} else {
		fmt.Fprintf(&buffer, "- Changed direct dependencies: `%s`\n", strings.Join(result.ChangedRoots, "`, `"))
	}

	if len(result.Findings) == 0 {
		buffer.WriteString("\nNo dependency changes were detected.\n")
		return buffer.Bytes()
	}

	buffer.WriteString("\n| Level | Finding | Package | Before | After | Detail |\n")
	buffer.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, finding := range result.Findings {
		fmt.Fprintf(&buffer, "| %s | `%s` | %s | %s | %s | %s |\n",
			finding.Level,
			finding.Code,
			markdownCode(finding.Package),
			markdownCode(versions(finding.Before)),
			markdownCode(versions(finding.After)),
			escapeTable(finding.Message),
		)
	}
	return buffer.Bytes()
}

func versions(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(values, ", ")
}

func markdownCode(value string) string {
	if value == "" || value == "—" {
		return "—"
	}
	return "`" + strings.ReplaceAll(value, "`", "\\`") + "`"
}

func escapeTable(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
