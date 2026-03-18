resource "docker_image" "nginx" {
  name = "nginx:latest"
}

resource "docker_container" "nginx_static" {
  name  = "nginx-static"
  image = docker_image.nginx.image_id

  ports {
    internal = 80
    external = 8080
  }

  volumes {
    host_path      = abspath("${path.module}/../registration-site")
    container_path = "/usr/share/nginx/html"
    read_only      = true
  }

  restart = "unless-stopped"
}
