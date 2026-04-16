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

  name    = each.key
  image   = each.value.image
  memory  = lookup(each.value, "memory", 256)
  command = lookup(each.value, "command", null)

  networks_advanced {
    name = var.network_name
  }

  dynamic "ports" {
    for_each = lookup(each.value, "ports", [])
    content {
      internal = ports.value.internal
      external = ports.value.external
    }
  }

  dynamic "volumes" {
    for_each = (
      lookup(each.value, "flag", null) != null &&
      lookup(lookup(each.value, "flag", {}), "type", "") == "file"
    ) ? [1] : []
    content {
      host_path      = abspath("${path.module}/../../../flags/${each.key}.txt")
      container_path = lookup(each.value.flag, "path", "/flag.txt")
      read_only      = true
    }
  }

}
