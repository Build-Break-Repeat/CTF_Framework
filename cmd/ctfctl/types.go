package main

type port struct {
	Internal int `json:"internal"`
	External int `json:"external"`
}

type challengeFlag struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`

	// file
	Path        string `json:"path,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`

	// sql
	Engine        string `json:"engine,omitempty"`         // "mysql" or "postgres"
	User          string `json:"user,omitempty"`
	Password      string `json:"password,omitempty"`
	Database      string `json:"database,omitempty"`
	Query         string `json:"query,omitempty"`          // use %s as placeholder for the flag value
	ReadyURL      string `json:"ready_url,omitempty"`      // poll this URL before injecting
	ReadyContains string `json:"ready_contains,omitempty"` // require this string in the ready response
	InitURL       string `json:"init_url,omitempty"`       // POST to this URL to initialize the app (e.g. DB setup)
	InitBody      string `json:"init_body,omitempty"`      // POST body for init_url

	// api
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"` // defaults to POST
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"` // use %s as placeholder for the flag value
}

type challenge struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Category    string         `json:"category,omitempty"`
	Points      int            `json:"points"`
	Path        string         `json:"path,omitempty"`
	Image       string         `json:"image,omitempty"`
	Memory      int            `json:"memory,omitempty"`
	Flag        *challengeFlag `json:"flag,omitempty"`
	Ports       []port         `json:"ports,omitempty"`
	Environment []string       `json:"environment,omitempty"`
}

type eventAdmin struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type eventConfig struct {
	Name       string     `json:"name,omitempty"`
	Teams      int        `json:"teams,omitempty"`
	FlagPrefix string     `json:"flag_prefix,omitempty"`
	SecretSeed string     `json:"secret_seed,omitempty"`
	Admin      eventAdmin `json:"admin"`
}

type challengeConfig struct {
	Event      eventConfig `json:"event"`
	Challenges []challenge `json:"challenges"`
}
