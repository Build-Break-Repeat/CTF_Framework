package main

import (
	"testing"
)

// ---- eventShow ----

func TestEventShow_succeeds(t *testing.T) {
	cfg := challengeConfig{
		Event: eventConfig{
			Name:        "Spring 2026 CTF",
			MaxTeamSize: 3,
			FlagPrefix:  "bbr",
			SecretSeed:  "supersecret",
			Admin:       eventAdmin{Username: "admin", Password: "pass"},
		},
	}
	withChallengeFile(t, cfg, func() {
		if err := eventShow(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestEventShow_emptyEvent(t *testing.T) {
	cfg := challengeConfig{}
	withChallengeFile(t, cfg, func() {
		if err := eventShow(); err != nil {
			t.Errorf("unexpected error for empty event: %v", err)
		}
	})
}

func TestEventShow_fileNotFound(t *testing.T) {
	old := challengeFile
	defer func() { challengeFile = old }()
	challengeFile = "/nonexistent/config.json"

	err := eventShow()
	if err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}

// ---- eventCommand dispatcher ----

func TestEventCommand_noArgs(t *testing.T) {
	err := eventCommand([]string{})
	if err == nil {
		t.Error("expected error for no args")
	}
}

func TestEventCommand_unknownSubcommand(t *testing.T) {
	err := eventCommand([]string{"bogus"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestEventCommand_show(t *testing.T) {
	cfg := challengeConfig{
		Event: eventConfig{Name: "Test Event", Admin: eventAdmin{Username: "admin"}},
	}
	withChallengeFile(t, cfg, func() {
		if err := eventCommand([]string{"show"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---- eventEdit via CLI flags (non-interactive path) ----

func TestEventEdit_allFlags(t *testing.T) {
	cfg := challengeConfig{
		Event: eventConfig{
			Name:        "Old Event",
			MaxTeamSize: 1,
			FlagPrefix:  "old",
			SecretSeed:  "oldseed",
			Admin:       eventAdmin{Username: "oldadmin", Password: "oldpass"},
		},
	}
	withChallengeFile(t, cfg, func() {
		args := []string{
			"--name", "New Event",
			"--max-team-size", "4",
			"--flag-prefix", "new",
			"--seed", "newseed",
			"--admin", "newadmin",
			"--password", "newpass",
		}
		if err := eventEdit(args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reloaded, err := loadChallengeConfig()
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.Event.Name != "New Event" {
			t.Errorf("Name = %q; want \"New Event\"", reloaded.Event.Name)
		}
		if reloaded.Event.MaxTeamSize != 4 {
			t.Errorf("MaxTeamSize = %d; want 4", reloaded.Event.MaxTeamSize)
		}
		if reloaded.Event.FlagPrefix != "new" {
			t.Errorf("FlagPrefix = %q; want \"new\"", reloaded.Event.FlagPrefix)
		}
		if reloaded.Event.SecretSeed != "newseed" {
			t.Errorf("SecretSeed = %q; want \"newseed\"", reloaded.Event.SecretSeed)
		}
		if reloaded.Event.Admin.Username != "newadmin" {
			t.Errorf("Admin.Username = %q; want \"newadmin\"", reloaded.Event.Admin.Username)
		}
		if reloaded.Event.Admin.Password != "newpass" {
			t.Errorf("Admin.Password = %q; want \"newpass\"", reloaded.Event.Admin.Password)
		}
	})
}

func TestEventEdit_fileNotFound(t *testing.T) {
	old := challengeFile
	defer func() { challengeFile = old }()
	challengeFile = "/nonexistent/config.json"

	err := eventEdit([]string{"--name", "test"})
	if err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}
