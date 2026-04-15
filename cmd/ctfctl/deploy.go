package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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
