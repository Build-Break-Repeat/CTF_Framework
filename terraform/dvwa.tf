resource "docker_image" "dvwa" {
  name         = "vulnerables/web-dvwa:latest"
  keep_locally = true
}

resource "docker_container" "dvwa" {
  name   = "dvwa-challenge"
  image  = docker_image.dvwa.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
