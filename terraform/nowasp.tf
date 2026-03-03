resource "docker_image" "nowasp" {
  name         = "citizenstig/nowasp:latest"
  keep_locally = true
}

resource "docker_container" "nowasp" {
  name  = "nowasp"
  image = docker_image.nowasp.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
