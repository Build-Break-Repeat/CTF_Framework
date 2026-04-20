package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func runCommand(name string, args ...string) error {
	fmt.Println("$", name, strings.Join(args, " "))
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
	fmt.Println("$ bash", strings.Join(args, " "))
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
		// Each argument is single-quoted and internal single quotes are escaped.
		cmdStr := shellQuote(binaryPath)
		for i := 1; i < len(os.Args); i++ {
			cmdStr += " " + shellQuote(os.Args[i])
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

	cmdStr := shellQuote(binaryPath)
	for i := 1; i < len(os.Args); i++ {
		cmdStr += " " + shellQuote(os.Args[i])
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

	_, err = exec.LookPath("python3")
	if err == nil {
		return runCommand("python3", "scripts/ctfd_bootstrap.py")
	}

	_, err = exec.LookPath("python")
	if err == nil {
		return runCommand("python", "scripts/ctfd_bootstrap.py")
	}

	return errors.New("python3 or python is required to bootstrap CTFd but neither was found")
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

	err = flagsEnsure()
	if err != nil {
		return err
	}

	err = flagsEnsure()
	if err != nil {
		return err
	}

	err = runScript("scripts/terraform_deploy.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	err = flagsInject()
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

	err = runScript("scripts/remove_firewall.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	err = runScript("scripts/terraform_destroy_challenges.sh", scriptFlags()...)
	if err != nil {
		return err
	}

	return runScript("scripts/terraform_destroy_bootstrap.sh", scriptFlags()...)
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

// getHostName returns the machine's hostname if it resolves via DNS,
// otherwise it falls back to the IP address. This way challenge URLs
// work even when DNS is not configured for the host.
func getHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return getHostIP()
	}

	// Try to resolve the hostname via DNS
	addrs, err := net.LookupHost(hostname)
	if err != nil || len(addrs) == 0 {
		return getHostIP()
	}

	return hostname
}

// challengeURL builds the access URL for a single port on a challenge.
func challengeURL(c challenge, p port, hostName string) string {
	path := c.Path
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return "http://" + hostName + ":" + strconv.Itoa(p.External) + path
}

func status() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	if len(cfg.Challenges) == 0 {
		fmt.Println("No challenges configured.")
		return nil
	}

	hostName := getHostName()

	fmt.Println()
	fmt.Println(bold("Challenge Status"))
	fmt.Println("----------------------------------------")
	fmt.Printf("  %-28s %-12s %s\n", "Challenge", "State", "URL")
	fmt.Println("----------------------------------------")

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		out, inspectErr := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", c.ID).Output()

		state := ""
		if inspectErr != nil {
			state = dim("not deployed")
		} else {
			raw := strings.TrimSpace(string(out))
			if raw == "" {
				state = dim("not deployed")
			} else {
				state = raw
			}
		}

		url := ""
		if len(c.Ports) > 0 {
			url = challengeURL(c, c.Ports[0], hostName)
		}

		fmt.Printf("  %-28s %-12s %s\n", c.Name, state, url)
	}

	fmt.Println("----------------------------------------")
	fmt.Println()
	return nil
}

func printChallengeURLs() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	hostName := getHostName()

	fmt.Println()
	fmt.Println(bold("CTF Deployment Complete"))
	fmt.Println()
	fmt.Println(bold("Challenges:"))
	fmt.Println("----------------------------------------")

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		for j := 0; j < len(c.Ports); j++ {
			url := challengeURL(c, c.Ports[j], hostName)
			fmt.Printf("  %-28s %s\n", c.Name, url)
		}
	}

	fmt.Println("----------------------------------------")
	return nil
}
