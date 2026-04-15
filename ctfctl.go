package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "strings"
)

// Challenge struct for reading challenges.json
type challenge struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Points int    `json:"points"`
    Port   int    `json:"port"`
}

// Wrapper struct for the JSON file
type challengeConfig struct {
    Challenges []challenge `json:"challenges"`
}

// executes a normal system command (terraform, python, etc.)
func runCommand(name string, args ...string) error {
    fmt.Println("$", name, args)
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    return cmd.Run()
}

// executes a bash script like deploy.sh or destroy.sh
func runScript(script string) error {
    fmt.Println("$ bash", script)
    cmd := exec.Command("bash", script)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    return cmd.Run()
}

//runs the terraform bootstrap and CTFd setup scripts
func bootstrap() error {
    // terraform init
    if err := runCommand("terraform", "-chdir=terraform/bootstrap", "init", "-input=false", "-upgrade"); err != nil {
        return err
    }

    // terraform apply
    if err := runCommand("terraform", "-chdir=terraform/bootstrap", "apply", "-auto-approve"); err != nil {
        return err
    }

    // try python3 first
    if err := runCommand("python3", "scripts/ctfd_bootstrap.py"); err == nil {
        return nil
    }

    // fallback to python
    return runCommand("python", "scripts/ctfd_bootstrap.py")
}

// usage prints the available commands
func usage() {
    fmt.Println("ctfctl deploy")
    fmt.Println("ctfctl destroy")
    fmt.Println("ctfctl rebuild")
    fmt.Println("ctfctl reset")
    fmt.Println("ctfctl bootstrap")
    fmt.Println("ctfctl challenge list")
}

//loads challenges.json and prints each challenge
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

//ctfctl challenge list
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

// gets host machine IP for external access
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

// prints challenge container URLs after deployment
func printChallengeURLs() error {
    data, err := os.ReadFile("challenges.json")
    if err != nil {
        return err
    }

    var cfg challengeConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return err
    }

    hostIP := getHostIP()

    fmt.Println("")
    fmt.Println("CTF Deployment Complete")
    fmt.Println("")
    fmt.Println("Challenges:")
    fmt.Println("----------------------------------------")

    for _, c := range cfg.Challenges {
        if c.Port != 0 {
            fmt.Printf("%-15s http://%s:%d\n", c.Name, hostIP, c.Port)
        }
    }

    fmt.Println("----------------------------------------")
    return nil
}

// main
func main() {

    // allow more than 2 arguments
    if len(os.Args) < 2 {
        usage()
        os.Exit(1)
    }

    cmd := os.Args[1]      // main command
    args := os.Args[2:]    // extra arguments (used for challenge list)
    var err error

    switch cmd {

    case "deploy":
        err = runScript("scripts/deploy.sh")
        if err == nil {
            _ = printChallengeURLs()
        }

    case "destroy":
        err = runScript("scripts/destroy.sh")

    case "rebuild":
        // destroy everything, then deploy everything
        if err = runScript("scripts/destroy.sh"); err == nil {
            err = runScript("scripts/deploy.sh")
            if err == nil {
                _ = printChallengeURLs()
            }
        }

    case "reset":
        // destroy only challenge containers, then redeploy challenges
        if err = runScript("scripts/reset_challenges.sh"); err == nil {
            err = runScript("scripts/deploy.sh")
            if err == nil {
                _ = printChallengeURLs()
            }
        }

    case "bootstrap":
        err = bootstrap()

    case "challenge":
        err = challengeCommand(args)

    default:
        usage()
        err = fmt.Errorf("unknown command: %s", cmd)
    }

    // print error and exit if something failed
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error:", err)
        os.Exit(1)
    }
}
