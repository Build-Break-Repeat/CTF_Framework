package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runScript(script string) error {
	cmd := exec.Command("bash", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	fmt.Println("$ bash", script)
	return cmd.Run()
}

func usage() {
	fmt.Println("ctfctl deploy")
	fmt.Println("ctfctl destroy")
	fmt.Println("ctfctl rebuild")
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