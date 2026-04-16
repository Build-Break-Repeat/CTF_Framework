variable "challenges" {
  type = any
}

resource "ctfd_challenge_standard" "dynamic" {
  for_each = var.challenges

  name     = each.key
  category = lookup(each.value, "category", "General")
  description = format(
    "## %s\n\n%s",
    lookup(each.value, "name", each.key),
    lookup(each.value, "description", "")
  )
  value = lookup(each.value, "points", 100)

  state        = lookup(each.value, "state", "visible")
  max_attempts = lookup(each.value, "max_attempts", 0)
}

resource "ctfd_flag" "flags" {
  for_each = var.challenges

  challenge_id = ctfd_challenge_standard.dynamic[each.key].id
  content      = trimspace(file("${path.module}/../../../flags/${each.key}.txt"))
}

terraform {
  required_providers {
    ctfd = {
      source = "ctfer-io/ctfd"
    }
  }
}
