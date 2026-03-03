resource "docker_image" "kali" {
  name         = "kasmweb/kali-rolling-desktop:1.16.0"
  keep_locally = true
}

resource "docker_container" "kali_desktop" {
  name   = "kali-desktop"
  image  = docker_image.kali.image_id
  memory = 512

  networks_advanced {
    name = docker_network.lab_network.name
  }

  ports {
    internal = 6901
    external = 6901
  }

  env = [
    "VNC_PW=password"
  ]

  volumes {
    host_path      = abspath("${path.module}/certs/self.pem")
    container_path = "/dockerstartup/vnc_ssl/self.pem"
  }

  volumes {
    host_path      = abspath("${path.module}/certs/self.key")
    container_path = "/dockerstartup/vnc_ssl/self.key"
  }
}
