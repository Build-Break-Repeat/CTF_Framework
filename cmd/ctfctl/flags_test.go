package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- computeFlag ----

func TestComputeFlag_format(t *testing.T) {
	prefix := "bbr"
	flag, err := computeFlag(prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(flag, prefix+"{") {
		t.Errorf("flag %q does not start with %q", flag, prefix+"{")
	}

	if !strings.HasSuffix(flag, "}") {
		t.Errorf("flag %q does not end with '}'", flag)
	}

	inner := strings.TrimPrefix(flag, prefix+"{")
	inner = strings.TrimSuffix(inner, "}")

	parts := strings.Split(inner, "_")
	if len(parts) != 3 {
		t.Errorf("flag inner %q does not have 3 underscore-separated parts", inner)
	}

	num := 0
	for _, c := range parts[2] {
		if c < '0' || c > '9' {
			t.Errorf("numeric part %q contains non-digit character", parts[2])
			break
		}
		num = num*10 + int(c-'0')
	}
	if num < 100 || num > 999 {
		t.Errorf("numeric suffix %d out of range [100, 999]", num)
	}
}

func TestComputeFlag_uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		f, err := computeFlag("CTF")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[f] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected varied flags, got too few unique values: %d", len(seen))
	}
}

func TestComputeFlag_emptyPrefix(t *testing.T) {
	flag, err := computeFlag("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(flag, "{") {
		t.Errorf("flag %q does not start with '{'", flag)
	}
}

func TestComputeFlag_specialCharPrefix(t *testing.T) {
	flag, err := computeFlag("my-prefix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(flag, "my-prefix{") {
		t.Errorf("flag %q does not start with 'my-prefix{'", flag)
	}
}

// ---- flagsGenerate ----

func TestFlagsGenerate_createsFiles(t *testing.T) {
	dir := t.TempDir()

	oldFile := challengeFile
	defer func() { challengeFile = oldFile }()

	cfg := challengeConfig{
		Event: eventConfig{FlagPrefix: "test"},
		Challenges: []challenge{
			{ID: "ch1", Name: "Chall 1", Flag: &challengeFlag{Type: "file"}},
			{ID: "ch2", Name: "Chall 2", Flag: &challengeFlag{Type: "sql"}},
		},
	}
	challengeFile = writeTempConfig(t, dir, cfg)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := flagsGenerate(); err != nil {
		t.Fatalf("flagsGenerate: %v", err)
	}

	for _, id := range []string{"ch1", "ch2"} {
		path := filepath.Join(dir, "flags", id+".txt")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("flag file %s not created: %v", path, err)
			continue
		}
		flag := strings.TrimSpace(string(data))
		if !strings.HasPrefix(flag, "test{") {
			t.Errorf("flag %q does not start with 'test{'", flag)
		}
	}
}

func TestFlagsGenerate_skipsNilFlag(t *testing.T) {
	dir := t.TempDir()

	oldFile := challengeFile
	defer func() { challengeFile = oldFile }()

	cfg := challengeConfig{
		Event:      eventConfig{FlagPrefix: "test"},
		Challenges: []challenge{{ID: "no-flag", Name: "No Flag", Flag: nil}},
	}
	challengeFile = writeTempConfig(t, dir, cfg)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := flagsGenerate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, "flags", "no-flag.txt")
	if _, err := os.Stat(path); err == nil {
		t.Error("expected no flag file for nil-flag challenge, but file was created")
	}
}

func TestFlagsGenerate_defaultPrefix(t *testing.T) {
	dir := t.TempDir()

	oldFile := challengeFile
	defer func() { challengeFile = oldFile }()

	cfg := challengeConfig{
		Event:      eventConfig{FlagPrefix: ""},
		Challenges: []challenge{{ID: "ch", Name: "Chall", Flag: &challengeFlag{Type: "file"}}},
	}
	challengeFile = writeTempConfig(t, dir, cfg)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := flagsGenerate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "flags", "ch.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "CTF{") {
		t.Errorf("expected default prefix 'CTF{', got %q", strings.TrimSpace(string(data)))
	}
}

// ---- flagsEnsure ----

func TestFlagsEnsure_doesNotOverwrite(t *testing.T) {
	dir := t.TempDir()

	oldFile := challengeFile
	defer func() { challengeFile = oldFile }()

	cfg := challengeConfig{
		Event:      eventConfig{FlagPrefix: "test"},
		Challenges: []challenge{{ID: "existing", Name: "Existing", Flag: &challengeFlag{Type: "file"}}},
	}
	challengeFile = writeTempConfig(t, dir, cfg)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	flagsDir := filepath.Join(dir, "flags")
	if err := os.MkdirAll(flagsDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := "test{original_flag_123}"
	if err := os.WriteFile(filepath.Join(flagsDir, "existing.txt"), []byte(existing+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := flagsEnsure(); err != nil {
		t.Fatalf("flagsEnsure: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(flagsDir, "existing.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != existing {
		t.Errorf("flagsEnsure overwrote existing flag: got %q, want %q", strings.TrimSpace(string(data)), existing)
	}
}

func TestFlagsEnsure_generatesForMissing(t *testing.T) {
	dir := t.TempDir()

	oldFile := challengeFile
	defer func() { challengeFile = oldFile }()

	cfg := challengeConfig{
		Event:      eventConfig{FlagPrefix: "ens"},
		Challenges: []challenge{{ID: "new-ch", Name: "New", Flag: &challengeFlag{Type: "file"}}},
	}
	challengeFile = writeTempConfig(t, dir, cfg)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := flagsEnsure(); err != nil {
		t.Fatalf("flagsEnsure: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "flags", "new-ch.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "ens{") {
		t.Errorf("expected flag with prefix 'ens{', got %q", strings.TrimSpace(string(data)))
	}
}

// ---- getOrCreateFlag ----

func TestGetOrCreateFlag_readsExisting(t *testing.T) {
	dir := t.TempDir()

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.MkdirAll(filepath.Join(dir, "flags"), 0755); err != nil {
		t.Fatal(err)
	}
	existing := "bbr{my_existing_flag_999}"
	if err := os.WriteFile(filepath.Join(dir, "flags", "ch1.txt"), []byte(existing+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &challengeConfig{
		Event:      eventConfig{FlagPrefix: "bbr"},
		Challenges: []challenge{{ID: "ch1", Flag: &challengeFlag{Type: "file"}}},
	}

	got, err := getOrCreateFlag(cfg, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != existing {
		t.Errorf("got %q; want %q", got, existing)
	}
}

func TestGetOrCreateFlag_generatesNew(t *testing.T) {
	dir := t.TempDir()

	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	cfg := &challengeConfig{
		Event:      eventConfig{FlagPrefix: "new"},
		Challenges: []challenge{{ID: "fresh", Flag: &challengeFlag{Type: "file"}}},
	}

	got, err := getOrCreateFlag(cfg, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "new{") {
		t.Errorf("expected flag starting with 'new{', got %q", got)
	}

	data, err := os.ReadFile(filepath.Join(dir, "flags", "fresh.txt"))
	if err != nil {
		t.Fatal("flag file not created")
	}
	if strings.TrimSpace(string(data)) != got {
		t.Errorf("file content %q does not match returned flag %q", strings.TrimSpace(string(data)), got)
	}
}

// ---- flagsCommand ----

func TestFlagsCommand_noArgs(t *testing.T) {
	err := flagsCommand([]string{})
	if err == nil {
		t.Error("expected error for no args")
	}
}

func TestFlagsCommand_unknown(t *testing.T) {
	err := flagsCommand([]string{"bogus"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

// ---- getCSRFToken ----

func TestGetCSRFToken_found(t *testing.T) {
	html := `<html><body>
		<form>
			<input type="hidden" name="csrf_token" value="abc123">
			<input type="text" name="username">
		</form>
	</body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer ts.Close()

	client := &http.Client{}
	token, _, err := getCSRFToken(ts.URL, client, "csrf_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "abc123" {
		t.Errorf("got token %q; want %q", token, "abc123")
	}
}

func TestGetCSRFToken_notFound(t *testing.T) {
	html := `<html><body><input type="text" name="username"></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer ts.Close()

	client := &http.Client{}
	_, _, err := getCSRFToken(ts.URL, client, "csrf_token")
	if err == nil {
		t.Error("expected error when field not found, got nil")
	}
}

func TestGetCSRFToken_singleQuoteValue(t *testing.T) {
	html := `<html><body><input type='hidden' name='token' value='xyz789'></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer ts.Close()

	client := &http.Client{}
	token, _, err := getCSRFToken(ts.URL, client, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "xyz789" {
		t.Errorf("got token %q; want %q", token, "xyz789")
	}
}

func TestGetCSRFToken_wrongFieldNameIgnored(t *testing.T) {
	// The value on the wrong field should not leak into the right field's result
	html := `<html><body>
		<input name="other_field" value="should_not_return_this">
		<input name="csrf_token" value="correct_token">
	</body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer ts.Close()

	client := &http.Client{}
	token, _, err := getCSRFToken(ts.URL, client, "csrf_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "correct_token" {
		t.Errorf("got token %q; want %q", token, "correct_token")
	}
}

func TestGetCSRFToken_serverError(t *testing.T) {
	client := &http.Client{}
	_, _, err := getCSRFToken("http://127.0.0.1:1", client, "token")
	if err == nil {
		t.Error("expected error for unreachable server, got nil")
	}
}

// ---- injectSQL engine validation ----

func TestInjectSQL_unknownEngine(t *testing.T) {
	c := challenge{
		ID:   "test",
		Flag: &challengeFlag{Type: "sql", Engine: "mssql"},
	}
	err := injectSQL(c, "flag{test}")
	if err == nil {
		t.Error("expected error for unknown engine, got nil")
	}
	if !strings.Contains(err.Error(), "unknown sql engine") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---- injectAPI default method ----

func TestInjectAPI_usesDefaultPOST(t *testing.T) {
	received := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Method
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := challenge{
		ID: "api-test",
		Flag: &challengeFlag{
			Type:   "api",
			URL:    ts.URL,
			Method: "",
			Body:   `{"flag": "%s"}`,
		},
	}

	err := injectAPI(c, "test{flag_123}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != "POST" {
		t.Errorf("expected POST method, got %q", received)
	}
}

func TestInjectAPI_customMethod(t *testing.T) {
	received := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Method
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := challenge{
		ID: "api-test",
		Flag: &challengeFlag{
			Type:   "api",
			URL:    ts.URL,
			Method: "PUT",
			Body:   `{"flag": "%s"}`,
		},
	}

	err := injectAPI(c, "test{flag_456}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != "PUT" {
		t.Errorf("expected PUT method, got %q", received)
	}
}

func TestInjectAPI_sendsHeaders(t *testing.T) {
	receivedCT := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCT = r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := challenge{
		ID: "api-test",
		Flag: &challengeFlag{
			Type:    "api",
			URL:     ts.URL,
			Method:  "POST",
			Body:    `{"flag": "%s"}`,
			Headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	if err := injectAPI(c, "test{flag_789}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedCT != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", receivedCT)
	}
}

func TestInjectAPI_substitutesFlagInBody(t *testing.T) {
	receivedBody := ""
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		io.Copy(&sb, r.Body)
		receivedBody = sb.String()
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c := challenge{
		ID: "api-test",
		Flag: &challengeFlag{
			Type:   "api",
			URL:    ts.URL,
			Method: "POST",
			Body:   `{"flag": "%s"}`,
		},
	}

	if err := injectAPI(c, "bbr{my_flag_999}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(receivedBody, "bbr{my_flag_999}") {
		t.Errorf("body %q does not contain flag", receivedBody)
	}
}

