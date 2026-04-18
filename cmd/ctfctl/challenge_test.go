package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupConfig writes cfg to a temporary config.json, points challengeFile at
// it, and returns the previous value of challengeFile so the caller can
// restore it with defer.
func setupConfig(t *testing.T, cfg challengeConfig) string {
	dir := t.TempDir()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("setupConfig: marshal failed: %v", err)
	}

	path := filepath.Join(dir, "config.json")
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("setupConfig: write failed: %v", err)
	}

	old := challengeFile
	challengeFile = path
	return old
}

// ---- generateId ----

func TestGenerateId_basic(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"SQL Injection Lab", "sql-injection-lab"},
		{"Hello World!", "hello-world"},
		{"  leading spaces  ", "leading-spaces"},
		{"multi---hyphens", "multi-hyphens"},
		{"already-lower", "already-lower"},
		{"123 Numbers", "123-numbers"},
		{"ALL CAPS", "all-caps"},
		{"trailing-", "trailing"},
	}

	for i := 0; i < len(cases); i++ {
		got := generateId(cases[i].input)
		if got != cases[i].want {
			t.Errorf("generateId(%q) = %q; want %q", cases[i].input, got, cases[i].want)
		}
	}
}

func TestGenerateId_emptyString(t *testing.T) {
	got := generateId("")
	if got != "" {
		t.Errorf("generateId(\"\") = %q; want \"\"", got)
	}
}

func TestGenerateId_onlySpecialChars(t *testing.T) {
	got := generateId("!!! @@@")
	if got != "" {
		t.Errorf("generateId(\"!!! @@@\") = %q; want \"\"", got)
	}
}

// ---- parsePort ----

func TestParsePort_valid(t *testing.T) {
	p, err := parsePort("80:8080")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Internal != 80 || p.External != 8080 {
		t.Errorf("got %+v; want {Internal:80 External:8080}", p)
	}
}

func TestParsePort_samePort(t *testing.T) {
	p, err := parsePort("443:443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Internal != 443 || p.External != 443 {
		t.Errorf("got %+v", p)
	}
}

func TestParsePort_missingColon(t *testing.T) {
	_, err := parsePort("8080")
	if err == nil {
		t.Error("expected error for missing colon, got nil")
	}
}

func TestParsePort_tooManyParts(t *testing.T) {
	_, err := parsePort("80:8080:extra")
	if err == nil {
		t.Error("expected error for too many parts, got nil")
	}
}

func TestParsePort_nonNumericInternal(t *testing.T) {
	_, err := parsePort("abc:8080")
	if err == nil {
		t.Error("expected error for non-numeric internal port, got nil")
	}
}

func TestParsePort_nonNumericExternal(t *testing.T) {
	_, err := parsePort("80:xyz")
	if err == nil {
		t.Error("expected error for non-numeric external port, got nil")
	}
}

func TestParsePort_emptyString(t *testing.T) {
	_, err := parsePort("")
	if err == nil {
		t.Error("expected error for empty string, got nil")
	}
}

// ---- parsePorts ----

func TestParsePorts_multiple(t *testing.T) {
	ports, err := parsePorts([]string{"80:8080", "22:2222"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if ports[0].Internal != 80 || ports[0].External != 8080 {
		t.Errorf("ports[0] = %+v", ports[0])
	}
	if ports[1].Internal != 22 || ports[1].External != 2222 {
		t.Errorf("ports[1] = %+v", ports[1])
	}
}

func TestParsePorts_empty(t *testing.T) {
	ports, err := parsePorts([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 0 {
		t.Errorf("expected 0 ports, got %d", len(ports))
	}
}

func TestParsePorts_invalidEntry(t *testing.T) {
	_, err := parsePorts([]string{"80:8080", "bad"})
	if err == nil {
		t.Error("expected error for invalid entry, got nil")
	}
}

// ---- loadChallengeConfig / saveChallengeConfig ----

func TestLoadChallengeConfig_valid(t *testing.T) {
	cfg := challengeConfig{
		Event: eventConfig{
			Name:        "Test CTF",
			FlagPrefix:  "test",
			MaxTeamSize: 2,
			Admin:       eventAdmin{Username: "admin", Password: "pass"},
		},
		Challenges: []challenge{
			{ID: "sqli", Name: "SQL Injection", Points: 100},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	got, err := loadChallengeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Event.Name != cfg.Event.Name {
		t.Errorf("Event.Name = %q; want %q", got.Event.Name, cfg.Event.Name)
	}
	if len(got.Challenges) != 1 {
		t.Fatalf("expected 1 challenge, got %d", len(got.Challenges))
	}
	if got.Challenges[0].ID != "sqli" {
		t.Errorf("Challenges[0].ID = %q; want \"sqli\"", got.Challenges[0].ID)
	}
}

func TestLoadChallengeConfig_fileNotFound(t *testing.T) {
	old := challengeFile
	defer func() { challengeFile = old }()
	challengeFile = "/nonexistent/path/config.json"

	_, err := loadChallengeConfig()
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadChallengeConfig_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("could not write test file: %v", err)
	}

	old := challengeFile
	defer func() { challengeFile = old }()
	challengeFile = path

	_, err = loadChallengeConfig()
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestSaveChallengeConfig_roundtrip(t *testing.T) {
	original := challengeConfig{
		Event: eventConfig{
			Name:       "Save Test",
			FlagPrefix: "save",
		},
		Challenges: []challenge{
			{ID: "ch1", Name: "Challenge One", Points: 50},
		},
	}

	old := setupConfig(t, original)
	defer func() { challengeFile = old }()

	original.Challenges = append(original.Challenges, challenge{ID: "ch2", Name: "Challenge Two", Points: 200})

	err := saveChallengeConfig(original)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	reloaded, err := loadChallengeConfig()
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if len(reloaded.Challenges) != 2 {
		t.Errorf("expected 2 challenges, got %d", len(reloaded.Challenges))
	}
	if reloaded.Challenges[1].ID != "ch2" {
		t.Errorf("second challenge ID = %q; want \"ch2\"", reloaded.Challenges[1].ID)
	}
}

// ---- challengeList ----

func TestChallengeList_succeeds(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "web1", Name: "Web One", Points: 100},
			{ID: "web2", Name: "Web Two", Points: 200},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeList()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeList_empty(t *testing.T) {
	cfg := challengeConfig{}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeList()
	if err != nil {
		t.Errorf("unexpected error for empty list: %v", err)
	}
}

// ---- challengeShow ----

func TestChallengeShow_found(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:     "test-ch",
				Name:   "Test Challenge",
				Points: 150,
				Image:  "nginx:latest",
				Memory: 256,
				Ports:  []port{{Internal: 80, External: 8080}},
			},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"test-ch"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeShow_notFound(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{{ID: "other"}},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"missing"})
	if err == nil {
		t.Error("expected error for missing ID, got nil")
	}
}

func TestChallengeShow_noArgs(t *testing.T) {
	cfg := challengeConfig{}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{})
	if err == nil {
		t.Error("expected error for no args, got nil")
	}
}

func TestChallengeShow_fileFlagType(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch-file", Name: "Challenge file", Flag: &challengeFlag{Type: "file"}},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch-file"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeShow_sqlFlagType(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch-sql", Name: "Challenge sql", Flag: &challengeFlag{Type: "sql"}},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch-sql"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeShow_apiFlagType(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch-api", Name: "Challenge api", Flag: &challengeFlag{Type: "api"}},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch-api"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeShow_envFlagType(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch-env", Name: "Challenge env", Flag: &challengeFlag{Type: "env"}},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch-env"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- command field roundtrip ----

func TestCommandField_preservedInRoundtrip(t *testing.T) {
	cmd := []string{"/bin/sh", "-c", "nginx -g 'daemon off;'"}
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch", Name: "Ch", Command: cmd},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := saveChallengeConfig(cfg)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	reloaded, err := loadChallengeConfig()
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	got := reloaded.Challenges[0].Command
	if len(got) != len(cmd) {
		t.Fatalf("command length = %d; want %d", len(got), len(cmd))
	}
	for i := 0; i < len(cmd); i++ {
		if got[i] != cmd[i] {
			t.Errorf("command[%d] = %q; want %q", i, got[i], cmd[i])
		}
	}
}

func TestChallengeShow_displaysCommand(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch", Name: "Ch", Command: []string{"/bin/sh", "-c", "echo hi"}},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestChallengeShow_emptyCommand(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "ch", Name: "Ch"},
		},
	}

	old := setupConfig(t, cfg)
	defer func() { challengeFile = old }()

	err := challengeShow([]string{"ch"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- challengeCommand dispatcher ----

func TestChallengeCommand_noArgs(t *testing.T) {
	err := challengeCommand([]string{})
	if err == nil {
		t.Error("expected error for no args")
	}
}

func TestChallengeCommand_unknownSubcommand(t *testing.T) {
	err := challengeCommand([]string{"bogus"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}
