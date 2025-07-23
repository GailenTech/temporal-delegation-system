package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"temporal-workflow/internal/handlers"
	"temporal-workflow/internal/models"
	"temporal-workflow/internal/services"
	"temporal-workflow/internal/workflows"
)

const (
	taskQueue = "purchase-approval-task-queue"
)

var temporalClient client.Client
var authService *services.AuthService
var delegationHandlers *handlers.DelegationHandlers

func main() {
	log.Println("Starting Purchase Approval Web Server...")

	// Connect to Temporal
	var err error
	
	// Read connection configuration from environment
	host := os.Getenv("TEMPORAL_HOST")
	port := os.Getenv("TEMPORAL_PORT")
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	
	// Set defaults if not provided
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "7233"
	}
	if namespace == "" {
		namespace = "default"
	}
	
	hostPort := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Connecting to Temporal at %s, namespace: %s", hostPort, namespace)
	
	temporalClient, err = client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer temporalClient.Close()

	// Initialize services
	authService = services.NewAuthService()
	delegationHandlers = handlers.NewDelegationHandlers(temporalClient)

	// Setup HTTP handlers
	http.HandleFunc("/", dashboardHandler)                    // Dashboard principal (redirige a login si no autenticado)
	http.HandleFunc("/dashboard", authService.RequireAuth(dashboardHandler))
	http.HandleFunc("/login-as/", authService.HandleLogin)    // Login simulado
	http.HandleFunc("/logout", authService.HandleLogout)      // Logout
	
	// Request handlers
	http.HandleFunc("/request/new", authService.RequireAuth(newRequestHandler))
	http.HandleFunc("/request/submit", authService.RequireAuth(submitHandler))
	http.HandleFunc("/status", authService.RequireAuth(statusHandler))
	
	// Approval handlers  
	http.HandleFunc("/approvals/pending", authService.RequireAuth(pendingApprovalsHandler))
	http.HandleFunc("/pending-approvals", authService.RequireAuth(pendingApprovalsHandler)) // Alias para delegaciones
	http.HandleFunc("/approval/", approvalHandler)  // No auth para compatibilidad
	http.HandleFunc("/approve", approveHandler)
	
	// Admin handlers
	http.HandleFunc("/admin/dashboard", authService.RequirePermission(
		func(p models.Permissions) bool { return p.CanViewAdminPanel }, 
		adminDashboardHandler))

	// Delegation handlers
	http.HandleFunc("/delegation/new", authService.RequirePermission(
		func(p models.Permissions) bool { return p.CanDelegate }, 
		delegationHandlers.CreateDelegationPage))
	http.HandleFunc("/delegation/create", authService.RequirePermission(
		func(p models.Permissions) bool { return p.CanDelegate }, 
		delegationHandlers.CreateDelegation))
	http.HandleFunc("/delegation/list", authService.RequireAuth(
		delegationHandlers.ListDelegations)) // Todos los usuarios autenticados pueden ver sus delegaciones
	http.HandleFunc("/delegation/received", authService.RequireAuth(
		delegationHandlers.ListReceivedDelegations)) // Solo delegaciones recibidas

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	log.Println("Web server starting on :8081")
	log.Println("Visit http://localhost:8081 to access the application")
	
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalln("Server failed to start:", err)
	}
}

// DashboardData datos para el dashboard
type DashboardData struct {
	User            *models.User        `json:"user"`
	Permissions     models.Permissions `json:"permissions"`
	Stats           models.DashboardStats `json:"stats"`
	RecentRequests  []models.PurchaseRequest `json:"recent_requests"`
	PendingApprovals []models.PurchaseRequest `json:"pending_approvals"`
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Si no está autenticado, mostrar página de login
	user := services.GetCurrentUser(r)
	if user == nil {
		// Redirected to login page (this will show the login form)
		authService.HandleLogin(w, r)
		return
	}

	// Obtener datos para el dashboard
	data := DashboardData{
		User:        user,
		Permissions: user.GetPermissions(),
		Stats:       getDashboardStats(*user),
		RecentRequests: getRecentRequests(*user),
		PendingApprovals: getPendingApprovals(*user),
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Dashboard - Sistema de Compras</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; background: #f8f9fa; }
        .header { background: #007cba; color: white; padding: 15px 0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header-content { max-width: 1200px; margin: 0 auto; padding: 0 20px; display: flex; justify-content: space-between; align-items: center; }
        .logo { font-size: 24px; font-weight: bold; }
        .user-info { display: flex; align-items: center; gap: 15px; }
        .user-info .name { font-weight: 500; }
        .user-info .role { background: rgba(255,255,255,0.2); padding: 4px 8px; border-radius: 12px; font-size: 12px; }
        .logout { background: rgba(255,255,255,0.2); border: 1px solid rgba(255,255,255,0.3); color: white; padding: 8px 16px; border-radius: 4px; text-decoration: none; }
        .logout:hover { background: rgba(255,255,255,0.3); }
        
        .container { max-width: 1200px; margin: 0 auto; padding: 30px 20px; }
        .welcome { margin-bottom: 30px; }
        .welcome h1 { margin: 0 0 10px 0; color: #333; }
        .welcome p { margin: 0; color: #666; }
        
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); border-left: 4px solid #007cba; }
        .stat-card h3 { margin: 0 0 10px 0; color: #666; font-size: 14px; font-weight: 500; }
        .stat-card .number { font-size: 32px; font-weight: bold; color: #007cba; margin: 0; }
        .stat-card.warning .number { color: #ff6b35; }
        .stat-card.success .number { color: #28a745; }
        
        .actions-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .action-card { background: white; border-radius: 8px; padding: 25px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
        .action-card h3 { margin: 0 0 15px 0; color: #333; }
        .action-card p { color: #666; font-size: 14px; margin: 0 0 20px 0; }
        .action-btn { display: inline-block; background: #007cba; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 500; }
        .action-btn:hover { background: #005a87; }
        .action-btn.secondary { background: #6c757d; }
        .action-btn.secondary:hover { background: #545b62; }
        .action-btn.warning { background: #ff6b35; }
        .action-btn.warning:hover { background: #e5592d; }
        
        .permission-note { background: #e9ecef; padding: 15px; border-radius: 6px; margin: 15px 0; color: #6c757d; font-size: 14px; }
        .hidden { display: none; }
        
        .recent-activity { background: white; border-radius: 8px; padding: 25px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .recent-activity h3 { margin: 0 0 20px 0; color: #333; }
        .activity-item { padding: 15px 0; border-bottom: 1px solid #eee; display: flex; justify-content: between; align-items: center; }
        .activity-item:last-child { border-bottom: none; }
        .activity-info { flex: 1; }
        .activity-date { color: #666; font-size: 12px; }
        .activity-status { padding: 4px 8px; border-radius: 12px; font-size: 11px; font-weight: 500; text-transform: uppercase; }
        .status-pending { background: #fff3cd; color: #856404; }
        .status-approved { background: #d1ecf1; color: #0c5460; }
        .status-completed { background: #d4edda; color: #155724; }
        .status-rejected { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">🛒 Sistema de Compras</div>
            <div class="user-info">
                <div>
                    <div class="name">{{.User.Name}}</div>
                    <div class="role">{{.User.GetRoleDisplayName}} - {{.User.Office}}</div>
                </div>
                <a href="/logout" class="logout">Salir</a>
            </div>
        </div>
    </div>

    <div class="container">
        <div class="welcome">
            <h1>¡Bienvenido, {{.User.Name}}!</h1>
            <p>Dashboard personalizado para {{.User.GetRoleDisplayName}} del departamento {{.User.Department}}</p>
        </div>

        <!-- Estadísticas -->
        <div class="stats-grid">
            <div class="stat-card">
                <h3>Mis Solicitudes</h3>
                <p class="number">{{.Stats.MyRequests}}</p>
            </div>
            {{if .Permissions.CanApprove}}
            <div class="stat-card warning">
                <h3>Pendientes de Aprobar</h3>
                <p class="number">{{.Stats.PendingApproval}}</p>
            </div>
            {{end}}
            {{if .Permissions.CanViewAllRequests}}
            <div class="stat-card success">
                <h3>Aprobadas Hoy</h3>
                <p class="number">{{.Stats.ApprovedToday}}</p>
            </div>
            <div class="stat-card">
                <h3>Monto Total</h3>
                <p class="number">€{{printf "%.0f" .Stats.TotalAmount}}</p>
            </div>
            {{end}}
        </div>

        <!-- Acciones principales -->
        <div class="actions-grid">
            <!-- Crear nueva solicitud - Todos pueden -->
            <div class="action-card">
                <h3>🛒 Nueva Solicitud</h3>
                <p>Solicita productos de Amazon para tu trabajo</p>
                {{if .Permissions.CanRequestForOthers}}
                <p><small>✨ Puedes solicitar para tu equipo</small></p>
                {{end}}
                <a href="/request/new" class="action-btn">Crear Solicitud</a>
            </div>

            <!-- Aprobaciones pendientes - Solo managers+ -->
            {{if .Permissions.CanApprove}}
            <div class="action-card">
                <h3>✅ Aprobaciones Pendientes</h3>
                <p>Revisa y aprueba solicitudes de tu equipo</p>
                {{if gt .Stats.PendingApproval 0}}
                <p><small>⚠️ Tienes {{.Stats.PendingApproval}} solicitudes esperando</small></p>
                {{end}}
                <a href="/approvals/pending" class="action-btn warning">Ver Pendientes ({{.Stats.PendingApproval}})</a>
            </div>
            {{end}}

            <!-- Panel de administración - Solo admin -->
            {{if .Permissions.CanViewAdminPanel}}
            <div class="action-card">
                <h3>⚙️ Panel de Admin</h3>
                <p>Administrar usuarios, reportes y configuración</p>
                <a href="/admin/dashboard" class="action-btn secondary">Ir a Admin</a>
            </div>
            {{end}}

            <!-- Delegaciones - Todos los usuarios -->
            <div class="action-card">
                {{if .Permissions.CanDelegate}}
                <h3>🔄 Gestionar Delegaciones</h3>
                <p>Crea y gestiona tus delegaciones de aprobación</p>
                <a href="/delegation/list" class="action-btn secondary">Gestionar Delegaciones</a>
                {{else}}
                <h3>📥 Delegaciones Recibidas</h3>
                <p>Ver delegaciones de aprobación que has recibido</p>
                <a href="/delegation/list" class="action-btn secondary">Ver Delegaciones</a>
                {{end}}
            </div>
        </div>

        <!-- Información de permisos -->
        <div class="permission-note">
            <strong>Tus permisos actuales:</strong>
            {{if .Permissions.MaxApprovalAmount}}
            Puedes auto-aprobar hasta €{{printf "%.0f" .Permissions.MaxApprovalAmount}} •
            {{end}}
            {{if .Permissions.CanApprove}}Puedes aprobar solicitudes • {{end}}
            {{if .Permissions.CanRequestForOthers}}Puedes solicitar para otros • {{end}}
            {{if .Permissions.CanViewAllRequests}}Puedes ver todas las solicitudes{{end}}
        </div>

        <!-- Actividad reciente -->
        <div class="recent-activity">
            <h3>📋 Actividad Reciente</h3>
            {{if .RecentRequests}}
                {{range .RecentRequests}}
                <div class="activity-item">
                    <div class="activity-info">
                        <strong>Solicitud {{.ID}}</strong><br>
                        <small>€{{printf "%.2f" .Cart.TotalAmount}} • {{len .Cart.Items}} productos • {{.EmployeeID}}</small>
                        <div class="activity-date">{{.CreatedAt.Format "2006-01-02 15:04"}}</div>
                    </div>
                    <span class="activity-status status-{{.Status}}">{{.Status}}</span>
                </div>
                {{end}}
            {{else}}
                <p style="color: #666; font-style: italic;">No hay actividad reciente</p>
            {{end}}
        </div>
    </div>

    <script>
        // Auto refresh cada 30 segundos para actualizaciones
        setTimeout(() => location.reload(), 30000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.New("dashboard").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func newRequestHandler(w http.ResponseWriter, r *http.Request) {
	user := services.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	perms := user.GetPermissions()

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Nueva Solicitud - Sistema de Compras</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input, textarea, select { width: 100%; padding: 8px; margin-bottom: 10px; }
        textarea { height: 100px; }
        button { background: #007cba; color: white; padding: 10px 20px; border: none; cursor: pointer; }
        button:hover { background: #005a87; }
        .url-input { margin-bottom: 10px; }
        .add-url { background: #28a745; margin-top: 10px; }
        .user-context { background: #e9ecef; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .back-link { margin-bottom: 20px; }
        .back-link a { color: #007cba; text-decoration: none; }
        .permission-info { background: #d1ecf1; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="back-link">
            <a href="/dashboard">← Volver al Dashboard</a>
        </div>

        <h1>Nueva Solicitud de Compra</h1>
        
        <div class="user-context">
            <strong>Solicitante:</strong> {{.Name}} ({{.GetRoleDisplayName}})
            {{if .MaxApproval}}
            <br><strong>Auto-aprobación:</strong> Hasta €{{printf "%.0f" .MaxApproval}}
            {{end}}
        </div>

        {{if .Permissions.CanRequestForOthers}}
        <div class="permission-info">
            ✨ Como {{.GetRoleDisplayName}}, puedes solicitar para miembros de tu equipo
        </div>
        {{end}}
        
        <form action="/request/submit" method="post">
            <input type="hidden" name="requested_by" value="{{.ID}}">

            {{if .Permissions.CanRequestForOthers}}
            <div class="form-group">
                <label for="on_behalf_of">Solicitar para:</label>
                <select id="on_behalf_of" name="on_behalf_of">
                    <option value="">Para mí ({{.Name}})</option>
                    {{range .Permissions.Subordinates}}
                    <option value="{{.}}">{{.}} (Subordinado)</option>
                    {{end}}
                </select>
            </div>
            {{end}}

            <div class="form-group">
                <label for="delivery_office">Oficina de Entrega:</label>
                <select id="delivery_office" name="delivery_office" required>
                    <option value="">Seleccionar oficina</option>
                    <option value="madrid" {{if eq .Office "madrid"}}selected{{end}}>Madrid - Oficina Central</option>
                    <option value="barcelona" {{if eq .Office "barcelona"}}selected{{end}}>Barcelona - Oficina Norte</option>
                    <option value="valencia">Valencia - Oficina Este</option>
                    <option value="sevilla">Sevilla - Oficina Sur</option>
                </select>
            </div>

            <div class="form-group">
                <label>URLs de Productos Amazon:</label>
                <div id="url-container">
                    <input type="url" name="product_urls" placeholder="https://amazon.es/dp/..." required>
                </div>
                <button type="button" class="add-url" onclick="addUrlField()">+ Añadir URL</button>
            </div>

            <div class="form-group">
                <label for="justification">Justificación de la Compra:</label>
                <textarea id="justification" name="justification" required 
                          placeholder="Explique por qué necesita estos productos y cómo beneficiarán a la empresa..."></textarea>
            </div>

            <button type="submit">Enviar Solicitud</button>
        </form>
    </div>

    <script>
        function addUrlField() {
            const container = document.getElementById('url-container');
            const newInput = document.createElement('input');
            newInput.type = 'url';
            newInput.name = 'product_urls';
            newInput.placeholder = 'https://amazon.es/dp/...';
            newInput.className = 'url-input';
            container.appendChild(newInput);
        }

        // Pre-fill some example URLs for testing
        window.onload = function() {
            const examples = [
                'https://amazon.es/dp/B08N5WRWNW',
                'https://amazon.es/dp/B07XJ8C8F5'
            ];
            
            const container = document.getElementById('url-container');
            const firstInput = container.querySelector('input');
            firstInput.value = examples[0];
            
            // Add second example
            addUrlField();
            const inputs = container.querySelectorAll('input');
            inputs[1].value = examples[1];
        }
    </script>
</body>
</html>`

	data := struct {
		*models.User
		Permissions models.Permissions
	}{
		User:        user,
		Permissions: perms,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.New("newRequest").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Funciones auxiliares para obtener datos del dashboard
func getDashboardStats(user models.User) models.DashboardStats {
	// Mock stats - en producción consultaría la base de datos
	return models.DashboardStats{
		MyRequests:      5,
		PendingApproval: 3,
		ApprovedToday:   12,
		TotalAmount:     2540.50,
	}
}

func getRecentRequests(user models.User) []models.PurchaseRequest {
	// Mock data - en producción consultaría workflows activos
	return []models.PurchaseRequest{
		{
			ID:         "req-001",
			EmployeeID: user.ID,
			Status:     models.StatusPending,
			Cart: models.Cart{
				TotalAmount: 89.99,
				Items: []models.CartItem{
					{Title: "Echo Dot", Price: 29.99, Quantity: 1},
					{Title: "Fire Stick", Price: 59.99, Quantity: 1},
				},
			},
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}
}

func getPendingApprovals(user models.User) []models.PurchaseRequest {
	if !user.GetPermissions().CanApprove {
		return []models.PurchaseRequest{}
	}
	
	// Mock pending approvals
	return []models.PurchaseRequest{
		{
			ID:         "req-002",
			EmployeeID: "empleado@empresa.com",
			Status:     models.StatusPending,
			Cart: models.Cart{
				TotalAmount: 150.00,
				Items: []models.CartItem{
					{Title: "Keyboard", Price: 150.00, Quantity: 1},
				},
			},
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}
}

func pendingApprovalsHandler(w http.ResponseWriter, r *http.Request) {
	user := services.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verificar si tiene permisos directos o delegados
	perms := user.GetPermissions()
	delegationID := r.URL.Query().Get("delegation")
	
	if !perms.CanApprove && delegationID == "" {
		http.Error(w, "No tienes permisos de aprobación. Usa una delegación específica.", http.StatusForbidden)
		return
	}

	pendingApprovals := getPendingApprovals(*user)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Aprobaciones Pendientes</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 1000px; margin: 0 auto; }
        .back-link { margin-bottom: 20px; }
        .back-link a { color: #007cba; text-decoration: none; }
        .approval-item { background: #f8f9fa; border: 1px solid #dee2e6; border-radius: 5px; padding: 20px; margin-bottom: 20px; }
        .approval-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .approval-actions { margin-top: 15px; }
        .approve-btn { background: #28a745; color: white; padding: 10px 20px; border: none; cursor: pointer; margin-right: 10px; }
        .reject-btn { background: #dc3545; color: white; padding: 10px 20px; border: none; cursor: pointer; }
        .view-btn { background: #007cba; color: white; padding: 10px 20px; border: none; cursor: pointer; text-decoration: none; display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <div class="back-link">
            <a href="/dashboard">← Volver al Dashboard</a>
        </div>

        <h1>Aprobaciones Pendientes</h1>
        {{if .DelegationID}}
        <div style="background: #e7f3ff; border-left: 4px solid #007bff; padding: 15px; margin-bottom: 20px;">
            <strong>🔄 Usando Delegación:</strong> {{.DelegationID}}<br>
            <small>Estás aprobando con permisos delegados de otro usuario</small>
        </div>
        {{end}}
        <p>Bienvenido {{.User.Name}}, puedes aprobar solicitudes hasta €{{printf "%.0f" .Permissions.MaxApprovalAmount}}</p>

        {{if .PendingApprovals}}
            {{range .PendingApprovals}}
            <div class="approval-item">
                <div class="approval-header">
                    <div>
                        <h3>Solicitud {{.ID}}</h3>
                        <p><strong>Empleado:</strong> {{.EmployeeID}}</p>
                        <p><strong>Total:</strong> €{{printf "%.2f" .Cart.TotalAmount}} • {{len .Cart.Items}} productos</p>
                        <p><strong>Fecha:</strong> {{.CreatedAt.Format "2006-01-02 15:04"}}</p>
                    </div>
                    <div class="approval-actions">
                        <a href="/approval/{{.ID}}" class="view-btn">Ver Detalles y Aprobar</a>
                    </div>
                </div>
            </div>
            {{end}}
        {{else}}
            <p style="color: #666; font-style: italic;">No hay aprobaciones pendientes</p>
        {{end}}
    </div>
</body>
</html>`

	data := struct {
		User             models.User
		Permissions      models.Permissions
		PendingApprovals []models.PurchaseRequest
		DelegationID     string
	}{
		User:             *user,
		Permissions:      user.GetPermissions(),
		PendingApprovals: pendingApprovals,
		DelegationID:     delegationID,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.New("pendingApprovals").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := services.GetCurrentUser(r)
	if user == nil || !user.GetPermissions().CanViewAdminPanel {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Admin dashboard placeholder
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Panel de Administración</title></head>
<body style="font-family: Arial, sans-serif; margin: 40px;">
    <a href="/dashboard">← Volver al Dashboard</a>
    <h1>🛠️ Panel de Administración</h1>
    <p>Funcionalidades admin en desarrollo...</p>
    <ul>
        <li>Gestión de usuarios</li>
        <li>Reportes financieros</li>
        <li>Configuración del sistema</li>
        <li>Auditoría de aprobaciones</li>
    </ul>
</body>
</html>`)
}

// homeHandler removed - now using dashboardHandler with auth

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Create purchase request
	request := models.PurchaseRequest{
		ID:             uuid.New().String(),
		EmployeeID:     r.FormValue("employee_id"),
		CreatedAt:      time.Now(),
		Status:         models.StatusPending,
		ProductURLs:    r.Form["product_urls"],
		Justification:  r.FormValue("justification"),
		DeliveryOffice: r.FormValue("delivery_office"),
		ApprovalFlow:   models.ApprovalFlow{},
	}

	// Start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        request.ID,
		TaskQueue: taskQueue,
	}

	workflowRun, err := temporalClient.ExecuteWorkflow(context.Background(), workflowOptions, workflows.PurchaseApprovalWorkflow, request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting workflow: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Started workflow for request %s (RunID: %s)", request.ID, workflowRun.GetRunID())

	// Redirect to status page
	http.Redirect(w, r, fmt.Sprintf("/status?id=%s", request.ID), http.StatusSeeOther)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("id")
	if requestID == "" {
		http.Error(w, "Request ID required", http.StatusBadRequest)
		return
	}

	// Query workflow status
	var result models.PurchaseRequest
	queryResult, err := temporalClient.QueryWorkflow(context.Background(), requestID, "", "getStatus")
	if err != nil {
		log.Printf("Error querying workflow status: %v", err)
		result = models.PurchaseRequest{
			ID:     requestID,
			Status: "unknown",
		}
	} else {
		err = queryResult.Get(&result)
		if err != nil {
			log.Printf("Error getting query result: %v", err)
		}
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Estado de Solicitud</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .status { padding: 20px; border-radius: 5px; margin: 20px 0; }
        .pending { background: #fff3cd; border: 1px solid #ffeaa7; }
        .approved { background: #d4edda; border: 1px solid #c3e6cb; }
        .rejected { background: #f8d7da; border: 1px solid #f5c6cb; }
        .completed { background: #d1ecf1; border: 1px solid #b6dfea; }
        .back-link { margin-top: 20px; }
        .refresh { background: #6c757d; color: white; padding: 10px 20px; border: none; cursor: pointer; margin-top: 10px; }
    </style>
    <script>
        setTimeout(function() {
            location.reload();
        }, 10000); // Auto-refresh every 10 seconds
    </script>
</head>
<body>
    <div class="container">
        <h1>Estado de la Solicitud</h1>
        <p><strong>ID:</strong> {{.ID}}</p>
        <p><strong>Empleado:</strong> {{.EmployeeID}}</p>
        <p><strong>Fecha:</strong> {{.CreatedAt.Format "2006-01-02 15:04:05"}}</p>
        
        <div class="status {{.Status}}">
            <h3>Estado Actual: {{.Status | title}}</h3>
            {{if eq .Status "pending"}}
                <p>Su solicitud está siendo procesada. Se han notificado los responsables para su aprobación.</p>
            {{else if eq .Status "approved"}}
                <p>Su solicitud ha sido aprobada y se está procesando la compra.</p>
            {{else if eq .Status "rejected"}}
                <p>Su solicitud ha sido rechazada.</p>
            {{else if eq .Status "completed"}}
                <p>¡Su compra ha sido completada exitosamente!</p>
            {{else}}
                <p>Estado desconocido. La solicitud está siendo procesada.</p>
            {{end}}
        </div>

        {{if .Cart.Items}}
        <h3>Productos Solicitados:</h3>
        <ul>
            {{range .Cart.Items}}
            <li>{{.Title}} - €{{.Price}} ({{if .IsValid}}✓ Válido{{else}}✗ Inválido{{end}})</li>
            {{end}}
        </ul>
        <p><strong>Total:</strong> €{{.Cart.TotalAmount}}</p>
        {{end}}

        <button class="refresh" onclick="location.reload()">Actualizar Estado</button>
        
        <div class="back-link">
            <a href="/">← Volver al Inicio</a>
        </div>
    </div>
</body>
</html>`

	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return fmt.Sprintf("%c%s", s[0]-32, s[1:])
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, _ := template.New("status").Funcs(funcMap).Parse(tmpl)
	t.Execute(w, result)
}

func approvalHandler(w http.ResponseWriter, r *http.Request) {
	// Extract request ID from URL path
	requestID := r.URL.Path[len("/approval/"):]
	if requestID == "" {
		http.Error(w, "Request ID required", http.StatusBadRequest)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Solicitud de Aprobación</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .approval-form { background: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0; }
        button { padding: 10px 20px; margin: 10px 5px; border: none; cursor: pointer; }
        .approve { background: #28a745; color: white; }
        .reject { background: #dc3545; color: white; }
        textarea { width: 100%; height: 80px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Solicitud de Aprobación</h1>
        <p><strong>ID de Solicitud:</strong> {{.}}</p>
        
        <div class="approval-form">
            <h3>Decisión de Aprobación</h3>
            <form action="/approve" method="post">
                <input type="hidden" name="request_id" value="{{.}}">
                <input type="hidden" name="responsible_id" value="responsible@empresa.com">
                
                <label>
                    <input type="radio" name="decision" value="approve" required> Aprobar
                </label><br>
                <label>
                    <input type="radio" name="decision" value="reject" required> Rechazar
                </label><br>
                
                <label for="reason">Comentarios:</label>
                <textarea name="reason" placeholder="Motivo de la decisión (opcional para aprobación, obligatorio para rechazo)..."></textarea>
                
                <button type="submit" class="approve">Enviar Decisión</button>
            </form>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, _ := template.New("approval").Parse(tmpl)
	t.Execute(w, requestID)
}

func approveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	requestID := r.FormValue("request_id")
	decision := r.FormValue("decision")
	reason := r.FormValue("reason")
	responsibleID := r.FormValue("responsible_id")

	if decision == "reject" && reason == "" {
		http.Error(w, "Reason required for rejection", http.StatusBadRequest)
		return
	}

	// Send signal to workflow
	approvalResponse := models.ApprovalResponse{
		RequestID:     requestID,
		ResponsibleID: responsibleID,
		Approved:      decision == "approve",
		Reason:        reason,
		RespondedAt:   time.Now(),
	}

	err = temporalClient.SignalWorkflow(context.Background(), requestID, "", "approval", approvalResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error sending approval signal: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Sent approval signal for request %s: %v", requestID, approvalResponse)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Decisión Registrada</title></head>
<body>
    <h1>Decisión Registrada</h1>
    <p>Su decisión ha sido registrada exitosamente.</p>
    <p><strong>Solicitud:</strong> %s</p>
    <p><strong>Decisión:</strong> %s</p>
    <p><a href="/status?id=%s">Ver estado de la solicitud</a></p>
</body>
</html>`, requestID, decision, requestID)
}