locals {
  challenges_list = jsondecode(file("${path.module}/../../challenges.json")).challenges

  challenges = {
    for c in local.challenges_list :
    c.id => c
  }
}

module "challenges" {
  source     = "../modules/challenges"
  challenges = local.challenges
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
