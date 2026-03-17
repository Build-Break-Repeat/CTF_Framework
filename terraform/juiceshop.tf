resource "docker_image" "juice_shop" {
  name         = "bkimminich/juice-shop:latest"
  keep_locally = true
}
resource "docker_container" "juice_shop" {
  name  = "juice-shop"
  image = docker_image.juice_shop.image_id
  memory = 256

  ports {
    internal = 3000
    external = 8003
  }

  networks_advanced {
    name = docker_network.lab_network.name
  }
}
