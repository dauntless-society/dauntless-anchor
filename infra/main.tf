resource "digitalocean_droplet" "anchor" {
  name   = var.node_name
  region = var.region
  size   = var.size
  image = var.image

  ssh_keys = [
    var.ssh_key_fingerprint
  ]

  user_data = file("${path.module}/cloud-init/anchor-node.yml")

  monitoring = true
#  project_id = "d2700191d-b95c-4ef1-aa4d-4c1f4c03a7f6"
}
