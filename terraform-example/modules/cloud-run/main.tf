# Cloud Run service for web frontend
resource "google_cloud_run_v2_service" "web_frontend" {
  name     = "${var.environment}-purchase-web"
  location = var.region
  
  template {
    containers {
      image = var.web_image
      ports {
        container_port = 8081
      }
      
      # Environment variables
      env {
        name  = "TEMPORAL_ADDRESS"
        value = var.temporal_address
      }
      
      env {
        name  = "ENVIRONMENT"
        value = var.environment
      }
      
      # Database connection from Secret Manager
      env {
        name = "DB_CONNECTION"
        value_source {
          secret_key_ref {
            secret  = var.db_secret_name
            version = "latest"
          }
        }
      }
      
      # Resource limits based on environment
      resources {
        limits = {
          cpu    = var.environment == "demo" ? "1" : "2"
          memory = var.environment == "demo" ? "512Mi" : "1Gi"
        }
      }
      
      # Startup and liveness probes
      startup_probe {
        http_get {
          path = "/health"
          port = 8081
        }
        initial_delay_seconds = 10
        timeout_seconds      = 5
        period_seconds       = 10
        failure_threshold    = 3
      }
      
      liveness_probe {
        http_get {
          path = "/health"
          port = 8081
        }
        initial_delay_seconds = 30
        timeout_seconds      = 5
        period_seconds       = 30
        failure_threshold    = 3
      }
    }
    
    # Scaling configuration
    scaling {
      min_instance_count = var.environment == "demo" ? 0 : 1
      max_instance_count = var.environment == "demo" ? 10 : 100
    }
    
    # Service account for secure access
    service_account = google_service_account.cloud_run_sa.email
    
    annotations = {
      "autoscaling.knative.dev/maxScale"        = var.environment == "demo" ? "10" : "100"
      "run.googleapis.com/execution-environment" = "gen2"
      "run.googleapis.com/cpu-throttling"       = "false"
    }
  }
  
  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Cloud Run service for workers
resource "google_cloud_run_v2_service" "workers" {
  name     = "${var.environment}-purchase-workers"
  location = var.region
  
  template {
    containers {
      image = var.worker_image
      
      env {
        name  = "TEMPORAL_ADDRESS"
        value = var.temporal_address
      }
      
      env {
        name  = "WORKER_TASK_QUEUE"
        value = "purchase-approval-task-queue"
      }
      
      env {
        name = "DB_CONNECTION"
        value_source {
          secret_key_ref {
            secret  = var.db_secret_name
            version = "latest"
          }
        }
      }
      
      resources {
        limits = {
          cpu    = var.environment == "demo" ? "1" : "2"
          memory = var.environment == "demo" ? "1Gi" : "2Gi"
        }
      }
      
      # Workers need to stay alive to poll for tasks
      startup_probe {
        tcp_socket {
          port = 8080
        }
        initial_delay_seconds = 15
        timeout_seconds      = 5
        period_seconds       = 10
        failure_threshold    = 3
      }
    }
    
    scaling {
      min_instance_count = var.environment == "demo" ? 0 : 1
      max_instance_count = var.environment == "demo" ? 5 : 20
    }
    
    service_account = google_service_account.cloud_run_sa.email
    
    annotations = {
      "autoscaling.knative.dev/maxScale" = var.environment == "demo" ? "5" : "20"
      "run.googleapis.com/execution-environment" = "gen2"
    }
  }
  
  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Service account for Cloud Run services
resource "google_service_account" "cloud_run_sa" {
  account_id   = "${var.environment}-cloud-run-sa"
  display_name = "Cloud Run Service Account for ${var.environment}"
}

# Grant permissions to access Secret Manager
resource "google_project_iam_member" "cloud_run_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

# Grant permissions to access Cloud SQL
resource "google_project_iam_member" "cloud_run_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

# IAM policy to allow unauthenticated access to web frontend
resource "google_cloud_run_service_iam_member" "web_public_access" {
  location = google_cloud_run_v2_service.web_frontend.location
  service  = google_cloud_run_v2_service.web_frontend.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "web_url" {
  value = google_cloud_run_v2_service.web_frontend.uri
}

output "worker_url" {
  value = google_cloud_run_v2_service.workers.uri
}