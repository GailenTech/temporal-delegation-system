# Operations Runbook: Temporal Purchase Approval System on GCP

## System Overview

The Temporal Purchase Approval System consists of:
- **Temporal Server**: Running on GKE cluster for workflow orchestration
- **Web Frontend**: Deployed on Cloud Run for user interface
- **Workers**: Deployed on Cloud Run for activity execution
- **Database**: Cloud SQL PostgreSQL for persistence
- **Search**: Elasticsearch for Temporal UI visibility

## 🚨 Emergency Procedures

### System Down (Complete Outage)

**Symptoms**: No response from web application, Temporal UI inaccessible
**Impact**: Users cannot submit or approve purchase requests

**Immediate Actions (5 minutes)**:
1. Check overall system status:
   ```bash
   # Check Cloud Run services
   gcloud run services list --region=us-central1
   
   # Check GKE cluster status
   kubectl get nodes
   kubectl get pods -n temporal-production
   ```

2. Check load balancer and ingress:
   ```bash
   # Check load balancer health
   gcloud compute backend-services get-health temporal-lb-backend
   ```

3. If GKE cluster is down:
   ```bash
   # Check cluster status
   gcloud container clusters describe production-temporal-cluster --region=us-central1
   
   # Scale up if needed
   gcloud container clusters resize production-temporal-cluster --num-nodes=3 --region=us-central1
   ```

4. If Cloud Run services are down:
   ```bash
   # Check service status
   gcloud run services describe production-purchase-web --region=us-central1
   
   # Force new deployment if needed
   gcloud run services update production-purchase-web --region=us-central1
   ```

### Database Issues

**Symptoms**: Application errors, workflow failures, data inconsistency
**Impact**: Cannot persist workflow state, data loss risk

**Immediate Actions**:
1. Check Cloud SQL instance status:
   ```bash
   gcloud sql instances describe production-temporal-db
   ```

2. Check database connections:
   ```bash
   # From a pod in the cluster
   kubectl exec -it temporal-server-xxx -- psql -h temporal-db-service -U temporal -d temporal -c "SELECT 1;"
   ```

3. If connection issues:
   ```bash
   # Restart Cloud SQL proxy
   kubectl rollout restart deployment/cloud-sql-proxy -n temporal-production
   ```

4. If performance issues:
   ```bash
   # Check database metrics
   gcloud sql instances describe production-temporal-db --format="table(currentDiskSize,maxDiskSize,gceZone,settings.tier)"
   ```

### High CPU/Memory Usage

**Symptoms**: Slow response times, workflow timeouts
**Impact**: Degraded performance, potential service failures

**Immediate Actions**:
1. Check resource usage:
   ```bash
   # Check GKE node resources
   kubectl top nodes
   kubectl top pods -n temporal-production
   
   # Check Cloud Run metrics
   gcloud run services describe production-purchase-web --region=us-central1 --format="table(metadata.name,status.traffic[].latestRevision)"
   ```

2. Scale up if needed:
   ```bash
   # Scale GKE nodes
   gcloud container clusters resize production-temporal-cluster --num-nodes=5 --region=us-central1
   
   # Increase Cloud Run resources
   gcloud run services update production-purchase-web --memory=2Gi --cpu=2 --region=us-central1
   ```

## 📊 Monitoring and Alerting

### Key Metrics to Monitor

#### Application Metrics
- **Request latency**: < 500ms for 95th percentile
- **Error rate**: < 1% for HTTP 5xx errors
- **Workflow success rate**: > 99%
- **Database connection pool**: < 80% utilization

#### Infrastructure Metrics
- **GKE node CPU**: < 70% average
- **GKE node memory**: < 80% average
- **Cloud SQL CPU**: < 70% average
- **Cloud SQL memory**: < 80% average
- **Disk usage**: < 80% of allocated space

### Monitoring Queries

#### Cloud Monitoring Queries
```sql
-- Request latency (Cloud Run)
fetch cloud_run_revision
| metric 'run.googleapis.com/request_latencies'
| filter resource.service_name == 'production-purchase-web'
| group_by 1m, [percentile(value.request_latencies, 95)]

-- Error rate (Cloud Run)
fetch cloud_run_revision
| metric 'run.googleapis.com/request_count'
| filter resource.service_name == 'production-purchase-web'
| filter metric.response_code_class != '2xx'
| group_by 1m, [sum(value.request_count)]

-- Database connections (Cloud SQL)
fetch cloudsql_database
| metric 'cloudsql.googleapis.com/database/postgresql/num_backends'
| filter resource.database_id == 'production-temporal-db'
| group_by 1m, [mean(value.num_backends)]
```

### Alert Configurations

#### Critical Alerts (Page immediately)
```yaml
# High error rate
alertPolicy:
  displayName: "High Error Rate - Purchase Web"
  conditions:
  - displayName: "Error rate > 5%"
    conditionThreshold:
      filter: 'resource.type="cloud_run_revision" AND resource.labels.service_name="production-purchase-web"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 0.05
      duration: 300s
  notificationChannels:
  - projects/PROJECT_ID/notificationChannels/pager-duty

# Database down
alertPolicy:  
  displayName: "Cloud SQL Instance Down"
  conditions:
  - displayName: "Instance state != RUNNABLE"
    conditionThreshold:
      filter: 'resource.type="cloudsql_database" AND resource.labels.database_id="production-temporal-db"'
      comparison: COMPARISON_NOT_EQUAL
      thresholdValue: 1
      duration: 60s
```

#### Warning Alerts (Email/Slack)
```yaml
# High latency
alertPolicy:
  displayName: "High Response Latency"
  conditions:
  - displayName: "95th percentile > 1s"
    conditionThreshold:
      filter: 'resource.type="cloud_run_revision"'
      comparison: COMPARISON_GREATER_THAN  
      thresholdValue: 1000
      duration: 600s
  notificationChannels:
  - projects/PROJECT_ID/notificationChannels/slack-alerts
```

## 🔧 Maintenance Procedures

### Regular Maintenance (Weekly)

#### System Health Check
```bash
#!/bin/bash
# Weekly health check script

echo "=== TEMPORAL SYSTEM HEALTH CHECK ==="
echo "Date: $(date)"
echo

# Check GKE cluster
echo "GKE Cluster Status:"
kubectl get nodes
kubectl get pods -n temporal-production

# Check Cloud Run services
echo -e "\nCloud Run Services:"
gcloud run services list --region=us-central1 --filter="metadata.name:production-purchase"

# Check database
echo -e "\nDatabase Status:"
gcloud sql instances describe production-temporal-db --format="table(state,settings.tier,settings.availabilityType)"

# Check resource usage
echo -e "\nResource Usage:"
kubectl top nodes
kubectl top pods -n temporal-production --sort-by=memory

# Check recent errors
echo -e "\nRecent Errors (last 1 hour):"
gcloud logging read "
  resource.type=\"cloud_run_revision\" 
  AND resource.labels.service_name=\"production-purchase-web\" 
  AND severity>=ERROR 
  AND timestamp>=\"$(date -u -d '1 hour ago' '+%Y-%m-%dT%H:%M:%SZ')\"
" --limit=10 --format="table(timestamp,jsonPayload.message)"
```

#### Database Maintenance
```bash
# Check database performance
gcloud sql operations list --instance=production-temporal-db --limit=5

# Check backup status
gcloud sql backups list --instance=production-temporal-db --limit=3

# Analyze slow queries (if enabled)
gcloud sql instances patch production-temporal-db --database-flags=log_min_duration_statement=1000
```

### Monthly Maintenance

#### Certificate Renewal
```bash
# Check certificate expiration
gcloud compute ssl-certificates list

# Renew if needed (managed certificates auto-renew)
gcloud compute ssl-certificates create temporal-ssl-cert-new \
    --domains=purchase.yourcompany.com \
    --global
```

#### Security Updates
```bash
# Update GKE cluster
gcloud container clusters upgrade production-temporal-cluster --master --region=us-central1

# Update node pools
gcloud container node-pools upgrade default-pool \
    --cluster=production-temporal-cluster --region=us-central1

# Rebuild and redeploy containers with latest base images
gcloud builds submit --config=cloudbuild.yaml
```

#### Backup Verification
```bash
# Test database backup restore
gcloud sql backups restore BACKUP_ID --restore-instance=test-restore-instance --source-instance=production-temporal-db

# Test data integrity
kubectl exec -it temporal-server-xxx -- temporal workflow list --namespace=default
```

## 🔍 Troubleshooting Guide

### Common Issues

#### Issue: Workflows Stuck in Running State
**Symptoms**: Workflows don't progress, activities timeout
**Cause**: Worker unavailable or activity failures

**Investigation**:
```bash
# Check worker status
kubectl logs -l app=temporal-server -n temporal-production | grep -i worker

# Check activity failures
temporal workflow show --workflow-id=WORKFLOW_ID --namespace=default

# Check worker Cloud Run logs
gcloud run services logs tail production-purchase-worker --region=us-central1
```

**Resolution**:
```bash
# Restart workers
gcloud run services update production-purchase-worker --region=us-central1

# If workers are healthy, check activities
temporal activity list --namespace=default
```

#### Issue: High Database Connection Count
**Symptoms**: "too many connections" errors
**Cause**: Connection leaks or high load

**Investigation**:
```bash
# Check active connections
gcloud sql instances describe production-temporal-db --format="value(currentDiskSize,settings.databaseFlags)"

# Check connection pool in applications
kubectl logs -l app=temporal-server -n temporal-production | grep -i "connection"
```

**Resolution**:
```bash
# Increase max connections temporarily
gcloud sql instances patch production-temporal-db --database-flags=max_connections=200

# Restart services to reset connection pools
kubectl rollout restart deployment/temporal-server -n temporal-production
```

#### Issue: Slow Query Performance
**Symptoms**: High database CPU, slow response times
**Cause**: Missing indexes, inefficient queries

**Investigation**:
```bash
# Enable slow query logging
gcloud sql instances patch production-temporal-db --database-flags=log_min_duration_statement=1000

# Check database performance insights
gcloud sql instances describe production-temporal-db --format="table(name,settings.insightsConfig)"
```

**Resolution**:
```bash
# Connect to database and analyze
kubectl exec -it temporal-server-xxx -- psql -h temporal-db-service -U temporal -d temporal

-- In psql:
-- Check running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';

-- Check missing indexes
SELECT schemaname, tablename, attname, n_distinct, correlation 
FROM pg_stats 
WHERE schemaname = 'temporal';
```

### Performance Tuning

#### GKE Performance Tuning
```bash
# Optimize node pool for Temporal workloads
gcloud container node-pools create temporal-optimized \
    --cluster=production-temporal-cluster \
    --machine-type=c2-standard-4 \
    --num-nodes=3 \
    --enable-autorepair \
    --enable-autoupgrade \
    --region=us-central1

# Add node selectors to Temporal pods
kubectl patch deployment temporal-server -n temporal-production -p '
{
  "spec": {
    "template": {
      "spec": {
        "nodeSelector": {
          "workload": "temporal"
        }
      }
    }
  }
}'
```

#### Cloud SQL Performance Tuning
```bash
# Increase instance size for better performance
gcloud sql instances patch production-temporal-db --tier=db-n1-standard-4

# Optimize database flags
gcloud sql instances patch production-temporal-db --database-flags=\
shared_preload_libraries=pg_stat_statements,\
max_connections=200,\
shared_buffers=1GB,\
effective_cache_size=3GB,\
work_mem=64MB,\
maintenance_work_mem=256MB
```

## 📞 Escalation Procedures

### On-Call Rotation
- **Level 1**: Site Reliability Engineer (first response)
- **Level 2**: Platform Team Lead (complex issues)
- **Level 3**: Engineering Manager (business impact)

### Contact Information
```yaml
oncall:
  level1:
    - name: "SRE Team"
      phone: "+1-555-SRE-TEAM"
      email: "sre-oncall@company.com"
      pagerduty: "service-temporal-l1"
  
  level2:
    - name: "Platform Lead"
      phone: "+1-555-PLATFORM"
      email: "platform-lead@company.com"
      pagerduty: "service-temporal-l2"
  
  vendors:
    - name: "GCP Support"
      phone: "+1-855-836-3987"
      case_url: "https://console.cloud.google.com/support"
```

### Escalation Triggers
- **Level 1→2**: Issue not resolved within 30 minutes
- **Level 2→3**: Business-critical system down > 1 hour
- **Level 3→Vendor**: Platform issue requiring GCP intervention

## 📚 Useful Commands Reference

### Quick Diagnostics
```bash
# System overview
kubectl get all -n temporal-production
gcloud run services list --region=us-central1

# Resource usage
kubectl top nodes
kubectl top pods -n temporal-production

# Recent errors
gcloud logging read "severity>=ERROR" --limit=20 --format="table(timestamp,resource.type,jsonPayload.message)"

# Database status
gcloud sql instances describe production-temporal-db --format="table(state,settings.tier,currentDiskSize)"
```

### Emergency Commands
```bash
# Scale up quickly
gcloud container clusters resize production-temporal-cluster --num-nodes=5 --region=us-central1

# Restart all services
kubectl rollout restart deployment -n temporal-production

# Emergency maintenance mode (redirect traffic)
gcloud run services update production-purchase-web --set-env-vars="MAINTENANCE_MODE=true" --region=us-central1
```