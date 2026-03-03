resource "docker_image" "metasploitable2" {
  name         = "tleemcjr/metasploitable2:latest"
  keep_locally = true
}

resource "docker_container" "metasploitable2" {
  name  = "metasploitable2"
  image = docker_image.metasploitable2.image_id
  memory = 256

  command = ["bash", "-c", "tail -f /dev/null"]

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
