output "namespace" {
  description = "Namespace MEMBRANE.AI was deployed into."
  value       = kubernetes_namespace.membrane.metadata[0].name
}

output "release_name" {
  description = "Helm release name."
  value       = helm_release.membrane.name
}

output "posture" {
  description = "Deployment posture this release was provisioned for."
  value       = var.posture
}
