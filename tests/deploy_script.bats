#!/usr/bin/env bats

@test "Deploy script loading without errors" {

	run bash -n scripts/deploy.sh

	[ "$status" -eq 0 ]
}
