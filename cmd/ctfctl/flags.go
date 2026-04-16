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

		// If the challenge specifies custom permissions for the flag file, apply them.
		// This is needed for file-type challenges where the web server needs to read the file.
		if c.Flag.Permissions != "" {
			var mode uint32
			_, parseErr := fmt.Sscanf(c.Flag.Permissions, "%o", &mode)
			if parseErr == nil {
				os.Chmod(flagFile, os.FileMode(mode))
			}
		}

		fmt.Printf("  %-30s %s\n", c.ID, flag)
	}

	fmt.Println()
	fmt.Println("Flags written to flags/.")
	fmt.Println("Run 'ctfctl deploy' or 'ctfctl reset' to inject them into containers.")
	return nil
}

func waitForHTTP(url string, contains string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	fmt.Println("  Waiting for", url+"...")

	for i := 0; i < 30; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		// If a string was specified, make sure the page contains it
		if contains != "" && !strings.Contains(string(body), contains) {
			time.Sleep(3 * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("timed out waiting for %s", url)
}

// getCSRFToken fetches a page and looks for a hidden form input named csrf
func getCSRFToken(url string, client *http.Client, fieldName string) (string, []*http.Cookie, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", nil, err
	}

	// Look for: name="fieldName" or name='fieldName'
	// Then find the value= attribute right after it
	page := string(body)
	token := ""

	// Try double quotes first, then single quotes
	searchDoubleQuote := `name="` + fieldName + `"`
	searchSingleQuote := `name='` + fieldName + `'`

	var namePos int
	if strings.Contains(page, searchDoubleQuote) {
		namePos = strings.Index(page, searchDoubleQuote)
	} else if strings.Contains(page, searchSingleQuote) {
		namePos = strings.Index(page, searchSingleQuote)
	} else {
		return "", nil, fmt.Errorf("could not find field %q on page", fieldName)
	}

	// Look for value= after where we found the name
	afterName := page[namePos:]

	var valueContent string
	if strings.Contains(afterName, `value="`) {
		start := strings.Index(afterName, `value="`) + len(`value="`)
		valueContent = afterName[start:]
		end := strings.Index(valueContent, `"`)
		if end != -1 {
			token = valueContent[:end]
		}
	} else if strings.Contains(afterName, `value='`) {
		start := strings.Index(afterName, `value='`) + len(`value='`)
		valueContent = afterName[start:]
		end := strings.Index(valueContent, `'`)
		if end != -1 {
			token = valueContent[:end]
		}
	}

	if token == "" {
		return "", nil, fmt.Errorf("could not read value of field %q on page", fieldName)
	}

	return token, resp.Cookies(), nil
}

func httpInit(url string, body string, tokenField string) error {
	// Don't follow redirects automatically - a 302 after a form POST is normal
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	fmt.Println("  Initializing via", url+"...")

	for i := 0; i < 5; i++ {
		postBody := body

		if tokenField != "" {
			token, cookies, tokenErr := getCSRFToken(url, client, tokenField)
			if tokenErr != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			postBody = body + "&" + tokenField + "=" + token

			req, reqErr := http.NewRequest("POST", url, strings.NewReader(postBody))
			if reqErr != nil {
				return reqErr
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Add the session cookies from the GET request
			for _, c := range cookies {
				req.AddCookie(c)
			}

			resp, doErr := client.Do(req)
			if doErr != nil {
				time.Sleep(2 * time.Second)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 400 {
				time.Sleep(2 * time.Second)
				continue
			}

			return nil
		}

		// No CSRF token needed, just POST directly
		resp, postErr := client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(postBody))
		if postErr != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			time.Sleep(2 * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to initialize %s after several attempts", url)
}

func injectSQL(c challenge, flag string) error {
	f := c.Flag

	if f.Engine != "mysql" && f.Engine != "postgres" {
		return fmt.Errorf("unknown sql engine %q - must be \"mysql\" or \"postgres\"", f.Engine)
	}

	// Wait for the web app to be ready before trying to touch the database
	if f.ReadyURL != "" {
		err := waitForHTTP(f.ReadyURL, f.ReadyContains)
		if err != nil {
			return fmt.Errorf("app never became ready: %w", err)
		}
	}

	// Some apps need a setup step before the database tables exist (e.g. DVWA's setup.php)
	if f.InitURL != "" {
		err := httpInit(f.InitURL, f.InitBody, f.InitTokenField)
		if err != nil {
			return fmt.Errorf("app init failed: %w", err)
		}
	}

	// Put the flag into the SQL query
	query := strings.ReplaceAll(f.Query, "%s", flag)

	fmt.Println("  Waiting for database...")

	// Try up to 20 times in case the database is still starting up
	for i := 0; i < 20; i++ {
		var err error

		if f.Engine == "mysql" {
			args := []string{"exec", c.ID, "mysql", "-u" + f.User}
			if f.Password != "" {
				args = append(args, "-p"+f.Password)
			}
			args = append(args, f.Database, "-e", query)

			cmd := exec.Command("docker", args...)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			err = cmd.Run()
		} else {
			// postgres uses an environment variable for the password
			cmd := exec.Command("docker", "exec",
				"-e", "PGPASSWORD="+f.Password,
				c.ID, "psql", "-U", f.User, "-d", f.Database, "-c", query)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			err = cmd.Run()
		}

		if err == nil {
			return nil
		}

		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("sql injection failed after several attempts")
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

	// Try up to 20 times in case the service is still starting up
	for i := 0; i < 20; i++ {
		req, err := http.NewRequest(method, f.URL, strings.NewReader(body))
		if err != nil {
			return err
		}

		for k, v := range f.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			time.Sleep(3 * time.Second)
			continue
		}

		return nil
	}

	return fmt.Errorf("api injection failed after several attempts")
}

// getOrCreateFlag returns the flag for a challenge. If no flag file exists yet it generates one.
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

	fmt.Printf("  Generated flag for %-20s %s\n", c.ID, flag)
	return flag, nil
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

		// Skip challenges that don't need active injection
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
