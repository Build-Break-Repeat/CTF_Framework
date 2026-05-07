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

func flagFilePerm(c challenge) os.FileMode {
	if c.Flag != nil && c.Flag.Permissions != "" {
		v, err := strconv.ParseUint(c.Flag.Permissions, 8, 32)
		if err == nil {
			return os.FileMode(v)
		}
	}
	return 0600
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
		err = os.WriteFile(flagFile, []byte(flag+"\n"), flagFilePerm(c))
		if err != nil {
			return fmt.Errorf("failed to write flag file for %s: %w", c.ID, err)
		}

		fmt.Printf("  %-30s %s\n", c.ID, flag)
	}

	fmt.Println()
	fmt.Println("Flags written to flags/.")
	fmt.Println("Run 'ctfctl deploy' or 'ctfctl ch reload' to deploy them into containers.")
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

// getCSRFToken fetches a page and looks for a hidden form input named fieldName,
// then returns its value. It searches within the <input> tag itself so that a
// value= attribute on a different element cannot be mistakenly returned.
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

	page := string(body)

	// Find an <input tag that contains name="fieldName" or name='fieldName'.
	// We scan for each <input opening and inspect only the characters up to
	// the closing > of that tag.
	searchTag := "<input"
	nameDoubleQuote := `name="` + fieldName + `"`
	nameSingleQuote := `name='` + fieldName + `'`

	pos := 0
	for {
		tagStart := strings.Index(page[pos:], searchTag)
		if tagStart == -1 {
			break
		}
		tagStart += pos

		// Find the end of this tag (closing >)
		tagEnd := strings.Index(page[tagStart:], ">")
		if tagEnd == -1 {
			break
		}
		tagEnd += tagStart

		tag := page[tagStart : tagEnd+1]

		if strings.Contains(tag, nameDoubleQuote) || strings.Contains(tag, nameSingleQuote) {
			// Extract value= from within this tag only
			token := ""
			if strings.Contains(tag, `value="`) {
				start := strings.Index(tag, `value="`) + len(`value="`)
				rest := tag[start:]
				end := strings.Index(rest, `"`)
				if end != -1 {
					token = rest[:end]
				}
			} else if strings.Contains(tag, `value='`) {
				start := strings.Index(tag, `value='`) + len(`value='`)
				rest := tag[start:]
				end := strings.Index(rest, `'`)
				if end != -1 {
					token = rest[:end]
				}
			}

			if token == "" {
				return "", nil, fmt.Errorf("found field %q but could not read its value", fieldName)
			}

			return token, resp.Cookies(), nil
		}

		pos = tagEnd + 1
	}

	return "", nil, fmt.Errorf("could not find field %q on page", fieldName)
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

// rewritePort replaces the port in any localhost URL with the challenge's first external port.
func rewritePort(url string, c challenge) string {
	if len(c.Ports) == 0 || url == "" {
		return url
	}
	port := strconv.Itoa(c.Ports[0].External)
	// Replace localhost:NNNN with the current external port
	if idx := strings.Index(url, "localhost:"); idx != -1 {
		start := idx + len("localhost:")
		end := start
		for end < len(url) && url[end] >= '0' && url[end] <= '9' {
			end++
		}
		return url[:start] + port + url[end:]
	}
	return url
}

func injectSQL(c challenge, flag string) error {
	f := c.Flag

	if f.Engine != "mysql" && f.Engine != "postgres" {
		return fmt.Errorf("unknown sql engine %q - must be \"mysql\" or \"postgres\"", f.Engine)
	}

	readyURL := rewritePort(f.ReadyURL, c)
	initURL := rewritePort(f.InitURL, c)

	// Wait for the web app to be ready before trying to touch the database
	if readyURL != "" {
		err := waitForHTTP(readyURL, f.ReadyContains)
		if err != nil {
			return fmt.Errorf("app never became ready: %w", err)
		}
	}

	// Some apps need a setup step before the database tables exist (e.g. DVWA's setup.php)
	if initURL != "" {
		err := httpInit(initURL, f.InitBody, f.InitTokenField)
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
			// Pass the password via MYSQL_PWD so it doesn't appear in the process list.
			args := []string{"exec", "-e", "MYSQL_PWD=" + f.Password, c.ID, "mysql", "-u" + f.User, f.Database, "-e", query}

			cmd := exec.Command("docker", args...)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			err = cmd.Run()
		} else {
			// Pass the password via PGPASSWORD so it doesn't appear in the process list.
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

	apiURL := rewritePort(f.URL, c)
	body := strings.ReplaceAll(f.Body, "%s", flag)

	fmt.Println("  Waiting for API...")

	client := &http.Client{Timeout: 5 * time.Second}

	// Try up to 20 times in case the service is still starting up
	for i := 0; i < 20; i++ {
		req, err := http.NewRequest(method, apiURL, strings.NewReader(body))
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

// flagsEnsure creates flag files for any challenge that doesn't have one yet.
// It never overwrites existing files, so it is safe to call before every deploy.
func flagsEnsure() error {
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
			continue
		}

		flagFile := "flags/" + c.ID + ".txt"

		if _, statErr := os.Stat(flagFile); statErr == nil {
			continue
		}

		flag, err := computeFlag(prefix)
		if err != nil {
			return fmt.Errorf("failed to generate flag for %s: %w", c.ID, err)
		}

		err = os.WriteFile(flagFile, []byte(flag+"\n"), flagFilePerm(c))
		if err != nil {
			return fmt.Errorf("failed to write flag file for %s: %w", c.ID, err)
		}

		fmt.Printf("  Generated flag for %-20s %s\n", c.ID, flag)
	}

	return nil
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
	failed := 0

	for i := 0; i < len(cfg.Challenges); i++ {
		c := cfg.Challenges[i]

		// file and env flags are handled automatically by Terraform/Docker — no active injection needed
		if c.Flag == nil || c.Flag.Type == "file" || c.Flag.Type == "env" || c.Flag.Type == "" {
			continue
		}

		flag, err := getOrCreateFlag(&cfg, i)
		if err != nil {
			fmt.Println("  Error generating flag for", c.ID+":", err)
			failed++
			continue
		}

		fmt.Printf("Injecting %s (%s)...\n", c.ID, c.Flag.Type)

		if c.Flag.Type == "sql" {
			err = injectSQL(c, flag)
		} else if c.Flag.Type == "api" {
			err = injectAPI(c, flag)
		} else {
			fmt.Println("  Unknown flag type:", c.Flag.Type)
			failed++
			continue
		}

		if err != nil {
			fmt.Println("  Error:", err)
			failed++
			continue
		}

		fmt.Println("  Done.")
		injected++
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("Flags injected: %d  failed: %d\n", injected, failed)
	} else {
		fmt.Printf("Flags injected: %d\n", injected)
	}
	return nil
}

func flagsHelp() {
	fmt.Println(bold("ctfctl flags"))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  ctfctl flags <subcommand>")
	fmt.Println()
	fmt.Println(bold("Subcommands:"))
	fmt.Printf("  %-20s %s\n", "generate", "Generate new flag files for all challenges")
	fmt.Printf("  %-20s %s\n", "inject", "Inject flags into running challenge containers (sql/api types)")
	fmt.Printf("  %-20s %s\n", "help", "Show this message")
	fmt.Println()
	fmt.Println(dim("Note: file and env type flags are handled automatically by Terraform."))
}

func flagsCommand(args []string) error {
	if len(args) == 0 {
		flagsHelp()
		return errors.New("flags subcommand required")
	}

	sub := args[0]

	if sub == "help" || sub == "--help" || sub == "-h" {
		flagsHelp()
		return nil
	}

	if sub == "generate" {
		return flagsGenerate()
	}

	if sub == "inject" {
		return flagsInject()
	}

	return errors.New("unknown flags subcommand: " + sub + " (run 'ctfctl flags help' for usage)")
}
