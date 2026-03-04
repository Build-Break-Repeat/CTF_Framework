resource "docker_image" "ctfd" {
  name         = "ctfd/ctfd:latest"
  keep_locally = true
}

resource "docker_container" "ctfd" {
  name   = "ctfd-scoreboard"
  image  = docker_image.ctfd.image_id
  memory = 512

  networks_advanced {
    name = docker_network.lab_network.name
  }
  ports {
    internal = 8000
    external = 8000
  }

}