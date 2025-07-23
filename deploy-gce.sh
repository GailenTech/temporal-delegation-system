#!/bin/bash
set -e

# Configuration
PROJECT_ID="temporal-demo-0723"
ZONE="europe-west1-b"
INSTANCE_NAME="temporal-demo"

echo "🚀 Deploying Temporal Workflow System to Google Compute Engine"

# Create VM instance with Docker
echo "🖥️  Creating VM instance..."
gcloud compute instances create ${INSTANCE_NAME} \
    --zone=${ZONE} \
    --machine-type=e2-standard-2 \
    --boot-disk-size=20GB \
    --boot-disk-type=pd-ssd \
    --image-family=ubuntu-2204-lts \
    --image-project=ubuntu-os-cloud \
    --metadata-from-file startup-script=startup-script.sh \
    --tags=http-server,https-server \
    --project=${PROJECT_ID} || echo "Instance may already exist"

# Create firewall rules
echo "🔥 Creating firewall rules..."
gcloud compute firewall-rules create allow-temporal-ports \
    --allow tcp:8080,tcp:8081,tcp:7233,tcp:7234 \
    --source-ranges 0.0.0.0/0 \
    --target-tags http-server \
    --project=${PROJECT_ID} || echo "Firewall rule may already exist"

# Wait for instance to be ready
echo "⏳ Waiting for instance to start..."
sleep 30

# Get external IP
EXTERNAL_IP=$(gcloud compute instances describe ${INSTANCE_NAME} --zone=${ZONE} --format="value(networkInterfaces[0].accessConfigs[0].natIP)" --project=${PROJECT_ID})

echo "✅ Deployment complete!"
echo ""
echo "🔗 Service URLs:"
echo "VM External IP: ${EXTERNAL_IP}"
echo "Temporal Web UI: http://${EXTERNAL_IP}:8080"
echo "Purchase Approval App: http://${EXTERNAL_IP}:8081"
echo ""
echo "🧪 SSH to instance:"
echo "gcloud compute ssh ${INSTANCE_NAME} --zone=${ZONE} --project=${PROJECT_ID}"