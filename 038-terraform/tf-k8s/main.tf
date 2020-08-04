provider "kubernetes" {
  config_context = "microk8s" 
  load_config_file = "false"
  host = "https://192.168.200.145:16443"
  client_certificate     = "${file("~/.kube/server.crt")}"
  client_key             = "${file("~/.kube/server.key")}"
  cluster_ca_certificate = "${file("~/.kube/ca.crt")}"
}

resource "kubernetes_deployment" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        test = "MyExampleApp"
      }
    }

    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
      }

      spec {
        container {
          image = "nginx:1.7.8"
          name  = "example"
        }
      }
    }
  }
}
