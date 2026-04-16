package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func runCommand(name string, args ...string) error {
	fmt.Println("$", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func scriptFlags() []string {
	if autoInstall {
		return []string{"-a"}
	}
	return []string{}
}

func runScript(script string, extraArgs ...string) error {
	args := append([]string{script}, extraArgs...)
	fmt.Println("$ bash", args)
	cmd := exec.Command("bash", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// verifyDockerGroup makes sure that if the current session does not have docker group
// access, it checks if user is in docker group and adds if needed, then
// it re-execs the current process under sg docker
func verifyDockerGroup() error {
	// Check if the current session already has docker group access
	out, err := exec.Command("id", "-Gn").Output()
	if err == nil {
		if strings.Contains(string(out), "docker") {
			return nil
		}
	}

	// Check if the user is in the docker group but the session hasn't loaded it
	user := os.Getenv("USER")
	out, err = exec.Command("id", "-Gn", user).Output()
	if err == nil && strings.Contains(string(out), "docker") {
		fmt.Println("[*] User is in the docker group but the current session hasn't loaded it")
		fmt.Println("[*] Reloading group membership...")

		binaryPath, err := os.Executable()
		if err != nil {
			return errors.New("failed to resolve executable path: " + err.Error())
		}

		// Build the command string to pass to sg docker -c "..."
		cmdStr := binaryPath
		for i := 1; i < len(os.Args); i++ {
			cmdStr += " " + os.Args[i]
		}

		sgPath, err := exec.LookPath("sg")
		if err != nil {
			return errors.New("sg command not found — cannot reload docker group")
		}

		// Replace this process with: sg docker -c "<binary> <args>"
		return syscall.Exec(sgPath, []string{"sg", "docker", "-c", cmdStr}, os.Environ())
	}

	// User is not in the docker group at all — attempt to add with sudo
	fmt.Println("[*] User '" + user + "' is not in the docker group")
	fmt.Println("[*] Adding to docker group...")

	err = runCommand("sudo", "usermod", "-aG", "docker", user)
	if err != nil {
		return errors.New("failed to add '" + user + "' to the docker group: " + err.Error())
	}

	fmt.Println("[*] Added '" + user + "' to the docker group. Reloading group membership...")

	binaryPath, err := os.Executable()
	if err != nil {
		return errors.New("failed to resolve executable path: " + err.Error())
	}

	cmdStr := binaryPath
	for i := 1; i < len(os.Args); i++ {
		cmdStr += " " + os.Args[i]
	}

	sgPath, err := exec.LookPath("sg")
	if err != nil {
		return errors.New("sg command not found — cannot reload docker group")
	}

	return syscall.Exec(sgPath, []string{"sg", "docker", "-c", cmdStr}, os.Environ())
}

func bootstrap() error {
	err := runScript("scripts/terraform_bootstrap.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	err = runCommand("python3", "scripts/ctfd_bootstrap.py")
	if err == nil {
		return nil
	}

	return runCommand("python", "scripts/ctfd_bootstrap.py")
}

func deploy() error {
	err := runScript("scripts/check_deps.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	err = verifyDockerGroup()
	if err != nil {
		return err
	}

	err = bootstrap()
	if err != nil {
		return err
	}

	err = runScript("scripts/terraform_deploy.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	return runScript("scripts/configure_firewall.sh", scriptFlags()...)
}

func destroy() error {
	err := verifyDockerGroup()
	if err != nil {
		return err
	}

	err = runScript("scripts/terraform_destroy_challenges.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	err = runScript("scripts/terraform_destroy_bootstrap.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	return runScript("scripts/remove_firewall.sh", scriptFlags()...)
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

func status() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	fmt.Println("Status:")
	fmt.Println("----------------------------------------")

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		out, err := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", c.ID).Output()

		state := ""
		if err != nil {
			state = "not found"
		} else {
			state = strings.TrimSpace(string(out))
		}

		fmt.Printf("%-30s %s\n", c.Name, state)
	}

	fmt.Println("----------------------------------------")
	return nil
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
