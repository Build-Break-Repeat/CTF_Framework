locals {
  challenges = jsondecode(file("${path.module}/challenges.json")).challenges
}

resource "docker_image" "challenge_images" {
  for_each = local.challenges

  name         = each.value.image
  keep_locally = true
}

resource "docker_container" "challenge_containers" {
  for_each = local.challenges

  name   = each.key
  image  = docker_image.challenge_images[each.key].image_id
  memory = each.value.memory

  networks_advanced {
    name = docker_network.lab_network.name
  }

  command = lookup(each.value, "command", null)

  dynamic "ports" {
    for_each = lookup(each.value, "ports", [])
    content {
      internal = ports.value.internal
      external = ports.value.external
    }
  }

  env = lookup(each.value, "env", [])
}
