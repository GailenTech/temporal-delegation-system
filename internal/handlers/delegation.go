package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"go.temporal.io/sdk/client"
	"temporal-workflow/internal/models"
	"temporal-workflow/internal/workflows"
)

// GetUserSession obtiene la sesión del usuario actual desde el contexto
func GetUserSession(r *http.Request) (*models.UserSession, error) {
	// Obtener usuario del contexto (puesto ahí por el middleware de auth)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}

	// Crear sesión mock - en producción esto vendría del servicio de auth
	session := &models.UserSession{
		User:        *user,
		Permissions: user.GetPermissions(),
		LoginTime:   time.Now().Add(-time.Hour), // Mock login hace 1 hora
		LastActive:  time.Now(),
	}

	return session, nil
}

// DelegationHandlers maneja las rutas relacionadas con delegaciones
type DelegationHandlers struct {
	temporalClient client.Client
}

// NewDelegationHandlers crea una nueva instancia de los handlers de delegación
func NewDelegationHandlers(temporalClient client.Client) *DelegationHandlers {
	return &DelegationHandlers{
		temporalClient: temporalClient,
	}
}

// CreateDelegationPage muestra el formulario para crear una nueva delegación
func (h *DelegationHandlers) CreateDelegationPage(w http.ResponseWriter, r *http.Request) {
	// Verificar autenticación
	session, err := GetUserSession(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Verificar permisos de delegación
	if !session.Permissions.CanDelegate {
		http.Error(w, "No tienes permisos para crear delegaciones", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nueva Delegación - Sistema de Compras</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 5px; font-weight: bold; color: #555; }
        input, select, textarea { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
        textarea { height: 80px; resize: vertical; }
        .button { background-color: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        .button:hover { background-color: #0056b3; }
        .button-secondary { background-color: #6c757d; margin-left: 10px; }
        .button-secondary:hover { background-color: #545b62; }
        .info-box { background-color: #e7f3ff; border-left: 4px solid #007bff; padding: 15px; margin-bottom: 20px; }
        .datetime-input { display: flex; gap: 10px; }
        .datetime-input input { flex: 1; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔄 Nueva Delegación de Aprobaciones</h1>
        
        <div class="info-box">
            <strong>📋 Información:</strong> Las delegaciones te permiten transferir temporalmente tus permisos de aprobación a otro usuario. 
            La delegación será gestionada automáticamente por el sistema según las fechas especificadas.
        </div>

        <form action="/delegation/create" method="POST">
            <div class="form-group">
                <label for="to_user_id">👤 Delegar a (Usuario destino):</label>
                <select id="to_user_id" name="to_user_id" required>
                    <option value="">Selecciona un usuario...</option>
                    {{range .Users}}
                    <option value="{{.ID}}">{{.Name}} ({{.GetRoleDisplayName}}) - {{.Department}}</option>
                    {{end}}
                </select>
            </div>

            <div class="form-group">
                <label for="start_date">📅 Fecha de inicio:</label>
                <input type="datetime-local" id="start_date" name="start_date" required 
                       min="{{.MinDate}}" value="{{.DefaultStartDate}}">
            </div>

            <div class="form-group">
                <label for="end_date">📅 Fecha de finalización:</label>
                <input type="datetime-local" id="end_date" name="end_date" required 
                       min="{{.MinDate}}">
            </div>

            <div class="form-group">
                <label for="max_amount">💰 Límite máximo de aprobación (€):</label>
                <input type="number" id="max_amount" name="max_amount" required 
                       min="1" max="{{.UserMaxAmount}}" step="0.01" 
                       placeholder="Máximo: {{.UserMaxAmount}}€">
            </div>

            <div class="form-group">
                <label for="reason">📝 Motivo de la delegación:</label>
                <textarea id="reason" name="reason" required 
                          placeholder="Ej: Vacaciones, viaje de negocios, ausencia temporal..."></textarea>
            </div>

            <div style="text-align: center; margin-top: 30px;">
                <button type="submit" class="button">✅ Crear Delegación</button>
                <button type="button" class="button button-secondary" onclick="window.location.href='/delegation/list'">❌ Cancelar</button>
            </div>
        </form>
    </div>

    <script>
        // Validación del formulario
        document.querySelector('form').addEventListener('submit', function(e) {
            const startDate = new Date(document.getElementById('start_date').value);
            const endDate = new Date(document.getElementById('end_date').value);
            
            if (endDate <= startDate) {
                alert('La fecha de finalización debe ser posterior a la fecha de inicio');
                e.preventDefault();
                return;
            }
            
            const maxDays = 30; // Máximo 30 días de delegación
            const diffDays = (endDate - startDate) / (1000 * 60 * 60 * 24);
            if (diffDays > maxDays) {
                alert('La delegación no puede exceder los ' + maxDays + ' días');
                e.preventDefault();
                return;
            }
        });

        // Auto-calcular fecha de fin mínima cuando cambia la fecha de inicio
        document.getElementById('start_date').addEventListener('change', function() {
            const startDate = new Date(this.value);
            const minEndDate = new Date(startDate.getTime() + 60 * 60 * 1000); // +1 hora mínimo
            document.getElementById('end_date').min = minEndDate.toISOString().slice(0, 16);
        });
    </script>
</body>
</html>`

	// Obtener lista de usuarios disponibles para delegación
	users := []models.User{}
	for _, user := range models.MockUsers {
		// No incluir al usuario actual ni a usuarios sin permisos de aprobación
		if user.ID != session.User.ID {
			perms := user.GetPermissions()
			if perms.CanApprove || user.Role == models.RoleManager || user.Role == models.RoleCEO {
				users = append(users, user)
			}
		}
	}

	data := struct {
		Users            []models.User
		MinDate          string
		DefaultStartDate string
		UserMaxAmount    float64
	}{
		Users:            users,
		MinDate:          time.Now().Format("2006-01-02T15:04"),
		DefaultStartDate: time.Now().Add(time.Hour).Format("2006-01-02T15:04"),
		UserMaxAmount:    session.Permissions.MaxApprovalAmount,
	}

	t, err := template.New("create_delegation").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

// CreateDelegation procesa la creación de una nueva delegación
func (h *DelegationHandlers) CreateDelegation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verificar autenticación
	session, err := GetUserSession(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Verificar permisos
	if !session.Permissions.CanDelegate {
		http.Error(w, "No tienes permisos para crear delegaciones", http.StatusForbidden)
		return
	}

	// Parsear formulario
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Extraer datos del formulario
	toUserID := r.FormValue("to_user_id")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")
	maxAmountStr := r.FormValue("max_amount")
	reason := r.FormValue("reason")

	// Validar datos
	if toUserID == "" || startDateStr == "" || endDateStr == "" || maxAmountStr == "" || reason == "" {
		http.Error(w, "Todos los campos son obligatorios", http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02T15:04", startDateStr)
	if err != nil {
		http.Error(w, "Fecha de inicio inválida", http.StatusBadRequest)
		return
	}

	endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
	if err != nil {
		http.Error(w, "Fecha de finalización inválida", http.StatusBadRequest)
		return
	}

	maxAmount, err := strconv.ParseFloat(maxAmountStr, 64)
	if err != nil {
		http.Error(w, "Monto máximo inválido", http.StatusBadRequest)
		return
	}

	// Validaciones de negocio
	if endDate.Before(startDate) {
		http.Error(w, "La fecha de finalización debe ser posterior a la fecha de inicio", http.StatusBadRequest)
		return
	}

	if maxAmount > session.Permissions.MaxApprovalAmount {
		http.Error(w, "El monto máximo excede tus permisos de aprobación", http.StatusBadRequest)
		return
	}

	// Crear delegación
	delegationID := fmt.Sprintf("delegation_%s_%d", session.User.ID, time.Now().Unix())
	delegation := models.Delegation{
		ID:         delegationID,
		FromUserID: session.User.ID,
		ToUserID:   toUserID,
		StartDate:  startDate,
		EndDate:    endDate,
		MaxAmount:  maxAmount,
		Reason:     reason,
		IsActive:   false,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
		CreatedBy:  session.User.ID,
	}

	// Iniciar workflow de delegación en Temporal
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("delegation-workflow-%s", delegationID),
		TaskQueue: "purchase-approval-task-queue",
	}

	input := workflows.DelegationWorkflowInput{
		Delegation: delegation,
	}

	workflowRun, err := h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflows.DelegationWorkflow, input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error starting delegation workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Guardar el workflow ID en la delegación
	delegation.WorkflowID = workflowRun.GetID()

	// TODO: Guardar delegación en base de datos

	// Respuesta JSON para AJAX o redirección
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"delegation_id": delegationID,
			"workflow_id":   workflowRun.GetID(),
			"message":       "Delegación creada exitosamente",
		})
	} else {
		// Redireccionar a la lista de delegaciones
		http.Redirect(w, r, "/delegation/list?success=created", http.StatusFound)
	}
}

// ListDelegations muestra todas las delegaciones del usuario actual
func (h *DelegationHandlers) ListDelegations(w http.ResponseWriter, r *http.Request) {
	// Verificar autenticación
	session, err := GetUserSession(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mis Delegaciones - Sistema de Compras</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        .tabs { display: flex; margin-bottom: 20px; border-bottom: 1px solid #ddd; }
        .tab { padding: 10px 20px; cursor: pointer; border-bottom: 2px solid transparent; }
        .tab.active { border-bottom-color: #007bff; background-color: #f8f9fa; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .delegation-card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin-bottom: 15px; }
        .delegation-card.active { border-color: #28a745; background-color: #f8fff9; }
        .delegation-card.expired { border-color: #dc3545; background-color: #fff8f8; }
        .delegation-card.pending { border-color: #ffc107; background-color: #fffdf7; }
        .status-badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status-active { background-color: #28a745; color: white; }
        .status-expired { background-color: #dc3545; color: white; }
        .status-pending { background-color: #ffc107; color: black; }
        .status-cancelled { background-color: #6c757d; color: white; }
        .button { background-color: #007bff; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; text-decoration: none; display: inline-block; }
        .button:hover { background-color: #0056b3; }
        .button-danger { background-color: #dc3545; }
        .button-danger:hover { background-color: #c82333; }
        .button-warning { background-color: #ffc107; color: black; }
        .button-warning:hover { background-color: #e0a800; }
        .no-delegations { text-align: center; color: #666; padding: 40px; }
        .actions { margin-top: 15px; }
        .actions button, .actions a { margin-right: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔄 Mis Delegaciones</h1>
        
        <div style="margin-bottom: 20px; display: flex; justify-content: space-between; align-items: center;">
            <div>
                {{if .User.Permissions.CanDelegate}}
                <a href="/delegation/new" class="button" style="background-color: #28a745; font-size: 16px; padding: 12px 24px;">➕ Nueva Delegación</a>
                {{end}}
                <a href="/dashboard" class="button" style="background-color: #6c757d; margin-left: 10px;">🏠 Volver al Dashboard</a>
            </div>
            <div style="color: #666; font-size: 14px;">
                {{if .User.Permissions.CanDelegate}}
                💡 <strong>Tip:</strong> Las delegaciones permiten transferir temporalmente tus permisos de aprobación
                {{else}}
                📥 <strong>Info:</strong> Aquí puedes ver las delegaciones que has recibido de otros usuarios
                {{end}}
            </div>
        </div>

        <div class="tabs">
            {{if .User.Permissions.CanDelegate}}
            <div class="tab active" onclick="showTab('created')">📤 Delegaciones Creadas ({{len .CreatedDelegations}})</div>
            <div class="tab" onclick="showTab('received')">📥 Delegaciones Recibidas ({{len .ReceivedDelegations}})</div>
            {{else}}
            <div class="tab active" onclick="showTab('received')">📥 Delegaciones Recibidas ({{len .ReceivedDelegations}})</div>
            {{end}}
        </div>

        {{if .User.Permissions.CanDelegate}}
        <div id="created" class="tab-content active">
            {{if .CreatedDelegations}}
                {{range .CreatedDelegations}}
                <div class="delegation-card {{.StatusClass}}">
                    <div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 15px;">
                        <div style="flex: 1;">
                            <div style="display: flex; align-items: center; margin-bottom: 10px;">
                                <h3 style="margin: 0; margin-right: 15px;">👤 {{.ToUserName}}</h3>
                                <span class="status-badge status-{{.StatusClass}}">{{.StatusText}}</span>
                            </div>
                            
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 15px; margin-bottom: 10px;">
                                <div>
                                    <strong>📅 Período:</strong><br>
                                    <span style="color: #666;">{{.StartDateFormatted}}</span><br>
                                    <span style="color: #666;">{{.EndDateFormatted}}</span>
                                </div>
                                <div>
                                    <strong>💰 Límites:</strong><br>
                                    <span style="color: #28a745;">Máximo: {{.MaxAmountFormatted}}€</span><br>
                                    <span style="color: #dc3545;">Usado: {{.UsedAmountFormatted}}€</span>
                                </div>
                            </div>
                            
                            <div style="margin-bottom: 10px;">
                                <strong>📝 Motivo:</strong><br>
                                <span style="color: #666; font-style: italic;">"{{.Reason}}"</span>
                            </div>
                            
                            <div style="font-size: 12px; color: #999;">
                                <strong>🆔 ID:</strong> {{.ID}}
                            </div>
                        </div>
                    </div>
                    <div class="actions">
                        {{if eq .Status "active"}}
                            <button class="button button-warning" onclick="extendDelegation('{{.ID}}')">⏰ Extender</button>
                            <button class="button button-warning" onclick="modifyAmount('{{.ID}}')">💰 Modificar Límite</button>
                            <button class="button button-danger" onclick="cancelDelegation('{{.ID}}')">❌ Cancelar</button>
                        {{end}}
                        {{if eq .Status "pending"}}
                            <button class="button button-warning" onclick="modifyDelegation('{{.ID}}')">✏️ Modificar</button>
                            <button class="button button-danger" onclick="cancelDelegation('{{.ID}}')">❌ Cancelar</button>
                        {{end}}
                        <button class="button" onclick="viewDelegationDetails('{{.ID}}')">👁️ Ver Detalles</button>
                    </div>
                </div>
                {{end}}
            {{else}}
                <div class="no-delegations">
                    <h3>📋 No has creado ninguna delegación</h3>
                    <p>Las delegaciones te permiten transferir temporalmente tus permisos de aprobación a otros usuarios.</p>
                    <a href="/delegation/new" class="button">➕ Crear Primera Delegación</a>
                </div>
            {{end}}
        </div>
        {{end}}

        <div id="received" class="tab-content{{if not .User.Permissions.CanDelegate}} active{{end}}">
            {{if .ReceivedDelegations}}
                {{range .ReceivedDelegations}}
                <div class="delegation-card {{.StatusClass}}">
                    <div style="display: flex; justify-content: space-between; align-items: start; margin-bottom: 15px;">
                        <div style="flex: 1;">
                            <div style="display: flex; align-items: center; margin-bottom: 10px;">
                                <h3 style="margin: 0; margin-right: 15px;">📥 De: {{.FromUserName}}</h3>
                                <span class="status-badge status-{{.StatusClass}}">{{.StatusText}}</span>
                            </div>
                            
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 15px; margin-bottom: 10px;">
                                <div>
                                    <strong>📅 Período:</strong><br>
                                    <span style="color: #666;">{{.StartDateFormatted}}</span><br>
                                    <span style="color: #666;">{{.EndDateFormatted}}</span>
                                </div>
                                <div>
                                    <strong>💰 Límites:</strong><br>
                                    <span style="color: #28a745;">Puedes aprobar: {{.MaxAmountFormatted}}€</span><br>
                                    <span style="color: #dc3545;">Ya usado: {{.UsedAmountFormatted}}€</span>
                                </div>
                            </div>
                            
                            <div style="margin-bottom: 10px;">
                                <strong>📝 Motivo:</strong><br>
                                <span style="color: #666; font-style: italic;">"{{.Reason}}"</span>
                            </div>
                            
                            <div style="font-size: 12px; color: #999;">
                                <strong>🆔 ID:</strong> {{.ID}}
                            </div>
                        </div>
                    </div>
                    <div class="actions">
                        <button class="button" onclick="viewDelegationDetails('{{.ID}}')">👁️ Ver Detalles</button>
                        {{if eq .Status "active"}}
                            <a href="/pending-approvals?delegation={{.ID}}" class="button" style="background-color: #28a745;">✅ Usar para Aprobaciones</a>
                        {{end}}
                    </div>
                </div>
                {{end}}
            {{else}}
                <div class="no-delegations">
                    <h3>📥 No has recibido ninguna delegación</h3>
                    <p>Cuando otros usuarios te deleguen permisos de aprobación, aparecerán aquí.</p>
                </div>
            {{end}}
        </div>
    </div>

    <script>
        function showTab(tabName) {
            // Ocultar todas las tabs
            document.querySelectorAll('.tab-content').forEach(content => {
                content.classList.remove('active');
            });
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });
            
            // Mostrar tab seleccionada
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }

        function viewDelegationDetails(delegationId) {
            window.location.href = '/delegation/details/' + delegationId;
        }

        function extendDelegation(delegationId) {
            const newEndDate = prompt('Nueva fecha de finalización (YYYY-MM-DD HH:MM):');
            if (newEndDate) {
                // TODO: Implementar llamada AJAX para extender delegación
                alert('Funcionalidad de extensión será implementada');
            }
        }

        function modifyAmount(delegationId) {
            const newAmount = prompt('Nuevo límite de aprobación (€):');
            if (newAmount && !isNaN(newAmount)) {
                // TODO: Implementar llamada AJAX para modificar monto
                alert('Funcionalidad de modificación será implementada');
            }
        }

        function cancelDelegation(delegationId) {
            const reason = prompt('Motivo de la cancelación:');
            if (reason) {
                if (confirm('¿Estás seguro de que quieres cancelar esta delegación?')) {
                    // TODO: Implementar llamada AJAX para cancelar delegación
                    alert('Funcionalidad de cancelación será implementada');
                }
            }
        }
    </script>
</body>
</html>`

	// TODO: Obtener delegaciones reales de la base de datos
	// Por ahora usamos datos mock
	data := struct {
		User                *models.UserSession
		CreatedDelegations  []DelegationView
		ReceivedDelegations []DelegationView
	}{
		User:                session,
		CreatedDelegations:  getMockCreatedDelegations(session.User.ID),
		ReceivedDelegations: getMockReceivedDelegations(session.User.ID),
	}

	t, err := template.New("list_delegations").Parse(tmpl)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

// DelegationView vista para mostrar delegaciones en el template
type DelegationView struct {
	ID                  string
	FromUserName        string
	ToUserName          string
	StartDateFormatted  string
	EndDateFormatted    string
	MaxAmountFormatted  string
	UsedAmountFormatted string
	Reason              string
	Status              string
	StatusText          string
	StatusClass         string
}

// getMockCreatedDelegations devuelve delegaciones mock creadas por el usuario
func getMockCreatedDelegations(userID string) []DelegationView {
	if userID != "manager@empresa.com" {
		return []DelegationView{}
	}

	return []DelegationView{
		{
			ID:                  "delegation_202507_001",
			FromUserName:        "Ana Manager",
			ToUserName:          "Juan Empleado (Empleado - IT)",
			StartDateFormatted:  "2025-07-24 09:00",
			EndDateFormatted:    "2025-07-28 18:00",
			MaxAmountFormatted:  "1500.00",
			UsedAmountFormatted: "0.00",
			Reason:              "Vacaciones de verano - Delegación temporal durante mi ausencia por vacaciones programadas",
			Status:              "pending",
			StatusText:          "⏳ Pendiente de Activación",
			StatusClass:         "pending",
		},
		{
			ID:                  "delegation_202507_002",
			FromUserName:        "Ana Manager", 
			ToUserName:          "Carlos CEO (CEO - Executive)",
			StartDateFormatted:  "2025-07-20 08:00",
			EndDateFormatted:    "2025-07-25 17:00",
			MaxAmountFormatted:  "2000.00",
			UsedAmountFormatted: "350.75",
			Reason:              "Conferencia en Barcelona - Delegación durante viaje de negocios",
			Status:              "active",
			StatusText:          "✅ Activa",
			StatusClass:         "active",
		},
		{
			ID:                  "delegation_202507_003",
			FromUserName:        "Ana Manager",
			ToUserName:          "Sofia Admin (Admin - IT)",
			StartDateFormatted:  "2025-07-15 09:00", 
			EndDateFormatted:    "2025-07-18 18:00",
			MaxAmountFormatted:  "1000.00",
			UsedAmountFormatted: "1000.00",
			Reason:              "Capacitación en Madrid - Curso de liderazgo corporativo",
			Status:              "expired",
			StatusText:          "⏰ Expirada",
			StatusClass:         "expired",
		},
	}
}

// getMockReceivedDelegations devuelve delegaciones mock recibidas por el usuario
func getMockReceivedDelegations(userID string) []DelegationView {
	// Mostrar delegaciones recibidas según el usuario
	switch userID {
	case "empleado@empresa.com":
		return []DelegationView{
			{
				ID:                  "delegation_recv_001",
				FromUserName:        "Ana Manager (Manager - IT)",
				ToUserName:          "Juan Empleado",
				StartDateFormatted:  "2025-07-20 09:00",
				EndDateFormatted:    "2025-07-25 18:00",
				MaxAmountFormatted:  "2000.00",
				UsedAmountFormatted: "350.75",
				Reason:              "Cobertura durante conferencia en Barcelona",
				Status:              "active",
				StatusText:          "✅ Activa - Puedes Aprobar",
				StatusClass:         "active",
			},
		}
	case "ceo@empresa.com":
		return []DelegationView{
			{
				ID:                  "delegation_recv_002",
				FromUserName:        "Director Financiero (CFO - Finance)",
				ToUserName:          "Carlos CEO",
				StartDateFormatted:  "2025-07-21 08:00",
				EndDateFormatted:    "2025-07-30 17:00",
				MaxAmountFormatted:  "5000.00",
				UsedAmountFormatted: "1200.00",
				Reason:              "Auditoría anual - Delegación de aprobaciones financieras",
				Status:              "active",
				StatusText:          "✅ Activa - Aprobaciones hasta 5K€",
				StatusClass:         "active",
			},
		}
	default:
		return []DelegationView{}
	}
}

// ListReceivedDelegations muestra solo las delegaciones recibidas por el usuario
func (h *DelegationHandlers) ListReceivedDelegations(w http.ResponseWriter, r *http.Request) {
	// Verificar autenticación
	_, err := GetUserSession(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Redirigir a la lista completa pero con focus en delegaciones recibidas
	http.Redirect(w, r, "/delegation/list#received", http.StatusFound)
}