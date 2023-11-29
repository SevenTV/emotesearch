terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.7.0"
    }

    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.18.1"
    }
  }
}

data "kubernetes_namespace" "app" {
  metadata {
    name = var.namespace
  }
}

// Define config secret
resource "kubernetes_secret" "app" {
  metadata {
    name      = "emotesearch"
    namespace = var.namespace
  }

  data = {
    "config.yaml" = templatefile("${path.module}/config.template.yaml", {
      mongo_uri        = var.infra.mongodb_uri
      mongo_username   = var.infra.mongodb_user_app.username
      mongo_password   = var.infra.mongodb_user_app.password
      mongo_database   = "7tv"
      mongo_collection = "emotes"
      meili_url        = "http://meilisearch.database.svc.cluster.local:7700"
      meili_key        = var.meilisearch_key
      meili_index      = "emotes"
    })
  }
}

resource "kubernetes_deployment" "app" {
  metadata {
    name      = "emotesearch"
    namespace = data.kubernetes_namespace.app.metadata[0].name
    labels    = {
      app = "emotesearch"
    }
  }

  lifecycle {
    replace_triggered_by = [kubernetes_secret.app]
  }

  timeouts {
    create = "4m"
    update = "2m"
    delete = "2m"
  }

  spec {
    selector {
      match_labels = {
        app = "emotesearch"
      }
    }

    replicas = 1

    template {
      metadata {
        labels = {
          app = "emotesearch"
        }
      }

      spec {
        container {
          name  = "emotesearch"
          image = local.image_url

          resources {
            requests = {
              cpu    = "100m"
              memory = "100Mi"
            }
            limits = {
              cpu    = "1"
              memory = "1Gi"
            }
          }

          volume_mount {
            name       = "config"
            mount_path = "/app/config.yaml"
            sub_path   = "config.yaml"
          }

          image_pull_policy = var.image_pull_policy
        }

        volume {
          name = "config"
          secret {
            secret_name = kubernetes_secret.app.metadata[0].name
          }
        }
      }
    }
  }
}