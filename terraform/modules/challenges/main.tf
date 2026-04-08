terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.6"
    }
  }
}

variable "challenges" {
  type = any
}

variable "network_name" {
  type = string
}

resource "docker_container" "challenge_containers" {
  for_each = var.challenges

  name  = each.key
  image = each.value.image
  memory = lookup(each.value, "memory", 256)

  networks_advanced {
    name = var.network_name
  }
}
