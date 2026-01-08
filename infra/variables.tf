variable "do_token" {
  type      = string
  sensitive = true
}

variable "region" {
  type    = string
  default = "nyc3"
}

variable "size" {
    type = string
    default = "s-1vcpu-1gb-intel"
}

variable "image" {
    type = string
    default = "fedora-42-x64"
}

variable "ssh_key_fingerprint" {
  type = string
}

variable "node_name" {
  type    = string
  default = "dauntless-anchor-01"
}
