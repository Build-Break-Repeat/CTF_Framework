output "challenge_containers" {
  value = [
    docker_container.kali_desktop.name,
    docker_container.dvwa.name,
    docker_container.sqli_labs.name,
    docker_container.nowasp.name,
    docker_container.metasploitable2.name,
    docker_container.juice_shop.name,
    docker_container.webgoat.name,
    docker_container.bwapp.name
  ]
}
