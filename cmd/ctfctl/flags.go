package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

var flagAdjectives = []string{
	"able", "brisk", "calm", "daring", "eager", "fancy", "gentle", "happy", "ideal", "jolly",
	"keen", "lively", "merry", "noble", "open", "proud", "quick", "ready", "sharp", "tidy",
	"urban", "vivid", "witty", "young", "zesty", "agile", "bold", "chill", "droll", "elegant",
	"fiery", "grand", "honest", "icy", "jaunty", "kind", "loyal", "mild", "neat", "odd",
	"plucky", "quiet", "rapid", "sly", "tough", "upbeat", "vast", "warm", "xenial", "yearly",
	"zonal", "apt", "blue", "crisp", "dry", "easy", "fair", "glad", "hazy", "intact",
	"jaded", "known", "light", "moody", "nice", "overt", "plain", "quick", "rural", "soft",
	"true", "ultra", "valid", "wise", "xeric", "youngish", "zany", "acute", "brief", "clean",
	"dense", "early", "faint", "giant", "humid", "ideal", "jolly", "knotty", "loose", "minor",
}

var flagNouns = []string{
	"apple", "bread", "chair", "desk", "earth", "flame", "glass", "house", "island", "jacket",
	"knife", "lamp", "mirror", "needle", "ocean", "paper", "queen", "river", "stone", "table",
	"umbrella", "vase", "window", "xylophone", "yard", "zebra", "anchor", "bridge", "cloud", "door",
	"engine", "forest", "garden", "hammer", "ink", "jewel", "key", "ladder", "market", "nest",
	"object", "pillow", "quill", "road", "shelf", "tower", "unit", "valley", "wheel", "xenon",
	"yarn", "zone", "artist", "beach", "circle", "drum", "energy", "field", "grain", "hill",
	"idea", "jar", "kettle", "leaf", "metal", "night", "orbit", "plant", "quest", "rain",
	"ship", "track", "user", "voice", "water", "xylem", "year", "zinc", "area", "block",
	"cell", "data", "event", "frame", "group", "heart", "image", "joint", "knife", "level",
}

func randIndex(n int) (int, error) {
	max := big.NewInt(int64(n))
	val, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(val.Int64()), nil
}

func computeFlag(prefix string) (string, error) {
	adjIdx, err := randIndex(len(flagAdjectives))
	if err != nil {
		return "", err
	}

	nounIdx, err := randIndex(len(flagNouns))
	if err != nil {
		return "", err
	}

	numVal, err := randIndex(900)
	if err != nil {
		return "", err
	}
	num := numVal + 100

	phrase := flagAdjectives[adjIdx] + "_" + flagNouns[nounIdx] + "_" + strconv.Itoa(num)
	return prefix + "{" + phrase + "}", nil
}

func flagsGenerate() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	prefix := cfg.Event.FlagPrefix
	if prefix == "" {
		prefix = "CTF"
	}

	err = os.MkdirAll("flags", 0755)
	if err != nil {
		return fmt.Errorf("failed to create flags directory: %w", err)
	}

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		if c.Flag == nil {
			fmt.Println("  Skipping", c.ID, "(no flag config)")
			continue
		}

		flag, err := computeFlag(prefix)
		if err != nil {
			return fmt.Errorf("failed to generate flag for %s: %w", c.ID, err)
		}

		flagFile := "flags/" + c.ID + ".txt"
		err = os.WriteFile(flagFile, []byte(flag+"\n"), 0600)
		if err != nil {
			return fmt.Errorf("failed to write flag file for %s: %w", c.ID, err)
		}

		cfg.Challenges[i].Flag.Content = flag

		fmt.Printf("  %-30s %s\n", c.ID, flag)
	}

	err = saveChallengeConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to save challenges.json: %w", err)
	}

	fmt.Println()
	fmt.Println("Flags written to flags/ and recorded in challenges.json.")
	fmt.Println("Run 'ctfctl flags inject' after containers are running to place them.")
	return nil
}

func flagsInject() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	injected := 0
	skipped := 0

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		if c.Flag == nil {
			skipped++
			continue
		}

		flagFile := "flags/" + c.ID + ".txt"
		data, err := os.ReadFile(flagFile)
		if err != nil {
			fmt.Println("  Skipping", c.ID, "— no flag file found (run 'ctfctl flags generate' first)")
			skipped++
			continue
		}

		flagContent := strings.TrimSpace(string(data))
		flagPath := c.Flag.Path
		owner := c.Flag.Owner
		if owner == "" {
			owner = "root"
		}
		perms := c.Flag.Permissions
		if perms == "" {
			perms = "0600"
		}

		// Write to a temp file then docker cp into the container to avoid shell quoting
		tmpFile, err := os.CreateTemp("", "ctfctl-flag-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file for %s: %w", c.ID, err)
		}
		tmpPath := tmpFile.Name()

		_, err = tmpFile.WriteString(flagContent + "\n")
		tmpFile.Close()
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write temp flag for %s: %w", c.ID, err)
		}

		err = runCommand("docker", "cp", tmpPath, c.ID+":"+flagPath)
		os.Remove(tmpPath)
		if err != nil {
			fmt.Println("  Warning: failed to copy flag into", c.ID, "—", err)
			skipped++
			continue
		}

		err = runCommand("docker", "exec", c.ID, "chmod", perms, flagPath)
		if err != nil {
			fmt.Println("  Warning: failed to chmod", flagPath, "in", c.ID, "—", err)
		}

		err = runCommand("docker", "exec", c.ID, "chown", owner+":"+owner, flagPath)
		if err != nil {
			fmt.Println("  Warning: failed to chown", flagPath, "in", c.ID, "—", err)
		}

		fmt.Printf("  %-30s → %s\n", c.ID, flagPath)
		injected++
	}

	fmt.Println()
	fmt.Printf("Done. %d injected, %d skipped.\n", injected, skipped)
	return nil
}

func flagsCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("flags subcommand required (generate, inject)")
	}

	sub := args[0]

	if sub == "generate" {
		return flagsGenerate()
	}

	if sub == "inject" {
		return flagsInject()
	}

	return errors.New("unknown flags subcommand: " + sub)
}
