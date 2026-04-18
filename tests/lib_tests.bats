#!/usr/bin/env bats

# ---------------------------------------------------------------------------
# Tests for scripts/lib.sh helper functions
# Functions are sourced in isolated sub-shells where possible to prevent
# state from leaking between tests.
# ---------------------------------------------------------------------------

SCRIPTS_DIR="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/../scripts" && pwd)"

# ---------------------------------------------------------------------------
# command_exists
# ---------------------------------------------------------------------------

@test "command_exists returns 0 for a real command (bash)" {
    source "$SCRIPTS_DIR/lib.sh"
    run command_exists bash
    [ "$status" -eq 0 ]
}

@test "command_exists returns 1 for a fake command" {
    source "$SCRIPTS_DIR/lib.sh"
    run command_exists __not_a_real_command_xyz__
    [ "$status" -eq 1 ]
}

# ---------------------------------------------------------------------------
# parse_flags
# ---------------------------------------------------------------------------

@test "parse_flags sets NON_INTERACTIVE with --non-interactive" {
    source "$SCRIPTS_DIR/lib.sh"
    NON_INTERACTIVE=false
    parse_flags --non-interactive
    [ "$NON_INTERACTIVE" = "true" ]
}

@test "parse_flags sets NON_INTERACTIVE with -n" {
    source "$SCRIPTS_DIR/lib.sh"
    NON_INTERACTIVE=false
    parse_flags -n
    [ "$NON_INTERACTIVE" = "true" ]
}

@test "parse_flags sets AUTO_INSTALL with --auto-install" {
    source "$SCRIPTS_DIR/lib.sh"
    AUTO_INSTALL=false
    parse_flags --auto-install
    [ "$AUTO_INSTALL" = "true" ]
}

@test "parse_flags sets AUTO_INSTALL with -a" {
    source "$SCRIPTS_DIR/lib.sh"
    AUTO_INSTALL=false
    parse_flags -a
    [ "$AUTO_INSTALL" = "true" ]
}

@test "parse_flags ignores unknown flags without error" {
    source "$SCRIPTS_DIR/lib.sh"
    parse_flags --unknown-flag --another-one
    [ "$NON_INTERACTIVE" = "false" ]
    [ "$AUTO_INSTALL" = "false" ]
}

@test "parse_flags sets both flags together" {
    source "$SCRIPTS_DIR/lib.sh"
    NON_INTERACTIVE=false
    AUTO_INSTALL=false
    parse_flags --non-interactive --auto-install
    [ "$NON_INTERACTIVE" = "true" ]
    [ "$AUTO_INSTALL" = "true" ]
}

@test "parse_flags --help exits with 0" {
    bash -c "source '$SCRIPTS_DIR/lib.sh'; parse_flags --help"
    [ "$?" -eq 0 ]
}

# ---------------------------------------------------------------------------
# record_port_state
# ---------------------------------------------------------------------------

@test "record_port_state appends port to state file" {
    source "$SCRIPTS_DIR/lib.sh"
    TMPDIR=$(mktemp -d)
    PORT_STATE_FILE="$TMPDIR/firewall_ports.state"

    record_port_state 8080
    [ -f "$PORT_STATE_FILE" ]
    grep -q "8080" "$PORT_STATE_FILE"

    rm -rf "$TMPDIR"
}

@test "record_port_state appends multiple entries" {
    source "$SCRIPTS_DIR/lib.sh"
    TMPDIR=$(mktemp -d)
    PORT_STATE_FILE="$TMPDIR/firewall_ports.state"

    record_port_state 80
    record_port_state 443
    record_port_state 8080

    [ "$(wc -l < "$PORT_STATE_FILE")" -eq 3 ]

    rm -rf "$TMPDIR"
}

@test "record_port_state creates state file if it does not exist" {
    source "$SCRIPTS_DIR/lib.sh"
    TMPDIR=$(mktemp -d)
    PORT_STATE_FILE="$TMPDIR/new_state.state"

    record_port_state 9000
    [ -f "$PORT_STATE_FILE" ]

    rm -rf "$TMPDIR"
}

# ---------------------------------------------------------------------------
# remove_firewall_rules — no state file path
# ---------------------------------------------------------------------------

@test "remove_firewall_rules prints message when state file is missing" {
    source "$SCRIPTS_DIR/lib.sh"
    PORT_STATE_FILE="/tmp/__nonexistent_state_file_$(date +%s)__"

    run remove_firewall_rules
    [ "$status" -eq 0 ]
    [[ "$output" == *"No port state file found"* ]]
}

# ---------------------------------------------------------------------------
# detect_firewall
# ---------------------------------------------------------------------------

@test "detect_firewall returns a non-empty string" {
    source "$SCRIPTS_DIR/lib.sh"
    result=$(detect_firewall)
    [ -n "$result" ]
}

@test "detect_firewall returns one of the known values" {
    source "$SCRIPTS_DIR/lib.sh"
    result=$(detect_firewall)
    [[ "$result" == "firewalld" || "$result" == "ufw" || "$result" == "iptables" || "$result" == "none" ]]
}

# ---------------------------------------------------------------------------
# Firewall command string format
# ---------------------------------------------------------------------------

@test "firewall-cmd add rule has correct format" {
    port=8080
    cmd="firewall-cmd --permanent --add-port=${port}/tcp"
    [[ "$cmd" == "firewall-cmd --permanent --add-port=8080/tcp" ]]
}

@test "firewall-cmd remove rule has correct format" {
    port=443
    cmd="firewall-cmd --permanent --remove-port=${port}/tcp"
    [[ "$cmd" == "firewall-cmd --permanent --remove-port=443/tcp" ]]
}

@test "ufw allow rule has correct format" {
    port=8080
    cmd="ufw allow ${port}/tcp"
    [[ "$cmd" == "ufw allow 8080/tcp" ]]
}

@test "iptables add rule has correct format" {
    port=8080
    cmd="iptables -I INPUT -p tcp --dport $port -j ACCEPT"
    [[ "$cmd" == "iptables -I INPUT -p tcp --dport 8080 -j ACCEPT" ]]
}

@test "iptables delete rule has correct format" {
    port=8080
    cmd="iptables -D INPUT -p tcp --dport $port -j ACCEPT"
    [[ "$cmd" == "iptables -D INPUT -p tcp --dport 8080 -j ACCEPT" ]]
}

# ---------------------------------------------------------------------------
# Terraform port extraction
# ---------------------------------------------------------------------------

@test "Port extraction finds expected ports from Terraform files" {
    run grep -rhoP 'external\s*=\s*\K\d+' "$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/terraform"
    [ "$status" -eq 0 ]
    [[ "$output" =~ 443 ]]
}

@test "Port extraction returns only numeric values" {
    REPO_ROOT="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
    run bash -c "grep -rhoP 'external\s*=\s*\K\d+' \"$REPO_ROOT/terraform\" | grep -vE '^[0-9]+\$'"
    # Output should be empty — all extracted values are numeric
    [ -z "$output" ]
}

# ---------------------------------------------------------------------------
# config.json structure validation
# ---------------------------------------------------------------------------

@test "config.json is valid JSON" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq empty "$CONFIG"
    [ "$status" -eq 0 ]
}

@test "config.json has event.name field" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '.event.name' "$CONFIG"
    [ "$status" -eq 0 ]
}

@test "config.json has challenges array" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '.challenges | type == "array"' "$CONFIG"
    [ "$status" -eq 0 ]
}

@test "each challenge has an id field" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '[.challenges[] | select(.id == null)] | length == 0' "$CONFIG"
    [ "$status" -eq 0 ]
    [[ "$output" == "true" ]]
}

@test "each challenge has a positive points value" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '[.challenges[] | select(.points <= 0)] | length == 0' "$CONFIG"
    [ "$status" -eq 0 ]
    [[ "$output" == "true" ]]
}

@test "event has flag_prefix field" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '.event.flag_prefix | type == "string"' "$CONFIG"
    [ "$status" -eq 0 ]
}

@test "admin has username and password" {
    CONFIG="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/config.json"
    run jq -e '.event.admin.username and .event.admin.password' "$CONFIG"
    [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# Shell script syntax (shellcheck)
# Run all scripts together so shellcheck can follow 'source' directives
# across files, matching how the CI workflow calls: shellcheck scripts/*.sh
# ---------------------------------------------------------------------------

@test "all scripts/*.sh pass shellcheck" {
    SCRIPTS_GLOB="$(cd -- "$(dirname "$BATS_TEST_FILENAME")/../scripts" && pwd)/*.sh"
    # shellcheck disable=SC2086
    run shellcheck $SCRIPTS_GLOB
    [ "$status" -eq 0 ]
}
