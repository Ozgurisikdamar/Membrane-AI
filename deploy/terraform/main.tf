# Deploys the MEMBRANE.AI Helm chart (D-030) onto an existing cluster. This
# module owns the namespace + release only; provisioning the cluster and the
# external Kafka/Redis/Postgres is left to the platform layer (cloud-neutral by
# design, D-011). For an air-gapped posture, mirror the images to an internal
# registry and set image_registry accordingly.

provider "kubernetes" {
  config_path    = var.kube_config_path
  config_context = var.kube_context != "" ? var.kube_context : null
}

provider "helm" {
  kubernetes = {
    config_path    = var.kube_config_path
    config_context = var.kube_context != "" ? var.kube_context : null
  }
}

resource "kubernetes_namespace" "membrane" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/part-of"      = "membrane"
      "membrane.ai/deployment-posture" = var.posture
    }
  }
}

resource "helm_release" "membrane" {
  name      = var.release_name
  namespace = kubernetes_namespace.membrane.metadata[0].name
  chart     = "${path.module}/../helm/membrane"

  # Render the chart's values from the module inputs (helm provider v3 list syntax).
  set = [
    { name = "image.registry", value = var.image_registry },
    { name = "image.tag", value = var.image_tag },
    { name = "infra.kafkaBrokers", value = var.kafka_brokers },
    { name = "infra.redisAddr", value = var.redis_addr },
    { name = "infra.otlpEndpoint", value = var.otlp_endpoint },
  ]

  # Credentials land in the chart's optional Secret; marked sensitive so plans
  # never print them.
  set_sensitive = [
    { name = "credentials.databaseUrl", value = var.database_url },
    { name = "credentials.githubToken", value = var.github_token },
    { name = "credentials.webhookSecret", value = var.webhook_secret },
  ]
}
