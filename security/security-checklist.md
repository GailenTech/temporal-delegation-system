# Security Checklist for Temporal Purchase Approval System on GCP

## Infrastructure Security

### Network Security
- [ ] **VPC-native GKE cluster** configured with private nodes
- [ ] **Private Google Access** enabled for nodes without external IPs  
- [ ] **Authorized networks** configured for GKE master API access
- [ ] **Network policies** implemented to restrict pod-to-pod communication
- [ ] **Cloud Armor** configured for DDoS protection and WAF rules
- [ ] **Load balancer** with SSL termination and HTTP to HTTPS redirect
- [ ] **VPC firewall rules** allowing only necessary traffic

### Identity and Access Management
- [ ] **Service accounts** with minimal required permissions
- [ ] **Workload Identity** enabled for pod-to-GCP service authentication
- [ ] **IAM conditions** applied for fine-grained access control
- [ ] **Regular access review** process established
- [ ] **Multi-factor authentication** required for admin access
- [ ] **Audit logging** enabled for all IAM changes

### Data Encryption
- [ ] **Encryption at rest** enabled for Cloud SQL and persistent volumes
- [ ] **Encryption in transit** configured for all service communications
- [ ] **Customer-managed encryption keys (CMEK)** for sensitive data
- [ ] **Secret Manager** used for all credentials and API keys
- [ ] **TLS 1.2+** enforced for all external communications

## Application Security

### Authentication & Authorization
- [ ] **OAuth 2.1/OIDC** integration for user authentication
- [ ] **JWT tokens** with proper validation and expiration
- [ ] **Role-based access control (RBAC)** implemented
- [ ] **Session management** with secure cookies and timeout
- [ ] **API authentication** for service-to-service communication

### Input Validation & Sanitization
- [ ] **Input validation** for all user inputs (URLs, forms)
- [ ] **SQL injection** protection in database queries
- [ ] **Cross-site scripting (XSS)** prevention in templates
- [ ] **CSRF protection** for state-changing operations
- [ ] **File upload validation** if applicable

### Secrets Management
- [ ] **No hardcoded secrets** in source code or containers
- [ ] **Environment-specific secrets** properly separated
- [ ] **Secret rotation** policies implemented
- [ ] **Least privilege access** to secrets

## Container Security

### Image Security
- [ ] **Base images** from trusted registries (distroless, alpine)
- [ ] **Vulnerability scanning** enabled in Cloud Build
- [ ] **Image signing** and verification process
- [ ] **Regular image updates** for security patches
- [ ] **Binary authorization** for production deployments

### Runtime Security
- [ ] **Non-root containers** running with specific user IDs
- [ ] **Read-only root filesystem** where possible
- [ ] **Security contexts** properly configured
- [ ] **Resource limits** set to prevent resource exhaustion
- [ ] **Pod security policies** or **Pod Security Standards** enforced

## Database Security

### Cloud SQL Security
- [ ] **Private IP** configuration (no public IP)
- [ ] **SSL connections** required for all clients
- [ ] **Database users** with minimal privileges
- [ ] **Regular backups** with encryption
- [ ] **Point-in-time recovery** enabled
- [ ] **Automated security updates** enabled

### Data Protection
- [ ] **Data classification** and handling procedures
- [ ] **Personal data encryption** for GDPR compliance
- [ ] **Data retention policies** implemented
- [ ] **Secure data deletion** procedures

## Monitoring and Compliance

### Logging and Monitoring
- [ ] **Audit logs** enabled for all GCP services
- [ ] **Application logging** with security events
- [ ] **Real-time alerting** for security incidents
- [ ] **Log retention** policies for compliance
- [ ] **SIEM integration** if required

### Compliance Requirements
- [ ] **SOC 2 Type II** compliance considerations
- [ ] **GDPR compliance** for personal data
- [ ] **PCI DSS compliance** if handling payment data
- [ ] **Industry-specific regulations** addressed

## Incident Response

### Security Incident Response Plan
- [ ] **Incident response playbook** documented
- [ ] **Emergency contacts** and escalation procedures
- [ ] **Forensic logging** capabilities enabled
- [ ] **Backup and recovery** procedures tested
- [ ] **Regular security drills** conducted

## Security Testing

### Automated Security Testing
- [ ] **Static Application Security Testing (SAST)** in CI/CD
- [ ] **Dynamic Application Security Testing (DAST)** for web app
- [ ] **Dependency vulnerability scanning** for Go modules
- [ ] **Infrastructure as Code scanning** for Terraform

### Manual Security Testing
- [ ] **Penetration testing** conducted annually
- [ ] **Code review** with security focus
- [ ] **Architecture security review** completed
- [ ] **Red team exercises** if applicable

## Security Configuration Examples

### GKE Security Configuration
```yaml
# Pod Security Policy (deprecated) or Pod Security Standards
apiVersion: v1
kind: Namespace
metadata:
  name: temporal-production
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Network Policy Example
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: temporal-server-policy
  namespace: temporal-production
spec:
  podSelector:
    matchLabels:
      app: temporal-server
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: temporal-client
    ports:
    - protocol: TCP
      port: 7233
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
```

### Cloud Armor Security Policy
```bash
# Create security policy
gcloud compute security-policies create temporal-security-policy \
    --description "Security policy for Temporal Purchase Approval System"

# Add DDoS protection rule
gcloud compute security-policies rules create 1000 \
    --security-policy temporal-security-policy \
    --expression "origin.region_code == 'CN' || origin.region_code == 'RU'" \
    --action "deny-403"

# Add rate limiting rule
gcloud compute security-policies rules create 2000 \
    --security-policy temporal-security-policy \
    --expression "rate(request.headers['x-forwarded-for'], 100)" \
    --action "throttle" \
    --rate-limit-threshold-count 100 \
    --rate-limit-threshold-interval-sec 60
```

## Continuous Security Monitoring

### Security Metrics to Track
- Failed authentication attempts
- Unusual API access patterns
- Resource usage anomalies
- Network traffic anomalies
- Certificate expiration dates
- Security patch compliance

### Alerting Configuration
```yaml
# Example Cloud Monitoring alert
displayName: "High Failed Authentication Rate"
conditions:
  - displayName: "Failed auth rate > 10/min"
    conditionThreshold:
      filter: 'resource.type="cloud_run_revision" AND log_name="projects/PROJECT_ID/logs/run.googleapis.com%2Fstderr"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 10
      duration: 300s
alertPolicy:
  notificationChannels:
  - projects/PROJECT_ID/notificationChannels/CHANNEL_ID
```

## Security Review Schedule
- **Weekly**: Review access logs and security alerts
- **Monthly**: Update security configurations and patches
- **Quarterly**: Security architecture review and testing
- **Annually**: Full penetration testing and compliance audit