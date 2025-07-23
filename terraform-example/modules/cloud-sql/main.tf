# Cloud SQL PostgreSQL for Temporal persistence
resource "google_sql_database_instance" "temporal_db" {
  name             = "${var.environment}-temporal-db"
  database_version = "POSTGRES_15"
  region          = var.region
  
  settings {
    tier = var.environment == "demo" ? "db-f1-micro" : "db-n1-standard-2"
    
    # Enable high availability for production
    availability_type = var.environment == "production" ? "REGIONAL" : "ZONAL"
    
    disk_type = "PD_SSD"
    disk_size = var.environment == "demo" ? 20 : 100
    disk_autoresize = true
    disk_autoresize_limit = var.environment == "demo" ? 50 : 500
    
    backup_configuration {
      enabled                        = true
      start_time                    = "03:00"
      point_in_time_recovery_enabled = var.environment != "demo"
      backup_retention_settings {
        retained_backups = var.environment == "demo" ? 7 : 30
      }
    }
    
    maintenance_window {
      day         = 1  # Sunday
      hour        = 4  # 4 AM
      update_track = "stable"
    }
    
    database_flags {
      name  = "max_connections"
      value = var.environment == "demo" ? "100" : "200"
    }
    
    database_flags {
      name  = "shared_preload_libraries"
      value = "pg_stat_statements"
    }
    
    ip_configuration {
      ipv4_enabled                                  = false
      private_network                              = var.vpc_network
      enable_private_path_for_google_cloud_services = true
    }
    
    insights_config {
      query_insights_enabled  = true
      query_string_length    = 1024
      record_application_tags = true
      record_client_address   = true
    }
  }
  
  deletion_protection = var.environment == "production"
}

# Create Temporal database
resource "google_sql_database" "temporal" {
  name     = "temporal"
  instance = google_sql_database_instance.temporal_db.name
}

# Create application database  
resource "google_sql_database" "temporal_visibility" {
  name     = "temporal_visibility"
  instance = google_sql_database_instance.temporal_db.name
}

# Create Temporal user with limited permissions
resource "google_sql_user" "temporal_user" {
  name     = "temporal"
  instance = google_sql_database_instance.temporal_db.name
  password = var.temporal_db_password
}

# Store connection details in Secret Manager
resource "google_secret_manager_secret" "temporal_db_connection" {
  secret_id = "${var.environment}-temporal-db-connection"
  
  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "temporal_db_connection" {
  secret = google_secret_manager_secret.temporal_db_connection.id
  
  secret_data = jsonencode({
    host     = google_sql_database_instance.temporal_db.private_ip_address
    port     = 5432
    database = "temporal"
    username = google_sql_user.temporal_user.name
    password = google_sql_user.temporal_user.password
  })
}

output "instance_connection_name" {
  value = google_sql_database_instance.temporal_db.connection_name
}

output "private_ip_address" {
  value = google_sql_database_instance.temporal_db.private_ip_address
}