resource "docker_image" "sqli_labs" {
  name         = "acgpiano/sqli-labs:latest"
  keep_locally = true
}

resource "docker_container" "sqli_labs" {
  name  = "sqli-labs"
  image = docker_image.sqli_labs.image_id
  memory = 256

  ports {
    internal = 80
    external = 8006
  }

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
