package main

type port struct {
	Internal int `json:"internal"`
	External int `json:"external"`
}

type challengeFlag struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
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
