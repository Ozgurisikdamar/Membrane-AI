# Provider-agnostic inputs (D-011): the same module deploys all three postures —
# Masked SaaS, Private VPC, Air-gapped on-prem — onto any Kubernetes. Infra
# (Kafka/Redis/Postgres/OTLP) is referenced as external endpoints (D-030), so
# the module stays cloud-neutral: point them at managed services or in-cluster.

variable "kube_config_path" {
  description = "Path to the kubeconfig used to reach the target cluster."
  type        = string
  default     = "~/.kube/config"
}

variable "kube_context" {
  description = "kubeconfig context to use (empty = current-context)."
  type        = string
  default     = ""
}

variable "namespace" {
  description = "Namespace to create and deploy MEMBRANE.AI into."
  type        = string
  default     = "membrane"
}

variable "release_name" {
  description = "Helm release name."
  type        = string
  default     = "membrane"
}

variable "posture" {
  description = "Deployment posture (D-011): saas | vpc | airgapped. Informational; surfaced as a label."
  type        = string
  default     = "vpc"
  validation {
    condition     = contains(["saas", "vpc", "airgapped"], var.posture)
    error_message = "posture must be one of: saas, vpc, airgapped."
  }
}

variable "image_registry" {
  description = "Registry/namespace prefix for the service images (empty = bare local names)."
  type        = string
  default     = ""
}

variable "image_tag" {
  description = "Image tag for all services."
  type        = string
  default     = "dev"
}

variable "kafka_brokers" {
  description = "External Kafka bootstrap brokers (host:port)."
  type        = string
}

variable "redis_addr" {
  description = "External Redis address (host:port)."
  type        = string
}

variable "database_url" {
  description = "Postgres/pgvector DSN for orchestrator + resolver."
  type        = string
  sensitive   = true
}

variable "otlp_endpoint" {
  description = "OTLP/gRPC collector address for tracing (empty = tracing off)."
  type        = string
  default     = ""
}

variable "otlp_insecure" {
  description = "Send traces over plaintext gRPC (dev collectors only)."
  type        = bool
  default     = false
}

variable "vllm_url" {
  description = "Tier-2 vLLM endpoint (OpenAI-compatible); empty = heuristic only."
  type        = string
  default     = ""
}

variable "vllm_model" {
  description = "Tier-2 vLLM model name."
  type        = string
  default     = "deepseek-coder"
}

variable "github_api_url" {
  description = "GitHub API base URL for the reporter (only for GitHub Enterprise)."
  type        = string
  default     = ""
}

variable "enforcement_mode" {
  description = "Reporter merge-gate posture: enforce (rejected blocks) | shadow (non-blocking)."
  type        = string
  default     = "enforce"
  validation {
    condition     = contains(["enforce", "shadow"], var.enforcement_mode)
    error_message = "enforcement_mode must be one of: enforce, shadow."
  }
}

variable "premium_enabled" {
  description = "Enable the tier-3 premium consensus (also needs a provider API key)."
  type        = bool
  default     = false
}

variable "premium_anthropic_model" {
  description = "Tier-3 Anthropic model id."
  type        = string
  default     = "claude-sonnet-4-6"
}

variable "premium_gemini_model" {
  description = "Tier-3 Gemini model id."
  type        = string
  default     = "gemini-2.5-pro"
}

variable "github_token" {
  description = "GitHub token for the reporter notifiers (optional)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "webhook_secret" {
  description = "HMAC secret for the ingestion Git webhook (optional but recommended)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "webhook_url" {
  description = "Slack-compatible webhook URL for the reporter (secret-bearing)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "anthropic_api_key" {
  description = "Tier-3 Anthropic API key (premium consensus)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "gemini_api_key" {
  description = "Tier-3 Gemini API key (premium consensus)."
  type        = string
  default     = ""
  sensitive   = true
}
