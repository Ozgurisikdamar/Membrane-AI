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
  # Defaults mirror values.yaml, so a bare apply renders like a bare helm install.
  set = [
    { name = "image.registry", value = var.image_registry },
    { name = "image.tag", value = var.image_tag },
    { name = "infra.kafkaBrokers", value = var.kafka_brokers },
    { name = "infra.redisAddr", value = var.redis_addr },
    { name = "infra.otlpEndpoint", value = var.otlp_endpoint },
    { name = "infra.otlpInsecure", value = tostring(var.otlp_insecure) },
    { name = "infra.vllmUrl", value = var.vllm_url },
    { name = "infra.vllmModel", value = var.vllm_model },
    { name = "infra.githubApiUrl", value = var.github_api_url },
    { name = "infra.enforcementMode", value = var.enforcement_mode },
    { name = "infra.premiumEnabled", value = tostring(var.premium_enabled) },
    { name = "infra.premiumAnthropicModel", value = var.premium_anthropic_model },
    { name = "infra.premiumGeminiModel", value = var.premium_gemini_model },
  ]

  # Credentials land in the chart's optional Secret; marked sensitive so plans
  # never print them. Each is optional (D-030): empty disables that feature.
  set_sensitive = [
    { name = "credentials.databaseUrl", value = var.database_url },
    { name = "credentials.githubToken", value = var.github_token },
    { name = "credentials.webhookSecret", value = var.webhook_secret },
    { name = "credentials.webhookUrl", value = var.webhook_url },
    { name = "credentials.anthropicApiKey", value = var.anthropic_api_key },
    { name = "credentials.geminiApiKey", value = var.gemini_api_key },
  ]
}
