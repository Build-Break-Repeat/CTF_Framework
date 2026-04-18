package main

import (
	"strings"
	"testing"
)

// ---- bold / dim ----

func TestBold_withColor(t *testing.T) {
	old := noColor
	defer func() { noColor = old }()
	noColor = false

	got := bold("hello")
	if !strings.Contains(got, "hello") {
		t.Errorf("bold(\"hello\") = %q; does not contain original string", got)
	}
	if !strings.HasPrefix(got, "\033[") {
		t.Errorf("bold(\"hello\") = %q; expected ANSI escape prefix", got)
	}
}

func TestBold_noColor(t *testing.T) {
	old := noColor
	defer func() { noColor = old }()
	noColor = true

	got := bold("hello")
	if got != "hello" {
		t.Errorf("bold(\"hello\") with noColor = %q; want \"hello\"", got)
	}
}

func TestDim_withColor(t *testing.T) {
	old := noColor
	defer func() { noColor = old }()
	noColor = false

	got := dim("world")
	if !strings.Contains(got, "world") {
		t.Errorf("dim(\"world\") = %q; does not contain original string", got)
	}
	if !strings.HasPrefix(got, "\033[") {
		t.Errorf("dim(\"world\") = %q; expected ANSI escape prefix", got)
	}
}

func TestDim_noColor(t *testing.T) {
	old := noColor
	defer func() { noColor = old }()
	noColor = true

	got := dim("world")
	if got != "world" {
		t.Errorf("dim(\"world\") with noColor = %q; want \"world\"", got)
	}
}

// ---- version constant ----

func TestVersion_nonEmpty(t *testing.T) {
	if version == "" {
		t.Error("version constant is empty")
	}
}

func TestVersion_format(t *testing.T) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Errorf("version %q does not appear to be semver (expected at least N.N)", version)
	}
}
