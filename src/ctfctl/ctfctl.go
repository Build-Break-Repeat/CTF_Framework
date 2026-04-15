package main

import (
	"bufio"
	"encoding/json"
	"errors"
	goflag "flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const version = "1.0.0"
const challengeFile = "challenges.json"

var noColor bool

func bold(s string) string {
	if noColor {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func dim(s string) string {
	if noColor {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

type port struct {
	Internal int `json:"internal"`
	External int `json:"external"`
}

type challengeFlag struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
}

type challenge struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Category    string         `json:"category,omitempty"`
	Points      int            `json:"points"`
	Path        string         `json:"path,omitempty"`
	Image       string         `json:"image,omitempty"`
	Memory      int            `json:"memory,omitempty"`
	Flag        *challengeFlag `json:"flag,omitempty"`
	Ports       []port         `json:"ports,omitempty"`
}

type challengeConfig struct {
	Event      json.RawMessage `json:"event,omitempty"`
	Challenges []challenge     `json:"challenges"`
}

var stdinReader = bufio.NewReader(os.Stdin)

// -----------------------
// Functions
// -----------------------

func promptField(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, _ := stdinReader.ReadString('\n')
	input := strings.TrimSpace(line)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptInt(label string, defaultVal int) int {
	var defStr string
	if defaultVal != 0 {
		defStr = strconv.Itoa(defaultVal)
	}
	raw := promptField(label, defStr)
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Invalid number, using %d\n", defaultVal)
		return defaultVal
	}
	return n
}

func getRequiredString(value string, label string) (string, error) {
	if value == "" {
		value = promptField(label, "")
	}
	if value == "" {
		return "", errors.New(label + " is required")
	}
	return value, nil
}

func getOptionalString(value string, label string) string {
	if value == "" {
		value = promptField(label, "")
	}
	return value
}

func getRequiredInt(value int, label string) (int, error) {
	if value == 0 {
		value = promptInt(label, 0)
	}
	if value == 0 {
		return 0, errors.New(label + " is required")
	}
	return value, nil
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

func parsePorts(inputs []string) ([]port, error) {
	var result []port

	for i := 0; i < len(inputs); i++ {
		p := inputs[i]

		parsed, err := parsePort(p)
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

func parsePort(s string) (port, error) {
	parts := strings.Split(s, ":")

	if len(parts) != 2 {
		return port{}, fmt.Errorf("invalid port format %q", s)
	}

	internal, err := strconv.Atoi(parts[0])
	if err != nil {
		return port{}, fmt.Errorf("invalid internal port")
	}

	external, err := strconv.Atoi(parts[1])
	if err != nil {
		return port{}, fmt.Errorf("invalid external port")
	}

	p := port{}
	p.Internal = internal
	p.External = external

	return p, nil
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
// Challenge commands
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

// -----------------------
// Challenge remove
// -----------------------

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

// -----------------------
// Challenge command router
// -----------------------

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

// -----------------------
// Deploy helpers
// -----------------------

func runCommand(name string, args ...string) error {
	fmt.Println("$", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runScript(script string) error {
	fmt.Println("$ bash", script)
	cmd := exec.Command("bash", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func getHostIP() string {
	out, err := exec.Command("hostname", "-I").Output()
	if err != nil {
		return "localhost"
	}
	parts := strings.Fields(string(out))
	if len(parts) > 0 {
		return parts[0]
	}
	return "localhost"
}

func printChallengeURLs() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	hostIP := getHostIP()

	fmt.Println("")
	fmt.Println("CTF Deployment Complete")
	fmt.Println("")
	fmt.Println("Challenges:")
	fmt.Println("----------------------------------------")

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]
		for j := 0; j < len(c.Ports); j++ {
			p := c.Ports[j]

			if p.External == 22 || p.External == 2222 {
				fmt.Printf("%-30s ssh msfadmin@%s -p %d\n", c.Name, hostIP, p.External)
				continue
			}

			url := "http://" + hostIP + ":" + strconv.Itoa(p.External)

			if c.Path != "" {
				if strings.HasPrefix(c.Path, "/") {
					url = url + c.Path
				} else {
					url = url + "/" + c.Path
				}
			}

			fmt.Printf("%-30s %s\n", c.Name, url)
		}
	}

	fmt.Println("----------------------------------------")
	return nil
}

func bootstrap() error {
	err := runCommand("terraform", "-chdir=terraform/bootstrap", "init", "-input=false", "-upgrade")
	if err != nil {
		return err
	}

	err = runCommand("terraform", "-chdir=terraform/bootstrap", "apply", "-auto-approve")
	if err != nil {
		return err
	}

	err = runCommand("python3", "scripts/ctfd_bootstrap.py")
	if err == nil {
		return nil
	}

	return runCommand("python", "scripts/ctfd_bootstrap.py")
}

// -----------------------
// Help
// -----------------------

func help() {
	fmt.Println(bold("ctfctl") + " " + dim("v"+version))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  ctfctl [--no-color] [--version] <command>")
	fmt.Println()
	fmt.Println(bold("Global flags:"))
	fmt.Printf("  %-30s %s\n", "--no-color", "Disable ANSI color output")
	fmt.Printf("  %-30s %s\n", "--version", "Print version and exit")
	fmt.Println()
	fmt.Println(bold("Commands:"))
	fmt.Printf("  %-30s %s\n", "deploy", "Deploy everything (deps -> bootstrap -> challenges -> firewall)")
	fmt.Printf("  %-30s %s\n", "bootstrap", "Install and configure CTFd (idempotent)")
	fmt.Printf("  %-30s %s\n", "destroy", "Tear down all containers and state")
	fmt.Printf("  %-30s %s\n", "rebuild", "destroy -> deploy in one shot")
	fmt.Printf("  %-30s %s\n", "challenge (ch)", "Manage challenges")
	fmt.Printf("  %-30s %s\n", "event", "Manage event configuration")
	fmt.Printf("  %-30s %s\n", "help", "Show this message")
}

// -----------------------
// Main
// -----------------------

func main() {
	var filteredArgs []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if a == "--no-color" {
			noColor = true
		} else if a == "--version" {
			fmt.Println("ctfctl v" + version)
			os.Exit(0)
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}

	if len(filteredArgs) == 0 {
		help()
		os.Exit(1)
	}

	cmd := filteredArgs[0]
	args := filteredArgs[1:]
	var err error

	if cmd == "help" {
		help()
	} else if cmd == "deploy" {
		err = runScript("scripts/deploy.sh")
		if err == nil {
			_ = printChallengeURLs()
		}
	} else if cmd == "destroy" {
		err = runScript("scripts/destroy.sh")
	} else if cmd == "rebuild" {
		err = runScript("scripts/destroy.sh")
		if err == nil {
			err = runScript("scripts/deploy.sh")
			if err == nil {
				_ = printChallengeURLs()
			}
		}
	} else if cmd == "reset" {
		err = runScript("scripts/reset_challenges.sh")
		if err == nil {
			err = runScript("scripts/deploy.sh")
			if err == nil {
				_ = printChallengeURLs()
			}
		}
	} else if cmd == "bootstrap" {
		err = bootstrap()
	} else if cmd == "challenge" || cmd == "ch" {
		err = challengeCommand(args)
	} else {
		help()
		err = errors.New("unknown command: " + cmd)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
