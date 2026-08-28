package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wangxpych/lockfile-review/internal/review"
)

func TestRenderFormats(t *testing.T) {
	t.Parallel()
	result := review.Result{
		LockfileVersion: "9.0",
		ChangedRoots:    []string{"typescript"},
		Findings: []review.Finding{{
			Code: review.CodeDowngrade, Level: review.LevelWarning, Package: "dependency",
			Before: []string{"2.0.0"}, After: []string{"1.0.0"}, Message: "version decreased",
		}},
	}
	policy := review.Policy{FailOnDowngrade: true}

	text, err := Render(result, policy, FormatText)
	if err != nil || !strings.Contains(string(text), "lockfile-review: failed") {
		t.Fatalf("text = %q, err = %v", text, err)
	}
	markdown, err := Render(result, policy, FormatMarkdown)
	if err != nil || !strings.Contains(string(markdown), "❌") || !strings.Contains(string(markdown), "unexpected-downgrade") {
		t.Fatalf("markdown = %q, err = %v", markdown, err)
	}
	data, err := Render(result, policy, FormatJSON)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["status"] != "failed" {
		t.Fatalf("status = %#v", decoded["status"])
	}
}

func TestRenderRejectsUnknownFormat(t *testing.T) {
	t.Parallel()
	if _, err := Render(review.Result{}, review.Policy{}, Format("xml")); err == nil {
		t.Fatal("Render(xml) error = nil")
	}
}
