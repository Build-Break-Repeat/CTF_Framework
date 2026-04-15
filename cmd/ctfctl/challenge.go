package main

import (
	"encoding/json"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func generateId(s string) string {
	s = strings.ToLower(s)
	result := ""
	prevHyphen := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result += string(c)
			prevHyphen = false
		} else {
			if !prevHyphen && len(result) > 0 {
				result += "-"
				prevHyphen = true
			}
		}
	}

	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}

	return result
}

func generateID(input string, name string) string {
	if input != "" {
		return input
	}

	defaultID := generateId(name)

	id := promptField("ID", defaultID)
	if id == "" {
		return defaultID
	}

	return id
}

func parsePort(s string) (port, error) {
	parts := strings.Split(s, ":")

	if len(parts) != 2 {
		return port{}, fmt.Errorf("invalid port format %q", s)
	}

	internal, err := strconv.Atoi(parts[0])
	if err != nil {
		return port{}, errors.New("invalid internal port")
	}

	external, err := strconv.Atoi(parts[1])
	if err != nil {
		return port{}, errors.New("invalid external port")
	}

	p := port{}
	p.Internal = internal
	p.External = external

	return p, nil
}

func parsePorts(inputs []string) ([]port, error) {
	var result []port

	for i := 0; i < len(inputs); i++ {
		parsed, err := parsePort(inputs[i])
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}

	return result, nil
}

func promptPorts() []port {
	var result []port

	fmt.Println("Ports (internal:external):")

	for {
		raw := promptField("Port", "")
		if raw == "" {
			break
		}

		parsed, err := parsePort(raw)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}

		result = append(result, parsed)
	}

	return result
}

func resolvePorts(inputs []string) ([]port, error) {
	if len(inputs) > 0 {
		return parsePorts(inputs)
	}
	return promptPorts(), nil
}

func loadChallengeConfig() (challengeConfig, error) {
	var cfg challengeConfig

	data, err := os.ReadFile(challengeFile)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func saveChallengeConfig(cfg challengeConfig) error {
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(challengeFile, out, 0644)
	if err != nil {
		return err
	}

	return nil
}

// -----------------------
// Subcommands
// -----------------------

func challengeList() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]
		fmt.Println(c.ID, "|", c.Name, "|", c.Points, "pts")
	}

	return nil
}

func challengeAdd(args []string) error {
	fs := goflag.NewFlagSet("challenge add", goflag.ContinueOnError)

	fsId := fs.String("id", "", "")
	fsName := fs.String("name", "", "")
	fsDescription := fs.String("description", "", "")
	fsCategory := fs.String("category", "", "")
	fsPoints := fs.Int("points", 0, "")
	fsImage := fs.String("image", "", "")
	fsMemory := fs.Int("memory", 0, "")
	fsFlagPath := fs.String("flag-path", "", "")

	var fsPorts []string
	fs.Func("port", "", func(v string) error {
		fsPorts = append(fsPorts, v)
		return nil
	})

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	c := challenge{}

	name, err := getRequiredString(*fsName, "Name")
	if err != nil {
		return err
	}
	c.Name = name

	c.ID = generateID(*fsId, c.Name)

	c.Description = getOptionalString(*fsDescription, "Description")
	c.Category = getOptionalString(*fsCategory, "Category")

	points, err := getRequiredInt(*fsPoints, "Points")
	if err != nil {
		return err
	}
	c.Points = points

	image, err := getRequiredString(*fsImage, "Docker image")
	if err != nil {
		return err
	}
	c.Image = image

	c.Memory = *fsMemory
	if c.Memory == 0 {
		c.Memory = promptInt("Memory (MB)", 256)
	}

	flagPath := *fsFlagPath
	if flagPath == "" {
		flagPath = promptField("Flag path", "/flag.txt")
	}

	if flagPath != "" {
		f := challengeFlag{}
		f.Type = "file"
		f.Path = flagPath
		f.Owner = "root"
		f.Permissions = "0600"
		c.Flag = &f
	}

	ports, err := resolvePorts(fsPorts)
	if err != nil {
		return err
	}
	c.Ports = ports

	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	for i := 0; i < len(cfg.Challenges); i++ {
		if cfg.Challenges[i].ID == c.ID {
			return errors.New("challenge already exists")
		}
	}

	cfg.Challenges = append(cfg.Challenges, c)

	err = saveChallengeConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Println("Added challenge:", c.Name, c.ID)

	return nil
}

func challengeRemove(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ctfctl challenge remove <id>")
	}

	id := args[0]

	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	index := -1
	for i := 0; i < len(cfg.Challenges); i++ {
		if cfg.Challenges[i].ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		fmt.Println("No challenge found with ID:", id)
		return nil
	}

	found := cfg.Challenges[index]
	fmt.Println("Remove challenge:", found.Name, "("+found.ID+")?", "[y/N]")

	line, _ := stdinReader.ReadString('\n')
	answer := strings.TrimSpace(line)

	if answer != "y" && answer != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	updated := []challenge{}
	for i := 0; i < len(cfg.Challenges); i++ {
		if cfg.Challenges[i].ID != id {
			updated = append(updated, cfg.Challenges[i])
		}
	}
	cfg.Challenges = updated

	err = saveChallengeConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Println("Removed challenge:", found.Name)
	return nil
}

func challengeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("challenge subcommand required (list, add, remove)")
	}

	sub := args[0]

	if sub == "list" {
		return challengeList()
	}

	if sub == "add" {
		return challengeAdd(args[1:])
	}

	if sub == "remove" {
		return challengeRemove(args[1:])
	}

	return errors.New("unknown challenge subcommand: " + sub)
}
