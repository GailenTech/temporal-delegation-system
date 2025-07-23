#!/bin/bash
set -euo pipefail

# One-click demo deployment script for Google Cloud
# Usage: ./scripts/deploy-demo.sh [PROJECT_ID] [REGION]

# Configuration
PROJECT_ID=${1:-"your-project-id"}
REGION=${2:-"us-central1"}
ENVIRONMENT="demo"
CLUSTER_NAME="${ENVIRONMENT}-temporal-cluster"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if gcloud is installed
    if ! command -v gcloud &> /dev/null; then
        log_error "gcloud CLI is not installed. Please install it first."
        exit 1
    fi
    
    # Check if terraform is installed
    if ! command -v terraform &> /dev/null; then
        log_error "Terraform is not installed. Please install it first."
        exit 1
    fi
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install it first."
        exit 1
    fi
    
    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log_error "Helm is not installed. Please install it first."
        exit 1
    fi
    
    log_success "All prerequisites are installed"
}

# Setup GCP project
setup_gcp_project() {
    log_info "Setting up GCP project: $PROJECT_ID"
    
    # Set project
    gcloud config set project $PROJECT_ID
    
    # Enable required APIs
    log_info "Enabling required GCP APIs..."
    gcloud services enable container.googleapis.com
    gcloud services enable sqladmin.googleapis.com
    gcloud services enable run.googleapis.com
    gcloud services enable secretmanager.googleapis.com
    gcloud services enable cloudbuild.googleapis.com
    gcloud services enable compute.googleapis.com
    
    log_success "GCP project setup completed"
}

# Deploy infrastructure with Terraform
deploy_infrastructure() {
    log_info "Deploying infrastructure with Terraform..."
    
    cd terraform-example/environments/demo
    
    # Initialize Terraform
    terraform init
    
    # Plan deployment
    terraform plan \
        -var="project_id=$PROJECT_ID" \
        -var="region=$REGION" \
        -var="environment=$ENVIRONMENT"
    
    # Apply deployment
    terraform apply -auto-approve \
        -var="project_id=$PROJECT_ID" \
        -var="region=$REGION" \
        -var="environment=$ENVIRONMENT"
    
    # Get outputs
    DB_CONNECTION_NAME=$(terraform output -raw db_connection_name)
    CLUSTER_ENDPOINT=$(terraform output -raw cluster_endpoint)
    
    cd ../../..
    
    log_success "Infrastructure deployment completed"
}

# Setup Kubernetes cluster
setup_kubernetes() {
    log_info "Setting up Kubernetes cluster..."
    
    # Get cluster credentials
    gcloud container clusters get-credentials $CLUSTER_NAME \
        --region $REGION --project $PROJECT_ID
    
    # Create namespace
    kubectl create namespace temporal-demo || log_warning "Namespace already exists"
    
    # Add Temporal Helm repo
    helm repo add temporalio https://go.temporal.io/helm-charts
    helm repo update
    
    log_success "Kubernetes setup completed"
}

# Deploy Temporal server
deploy_temporal() {
    log_info "Deploying Temporal server..."
    
    # Create secret for database connection
    kubectl create secret generic temporal-db-secret \
        --from-literal=password="$(openssl rand -base64 32)" \
        --namespace temporal-demo || log_warning "Secret already exists"
    
    # Deploy Temporal using Helm
    helm upgrade --install temporal-server temporalio/temporal \
        --namespace temporal-demo \
        --values helm/temporal-server/values-demo.yaml \
        --set server.config.persistence.default.sql.password.secretName="temporal-db-secret" \
        --set server.config.persistence.default.sql.password.secretKey="password" \
        --set global.image.repository="temporalio/server" \
        --set global.image.tag="1.22.0" \
        --wait --timeout=10m
    
    log_success "Temporal server deployment completed"
}

# Build and deploy application
deploy_application() {
    log_info "Building and deploying application..."
    
    # Build images
    log_info "Building Docker images..."
    docker build -t gcr.io/$PROJECT_ID/temporal-purchase-web:demo -f Dockerfile.web .
    docker build -t gcr.io/$PROJECT_ID/temporal-purchase-worker:demo -f Dockerfile.worker .
    
    # Push images
    log_info "Pushing images to Container Registry..."
    docker push gcr.io/$PROJECT_ID/temporal-purchase-web:demo
    docker push gcr.io/$PROJECT_ID/temporal-purchase-worker:demo
    
    # Get Temporal server service endpoint
    TEMPORAL_ADDRESS=$(kubectl get service temporal-server-frontend \
        --namespace temporal-demo \
        -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):7233
    
    # Deploy web service to Cloud Run
    log_info "Deploying web service to Cloud Run..."
    gcloud run deploy ${ENVIRONMENT}-purchase-web \
        --image gcr.io/$PROJECT_ID/temporal-purchase-web:demo \
        --region $REGION \
        --platform managed \
        --allow-unauthenticated \
        --set-env-vars="ENVIRONMENT=$ENVIRONMENT,TEMPORAL_ADDRESS=$TEMPORAL_ADDRESS" \
        --max-instances=10 \
        --memory=512Mi \
        --cpu=1 \
        --timeout=300
    
    # Deploy worker service to Cloud Run
    log_info "Deploying worker service to Cloud Run..."
    gcloud run deploy ${ENVIRONMENT}-purchase-worker \
        --image gcr.io/$PROJECT_ID/temporal-purchase-worker:demo \
        --region $REGION \
        --platform managed \
        --no-allow-unauthenticated \
        --set-env-vars="ENVIRONMENT=$ENVIRONMENT,TEMPORAL_ADDRESS=$TEMPORAL_ADDRESS" \
        --max-instances=5 \
        --memory=1Gi \
        --cpu=1 \
        --timeout=3600
    
    log_success "Application deployment completed"
}

# Get deployment information
get_deployment_info() {
    log_info "Getting deployment information..."
    
    # Get web service URL
    WEB_URL=$(gcloud run services describe ${ENVIRONMENT}-purchase-web \
        --region $REGION --format="value(status.url)")
    
    # Get Temporal UI URL
    TEMPORAL_UI_EXTERNAL_IP=$(kubectl get service temporal-server-web \
        --namespace temporal-demo \
        -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    echo ""
    log_success "🎉 Demo deployment completed successfully!"
    echo ""
    echo "📋 Deployment Information:"
    echo "   Project ID: $PROJECT_ID"
    echo "   Region: $REGION"
    echo "   Environment: $ENVIRONMENT"
    echo ""
    echo "🌐 Application URLs:"
    echo "   Purchase System: $WEB_URL"
    echo "   Temporal UI: http://$TEMPORAL_UI_EXTERNAL_IP:8080"
    echo ""
    echo "🔧 Management Commands:"
    echo "   View logs: gcloud run logs tail ${ENVIRONMENT}-purchase-web --region $REGION"
    echo "   Scale down: gcloud run services update ${ENVIRONMENT}-purchase-web --region $REGION --max-instances=0"
    echo "   Delete: ./scripts/cleanup-demo.sh $PROJECT_ID $REGION"
    echo ""
    
    # Save deployment info to file
    cat > deployment-info.json <<EOF
{
    "project_id": "$PROJECT_ID",
    "region": "$REGION", 
    "environment": "$ENVIRONMENT",
    "web_url": "$WEB_URL",
    "temporal_ui": "http://$TEMPORAL_UI_EXTERNAL_IP:8080",
    "deployed_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log_info "Deployment info saved to deployment-info.json"
}

# Main execution
main() {
    echo "🚀 Starting Temporal Purchase Approval Demo Deployment"
    echo "   Project: $PROJECT_ID"
    echo "   Region: $REGION"
    echo ""
    
    check_prerequisites
    setup_gcp_project
    deploy_infrastructure
    setup_kubernetes
    deploy_temporal
    deploy_application
    get_deployment_info
    
    log_success "🎉 All done! Your demo environment is ready."
}

# Run main function
main "$@"