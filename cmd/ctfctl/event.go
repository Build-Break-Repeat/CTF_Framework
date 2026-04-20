package main

import (
	"errors"
	goflag "flag"
	"fmt"
	"strings"
)

// -----------------------
// Subcommands
// -----------------------

func eventShow() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	e := cfg.Event

	fmt.Println()
	fmt.Println(bold("Event Configuration"))
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  %-16s %s\n", "Name:", e.Name)
	fmt.Printf("  %-16s %d\n", "Max team size:", e.MaxTeamSize)
	fmt.Printf("  %-16s %s\n", "Flag prefix:", e.FlagPrefix)
	fmt.Printf("  %-16s %s\n", "Seed:", e.SecretSeed)
	fmt.Printf("  %-16s %s\n", "Admin:", e.Admin.Username)
	fmt.Println()

	return nil
}

func eventEdit(args []string) error {
	fs := goflag.NewFlagSet("event edit", goflag.ContinueOnError)

	fsName := fs.String("name", "", "")
	fsMaxTeamSize := fs.Int("max-team-size", 0, "")
	fsFlagPrefix := fs.String("flag-prefix", "", "")
	fsSeed := fs.String("seed", "", "")
	fsAdmin := fs.String("admin", "", "")
	fsPassword := fs.String("password", "", "")

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	e := cfg.Event

	if *fsName != "" {
		e.Name = *fsName
	} else {
		e.Name = promptField("Event name", e.Name)
	}

	if *fsMaxTeamSize != 0 {
		e.MaxTeamSize = *fsMaxTeamSize
	} else {
		e.MaxTeamSize = promptInt("Max team size (1 = solo event)", e.MaxTeamSize)
	}

	if *fsFlagPrefix != "" {
		e.FlagPrefix = *fsFlagPrefix
	} else {
		e.FlagPrefix = promptField("Flag prefix", e.FlagPrefix)
	}

	if *fsSeed != "" {
		e.SecretSeed = *fsSeed
	} else {
		e.SecretSeed = promptField("Seed", e.SecretSeed)
	}

	if *fsAdmin != "" {
		e.Admin.Username = *fsAdmin
	} else {
		e.Admin.Username = promptField("Admin username", e.Admin.Username)
	}

	if *fsPassword != "" {
		e.Admin.Password = *fsPassword
	} else {
		e.Admin.Password = promptField("Admin password", e.Admin.Password)
	}

	cfg.Event = e

	err = saveChallengeConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Println("Event updated.")
	return nil
}

func eventHelp() {
	fmt.Println(bold("ctfctl event"))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  ctfctl event <subcommand>")
	fmt.Println()
	fmt.Println(bold("Subcommands:"))
	fmt.Printf("  %-20s %s\n", "show", "Display current event configuration")
	fmt.Printf("  %-20s %s\n", "edit", "Edit event settings (name, team size, flag prefix, admin)")
	fmt.Printf("  %-20s %s\n", "help", "Show this message")
}

func eventCommand(args []string) error {
	if len(args) == 0 {
		eventHelp()
		return errors.New("event subcommand required")
	}

	sub := args[0]

	if sub == "help" || sub == "--help" || sub == "-h" {
		eventHelp()
		return nil
	}

	if sub == "show" {
		return eventShow()
	}

	if sub == "edit" {
		return eventEdit(args[1:])
	}

	return errors.New("unknown event subcommand: " + sub + " (run 'ctfctl event help' for usage)")
}
