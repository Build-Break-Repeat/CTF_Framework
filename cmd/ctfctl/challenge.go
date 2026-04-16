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

func promptEnv() []string {
	var result []string

	fmt.Println("Environment variables (KEY=VALUE):")

	for {
		raw := promptField("Env var", "")
		if raw == "" {
			break
		}

		result = append(result, raw)
	}

	return result
}

func resolveEnv(inputs []string) []string {
	if len(inputs) > 0 {
		return inputs
	}
	return promptEnv()
}

func editEnv(current []string) []string {
	result := []string{}
	for i := 0; i < len(current); i++ {
		result = append(result, current[i])
	}

	for {
		fmt.Println("Environment variables:")
		if len(result) == 0 {
			fmt.Println("  (none)")
		}
		for i := 0; i < len(result); i++ {
			fmt.Printf("  %d) %s\n", i+1, result[i])
		}

		fmt.Print("  [a]dd  [r]emove <n>  [done]: ")
		line, _ := stdinReader.ReadString('\n')
		input := strings.TrimSpace(line)

		if input == "" || input == "done" || input == "d" {
			break
		}

		if input == "a" {
			raw := promptField("Env var (KEY=VALUE)", "")
			if raw == "" {
				continue
			}
			result = append(result, raw)
			continue
		}

		if strings.HasPrefix(input, "r ") {
			numStr := strings.TrimPrefix(input, "r ")
			n, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || n < 1 || n > len(result) {
				fmt.Println("  Invalid selection")
				continue
			}
			updated := []string{}
			for i := 0; i < len(result); i++ {
				if i+1 != n {
					updated = append(updated, result[i])
				}
			}
			result = updated
			continue
		}

		fmt.Println("  Unknown option")
	}

	return result
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

func challengeShow(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ctfctl challenge show <id>")
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
		return errors.New("no challenge found with ID: " + id)
	}

	c := cfg.Challenges[index]

	fmt.Println("ID:         ", c.ID)
	fmt.Println("Name:       ", c.Name)
	fmt.Println("Description:", c.Description)
	fmt.Println("Category:   ", c.Category)
	fmt.Println("Points:     ", strconv.Itoa(c.Points))
	fmt.Println("Image:      ", c.Image)
	fmt.Println("Memory:     ", strconv.Itoa(c.Memory)+"MB")

	if c.Flag != nil {
		fmt.Println("Flag path:  ", c.Flag.Path)
	}

	fmt.Println("Ports:")
	if len(c.Ports) == 0 {
		fmt.Println("  (none)")
	}
	for i := 0; i < len(c.Ports); i++ {
		p := c.Ports[i]
		fmt.Println(" ", strconv.Itoa(p.Internal), "->", strconv.Itoa(p.External))
	}

	return nil
}

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
	fsPath := fs.String("path", "", "")

	var fsPorts []string
	fs.Func("port", "", func(v string) error {
		fsPorts = append(fsPorts, v)
		return nil
	})

	var fsEnvs []string
	fs.Func("env", "", func(v string) error {
		fsEnvs = append(fsEnvs, v)
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

	c.Path = getOptionalString(*fsPath, "URL path")
	c.Environment = resolveEnv(fsEnvs)

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

func editPorts(current []port) []port {
	result := []port{}
	for i := 0; i < len(current); i++ {
		result = append(result, current[i])
	}

	for {
		fmt.Println("Ports:")
		if len(result) == 0 {
			fmt.Println("  (none)")
		}
		for i := 0; i < len(result); i++ {
			fmt.Printf("  %d) %d:%d\n", i+1, result[i].Internal, result[i].External)
		}

		fmt.Print("  [a]dd  [r]emove <n>  [done]: ")
		line, _ := stdinReader.ReadString('\n')
		input := strings.TrimSpace(line)

		if input == "" || input == "done" || input == "d" {
			break
		}

		if input == "a" {
			raw := promptField("Port (internal:external)", "")
			if raw == "" {
				continue
			}
			p, err := parsePort(raw)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			result = append(result, p)
			continue
		}

		if strings.HasPrefix(input, "r ") {
			numStr := strings.TrimPrefix(input, "r ")
			n, err := strconv.Atoi(strings.TrimSpace(numStr))
			if err != nil || n < 1 || n > len(result) {
				fmt.Println("  Invalid selection")
				continue
			}
			updated := []port{}
			for i := 0; i < len(result); i++ {
				if i+1 != n {
					updated = append(updated, result[i])
				}
			}
			result = updated
			continue
		}

		fmt.Println("  Unknown option")
	}

	return result
}

func challengeEdit(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ctfctl challenge edit <id> [flags]")
	}

	id := args[0]

	fs := goflag.NewFlagSet("challenge edit", goflag.ContinueOnError)

	fsName := fs.String("name", "", "")
	fsDescription := fs.String("description", "", "")
	fsCategory := fs.String("category", "", "")
	fsPoints := fs.Int("points", 0, "")
	fsImage := fs.String("image", "", "")
	fsMemory := fs.Int("memory", 0, "")
	fsFlagPath := fs.String("flag-path", "", "")
	fsPath := fs.String("path", "", "")

	var fsPorts []string
	fs.Func("port", "", func(v string) error {
		fsPorts = append(fsPorts, v)
		return nil
	})

	var fsEnvs []string
	fs.Func("env", "", func(v string) error {
		fsEnvs = append(fsEnvs, v)
		return nil
	})

	err := fs.Parse(args[1:])
	if err != nil {
		return err
	}

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
		return errors.New("no challenge found with ID: " + id)
	}

	c := cfg.Challenges[index]

	if *fsName != "" {
		c.Name = *fsName
	} else {
		c.Name = promptField("Name", c.Name)
	}

	if *fsDescription != "" {
		c.Description = *fsDescription
	} else {
		c.Description = promptField("Description", c.Description)
	}

	if *fsCategory != "" {
		c.Category = *fsCategory
	} else {
		c.Category = promptField("Category", c.Category)
	}

	if *fsPoints != 0 {
		c.Points = *fsPoints
	} else {
		c.Points = promptInt("Points", c.Points)
	}

	if *fsImage != "" {
		c.Image = *fsImage
	} else {
		c.Image = promptField("Docker image", c.Image)
	}

	if *fsMemory != 0 {
		c.Memory = *fsMemory
	} else {
		c.Memory = promptInt("Memory (MB)", c.Memory)
	}

	existingFlagPath := ""
	if c.Flag != nil {
		existingFlagPath = c.Flag.Path
	}

	flagPath := ""
	if *fsFlagPath != "" {
		flagPath = *fsFlagPath
	} else {
		flagPath = promptField("Flag path", existingFlagPath)
	}

	if flagPath != "" {
		if c.Flag == nil {
			f := challengeFlag{}
			f.Type = "file"
			f.Owner = "root"
			f.Permissions = "0600"
			c.Flag = &f
		}
		c.Flag.Path = flagPath
	}

	if len(fsPorts) > 0 {
		ports, err := parsePorts(fsPorts)
		if err != nil {
			return err
		}
		c.Ports = ports
	} else {
		c.Ports = editPorts(c.Ports)
	}

	if *fsPath != "" {
		c.Path = *fsPath
	} else {
		c.Path = promptField("URL path", c.Path)
	}

	if len(fsEnvs) > 0 {
		c.Environment = fsEnvs
	} else {
		c.Environment = editEnv(c.Environment)
	}

	cfg.Challenges[index] = c

	err = saveChallengeConfig(cfg)
	if err != nil {
		return err
	}

	fmt.Println("Updated challenge:", c.Name)
	return nil
}

func challengeReset(args []string) error {
	// No ID — reset all challenges
	if len(args) == 0 {
		err := runScript("scripts/terraform_destroy_challenges.sh", scriptFlags()...)
		if err != nil {
			return err
		}
		return runScript("scripts/terraform_deploy.sh", scriptFlags()...)
	}

	// Specific challenge by ID
	id := args[0]

	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	found := false
	for i := 0; i < len(cfg.Challenges); i++ {
		if cfg.Challenges[i].ID == id {
			found = true
			break
		}
	}

	if !found {
		fmt.Println("No challenge found with ID:", id)
		return nil
	}

	target := "module.challenges.docker_container.challenge_containers[\"" + id + "\"]"

	err = runCommand("sudo", "terraform", "-chdir=terraform/challenges", "destroy", "-auto-approve", "-target="+target)
	if err != nil {
		return err
	}

	return runCommand("sudo", "terraform", "-chdir=terraform/challenges", "apply", "-auto-approve", "-target="+target)
}

func challengeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("challenge subcommand required (list, add, remove, reset, edit)")
	}

	sub := args[0]

	if sub == "list" {
		return challengeList()
	}

	if sub == "show" {
		return challengeShow(args[1:])
	}

	if sub == "add" {
		return challengeAdd(args[1:])
	}

	if sub == "remove" {
		return challengeRemove(args[1:])
	}

	if sub == "reset" {
		return challengeReset(args[1:])
	}

	if sub == "edit" {
		return challengeEdit(args[1:])
	}

	return errors.New("unknown challenge subcommand: " + sub)
}
