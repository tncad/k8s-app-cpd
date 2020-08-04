# provider with remote API enabled
provider "docker" {
  host = "http://127.0.0.1:2375"
}

# image resource
resource "docker_image" "nginx" {
  name         = "nginx:latest"
  keep_locally = false
}

# container resource
resource "docker_container" "nginx" {
  image = "${docker_image.nginx.latest}"
  name  = "tutorial"
  ports {
    internal = 80
    external = 8000
  }
}

