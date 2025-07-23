# GKE Cluster for Temporal Server
resource "google_container_cluster" "temporal_cluster" {
  name     = "${var.environment}-temporal-cluster"
  location = var.region
  
  # Remove default node pool and create custom one
  remove_default_node_pool = true
  initial_node_count       = 1
  
  # Enable workload identity for secure pod-to-GCP authentication
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
  
  # Enable network policy for security
  network_policy {
    enabled = true
  }
  
  # Enable VPC-native networking
  ip_allocation_policy {
    cluster_ipv4_cidr_block  = var.cluster_ipv4_cidr
    services_ipv4_cidr_block = var.services_ipv4_cidr
  }
  
  # Enable private cluster for security
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false  # Set true for production
    master_ipv4_cidr_block = var.master_ipv4_cidr
  }
  
  master_auth {
    client_certificate_config {
      issue_client_certificate = false
    }
  }
  
  # Resource labels for cost tracking
  resource_labels = {
    environment = var.environment
    component   = "temporal-cluster"
    team        = var.team
  }
}

# Temporal Server Node Pool
resource "google_container_node_pool" "temporal_nodes" {
  name       = "${var.environment}-temporal-nodes"
  location   = var.region
  cluster    = google_container_cluster.temporal_cluster.name
  
  node_count = var.environment == "demo" ? 1 : 3
  
  node_config {
    machine_type = var.environment == "demo" ? "e2-small" : "e2-medium"
    disk_size_gb = var.environment == "demo" ? 50 : 100
    disk_type    = "pd-ssd"
    
    # Enable workload identity
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
    
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
    
    labels = {
      environment = var.environment
      role        = "temporal-server"
    }
    
    taint {
      key    = "temporal-server"
      value  = "true"
      effect = "NO_SCHEDULE"
    }
  }
  
  # Enable auto-scaling for production
  dynamic "autoscaling" {
    for_each = var.environment != "demo" ? [1] : []
    content {
      min_node_count = 1
      max_node_count = 5
    }
  }
  
  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# Output cluster credentials
output "cluster_endpoint" {
  value = google_container_cluster.temporal_cluster.endpoint
}

output "cluster_ca_certificate" {
  value = base64decode(google_container_cluster.temporal_cluster.master_auth.0.cluster_ca_certificate)
}