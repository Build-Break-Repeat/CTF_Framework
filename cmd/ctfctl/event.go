package main

import (
	"errors"
	goflag "flag"
	"fmt"
	"strconv"
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

	fmt.Println("Event:")
	fmt.Println("  Name:        ", e.Name)
	fmt.Println("  Max team size:", strconv.Itoa(e.MaxTeamSize))
	fmt.Println("  Flag prefix: ", e.FlagPrefix)
	fmt.Println("  Seed:        ", e.SecretSeed)
	fmt.Println("  Admin:       ", e.Admin.Username)

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

func eventCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("event subcommand required (show, edit)")
	}

	sub := args[0]

	if sub == "show" {
		return eventShow()
	}

	if sub == "edit" {
		return eventEdit(args[1:])
	}

	return errors.New("unknown event subcommand: " + sub)
}
