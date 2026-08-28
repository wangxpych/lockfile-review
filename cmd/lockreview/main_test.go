package main

import (
	"reflect"
	"testing"
)

func TestParseOptions(t *testing.T) {
	t.Setenv("INPUT_FAIL-ON-UNRELATED", "false")
	t.Setenv("INPUT_EXPECTED-PACKAGES", "one,two")
	options, err := parseOptions([]string{"--format", "json", "--fail-on-downgrade=false"})
	if err != nil {
		t.Fatal(err)
	}
	if options.format != "json" || options.failOnUnrelated || options.failOnDowngrade {
		t.Fatalf("options = %#v", options)
	}
	if got := splitCommaList(options.expectedPackages); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("expected packages = %#v", got)
	}
}

func TestParseOptionsRejectsUnknownFormat(t *testing.T) {
	if _, err := parseOptions([]string{"--format", "xml"}); err == nil {
		t.Fatal("unknown format succeeded")
	}
}

func TestSplitCommaList(t *testing.T) {
	t.Parallel()
	if got := splitCommaList(" one, ,two "); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("splitCommaList() = %#v", got)
	}
}
