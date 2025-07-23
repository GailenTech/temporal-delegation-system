#!/bin/bash

# Update system
apt-get update

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
systemctl enable docker
systemctl start docker

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/download/v2.21.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Create application directory
mkdir -p /app
cd /app

# Create docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  postgresql:
    container_name: temporal-postgresql
    environment:
      POSTGRES_PASSWORD: temporal
      POSTGRES_USER: temporal
    image: postgres:13
    ports:
      - 5432:5432
    volumes:
      - postgres-data:/var/lib/postgresql/data

  temporal:
    container_name: temporal
    depends_on:
      - postgresql
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
    image: temporalio/auto-setup:1.28.0
    ports:
      - 7233:7233
    volumes:
      - ./dynamicconfig:/etc/temporal/config/dynamicconfig
    labels:
      kompose.volume.type: configMap

  temporal-admin-tools:
    container_name: temporal-admin-tools
    depends_on:
      - temporal
    environment:
      - TEMPORAL_CLI_ADDRESS=temporal:7233
    image: temporalio/admin-tools:1.28.0
    stdin_open: true
    tty: true

  temporal-ui:
    container_name: temporal-ui
    depends_on:
      - temporal
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
      - TEMPORAL_CORS_ORIGINS=http://localhost:3000
    image: temporalio/ui:2.33.0
    ports:
      - 8080:8080

  web:
    container_name: purchase-web
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal
      - TEMPORAL_PORT=7233
      - TEMPORAL_NAMESPACE=default
    image: europe-west1-docker.pkg.dev/temporal-demo-0723/temporal-demo/web:latest
    ports:
      - 8081:8081

  worker:
    container_name: purchase-worker
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal
      - TEMPORAL_PORT=7233
      - TEMPORAL_NAMESPACE=default
    image: europe-west1-docker.pkg.dev/temporal-demo-0723/temporal-demo/worker:latest

volumes:
  postgres-data:
EOF

# Create dynamic config directory
mkdir -p dynamicconfig

# Create temporal development configuration
cat > dynamicconfig/development-sql.yaml << 'EOF'
system.forceSearchAttributesCacheRefreshOnRead:
  - value: true
    constraints: {}

history.persistenceMaxQPS:
  - value: 3000
    constraints: {}

frontend.persistenceMaxQPS:
  - value: 3000
    constraints: {}

matching.persistenceMaxQPS:
  - value: 3000
    constraints: {}

worker.persistenceMaxQPS:
  - value: 3000
    constraints: {}

system.historyStreamFromMatchingError:
  - value: true
    constraints: {}
EOF

# Authenticate with Google Cloud to pull images
export GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcp-key.json
curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token | jq -r .access_token > /tmp/token
docker login -u oauth2accesstoken --password-stdin https://europe-west1-docker.pkg.dev < /tmp/token

# Start services
docker-compose up -d

# Wait for services to be ready
sleep 60

# Show status
docker-compose ps