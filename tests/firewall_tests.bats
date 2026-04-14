#!/usr/bin/env bats

@test "Port extraction from Terraform files" {
	run grep -rhoP 'external\s*=\s*\K\d+' terraform

	[ "$status" -eq 0 ]
	[[ "$output" =~ 80 ]]
}

@test "Firewall add rule command building" {

	port=8080
	cmd="firewall-cmd --permanent --add-port=${port}/tcp"

	[[ "$cmd" == "firewall-cmd --permanent --add-port=8080/tcp" ]]
}
