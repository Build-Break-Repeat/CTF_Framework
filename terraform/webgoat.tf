resource "docker_image" "webgoat" {
  name         = "webgoat/webgoat:latest"
  keep_locally = true
}
resource "docker_container" "webgoat" {
  name  = "webgoat"
  image = docker_image.webgoat.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
