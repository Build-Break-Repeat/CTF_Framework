locals {
  ctfd_api_key = trimspace(file("${path.module}/../ctfd_token.txt"))
}

provider "ctfd" {
  url     = "http://localhost:8000"
  api_key = local.ctfd_api_key
}
