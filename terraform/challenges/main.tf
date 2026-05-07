variable "challenge_host" {
  type    = string
  default = "localhost"
}

locals {
  config          = jsondecode(file("${path.module}/../../config.json"))
  challenges_list = local.config.challenges
  base_url        = lookup(local.config.event, "base_url", "")

  challenges = {
    for c in local.challenges_list :
    c.id => merge(c, {
      url = length(lookup(c, "ports", [])) > 0 ? (
        local.base_url != ""
          ? "${local.base_url}:${c.ports[0].external}"
          : "http://${var.challenge_host}:${c.ports[0].external}"
      ) : ""
    })
  }
}

module "challenges" {
  source       = "../modules/challenges"
  challenges   = local.challenges
  network_name = data.docker_network.lab_network.name
  providers = {
    docker = docker
  }
}

module "ctfd" {
  source     = "../modules/ctfd"
  challenges = local.challenges
}

data "docker_network" "lab_network" {
  name = "lab-network"
}

terraform {
  required_providers {
    docker = {
      source = "kreuzwerker/docker"
    }
    ctfd = {
      source  = "ctfer-io/ctfd"
      version = "2.7.5"
    }
  }
}
