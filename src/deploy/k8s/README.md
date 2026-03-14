# Kubernetes Deployment

Helm chart and manifests for distributed Werd deployment. Planned for Phase 9.

## Strategy

1. Generate baseline manifests from compose via Kompose
2. Refine: add probes, resource limits, Secrets, Ingress
3. Package as Helm chart
4. Use operators for stateful services (CloudNativePG, Redis operator)

## Local Testing with k3s

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable traefik --disable servicelb" sh -
helm install werd ./helm/werd --namespace werd --create-namespace --values values.yaml
```
