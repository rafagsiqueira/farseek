terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "7.14.1"
    }
  }
}

provider "google" {
  project     = "farseek-482420"
}