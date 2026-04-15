terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.6"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

resource "random_password" "db_password" {
  length  = 32
  special = false
}

resource "docker_network" "lab_network" {
  name = "lab-network"
}

resource "docker_container" "ctfd" {
  name  = "ctfd"
  image = "ctfd/ctfd:latest"

  networks_advanced {
    name = docker_network.lab_network.name
  }
  ports {
    internal = 8000
    external = 8000
    ip       = "127.0.0.1"
  }

  env = [
    "DATABASE_URL=mysql+pymysql://ctfd:${random_password.db_password.result}@ctfd-db/ctfd"
  ]

  depends_on = [docker_container.ctfd_db]
}

resource "docker_volume" "caddy_data" {
  name = "caddy-data"
}

resource "docker_container" "caddy" {
  name  = "caddy"
  image = "caddy:latest"

  networks_advanced {
    name = docker_network.lab_network.name
  }

  ports {
    internal = 80
    external = 80
  }

  ports {
    internal = 443
    external = 443
  }

  volumes {
    volume_name    = docker_volume.caddy_data.name
    container_path = "/data"
  }

  upload {
    content = <<-EOT
      :443 {
          reverse_proxy ctfd:8000
          tls internal {
              on_demand
          }
      }

      :80 {
          redir https://{host}{uri} permanent
      }
    EOT
    file    = "/etc/caddy/Caddyfile"
  }

  depends_on = [docker_container.ctfd]
}

resource "docker_container" "ctfd_db" {
  name  = "ctfd-db"
  image = "mariadb:10"

  networks_advanced {
    name = docker_network.lab_network.name
  }

  env = [
    "MARIADB_DATABASE=ctfd",
    "MARIADB_USER=ctfd",
    "MARIADB_PASSWORD=${random_password.db_password.result}",
    "MARIADB_RANDOM_ROOT_PASSWORD=yes"
  ]
}

