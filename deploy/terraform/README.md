# Terraform — MEMBRANE.AI deployment

A cloud-neutral module that deploys the Helm chart (`../helm/membrane`, D-030)
onto an existing Kubernetes cluster. It owns the **namespace + Helm release**;
the cluster itself and the external Kafka/Redis/Postgres are the platform
layer's concern, so the same module serves all three postures (D-011).

| Posture | How |
| --- | --- |
| **Masked SaaS** | Point `kafka_brokers` / `redis_addr` / `database_url` at managed services; `image_registry` at your registry. |
| **Private VPC** | Same, inside the customer's VPC; `otlp_endpoint` at their collector. |
| **Air-gapped** | Mirror images to an internal registry, set `image_registry`; all endpoints internal; leave `github_token` empty. |

## Use

```sh
cd deploy/terraform
terraform init
terraform apply \
  -var 'kafka_brokers=kafka:9092' \
  -var 'redis_addr=redis:6379' \
  -var 'database_url=postgres://membrane:membrane@postgres:5432/membrane' \
  -var 'posture=vpc'
```

Apply the SQL migrations (`../migrations/*.up.sql`) to the database before first
use — there is no in-chart migration job (pre-1.0 policy D-022).

Validate locally without a cluster: `terraform init -backend=false && terraform validate`.
