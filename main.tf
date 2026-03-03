#platform/provider terraform will build with (ie docker): https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs

terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.6"
    }
  }
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
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

#DVWA Container
#######################################################################################################################
resource "docker_image" "dvwa" {
  name         = "vulnerables/web-dvwa:latest"
  keep_locally = true
}

resource "docker_container" "dvwa" {
  name  = "dvwa-challenge"
  image = docker_image.dvwa.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}

#SQLI-Labs Container
#######################################################################################################################
resource "docker_image" "sqli_labs" {
  name         = "acgpiano/sqli-labs:latest"
  keep_locally = true
}

resource "docker_container" "sqli_labs" {
  name  = "sqli-labs"
  image = docker_image.sqli_labs.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}

#NOWASP Container
#######################################################################################################################
resource "docker_image" "nowasp" {
  name         = "citizenstig/nowasp:latest"
  keep_locally = true
}

resource "docker_container" "nowasp" {
  name  = "nowasp"
  image = docker_image.nowasp.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}

#Metasploitable2 Container
#######################################################################################################################
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

#OWASP Juice Shop
#######################################################################################################################
resource "docker_image" "juice_shop" {
  name         = "bkimminich/juice-shop:latest"
  keep_locally = true
}
resource "docker_container" "juice_shop" {
  name  = "juice-shop"
  image = docker_image.juice_shop.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
  }
}

#WebGoat Container
#######################################################################################################################
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

#bWAPP Container
#######################################################################################################################
resource "docker_image" "bwapp" {
  name         = "raesene/bwapp:latest"
  keep_locally = true
}
resource "docker_container" "bwapp" {
  name  = "bwapp"
  image = docker_image.bwapp.image_id
  memory = 256

  networks_advanced {
    name = docker_network.lab_network.name
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
