locals {
  token_file   = "${path.module}/../ctfd_token.txt"
  ctfd_api_key = fileexists(local.token_file) ? trimspace(file(local.token_file)) : ""
}

provider "ctfd" {
  url     = "http://localhost:8000"
  api_key = local.ctfd_api_key
}
