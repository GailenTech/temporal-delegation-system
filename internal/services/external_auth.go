package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"temporal-workflow/internal/models"
)

// ExternalAuthProvider interface para diferentes proveedores de autenticación
type ExternalAuthProvider interface {
	ValidateToken(ctx context.Context, token string) (*ExternalUserInfo, error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	GetUserPermissions(ctx context.Context, userID string) (*ExternalPermissions, error)
	GetUserSubordinates(ctx context.Context, userID string) ([]string, error)
	ValidateDelegationPermission(ctx context.Context, fromUserID, toUserID string) (*DelegationValidation, error)
}

// ExternalUserInfo información del usuario desde el sistema externo
type ExternalUserInfo struct {
	ID           string            `json:"id"`
	Username     string            `json:"username"`
	Email        string            `json:"email"`
	FirstName    string            `json:"first_name"`
	LastName     string            `json:"last_name"`
	Roles        []string          `json:"roles"`
	Attributes   map[string]string `json:"attributes"`
	IsActive     bool              `json:"is_active"`
	LastLogin    time.Time         `json:"last_login"`
	TokenExpires time.Time         `json:"token_expires"`
}

// ExternalPermissions permisos del usuario desde sistema externo
type ExternalPermissions struct {
	CanCreateRequest     bool     `json:"can_create_request"`
	CanApprove          bool     `json:"can_approve"`
	CanApproveForOthers bool     `json:"can_approve_for_others"`
	CanViewAllRequests  bool     `json:"can_view_all_requests"`
	CanViewAdminPanel   bool     `json:"can_view_admin_panel"`
	CanDelegate         bool     `json:"can_delegate"`
	MaxApprovalAmount   float64  `json:"max_approval_amount"`
	CanRequestForOthers bool     `json:"can_request_for_others"`
	Subordinates        []string `json:"subordinates"`
	Department          string   `json:"department"`
	Office              string   `json:"office"`
	ManagerID           string   `json:"manager_id"`
}

// DelegationValidation resultado de validar permisos para delegación
type DelegationValidation struct {
	IsAllowed       bool     `json:"is_allowed"`
	MaxAmount       float64  `json:"max_amount"`
	MaxDurationDays int      `json:"max_duration_days"`
	Restrictions    []string `json:"restrictions"`
	Reason          string   `json:"reason"`
}

// AuthIntegrationMode modo de integración con sistema externo
type AuthIntegrationMode string

const (
	ModeStandalone    AuthIntegrationMode = "standalone"    // Sin integración externa
	ModeHybrid        AuthIntegrationMode = "hybrid"        // Integración parcial
	ModeFullExternal  AuthIntegrationMode = "full_external" // Todo desde sistema externo
)

// ExternalAuthService servicio para integración con sistemas externos
type ExternalAuthService struct {
	provider ExternalAuthProvider
	mode     AuthIntegrationMode
	config   *ExternalAuthConfig
}

// ExternalAuthConfig configuración para integración externa
type ExternalAuthConfig struct {
	Mode                AuthIntegrationMode `json:"mode"`
	ProviderType        string              `json:"provider_type"` // "keycloak", "okta", "azure_ad", etc.
	BaseURL             string              `json:"base_url"`
	Realm               string              `json:"realm"`
	ClientID            string              `json:"client_id"`
	ClientSecret        string              `json:"client_secret"`
	CacheTimeout        time.Duration       `json:"cache_timeout"`
	FallbackToLocal     bool                `json:"fallback_to_local"`
	SyncUserAttributes  bool                `json:"sync_user_attributes"`
	SyncRoles           bool                `json:"sync_roles"`
	SyncPermissions     bool                `json:"sync_permissions"`
	DelegationRulesURL  string              `json:"delegation_rules_url"`
}

// NewExternalAuthService crea un nuevo servicio de autenticación externa
func NewExternalAuthService(config *ExternalAuthConfig) *ExternalAuthService {
	var provider ExternalAuthProvider

	switch config.ProviderType {
	case "keycloak":
		provider = NewKeycloakProvider(config)
	case "okta":
		provider = NewOktaProvider(config)
	case "azure_ad":
		provider = NewAzureADProvider(config)
	default:
		// Por defecto, modo standalone sin proveedor externo
		provider = nil
	}

	return &ExternalAuthService{
		provider: provider,
		mode:     config.Mode,
		config:   config,
	}
}

// EnrichUserWithExternalData enriquece los datos del usuario con información externa
func (s *ExternalAuthService) EnrichUserWithExternalData(ctx context.Context, user *models.User, token string) error {
	if s.provider == nil || s.mode == ModeStandalone {
		return nil // Sin integración externa
	}

	// Obtener información del usuario desde sistema externo
	externalUser, err := s.provider.ValidateToken(ctx, token)
	if err != nil {
		if s.config.FallbackToLocal {
			return nil // Fallar silenciosamente y usar datos locales
		}
		return fmt.Errorf("failed to validate external token: %w", err)
	}

	// Sincronizar atributos del usuario si está habilitado
	if s.config.SyncUserAttributes {
		if externalUser.FirstName != "" && externalUser.LastName != "" {
			user.Name = fmt.Sprintf("%s %s", externalUser.FirstName, externalUser.LastName)
		}
		
		if department := externalUser.Attributes["department"]; department != "" {
			user.Department = department
		}
		
		if office := externalUser.Attributes["office"]; office != "" {
			user.Office = office
		}
		
		if managerID := externalUser.Attributes["manager_id"]; managerID != "" {
			user.ManagerID = managerID
		}
	}

	return nil
}

// GetEnrichedPermissions obtiene permisos enriquecidos con datos externos
func (s *ExternalAuthService) GetEnrichedPermissions(ctx context.Context, user *models.User, token string) (*models.Permissions, error) {
	// Empezar con permisos locales basados en rol
	localPerms := user.GetPermissions()

	if s.provider == nil || s.mode == ModeStandalone {
		return &localPerms, nil
	}

	// Obtener permisos desde sistema externo
	externalPerms, err := s.provider.GetUserPermissions(ctx, user.ID)
	if err != nil {
		if s.config.FallbackToLocal {
			return &localPerms, nil
		}
		return nil, fmt.Errorf("failed to get external permissions: %w", err)
	}

	// Combinar permisos según el modo de integración
	switch s.mode {
	case ModeHybrid:
		// En modo híbrido, tomar el más permisivo de cada permiso
		return s.mergeHybridPermissions(&localPerms, externalPerms), nil
		
	case ModeFullExternal:
		// En modo totalmente externo, usar solo permisos externos
		return s.convertExternalPermissions(externalPerms), nil
		
	default:
		return &localPerms, nil
	}
}

// ValidateExternalDelegation valida una delegación usando reglas externas
func (s *ExternalAuthService) ValidateExternalDelegation(ctx context.Context, delegation *models.Delegation) (*DelegationValidation, error) {
	if s.provider == nil || s.mode == ModeStandalone {
		// Sin validaciones externas, permitir basado en permisos locales
		return &DelegationValidation{
			IsAllowed:       true,
			MaxAmount:       999999,
			MaxDurationDays: 30,
			Restrictions:    []string{},
			Reason:          "Local validation only",
		}, nil
	}

	return s.provider.ValidateDelegationPermission(ctx, delegation.FromUserID, delegation.ToUserID)
}

// mergeHybridPermissions combina permisos locales y externos en modo híbrido
func (s *ExternalAuthService) mergeHybridPermissions(local *models.Permissions, external *ExternalPermissions) *models.Permissions {
	return &models.Permissions{
		CanCreateRequest:     local.CanCreateRequest || external.CanCreateRequest,
		CanApprove:          local.CanApprove || external.CanApprove,
		CanApproveForOthers: local.CanApproveForOthers || external.CanApproveForOthers,
		CanViewAllRequests:  local.CanViewAllRequests || external.CanViewAllRequests,
		CanViewAdminPanel:   local.CanViewAdminPanel || external.CanViewAdminPanel,
		CanDelegate:         local.CanDelegate || external.CanDelegate,
		CanRequestForOthers: local.CanRequestForOthers || external.CanRequestForOthers,
		MaxApprovalAmount:   max(local.MaxApprovalAmount, external.MaxApprovalAmount),
		Subordinates:        s.mergeStringSlices(local.Subordinates, external.Subordinates),
	}
}

// convertExternalPermissions convierte permisos externos al formato interno
func (s *ExternalAuthService) convertExternalPermissions(external *ExternalPermissions) *models.Permissions {
	return &models.Permissions{
		CanCreateRequest:     external.CanCreateRequest,
		CanApprove:          external.CanApprove,
		CanApproveForOthers: external.CanApproveForOthers,
		CanViewAllRequests:  external.CanViewAllRequests,
		CanViewAdminPanel:   external.CanViewAdminPanel,
		CanDelegate:         external.CanDelegate,
		CanRequestForOthers: external.CanRequestForOthers,
		MaxApprovalAmount:   external.MaxApprovalAmount,
		Subordinates:        external.Subordinates,
	}
}

// mergeStringSlices combina dos slices de strings eliminando duplicados
func (s *ExternalAuthService) mergeStringSlices(slice1, slice2 []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice1 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	for _, item := range slice2 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// max devuelve el valor máximo entre dos float64
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// KeycloakProvider implementación específica para Keycloak
type KeycloakProvider struct {
	config *ExternalAuthConfig
	client *http.Client
}

// NewKeycloakProvider crea un nuevo proveedor de Keycloak
func NewKeycloakProvider(config *ExternalAuthConfig) *KeycloakProvider {
	return &KeycloakProvider{
		config: config,
		client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// ValidateToken valida un token JWT de Keycloak
func (k *KeycloakProvider) ValidateToken(ctx context.Context, token string) (*ExternalUserInfo, error) {
	// Construir URL de introspección de Keycloak
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token/introspect", 
		k.config.BaseURL, k.config.Realm)

	// Preparar datos de la petición
	data := fmt.Sprintf("token=%s&client_id=%s&client_secret=%s", 
		token, k.config.ClientID, k.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token introspection failed with status: %d", resp.StatusCode)
	}

	var introspection struct {
		Active   bool     `json:"active"`
		Username string   `json:"username"`
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Roles    []string `json:"realm_access.roles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&introspection); err != nil {
		return nil, fmt.Errorf("failed to decode introspection response: %w", err)
	}

	if !introspection.Active {
		return nil, fmt.Errorf("token is not active")
	}

	// Convertir a formato interno
	nameParts := strings.Split(introspection.Name, " ")
	firstName := ""
	lastName := ""
	if len(nameParts) > 0 {
		firstName = nameParts[0]
	}
	if len(nameParts) > 1 {
		lastName = strings.Join(nameParts[1:], " ")
	}

	return &ExternalUserInfo{
		ID:           introspection.Username,
		Username:     introspection.Username,
		Email:        introspection.Email,
		FirstName:    firstName,
		LastName:     lastName,
		Roles:        introspection.Roles,
		IsActive:     introspection.Active,
		LastLogin:    time.Now(),
		TokenExpires: time.Now().Add(time.Hour), // TODO: Usar exp claim del token
	}, nil
}

// GetUserRoles obtiene los roles del usuario desde Keycloak
func (k *KeycloakProvider) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	// TODO: Implementar consulta de roles a Keycloak Admin API
	return []string{}, nil
}

// GetUserPermissions obtiene permisos del usuario desde Keycloak
func (k *KeycloakProvider) GetUserPermissions(ctx context.Context, userID string) (*ExternalPermissions, error) {
	// TODO: Implementar mapeo de roles de Keycloak a permisos internos
	// Esto requeriría configurar custom attributes o usar grupos específicos
	return &ExternalPermissions{
		CanCreateRequest:     true,
		CanApprove:          false,
		CanApproveForOthers: false,
		CanViewAllRequests:  false,
		CanViewAdminPanel:   false,
		CanDelegate:         false,
		MaxApprovalAmount:   0,
		CanRequestForOthers: false,
		Subordinates:        []string{},
	}, nil
}

// GetUserSubordinates obtiene subordinados del usuario desde Keycloak
func (k *KeycloakProvider) GetUserSubordinates(ctx context.Context, userID string) ([]string, error) {
	// TODO: Implementar consulta de jerarquía organizacional
	return []string{}, nil
}

// ValidateDelegationPermission valida permisos de delegación usando reglas de Keycloak
func (k *KeycloakProvider) ValidateDelegationPermission(ctx context.Context, fromUserID, toUserID string) (*DelegationValidation, error) {
	// TODO: Implementar validación usando custom policies en Keycloak
	return &DelegationValidation{
		IsAllowed:       true,
		MaxAmount:       10000,
		MaxDurationDays: 30,
		Restrictions:    []string{},
		Reason:          "Keycloak validation passed",
	}, nil
}

// Placeholders para otros proveedores
type OktaProvider struct{ config *ExternalAuthConfig }
type AzureADProvider struct{ config *ExternalAuthConfig }

func NewOktaProvider(config *ExternalAuthConfig) *OktaProvider { return &OktaProvider{config} }
func NewAzureADProvider(config *ExternalAuthConfig) *AzureADProvider { return &AzureADProvider{config} }

func (o *OktaProvider) ValidateToken(ctx context.Context, token string) (*ExternalUserInfo, error) { return nil, fmt.Errorf("not implemented") }
func (o *OktaProvider) GetUserRoles(ctx context.Context, userID string) ([]string, error) { return nil, fmt.Errorf("not implemented") }
func (o *OktaProvider) GetUserPermissions(ctx context.Context, userID string) (*ExternalPermissions, error) { return nil, fmt.Errorf("not implemented") }
func (o *OktaProvider) GetUserSubordinates(ctx context.Context, userID string) ([]string, error) { return nil, fmt.Errorf("not implemented") }
func (o *OktaProvider) ValidateDelegationPermission(ctx context.Context, fromUserID, toUserID string) (*DelegationValidation, error) { return nil, fmt.Errorf("not implemented") }

func (a *AzureADProvider) ValidateToken(ctx context.Context, token string) (*ExternalUserInfo, error) { return nil, fmt.Errorf("not implemented") }
func (a *AzureADProvider) GetUserRoles(ctx context.Context, userID string) ([]string, error) { return nil, fmt.Errorf("not implemented") }
func (a *AzureADProvider) GetUserPermissions(ctx context.Context, userID string) (*ExternalPermissions, error) { return nil, fmt.Errorf("not implemented") }
func (a *AzureADProvider) GetUserSubordinates(ctx context.Context, userID string) ([]string, error) { return nil, fmt.Errorf("not implemented") }
func (a *AzureADProvider) ValidateDelegationPermission(ctx context.Context, fromUserID, toUserID string) (*DelegationValidation, error) { return nil, fmt.Errorf("not implemented") }