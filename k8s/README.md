# Pantheon Base Kubernetes Deployment

Production-ready Kubernetes manifests for deploying Pantheon Base to any Kubernetes cluster.

## 📋 Prerequisites

- **Kubernetes cluster**: 1.24+ (tested on 1.28)
- **kubectl**: Configured and authenticated
- **NGINX Ingress Controller**: Installed in cluster
- **cert-manager**: For automatic TLS certificate management (recommended)
- **External dependencies**:
  - MySQL 8.0+ (managed service or external instance)
  - Redis 7.0+ (managed service or external instance)

## 🚀 Quick Start

### 1. Create Namespace

```bash
kubectl apply -f namespace.yaml
```

### 2. Configure Secrets

**Important**: Never commit actual secrets to version control.

```bash
# Copy the example
cp secret.yaml.example secret.yaml

# Edit with your actual credentials
vim secret.yaml

# Apply the secret
kubectl apply -f secret.yaml
```

**Required secrets**:
- `PANTHEON_DSN`: MySQL connection string
- `PANTHEON_REDIS_ADDR`: Redis host:port
- `PANTHEON_REDIS_PASSWORD`: Redis password
- `PANTHEON_METRICS_BEARER_TOKEN`: Metrics endpoint auth token

### 3. Apply ConfigMap

```bash
kubectl apply -f configmap.yaml
```

**Review configuration** in `configmap.yaml` before applying. Key settings:
- `PANTHEON_ENV`: Set to `production`
- `PANTHEON_AUTO_MIGRATE`: Set to `false` (run migrations manually)
- Rate limiting: Adjust `PANTHEON_API_RATE_LIMIT_MAX` based on load

### 4. Deploy Backend

```bash
kubectl apply -f backend/
```

This creates:
- **Deployment**: 3 replicas with rolling update strategy
- **Service**: ClusterIP service on port 8080
- **HPA**: Autoscaler (3-10 replicas based on CPU/memory)

### 5. Configure Ingress

**Edit `ingress.yaml`**:
- Replace `pantheon.example.com` with your actual domain
- Update `cert-manager.io/cluster-issuer` if using different issuer
- Adjust rate limiting annotations if needed

```bash
kubectl apply -f ingress.yaml
```

### 6. Verify Deployment

```bash
# Check all resources
kubectl get all -n pantheon

# Check pods
kubectl get pods -n pantheon
# Expected: 3 pods in Running state

# Check service
kubectl get svc -n pantheon
# Expected: pantheon-backend ClusterIP

# Check ingress
kubectl get ingress -n pantheon
# Expected: pantheon-ingress with your domain

# View logs
kubectl logs -n pantheon -l app=pantheon-backend --tail=100

# Test health endpoint
curl https://pantheon.example.com/health
# Expected: {"status":"ok"}
```

## 📊 Resource Requirements

### Per Pod

| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 250m | 1000m (1 core) |
| Memory | 512Mi | 1Gi |

### Cluster Sizing Recommendations

**Small** (Development/Staging):
- 3 nodes: 2 vCPU, 4GB RAM each
- Total: 6 vCPU, 12GB RAM

**Medium** (Production - Low/Medium Traffic):
- 5 nodes: 4 vCPU, 8GB RAM each
- Total: 20 vCPU, 40GB RAM
- Supports up to 10 pods (autoscaling)

**Large** (Production - High Traffic):
- 10+ nodes: 8 vCPU, 16GB RAM each
- Use autoscaling node groups
- Supports 20+ pods

## 🔄 Operations

### Scaling

**Manual Scaling**:
```bash
kubectl scale deployment pantheon-backend -n pantheon --replicas=5
```

**Autoscaling**:

HPA is configured to scale 3-10 replicas based on:
- CPU utilization: 70% threshold
- Memory utilization: 80% threshold

```bash
# Check HPA status
kubectl get hpa -n pantheon

# Describe autoscaler
kubectl describe hpa pantheon-backend-hpa -n pantheon
```

### Updating

**Update to new version**:
```bash
# Option 1: Set image directly
kubectl set image deployment/pantheon-backend \
  backend=ghcr.io/duanxldragon/pantheon-base/backend:v0.11.2 \
  -n pantheon

# Option 2: Edit deployment
kubectl edit deployment pantheon-backend -n pantheon

# Watch rollout
kubectl rollout status deployment/pantheon-backend -n pantheon
```

**Rollback**:
```bash
# Rollback to previous version
kubectl rollout undo deployment/pantheon-backend -n pantheon

# Rollback to specific revision
kubectl rollout history deployment/pantheon-backend -n pantheon
kubectl rollout undo deployment/pantheon-backend -n pantheon --to-revision=2
```

### Restarting

```bash
# Rolling restart (zero downtime)
kubectl rollout restart deployment/pantheon-backend -n pantheon
```

## 🔍 Monitoring & Debugging

### Logs

```bash
# Tail logs from all pods
kubectl logs -n pantheon -l app=pantheon-backend --tail=100 -f

# Logs from specific pod
kubectl logs -n pantheon pantheon-backend-xxxxx-yyyyy --tail=200

# Previous container logs (if pod crashed)
kubectl logs -n pantheon pantheon-backend-xxxxx-yyyyy --previous
```

### Metrics

Metrics are exposed at `/metrics` endpoint (protected by bearer token):

```bash
# Port forward to local machine
kubectl port-forward -n pantheon svc/pantheon-backend 8080:8080

# Access metrics (replace TOKEN)
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/metrics
```

### Resource Usage

```bash
# Pod resource usage
kubectl top pods -n pantheon

# Node resource usage
kubectl top nodes

# Detailed pod info
kubectl describe pod -n pantheon -l app=pantheon-backend
```

### Events

```bash
# Recent events in namespace
kubectl get events -n pantheon --sort-by='.lastTimestamp'

# Watch events live
kubectl get events -n pantheon --watch
```

## 🔧 Troubleshooting

### Pod Not Starting

**Symptoms**: Pods in `Pending`, `CrashLoopBackOff`, or `Error` state.

**Diagnosis**:
```bash
# Describe pod to see events
kubectl describe pod -n pantheon pantheon-backend-xxxxx-yyyyy

# Check logs
kubectl logs -n pantheon pantheon-backend-xxxxx-yyyyy

# Check previous logs if crashed
kubectl logs -n pantheon pantheon-backend-xxxxx-yyyyy --previous
```

**Common causes**:
- Database connection failure (check `PANTHEON_DSN` in secret)
- Redis connection failure (check `PANTHEON_REDIS_ADDR`)
- Insufficient resources (check node capacity)
- Image pull errors (check image name and registry access)

### Database Connection Issues

```bash
# Verify secret
kubectl get secret pantheon-secret -n pantheon -o yaml

# Decode secret (for debugging)
kubectl get secret pantheon-secret -n pantheon -o jsonpath='{.data.PANTHEON_DSN}' | base64 -d

# Test connection from pod
kubectl exec -it -n pantheon deployment/pantheon-backend -- sh
# Inside pod:
nc -zv mysql-host.example.com 3306
```

### Ingress Not Working

```bash
# Check ingress status
kubectl describe ingress pantheon-ingress -n pantheon

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=100

# Verify DNS
nslookup pantheon.example.com

# Test service directly (bypass ingress)
kubectl port-forward -n pantheon svc/pantheon-backend 8080:8080
curl http://localhost:8080/health
```

### TLS Certificate Issues

```bash
# Check certificate
kubectl get certificate -n pantheon

# Describe certificate
kubectl describe certificate pantheon-tls -n pantheon

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager --tail=100
```

### High CPU/Memory Usage

```bash
# Check resource usage
kubectl top pods -n pantheon

# Check if HPA is scaling
kubectl get hpa -n pantheon

# Review limits
kubectl get deployment pantheon-backend -n pantheon -o yaml | grep -A 5 resources
```

## 🔐 Security Notes

### Secret Management

**Production recommendations**:
- Use **Sealed Secrets** (bitnami-labs/sealed-secrets)
- Use **External Secrets Operator** (external-secrets/external-secrets)
- Use cloud provider secret management (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault)

**Never**:
- Commit actual secrets to git
- Store secrets in plain text
- Share secrets via insecure channels

### Network Security

**Recommended**:
```bash
# Add NetworkPolicy to restrict pod-to-pod traffic
kubectl apply -f network-policy.yaml
```

### RBAC

Follow least-privilege principle:
- Service accounts should have minimal permissions
- Use separate service accounts for different components
- Review ClusterRole bindings regularly

### Image Security

**Best practices**:
- Scan images for vulnerabilities before deployment
- Use specific image tags (not `:latest`)
- Pull from trusted registries only
- Enable image pull secrets if using private registry

## 🚀 CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy to Kubernetes

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Configure kubectl
        uses: azure/k8s-set-context@v3
        with:
          method: kubeconfig
          kubeconfig: ${{ secrets.KUBE_CONFIG }}

      - name: Deploy to cluster
        run: |
          kubectl apply -f k8s/namespace.yaml
          kubectl apply -f k8s/configmap.yaml
          kubectl apply -f k8s/backend/
          kubectl set image deployment/pantheon-backend \
            backend=ghcr.io/duanxldragon/pantheon-base/backend:${{ github.ref_name }} \
            -n pantheon
          kubectl rollout status deployment/pantheon-backend -n pantheon --timeout=5m
```

## 🗄️ Database Migrations

**Important**: Always run migrations manually before deploying new versions.

```bash
# Get a shell in a pod
kubectl exec -it -n pantheon deployment/pantheon-backend -- sh

# Inside pod, run migration tool (if available)
# OR connect to database directly and run migration SQL
```

**Recommendation**: Use a separate Job for migrations:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: pantheon-migration
  namespace: pantheon
spec:
  template:
    spec:
      containers:
      - name: migration
        image: ghcr.io/duanxldragon/pantheon-base/backend:v0.11.2
        command: ["/app/migrate"]
        envFrom:
        - secretRef:
            name: pantheon-secret
      restartPolicy: Never
```

## 📝 Maintenance

### Regular Tasks

**Weekly**:
- Review pod logs for errors
- Check resource usage trends
- Review HPA scaling events

**Monthly**:
- Update Kubernetes cluster
- Update application images
- Review and rotate secrets
- Check for security updates

**Quarterly**:
- Review and optimize resource limits
- Audit RBAC permissions
- Review network policies
- Update disaster recovery procedures

## 🆘 Support

For issues or questions:
- **Documentation**: [docs/DEPLOYMENT_GUIDE.md](../docs/DEPLOYMENT_GUIDE.md)
- **Architecture**: [docs/designs/](../docs/designs/)
- **GitHub Issues**: https://github.com/duanxldragon/pantheon-base/issues

## 📚 Additional Resources

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Horizontal Pod Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
