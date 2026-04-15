package main

import (
	"errors"
	"fmt"
	"os"
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
	fmt.Printf("  %-30s %s\n", "status", "Show running state of all challenge containers")
	fmt.Printf("  %-30s %s\n", "help", "Show this message")
}

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
		err = challengeReset([]string{})
		if err == nil {
			_ = printChallengeURLs()
		}
	} else if cmd == "bootstrap" {
		err = bootstrap()
	} else if cmd == "challenge" || cmd == "ch" {
		err = challengeCommand(args)
	} else if cmd == "event" {
		err = eventCommand(args)
	} else if cmd == "status" {
		err = status()
	} else {
		help()
		err = errors.New("unknown command: " + cmd)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
