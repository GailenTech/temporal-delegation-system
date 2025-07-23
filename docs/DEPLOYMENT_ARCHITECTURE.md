# Google Cloud Deployment Architecture
## Temporal Purchase Approval System

## Executive Summary

This document provides a comprehensive deployment architecture for the Temporal Purchase Approval System on Google Cloud Platform. The recommended approach uses a **hybrid Cloud Run + GKE architecture** that balances cost-effectiveness for demos with enterprise-grade scalability for production.

## Architecture Overview

### Recommended Hybrid Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                Google Cloud Load Balancer                   │
│                     (Global HTTPS)                         │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
        ▼             ▼             ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│ Cloud Run   │ │ Cloud Run   │ │    GKE      │
│ Web Server  │ │ Workers     │ │ Temporal    │
│(Serverless) │ │(Auto-scale) │ │ Cluster     │
└─────────────┘ └─────────────┘ └─────────────┘
        │               │               │
        └───────────────┼───────────────┘
                        │
                        ▼
              ┌─────────────────┐
              │   Cloud SQL     │
              │  (PostgreSQL)   │
              └─────────────────┘
```

### Component Distribution

**Cloud Run (Serverless)**:
- Web frontend (Go server with HTML templates)
- Worker processes (auto-scaling based on Temporal task queue)
- Cost-effective for variable workloads

**GKE Cluster**:
- Temporal Server (persistent, always-on)
- Elasticsearch (for Temporal UI visibility)
- Temporal UI (web interface)

**Managed Services**:
- Cloud SQL PostgreSQL (Temporal persistence)
- Cloud Load Balancing (traffic distribution)
- Secret Manager (credentials management)

## Cost Analysis

### Environment Comparison

| Environment | Monthly Cost | Use Case | Key Features |
|-------------|--------------|----------|--------------|
| **Demo** | $65-85 | Demonstrations, POCs | Single node, preemptible instances |
| **Staging** | $150-200 | Testing, development | Basic HA, development features |
| **Production** | $500-650 | Live business use | Full HA, monitoring, security |
| **Enterprise** | $1,200-2,000 | Multi-region scale | Global deployment, premium support |

### Cost Breakdown (Production)

| Component | Monthly Cost | Percentage |
|-----------|--------------|------------|
| GKE Cluster (3 nodes) | $150 | 25% |
| Cloud SQL (HA) | $180 | 30% |
| Cloud Run Services | $75 | 12% |
| Load Balancer + SSL | $25 | 4% |
| Storage + Networking | $70 | 12% |
| Monitoring + Ops | $35 | 6% |
| Security + Compliance | $65 | 11% |
| **Total** | **$600** | **100%** |

## Deployment Options

### Option 1: Demo Environment (Recommended Start)

**Target**: Quick demos, POCs, development
**Cost**: $65-85/month
**Setup Time**: 30 minutes with automation

**Configuration**:
- 1 x e2-small GKE node (preemptible)
- db-f1-micro Cloud SQL instance
- Cloud Run with minimal allocation
- Basic monitoring

**Deploy Command**:
```bash
./scripts/deploy-demo.sh your-project-id us-central1
```

### Option 2: Production Environment

**Target**: Live business operations
**Cost**: $500-650/month
**Setup Time**: 2-3 hours with full configuration

**Configuration**:
- 3 x e2-standard-2 GKE nodes (multi-zone)
- db-n1-standard-2 Cloud SQL (HA)
- Cloud Run with auto-scaling
- Full monitoring and alerting

### Option 3: Enterprise Scale

**Target**: Large organizations, multi-region
**Cost**: $1,200-2,000/month
**Setup Time**: Full project setup

**Configuration**:
- Multi-region GKE clusters
- Global load balancing
- Premium support and SLAs
- Advanced security features

## Implementation Guide

### Prerequisites

1. **Google Cloud Project** with billing enabled
2. **Required APIs** enabled:
   - Container Engine API
   - Cloud SQL Admin API
   - Cloud Run API
   - Secret Manager API
   - Cloud Build API

3. **Local Tools**:
   - gcloud CLI
   - kubectl
   - terraform
   - helm
   - docker

### Quick Start Deployment

1. **Clone Repository**:
```bash
git clone <repository-url>
cd temporal-workflow
```

2. **Run Deployment Script**:
```bash
./scripts/deploy-demo.sh your-project-id us-central1
```

3. **Access Applications**:
   - Purchase System: https://your-cloud-run-url
   - Temporal UI: http://temporal-ui-ip:8080

### Manual Deployment Steps

#### Step 1: Infrastructure Setup
```bash
cd terraform-example/environments/demo
terraform init
terraform plan -var="project_id=your-project"
terraform apply
```

#### Step 2: Kubernetes Configuration
```bash
gcloud container clusters get-credentials demo-temporal-cluster --region=us-central1
kubectl create namespace temporal-demo
```

#### Step 3: Temporal Server Deployment
```bash
helm repo add temporalio https://go.temporal.io/helm-charts
helm install temporal-server temporalio/temporal \
  --namespace temporal-demo \
  --values helm/temporal-server/values-demo.yaml
```

#### Step 4: Application Deployment
```bash
# Build and push images
docker build -t gcr.io/PROJECT_ID/temporal-purchase-web:latest -f Dockerfile.web .
docker push gcr.io/PROJECT_ID/temporal-purchase-web:latest

# Deploy to Cloud Run
gcloud run deploy demo-purchase-web \
  --image gcr.io/PROJECT_ID/temporal-purchase-web:latest \
  --region us-central1 \
  --allow-unauthenticated
```

## Security Configuration

### Network Security
- **VPC-native GKE cluster** with private nodes
- **Cloud Armor** for DDoS protection and WAF
- **Network policies** restricting pod communication
- **SSL termination** at load balancer

### Access Control
- **Workload Identity** for pod authentication
- **IAM policies** with least privilege
- **Service accounts** for each component
- **Secret Manager** for credentials

### Compliance Features
- **Audit logging** for all operations
- **Encryption** at rest and in transit
- **Vulnerability scanning** for containers
- **Regular security updates**

## Monitoring and Operations

### Key Metrics
- **Request latency**: < 500ms (95th percentile)
- **Error rate**: < 1% for HTTP 5xx
- **Workflow success rate**: > 99%
- **Resource utilization**: < 80% average

### Alerting
- **Critical alerts**: Page immediately for outages
- **Warning alerts**: Email/Slack for performance issues
- **Uptime monitoring**: External monitoring service

### Operations Tools
- **Cloud Monitoring**: Metrics and dashboards
- **Cloud Logging**: Centralized log management
- **Cloud Trace**: Request tracing
- **Cloud Debugger**: Live debugging

## Scaling Strategy

### Phase 1: Demo (Months 1-2)
- Single environment deployment
- Basic monitoring setup
- Core functionality validation

### Phase 2: Staging (Months 2-3)
- Add staging environment
- CI/CD pipeline implementation
- Integration testing automation

### Phase 3: Production (Months 3-6)
- Production deployment with HA
- Advanced monitoring and alerting
- Security hardening

### Phase 4: Enterprise (Months 6-12)
- Multi-region deployment
- Advanced enterprise features
- Disaster recovery implementation

## Migration Path

### From Docker Compose (Current)
1. **Assessment**: Analyze current Docker setup
2. **Infrastructure**: Deploy GCP infrastructure
3. **Database Migration**: Migrate PostgreSQL to Cloud SQL
4. **Application Migration**: Deploy to Cloud Run
5. **Testing**: Validate functionality
6. **Cutover**: Switch traffic to new environment

### Database Migration
```bash
# Export from local PostgreSQL
pg_dump -h localhost -U temporal temporal > temporal_backup.sql

# Import to Cloud SQL
gcloud sql import sql production-temporal-db gs://your-bucket/temporal_backup.sql \
  --database=temporal
```

## Disaster Recovery

### Backup Strategy
- **Database**: Daily automated backups with 30-day retention
- **Application**: Container images in multiple registries
- **Configuration**: Infrastructure as Code in version control

### Recovery Procedures
- **RTO**: 4 hours for full system recovery
- **RPO**: 1 hour maximum data loss
- **Multi-region**: Active-passive setup for critical environments

## Cost Optimization

### Short-term Savings
- **Preemptible instances** for development (80% cost reduction)
- **Right-sizing** resources based on actual usage
- **Auto-scaling** to match demand

### Long-term Savings
- **Committed use discounts** for predictable workloads
- **Custom machine types** for specific requirements
- **Resource scheduling** for development environments

## Troubleshooting

### Common Issues

**Temporal Server Not Starting**:
```bash
kubectl logs -l app=temporal-server -n temporal-demo
kubectl describe pod -l app=temporal-server -n temporal-demo
```

**Database Connection Issues**:
```bash
kubectl exec -it temporal-server-xxx -- psql -h temporal-db-service -U temporal -d temporal
```

**Cloud Run Cold Starts**:
```bash
gcloud run services update service-name --min-instances=1
```

## Support and Maintenance

### Regular Tasks
- **Weekly**: Health checks and resource monitoring
- **Monthly**: Security updates and certificate renewal
- **Quarterly**: Performance tuning and cost optimization

### Support Channels
- **Level 1**: Internal SRE team
- **Level 2**: Platform engineering team
- **Level 3**: Google Cloud support

## Getting Started Checklist

- [ ] GCP project created with billing enabled
- [ ] Required APIs enabled
- [ ] Local tools installed (gcloud, kubectl, terraform, helm)
- [ ] Repository cloned and configured
- [ ] Environment variables set
- [ ] Deployment script executed
- [ ] Applications accessible and functional
- [ ] Monitoring and alerting configured
- [ ] Security checklist completed
- [ ] Documentation reviewed and understood

## Resources

### Documentation
- [Terraform modules](terraform-example/)
- [Helm charts](helm/)
- [Deployment scripts](scripts/)
- [Security checklist](security/security-checklist.md)
- [Operations runbook](ops/runbook.md)

### Tools
- [Cost calculator](scripts/cost-calculator.py)
- [Health check script](scripts/health-check.sh)
- [Deployment automation](scripts/deploy-demo.sh)

### External Links
- [Temporal.io Documentation](https://docs.temporal.io/)
- [Google Cloud Architecture Center](https://cloud.google.com/architecture)
- [GKE Best Practices](https://cloud.google.com/kubernetes-engine/docs/best-practices)
- [Cloud Run Best Practices](https://cloud.google.com/run/docs/best-practices)