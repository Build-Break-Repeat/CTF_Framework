package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
	fmt.Println("Run 'ctfctl deploy' or 'ctfctl reset' to mount them into containers.")
	return nil
}

// runCommandSilent runs a command discarding all output, returning only exit status.
func runCommandSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// withRetry calls fn up to maxTries times, sleeping delay between attempts.
func withRetry(maxTries int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i < maxTries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i < maxTries-1 {
			time.Sleep(delay)
		}
	}
	return lastErr
}

func injectSQL(c challenge, flag string) error {
	f := c.Flag

	if f.Engine != "mysql" && f.Engine != "postgres" {
		return fmt.Errorf("unknown sql engine %q — must be \"mysql\" or \"postgres\"", f.Engine)
	}

	query := strings.ReplaceAll(f.Query, "%s", flag)

	fmt.Println("  Waiting for database...")

	err := withRetry(20, 3*time.Second, func() error {
		if f.Engine == "mysql" {
			return runCommandSilent("docker", "exec", c.ID,
				"mysql", "-u"+f.User, "-p"+f.Password, f.Database,
				"-e", query)
		}
		// postgres
		return runCommandSilent("docker", "exec",
			"-e", "PGPASSWORD="+f.Password,
			c.ID, "psql", "-U", f.User, "-d", f.Database, "-c", query)
	})

	if err != nil {
		return fmt.Errorf("sql injection failed: %w", err)
	}
	return nil
}

func injectAPI(c challenge, flag string) error {
	f := c.Flag

	method := f.Method
	if method == "" {
		method = "POST"
	}

	body := strings.ReplaceAll(f.Body, "%s", flag)

	fmt.Println("  Waiting for API...")

	client := &http.Client{Timeout: 5 * time.Second}

	err := withRetry(20, 3*time.Second, func() error {
		req, err := http.NewRequest(method, f.URL, strings.NewReader(body))
		if err != nil {
			return err
		}

		for k, v := range f.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("API returned status %d", resp.StatusCode)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("api injection failed: %w", err)
	}
	return nil
}

// getOrCreateFlag returns the flag value for a challenge, generating and saving it if missing.
func getOrCreateFlag(cfg *challengeConfig, i int) (string, error) {
	c := cfg.Challenges[i]
	flagFile := "flags/" + c.ID + ".txt"

	data, err := os.ReadFile(flagFile)
	if err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	prefix := cfg.Event.FlagPrefix
	if prefix == "" {
		prefix = "CTF"
	}

	err = os.MkdirAll("flags", 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create flags directory: %w", err)
	}

	flag, err := computeFlag(prefix)
	if err != nil {
		return "", fmt.Errorf("failed to generate flag for %s: %w", c.ID, err)
	}

	err = os.WriteFile(flagFile, []byte(flag+"\n"), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write flag file for %s: %w", c.ID, err)
	}

	cfg.Challenges[i].Flag.Content = flag
	fmt.Printf("  Generated flag for %-20s %s\n", c.ID, flag)
	return flag, nil
}

func flagsInject() error {
	cfg, err := loadChallengeConfig()
	if err != nil {
		return err
	}

	// Check whether any challenges actually need injection before doing anything.
	hasInjectable := false
	for i := 0; i < len(cfg.Challenges); i++ {
		t := ""
		if cfg.Challenges[i].Flag != nil {
			t = cfg.Challenges[i].Flag.Type
		}
		if t == "sql" || t == "api" {
			hasInjectable = true
			break
		}
	}
	if !hasInjectable {
		return nil
	}

	injected := 0
	skipped := 0
	cfgDirty := false

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		if c.Flag == nil || c.Flag.Type == "file" || c.Flag.Type == "env" || c.Flag.Type == "" {
			skipped++
			continue
		}

		flag, err := getOrCreateFlag(&cfg, i)
		if err != nil {
			fmt.Println("  Error generating flag for", c.ID+":", err)
			skipped++
			continue
		}
		if cfg.Challenges[i].Flag.Content == flag && c.Flag.Content != flag {
			cfgDirty = true
		}

		fmt.Printf("Injecting %s (%s)...\n", c.ID, c.Flag.Type)

		if c.Flag.Type == "sql" {
			err = injectSQL(c, flag)
		} else if c.Flag.Type == "api" {
			err = injectAPI(c, flag)
		} else {
			fmt.Println("  Unknown flag type:", c.Flag.Type)
			skipped++
			continue
		}

		if err != nil {
			fmt.Println("  Error:", err)
			skipped++
			continue
		}

		fmt.Println("  Done.")
		injected++
	}

	if cfgDirty {
		_ = saveChallengeConfig(cfg)
	}

	fmt.Println()
	fmt.Printf("Flags injected: %d  skipped: %d\n", injected, skipped)
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
