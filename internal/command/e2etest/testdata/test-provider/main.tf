terraform {
  required_providers {
    simple = {
      source = "hashicorp/simple"
    }
  }
}

resource "simple_resource" "test" {
}
