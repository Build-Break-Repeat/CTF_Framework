package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

func promptField(label string, defaultVal string) string {
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
