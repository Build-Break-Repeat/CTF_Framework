package main

import (
	"strings"
	"testing"
)

// ---- shellQuote ----

func TestShellQuote_noSpecialChars(t *testing.T) {
	got := shellQuote("hello")
	if got != "'hello'" {
		t.Errorf("shellQuote(\"hello\") = %q; want \"'hello'\"", got)
	}
}

func TestShellQuote_withSingleQuote(t *testing.T) {
	got := shellQuote("it's")
	// Single quotes inside should be escaped as: '\''
	if !strings.Contains(got, `'\''`) {
		t.Errorf("shellQuote(\"it's\") = %q; does not contain escaped single quote", got)
	}
	// Should still be wrapped in outer single quotes
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("shellQuote result %q should start and end with single quote", got)
	}
}

func TestShellQuote_emptyString(t *testing.T) {
	got := shellQuote("")
	if got != "''" {
		t.Errorf("shellQuote(\"\") = %q; want \"''\"", got)
	}
}

func TestShellQuote_multipleSpecialChars(t *testing.T) {
	input := "a'b'c"
	got := shellQuote(input)
	// Should escape both single quotes
	if strings.Count(got, `'\''`) != 2 {
		t.Errorf("shellQuote(%q) = %q; expected 2 escaped single quotes", input, got)
	}
}

func TestShellQuote_spacesAndSymbols(t *testing.T) {
	got := shellQuote("hello world $VAR")
	// Spaces and $ should be safe inside single quotes; no escaping needed
	if !strings.Contains(got, "hello world $VAR") {
		t.Errorf("shellQuote(\"hello world $VAR\") = %q; should contain original string", got)
	}
}

// ---- scriptFlags ----

func TestScriptFlags_autoInstallTrue(t *testing.T) {
	old := autoInstall
	defer func() { autoInstall = old }()

	autoInstall = true
	flags := scriptFlags()
	if len(flags) != 1 || flags[0] != "-a" {
		t.Errorf("scriptFlags() with autoInstall=true = %v; want [\"-a\"]", flags)
	}
}

func TestScriptFlags_autoInstallFalse(t *testing.T) {
	old := autoInstall
	defer func() { autoInstall = old }()

	autoInstall = false
	flags := scriptFlags()
	if len(flags) != 0 {
		t.Errorf("scriptFlags() with autoInstall=false = %v; want []", flags)
	}
}

// ---- getHostIP ----

func TestGetHostIP_returnsString(t *testing.T) {
	// Just verify it returns something non-empty (either a real IP or "localhost")
	ip := getHostIP()
	if ip == "" {
		t.Error("getHostIP() returned empty string")
	}
}

// ---- printChallengeURLs ----

func TestPrintChallengeURLs_SSHPort(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:    "metasploitable",
				Name:  "Metasploitable",
				Ports: []port{{Internal: 22, External: 22}},
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		err := printChallengeURLs()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPrintChallengeURLs_WebPort(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:    "webchall",
				Name:  "Web Challenge",
				Ports: []port{{Internal: 80, External: 8080}},
				Path:  "/login",
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		err := printChallengeURLs()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPrintChallengeURLs_NoPorts(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "noport", Name: "No Port Challenge"},
		},
	}
	withChallengeFile(t, cfg, func() {
		err := printChallengeURLs()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPrintChallengeURLs_PathWithLeadingSlash(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:    "ch",
				Name:  "Ch",
				Ports: []port{{Internal: 80, External: 8080}},
				Path:  "/admin",
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := printChallengeURLs(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPrintChallengeURLs_PathWithoutLeadingSlash(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:    "ch",
				Name:  "Ch",
				Ports: []port{{Internal: 80, External: 8080}},
				Path:  "admin",
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := printChallengeURLs(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPrintChallengeURLs_Port2222IsSSH(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:    "ssh-ch",
				Name:  "SSH Challenge",
				Ports: []port{{Internal: 22, External: 2222}},
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := printChallengeURLs(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
