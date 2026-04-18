package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

	for _, c := range cases {
		got := generateId(c.input)
		if got != c.want {
			t.Errorf("generateId(%q) = %q; want %q", c.input, got, c.want)
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

func writeTempConfig(t *testing.T, dir string, cfg challengeConfig) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func withChallengeFile(t *testing.T, cfg challengeConfig, fn func()) {
	t.Helper()
	dir := t.TempDir()
	old := challengeFile
	t.Cleanup(func() { challengeFile = old })
	challengeFile = writeTempConfig(t, dir, cfg)
	fn()
}

func TestLoadChallengeConfig_valid(t *testing.T) {
	want := challengeConfig{
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

	withChallengeFile(t, want, func() {
		got, err := loadChallengeConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Event.Name != want.Event.Name {
			t.Errorf("Event.Name = %q; want %q", got.Event.Name, want.Event.Name)
		}
		if len(got.Challenges) != 1 || got.Challenges[0].ID != "sqli" {
			t.Errorf("unexpected challenges: %+v", got.Challenges)
		}
	})
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
	if err := os.WriteFile(path, []byte("{invalid json}"), 0644); err != nil {
		t.Fatal(err)
	}

	old := challengeFile
	defer func() { challengeFile = old }()
	challengeFile = path

	_, err := loadChallengeConfig()
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

	withChallengeFile(t, original, func() {
		original.Challenges = append(original.Challenges, challenge{ID: "ch2", Name: "Challenge Two", Points: 200})
		if err := saveChallengeConfig(original); err != nil {
			t.Fatalf("save: %v", err)
		}

		reloaded, err := loadChallengeConfig()
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if len(reloaded.Challenges) != 2 {
			t.Errorf("expected 2 challenges, got %d", len(reloaded.Challenges))
		}
		if reloaded.Challenges[1].ID != "ch2" {
			t.Errorf("second challenge ID = %q; want \"ch2\"", reloaded.Challenges[1].ID)
		}
	})
}

// ---- challengeList ----

func TestChallengeList_succeeds(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{ID: "web1", Name: "Web One", Points: 100},
			{ID: "web2", Name: "Web Two", Points: 200},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := challengeList(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestChallengeList_empty(t *testing.T) {
	cfg := challengeConfig{}
	withChallengeFile(t, cfg, func() {
		if err := challengeList(); err != nil {
			t.Errorf("unexpected error for empty list: %v", err)
		}
	})
}

// ---- challengeShow ----

func TestChallengeShow_found(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{
			{
				ID:      "test-ch",
				Name:    "Test Challenge",
				Points:  150,
				Image:   "nginx:latest",
				Memory:  256,
				Ports:   []port{{Internal: 80, External: 8080}},
			},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := challengeShow([]string{"test-ch"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestChallengeShow_notFound(t *testing.T) {
	cfg := challengeConfig{
		Challenges: []challenge{{ID: "other"}},
	}
	withChallengeFile(t, cfg, func() {
		err := challengeShow([]string{"missing"})
		if err == nil {
			t.Error("expected error for missing ID, got nil")
		}
	})
}

func TestChallengeShow_noArgs(t *testing.T) {
	cfg := challengeConfig{}
	withChallengeFile(t, cfg, func() {
		err := challengeShow([]string{})
		if err == nil {
			t.Error("expected error for no args, got nil")
		}
	})
}

func TestChallengeShow_withFlagTypes(t *testing.T) {
	flagTypes := []string{"file", "sql", "api", "env"}
	for _, ft := range flagTypes {
		ft := ft
		t.Run(ft, func(t *testing.T) {
			cfg := challengeConfig{
				Challenges: []challenge{
					{
						ID:   "ch-" + ft,
						Name: "Challenge " + ft,
						Flag: &challengeFlag{Type: ft},
					},
				},
			}
			withChallengeFile(t, cfg, func() {
				if err := challengeShow([]string{"ch-" + ft}); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		})
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
