#platform/provider terraform will build with (ie docker): https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs

terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.6.2"
    }
  }
}


#Internal network for the lab
#######################################################################################################################
resource "docker_network" "lab_network" {
  name = "lab-network"
}


#Kali desktop image (HTTPS access) (Keep_Locally forces terraform to keep images and not delete after destroy operation)
#######################################################################################################################
resource "docker_image" "kali" {
  name         = "kasmweb/kali-rolling-desktop:1.16.0"
  keep_locally = true
}

#Create and assign settings to docker container from kali image
#######################################################################################################################
resource "docker_container" "kali_desktop" {
  name  = "kali-desktop"
  image = docker_image.kali.image_id
  memory = 512

  #Attach to internal network
  networks_advanced {
    name = docker_network.lab_network.name
  }

  #HTTPS VNC access
  ports {
    internal = 6901
    external = 6901
  }

  #set VNC password to "password" - by default username is "kasm_user"
  env = [
    "VNC_PW=password"
  ]


  #Mount SSL certificate and key for HTTPS (http does not work, certs required for https)
  volumes {
    host_path      = abspath("${path.module}/certs/self.pem")
    container_path = "/dockerstartup/vnc_ssl/self.pem"
  }

  volumes {
    host_path      = abspath("${path.module}/certs/self.key")
    container_path = "/dockerstartup/vnc_ssl/self.key"
  }
}


# Image/container template (internal network)
#######################################################################################################################
#resource "docker_image" "#RESOURCENAME" {
#  name         = "#PICKEDIMAGETOBUILDFROM"
#  keep_locally = true
#}

#resource "docker_container" "#RESOURCENAME" {
#  name  = "#NAMECONTAINERAFTERCREATE"
#  image = docker_image.#RESOURCENAME.image_id
#
#  networks_advanced {
#    name = docker_network.lab_network.name
#  }
#}
