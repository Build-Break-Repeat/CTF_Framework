package main

import (
	"bytes"
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

func promptCommand() []string {
	fmt.Println("Command override (leave blank for image default):")
	fmt.Println("  Enter each argument on its own line. Leave blank when done.")

	var result []string
	for {
		raw := promptField("Arg", "")
		if raw == "" {
			break
		}
		result = append(result, raw)
	}
	return result
}

func resolveCommand(inputs []string) []string {
	if len(inputs) > 0 {
		return inputs
	}
	return promptCommand()
}

func editCommand(current []string) []string {
	result := []string{}
	for i := 0; i < len(current); i++ {
		result = append(result, current[i])
	}

	for {
		fmt.Println("Command override:")
		if len(result) == 0 {
			fmt.Println("  (none — image default)")
		}
		for i := 0; i < len(result); i++ {
			fmt.Printf("  %d) %s\n", i+1, result[i])
		}

		fmt.Print("  [a]dd  [r]emove <n>  [c]lear  [done]: ")
		line, _ := stdinReader.ReadString('\n')
		input := strings.TrimSpace(line)

		if input == "" || input == "done" || input == "d" {
			break
		}

		if input == "a" {
			raw := promptField("Arg", "")
			if raw == "" {
				continue
			}
			result = append(result, raw)
			continue
		}

		if input == "c" {
			result = []string{}
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
	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(cfg)
	if err != nil {
		return err
	}

	// json.Encoder.Encode adds a trailing newline — trim it so the file ends cleanly
	out := bytes.TrimRight(buf.Bytes(), "\n")

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

	fmt.Println()
	fmt.Println(bold(c.Name) + " " + dim("("+c.ID+")"))
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("  %-16s %s\n", "Category:", c.Category)
	fmt.Printf("  %-16s %d pts\n", "Points:", c.Points)
	fmt.Printf("  %-16s %s\n", "Description:", c.Description)
	fmt.Printf("  %-16s %s\n", "Image:", c.Image)
	fmt.Printf("  %-16s %dMB\n", "Memory:", c.Memory)

	if len(c.Command) > 0 {
		fmt.Printf("  %-16s %s\n", "Command:", strings.Join(c.Command, " "))
	}

	if c.Flag != nil {
		fmt.Printf("  %-16s %s\n", "Flag type:", c.Flag.Type)
		if c.Flag.Type == "file" || c.Flag.Type == "" {
			fmt.Printf("  %-16s %s\n", "Flag path:", c.Flag.Path)
			fmt.Printf("  %-16s %s\n", "Flag owner:", c.Flag.Owner)
			fmt.Printf("  %-16s %s\n", "Flag perms:", c.Flag.Permissions)
		} else if c.Flag.Type == "sql" {
			fmt.Printf("  %-16s %s\n", "DB engine:", c.Flag.Engine)
			fmt.Printf("  %-16s %s\n", "DB user:", c.Flag.User)
			fmt.Printf("  %-16s %s\n", "DB name:", c.Flag.Database)
			fmt.Printf("  %-16s %s\n", "DB query:", c.Flag.Query)
			fmt.Printf("  %-16s %s\n", "Ready URL:", c.Flag.ReadyURL)
			fmt.Printf("  %-16s %s\n", "Init URL:", c.Flag.InitURL)
		} else if c.Flag.Type == "api" {
			fmt.Printf("  %-16s %s\n", "API URL:", c.Flag.URL)
			fmt.Printf("  %-16s %s\n", "API method:", c.Flag.Method)
			fmt.Printf("  %-16s %s\n", "API body:", c.Flag.Body)
		} else if c.Flag.Type == "env" {
			fmt.Printf("  %-16s %s\n", "Flag:", "injected via environment variable")
		}
	}



	fmt.Printf("  %-16s", "Ports:")
	if len(c.Ports) == 0 {
		fmt.Println(" (none)")
	} else {
		fmt.Println()
		for i := 0; i < len(c.Ports); i++ {
			p := c.Ports[i]
			fmt.Printf("    %d -> %d\n", p.Internal, p.External)
		}
	}

	if len(c.Ports) > 0 {
		url := challengeURL(c, c.Ports[0], getHostName())
		fmt.Printf("  %-16s %s\n", "URL:", url)
	}

	fmt.Println()
	return nil
}

func challengeList() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	if len(cfg.Challenges) == 0 {
		fmt.Println("No challenges configured.")
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-20s %-30s %-10s %s\n", bold("ID"), bold("Name"), bold("Points"), bold("Category"))
	fmt.Println("  " + strings.Repeat("-", 70))

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]
		fmt.Printf("  %-20s %-30s %-10d %s\n", c.ID, c.Name, c.Points, c.Category)
	}

	fmt.Println()
	return nil
}

func promptFlagConfig(existing *challengeFlag) *challengeFlag {
	flagType := ""
	if existing != nil {
		flagType = existing.Type
	}
	flagType = promptField("Flag type (file, sql, api, env, or leave blank for none)", flagType)

	if flagType == "" {
		return nil
	}

	f := challengeFlag{}
	f.Type = flagType

	if flagType == "file" {
		existingPath := "/flag.txt"
		existingOwner := "root"
		existingPerms := "0600"
		if existing != nil && existing.Type == "file" {
			existingPath = existing.Path
			existingOwner = existing.Owner
			existingPerms = existing.Permissions
		}
		f.Path = promptField("Flag file path inside container", existingPath)
		f.Owner = promptField("File owner", existingOwner)
		f.Permissions = promptField("File permissions (octal)", existingPerms)

	} else if flagType == "sql" {
		existingEngine := "mysql"
		existingUser := ""
		existingPassword := ""
		existingDatabase := ""
		existingQuery := ""
		existingReadyURL := ""
		existingReadyContains := ""
		existingInitURL := ""
		existingInitBody := ""
		existingInitTokenField := ""
		if existing != nil && existing.Type == "sql" {
			existingEngine = existing.Engine
			existingUser = existing.User
			existingPassword = existing.Password
			existingDatabase = existing.Database
			existingQuery = existing.Query
			existingReadyURL = existing.ReadyURL
			existingReadyContains = existing.ReadyContains
			existingInitURL = existing.InitURL
			existingInitBody = existing.InitBody
			existingInitTokenField = existing.InitTokenField
		}
		f.Engine = promptField("DB engine (mysql or postgres)", existingEngine)
		f.User = promptField("DB user", existingUser)
		f.Password = promptField("DB password", existingPassword)
		f.Database = promptField("DB name", existingDatabase)
		f.Query = promptField("SQL query (use %s as flag placeholder)", existingQuery)
		f.ReadyURL = promptField("Ready URL (poll before injecting, optional)", existingReadyURL)
		f.ReadyContains = promptField("Ready URL must contain (optional)", existingReadyContains)
		f.InitURL = promptField("Init URL (POST to initialize app, optional)", existingInitURL)
		f.InitBody = promptField("Init POST body (optional)", existingInitBody)
		f.InitTokenField = promptField("Init CSRF token field name (optional)", existingInitTokenField)

	} else if flagType == "api" {
		existingURL := ""
		existingMethod := "POST"
		existingBody := ""
		if existing != nil && existing.Type == "api" {
			existingURL = existing.URL
			existingMethod = existing.Method
			existingBody = existing.Body
		}
		f.URL = promptField("API URL", existingURL)
		f.Method = promptField("HTTP method", existingMethod)
		f.Body = promptField("Request body (use %s as flag placeholder)", existingBody)

		fmt.Println("Headers (Name: Value, leave blank when done):")
		if existing != nil && existing.Type == "api" && len(existing.Headers) > 0 {
			f.Headers = existing.Headers
			for k, v := range f.Headers {
				fmt.Println("  Existing header:", k+": "+v)
			}
		}
		headers := map[string]string{}
		for {
			raw := promptField("Header", "")
			if raw == "" {
				break
			}
			parts := strings.SplitN(raw, ":", 2)
			if len(parts) != 2 {
				fmt.Println("  Invalid format, use Name: Value")
				continue
			}
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		if len(headers) > 0 {
			f.Headers = headers
		}

	} else if flagType == "env" {
		fmt.Println("  (flag will be injected via environment variable — set the variable name in the Environment list)")
	} else {
		fmt.Println("  Unknown flag type:", flagType)
	}

	return &f
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

	var fsCommand []string
	fs.Func("command", "", func(v string) error {
		fsCommand = append(fsCommand, v)
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

	if *fsFlagPath != "" {
		f := challengeFlag{}
		f.Type = "file"
		f.Path = *fsFlagPath
		f.Owner = "root"
		f.Permissions = "0600"
		c.Flag = &f
	} else {
		c.Flag = promptFlagConfig(nil)
	}

	c.Command = resolveCommand(fsCommand)

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
	fs := goflag.NewFlagSet("challenge edit", goflag.ContinueOnError)

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

	var fsCommand []string
	fs.Func("command", "", func(v string) error {
		fsCommand = append(fsCommand, v)
		return nil
	})

	var fsEnvs []string
	fs.Func("env", "", func(v string) error {
		fsEnvs = append(fsEnvs, v)
		return nil
	})

	// If first arg looks like an ID (not a flag), consume it before parsing
	remainingArgs := args
	id := ""
	if len(remainingArgs) > 0 && !strings.HasPrefix(remainingArgs[0], "-") {
		id = remainingArgs[0]
		remainingArgs = remainingArgs[1:]
	}

	err := fs.Parse(remainingArgs)
	if err != nil {
		return err
	}

	if *fsId != "" {
		id = *fsId
	}

	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	if id == "" {
		id, err = pickChallenge(cfg)
		if err != nil {
			return err
		}
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

	if *fsFlagPath != "" {
		// --flag-path CLI flag: shortcut for file-type only
		if c.Flag == nil {
			f := challengeFlag{}
			f.Type = "file"
			f.Owner = "root"
			f.Permissions = "0600"
			c.Flag = &f
		}
		c.Flag.Path = *fsFlagPath
	} else {
		c.Flag = promptFlagConfig(c.Flag)
	}

	if len(fsCommand) > 0 {
		c.Command = fsCommand
	} else {
		c.Command = editCommand(c.Command)
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

func reloadScriptFlags() []string {
	flags := []string{"-n"}
	if autoInstall {
		flags = append(flags, "-a")
	}
	return flags
}

func challengeReload(args []string) error {
	if len(args) == 0 {
		err := runScript("scripts/terraform_destroy_challenges.sh", reloadScriptFlags()...)
		if err != nil {
			return err
		}

		err = flagsEnsure()
		if err != nil {
			return err
		}

		err = runScript("scripts/terraform_deploy.sh", reloadScriptFlags()...)
		if err != nil {
			return err
		}

		return flagsInject()
	}

	// Specific challenge by ID — reload just that one container
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
		return errors.New("no challenge found with ID: " + id)
	}

	target := "module.challenges.docker_container.challenge_containers[\"" + id + "\"]"

	err = runCommand("terraform", "-chdir=terraform/challenges", "init", "-input=false", "-upgrade")
	if err != nil {
		return err
	}

	err = runCommand("sudo", "terraform", "-chdir=terraform/challenges", "destroy", "-auto-approve", "-target="+target)
	if err != nil {
		return err
	}

	err = flagsEnsure()
	if err != nil {
		return err
	}

	err = runCommand("sudo", "terraform", "-chdir=terraform/challenges", "apply", "-auto-approve", "-target="+target)
	if err != nil {
		return err
	}

	return flagsInject()
}

func pickChallenge(cfg challengeConfig) (string, error) {
	if len(cfg.Challenges) == 0 {
		return "", errors.New("no challenges found in config")
	}

	fmt.Println("Select a challenge:")
	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]
		fmt.Printf("  %d) %s (%s)\n", i+1, c.Name, c.ID)
	}

	for {
		fmt.Print("  Enter number: ")
		line, _ := stdinReader.ReadString('\n')
		input := strings.TrimSpace(line)

		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(cfg.Challenges) {
			fmt.Println("  Invalid selection, try again")
			continue
		}

		return cfg.Challenges[n-1].ID, nil
	}
}

func challengePull() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		if c.Image == "" {
			fmt.Println("  Skipping", c.ID, "(no image configured)")
			continue
		}

		fmt.Println("Pulling", c.Image, "...")
		err = runCommand("docker", "pull", c.Image)
		if err != nil {
			fmt.Println("  Warning: failed to pull", c.Image+":", err)
		}
	}

	return nil
}

func challengeHelp() {
	fmt.Println(bold("ctfctl challenge") + " " + dim("(ch)"))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  ctfctl challenge <subcommand> [args]")
	fmt.Println()
	fmt.Println(bold("Subcommands:"))
	fmt.Printf("  %-20s %s\n", "list", "List all challenges")
	fmt.Printf("  %-20s %s\n", "show <id>", "Show details of a challenge")
	fmt.Printf("  %-20s %s\n", "add", "Add a new challenge")
	fmt.Printf("  %-20s %s\n", "edit [id]", "Edit a challenge (select from list if no ID given)")
	fmt.Printf("  %-20s %s\n", "remove <id>", "Remove a challenge")
	fmt.Printf("  %-20s %s\n", "reload [id]", "Destroy and redeploy all (or one) challenge container(s)")
	fmt.Printf("  %-20s %s\n", "pull", "Pull Docker images for all challenges")
	fmt.Printf("  %-20s %s\n", "help", "Show this message")
}

func challengeCommand(args []string) error {
	if len(args) == 0 {
		challengeHelp()
		return errors.New("challenge subcommand required")
	}

	sub := args[0]

	if sub == "help" || sub == "--help" || sub == "-h" {
		challengeHelp()
		return nil
	}

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

	if sub == "reload" {
		return challengeReload(args[1:])
	}

	if sub == "pull" {
		return challengePull()
	}

	if sub == "edit" {
		return challengeEdit(args[1:])
	}

	return errors.New("unknown challenge subcommand: " + sub + " (run 'ctfctl challenge help' for usage)")
}
