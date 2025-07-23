#!/bin/bash
set -e

# Configuration
PROJECT_ID="temporal-demo-0723"
REGION="europe-west1"
REGISTRY="europe-west1-docker.pkg.dev"

echo "🚀 Deploying Temporal Workflow System to Cloud Run"

# Deploy PostgreSQL for Temporal
echo "📊 Creating Cloud SQL PostgreSQL instance..."
gcloud sql instances create temporal-postgres \
    --database-version=POSTGRES_14 \
    --cpu=1 \
    --memory=3840MB \
    --region=${REGION} \
    --root-password=temporal123 \
    --availability-type=zonal \
    --storage-size=10GB \
    --storage-type=SSD \
    --project=${PROJECT_ID} || echo "Instance may already exist"

# Create databases
echo "📊 Creating databases..."
gcloud sql databases create temporal \
    --instance=temporal-postgres \
    --project=${PROJECT_ID} || echo "Database may already exist"

gcloud sql databases create temporal_visibility \
    --instance=temporal-postgres \
    --project=${PROJECT_ID} || echo "Database may already exist"

# Get the Cloud SQL connection name
INSTANCE_CONNECTION_NAME=$(gcloud sql instances describe temporal-postgres --format="value(connectionName)" --project=${PROJECT_ID})

# Deploy Temporal Server using the official Docker image
echo "⚡ Deploying Temporal Server..."
gcloud run deploy temporal-server \
    --image=temporalio/auto-setup:1.28.0 \
    --platform=managed \
    --region=${REGION} \
    --allow-unauthenticated \
    --port=7233 \
    --memory=1Gi \
    --cpu=1 \
    --min-instances=0 \
    --max-instances=3 \
    --set-env-vars="DB=postgresql,DB_PORT=5432,POSTGRES_USER=postgres,POSTGRES_PWD=temporal123,POSTGRES_SEEDS=temporal" \
    --set-env-vars="DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml" \
    --add-cloudsql-instances=${INSTANCE_CONNECTION_NAME} \
    --project=${PROJECT_ID}

# Deploy Temporal Web UI
echo "🌐 Deploying Temporal Web UI..."
gcloud run deploy temporal-web \
    --image=temporalio/ui:2.33.0 \
    --platform=managed \
    --region=${REGION} \
    --allow-unauthenticated \
    --port=8080 \
    --memory=512Mi \
    --cpu=0.5 \
    --min-instances=0 \
    --max-instances=2 \
    --set-env-vars="TEMPORAL_ADDRESS=temporal-server-url-placeholder:7233" \
    --set-env-vars="TEMPORAL_CORS_ORIGINS=*" \
    --project=${PROJECT_ID}

# Get Temporal Server URL and update Web UI
TEMPORAL_SERVER_URL=$(gcloud run services describe temporal-server --region=${REGION} --format="value(status.url)" --project=${PROJECT_ID})

# Update Web UI with correct Temporal Server URL
gcloud run services update temporal-web \
    --region=${REGION} \
    --set-env-vars="TEMPORAL_ADDRESS=${TEMPORAL_SERVER_URL#https://}:443" \
    --project=${PROJECT_ID}

# Deploy our Web Application
echo "🖥️  Deploying Purchase Approval Web App..."
gcloud run deploy purchase-web \
    --image=${REGISTRY}/${PROJECT_ID}/temporal-demo/web:latest \
    --platform=managed \
    --region=${REGION} \
    --allow-unauthenticated \
    --port=8081 \
    --memory=512Mi \
    --cpu=0.5 \
    --min-instances=0 \
    --max-instances=3 \
    --set-env-vars="TEMPORAL_HOST=${TEMPORAL_SERVER_URL#https://},TEMPORAL_PORT=443,TEMPORAL_NAMESPACE=default" \
    --project=${PROJECT_ID}

# Deploy our Worker
echo "⚙️  Deploying Purchase Approval Worker..."
gcloud run deploy purchase-worker \
    --image=${REGISTRY}/${PROJECT_ID}/temporal-demo/worker:latest \
    --platform=managed \
    --region=${REGION} \
    --allow-unauthenticated \
    --port=8080 \
    --memory=512Mi \
    --cpu=0.5 \
    --min-instances=1 \
    --max-instances=2 \
    --set-env-vars="TEMPORAL_HOST=${TEMPORAL_SERVER_URL#https://},TEMPORAL_PORT=443,TEMPORAL_NAMESPACE=default" \
    --project=${PROJECT_ID}

echo "✅ Deployment complete!"
echo ""
echo "🔗 Service URLs:"
echo "Temporal Server: ${TEMPORAL_SERVER_URL}"
TEMPORAL_WEB_URL=$(gcloud run services describe temporal-web --region=${REGION} --format="value(status.url)" --project=${PROJECT_ID})
echo "Temporal Web UI: ${TEMPORAL_WEB_URL}"
PURCHASE_WEB_URL=$(gcloud run services describe purchase-web --region=${REGION} --format="value(status.url)" --project=${PROJECT_ID})
echo "Purchase Approval App: ${PURCHASE_WEB_URL}"

echo ""
echo "🧪 Test the system:"
echo "1. Visit the Purchase Approval App: ${PURCHASE_WEB_URL}"
echo "2. Monitor workflows in Temporal UI: ${TEMPORAL_WEB_URL}"