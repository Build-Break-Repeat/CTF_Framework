package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type challenge struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Points int    `json:"points"`
}

type challengeConfig struct {
	Challenges []challenge `json:"challenges"`
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Println("$", name, args)
	return cmd.Run()
}

func runScript(script string) error {
	cmd := exec.Command("bash", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Println("$ bash", script)
	return cmd.Run()
}

func bootstrap() error {
	if err := runCommand("terraform", "-chdir=terraform/bootstrap", "init", "-input=false", "-upgrade"); err != nil {
		return err
	}
	if err := runCommand("terraform", "-chdir=terraform/bootstrap", "apply", "-auto-approve"); err != nil {
		return err
	}

	if err := runCommand("python3", "scripts/ctfd_bootstrap.py"); err == nil {
		return nil
	}

	return runCommand("python", "scripts/ctfd_bootstrap.py")
}

func usage() {
	fmt.Println("ctfctl deploy")
	fmt.Println("ctfctl destroy")
	fmt.Println("ctfctl rebuild")
	fmt.Println("ctfctl bootstrap")
	fmt.Println("ctfctl challenge list")
}

func challengeList() error {
	data, err := os.ReadFile("challenges.json")
	if err != nil {
		return err
	}

	var cfg challengeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	for _, c := range cfg.Challenges {
		fmt.Printf("%s | %s | %d pts\n", c.ID, c.Name, c.Points)
	}
	return nil
}

func challengeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("challenge subcommand required (list)")
	}

	switch args[0] {
	case "list":
		return challengeList()
	default:
		return fmt.Errorf("unknown challenge subcommand: %s", args[0])
	}
}

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}

	var err error

	switch os.Args[1] {
	case "deploy":
		err = runScript("scripts/deploy.sh")
	case "destroy":
		err = runScript("scripts/destroy.sh")
	case "rebuild":
		if err = runScript("scripts/destroy.sh"); err == nil {
			err = runScript("scripts/deploy.sh")
		}
	case "bootstrap":
		err = bootstrap()
	case "challenge":
		err = challengeCommand(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}