resource "docker_image" "bwapp" {
  name         = "raesene/bwapp:latest"
  keep_locally = true
}
resource "docker_container" "bwapp" {
  name   = "bwapp"
  image  = docker_image.bwapp.image_id
  memory = 256

  ports {
    internal = 80
    external = 8002
  }

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
