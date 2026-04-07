output "challenge_container_names" {
  value = keys(docker_container.challenge_containers)
}
