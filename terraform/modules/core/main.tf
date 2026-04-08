terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.6"
    }
  }
}

resource "docker_network" "lab_network" {
  name = "lab-network"
}

resource "docker_container" "ctfd" {
  name  = "ctfd"
  image = "ctfd/ctfd:latest"

  networks_advanced {
    name = docker_network.lab_network.name
  }
  ports {
    internal = 8000
    external = 8000
  }
}

resource "docker_container" "ctfd_db" {
  name  = "ctfd-db"
  image = "mariadb:10"

  networks_advanced {
    name = docker_network.lab_network.name
  }
}

