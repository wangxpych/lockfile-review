package manifest

import "testing"

func TestParseAndDirect(t *testing.T) {
	t.Parallel()
	parsed, err := Parse([]byte(`{
  "dependencies": {"runtime": "^1.0.0"},
  "devDependencies": {"tool": "^2.0.0"},
  "optionalDependencies": {"native": "^3.0.0"},
  "peerDependencies": {"host": "^4.0.0"}
}`))
	if err != nil {
		t.Fatal(err)
	}
	direct := parsed.Direct()
	for name, scope := range map[string]Scope{
		"runtime": ScopeProduction,
		"tool":    ScopeDevelopment,
		"native":  ScopeOptional,
		"host":    ScopePeer,
	} {
		if direct[name].Scope != scope {
			t.Fatalf("%s scope = %q, want %q", name, direct[name].Scope, scope)
		}
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte(`{`)); err == nil {
		t.Fatal("Parse() error = nil, want invalid JSON error")
	}
}
