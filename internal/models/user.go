package models

import "time"

// User representa un usuario del sistema
type User struct {
	ID          string    `json:"id"`           // email del usuario
	Name        string    `json:"name"`         // Nombre completo
	Role        UserRole  `json:"role"`         // Rol del usuario
	Department  string    `json:"department"`   // Departamento
	Office      string    `json:"office"`       // Oficina
	ManagerID   string    `json:"manager_id"`   // ID del supervisor directo
	MaxApproval float64   `json:"max_approval"` // Límite de auto-aprobación
	CreatedAt   time.Time `json:"created_at"`
	LastLogin   time.Time `json:"last_login"`
}

// UserRole define los roles disponibles
type UserRole string

const (
	RoleEmployee UserRole = "employee" // Empleado normal
	RoleManager  UserRole = "manager"  // Manager de equipo
	RoleCEO      UserRole = "ceo"      // CEO/Director general
	RoleAdmin    UserRole = "admin"    // Administrador del sistema
)

// Permissions estructura los permisos del usuario
type Permissions struct {
	CanCreateRequest     bool     `json:"can_create_request"`
	CanApprove          bool     `json:"can_approve"`
	CanApproveForOthers bool     `json:"can_approve_for_others"`
	CanViewAllRequests  bool     `json:"can_view_all_requests"`
	CanViewAdminPanel   bool     `json:"can_view_admin_panel"`
	CanDelegate         bool     `json:"can_delegate"`
	MaxApprovalAmount   float64  `json:"max_approval_amount"`
	CanRequestForOthers bool     `json:"can_request_for_others"`
	Subordinates        []string `json:"subordinates"` // IDs de subordinados
}

// GetPermissions calcula los permisos basados en el rol
func (u *User) GetPermissions() Permissions {
	perms := Permissions{
		CanCreateRequest: true, // Todos pueden crear solicitudes
		Subordinates:     []string{},
	}

	switch u.Role {
	case RoleEmployee:
		// Empleados solo pueden crear solicitudes
		perms.MaxApprovalAmount = 0

	case RoleManager:
		perms.CanApprove = true
		perms.CanApproveForOthers = true
		perms.CanDelegate = true
		perms.CanRequestForOthers = true
		perms.MaxApprovalAmount = u.MaxApproval
		// TODO: Cargar subordinados desde DB/LDAP
		perms.Subordinates = getSubordinates(u.ID)

	case RoleCEO:
		perms.CanApprove = true
		perms.CanApproveForOthers = true
		perms.CanViewAllRequests = true
		perms.CanDelegate = true
		perms.CanRequestForOthers = true
		perms.MaxApprovalAmount = 999999 // Sin límite práctico

	case RoleAdmin:
		// Admins tienen todos los permisos
		perms.CanApprove = true
		perms.CanApproveForOthers = true
		perms.CanViewAllRequests = true
		perms.CanViewAdminPanel = true
		perms.CanDelegate = true
		perms.CanRequestForOthers = true
		perms.MaxApprovalAmount = 999999
	}

	return perms
}

// CanApproveAmount verifica si el usuario puede aprobar un monto específico
func (u *User) CanApproveAmount(amount float64) bool {
	perms := u.GetPermissions()
	return perms.CanApprove && amount <= perms.MaxApprovalAmount
}

// GetRoleDisplayName devuelve el nombre legible del rol
func (u *User) GetRoleDisplayName() string {
	switch u.Role {
	case RoleEmployee:
		return "Empleado"
	case RoleManager:
		return "Manager"
	case RoleCEO:
		return "CEO"
	case RoleAdmin:
		return "Administrador"
	default:
		return "Desconocido"
	}
}

// UserSession información de sesión del usuario
type UserSession struct {
	User        User        `json:"user"`
	Permissions Permissions `json:"permissions"`
	LoginTime   time.Time   `json:"login_time"`
	LastActive  time.Time   `json:"last_active"`
}

// DashboardStats estadísticas para el dashboard
type DashboardStats struct {
	MyRequests      int `json:"my_requests"`
	PendingApproval int `json:"pending_approval"`
	ApprovedToday   int `json:"approved_today"`
	TotalAmount     float64 `json:"total_amount"`
}

// Delegation delegación temporal de aprobaciones
type Delegation struct {
	ID          string    `json:"id"`
	FromUserID  string    `json:"from_user_id"`
	ToUserID    string    `json:"to_user_id"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	MaxAmount   float64   `json:"max_amount"`
	Reason      string    `json:"reason"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
	CreatedBy   string    `json:"created_by"`   // Para auditoría
	WorkflowID  string    `json:"workflow_id"`  // ID del workflow de Temporal
}

// DelegationStatus estado detallado de una delegación
type DelegationStatus struct {
	DelegationID string    `json:"delegation_id"`
	IsActive     bool      `json:"is_active"`
	CurrentPhase string    `json:"current_phase"` // "pending", "active", "expired", "cancelled"
	StartedAt    time.Time `json:"started_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastModified time.Time `json:"last_modified"`
	UsedAmount   float64   `json:"used_amount"`    // Monto ya usado de la delegación
}

// DelegatedPermissionResult resultado de verificación de permisos delegados
type DelegatedPermissionResult struct {
	IsAllowed       bool      `json:"is_allowed"`
	DelegationID    string    `json:"delegation_id"`
	OriginalUserID  string    `json:"original_user_id"`  // Usuario que delegó
	UsedAmount      float64   `json:"used_amount"`
	RemainingAmount float64   `json:"remaining_amount"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// DelegationEvent evento para notificaciones de delegación
type DelegationEvent struct {
	EventType    string    `json:"event_type"`    // "created", "activated", "expired", "cancelled", "modified"
	DelegationID string    `json:"delegation_id"`
	FromUserID   string    `json:"from_user_id"`
	ToUserID     string    `json:"to_user_id"`
	Amount       float64   `json:"amount"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Mock users para desarrollo
var MockUsers = map[string]User{
	"empleado@empresa.com": {
		ID:          "empleado@empresa.com",
		Name:        "Juan Empleado",
		Role:        RoleEmployee,
		Department:  "IT",
		Office:      "madrid",
		ManagerID:   "manager@empresa.com",
		MaxApproval: 0,
		CreatedAt:   time.Now(),
	},
	"manager@empresa.com": {
		ID:          "manager@empresa.com",
		Name:        "Ana Manager",
		Role:        RoleManager,
		Department:  "IT",
		Office:      "madrid",
		ManagerID:   "ceo@empresa.com",
		MaxApproval: 2000,
		CreatedAt:   time.Now(),
	},
	"ceo@empresa.com": {
		ID:          "ceo@empresa.com",
		Name:        "Carlos CEO",
		Role:        RoleCEO,
		Department:  "Executive",
		Office:      "madrid",
		ManagerID:   "",
		MaxApproval: 999999,
		CreatedAt:   time.Now(),
	},
	"admin@empresa.com": {
		ID:          "admin@empresa.com",
		Name:        "Sofia Admin",
		Role:        RoleAdmin,
		Department:  "IT",
		Office:      "madrid",
		ManagerID:   "",
		MaxApproval: 999999,
		CreatedAt:   time.Now(),
	},
}

// GetUser obtiene un usuario por ID (mock implementation)
func GetUser(userID string) (*User, bool) {
	user, exists := MockUsers[userID]
	if exists {
		return &user, true
	}
	return nil, false
}

// getSubordinates obtiene la lista de subordinados (mock)
func getSubordinates(userID string) []string {
	subordinates := map[string][]string{
		"manager@empresa.com": {"empleado@empresa.com", "dev@empresa.com"},
		"ceo@empresa.com":     {"manager@empresa.com", "manager2@empresa.com"},
	}
	
	if subs, exists := subordinates[userID]; exists {
		return subs
	}
	return []string{}
}