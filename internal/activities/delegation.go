package activities

import (
	"context"
	"fmt"
	"time"

	"temporal-workflow/internal/models"
)

// DelegationValidationResult resultado de la validación de delegación
type DelegationValidationResult struct {
	IsValid          bool     `json:"is_valid"`
	ValidationErrors []string `json:"validation_errors"`
}

// ModifyDelegationSignal señal para modificar una delegación activa
type ModifyDelegationSignal struct {
	Action       string    `json:"action"`        // "extend", "modify_amount"
	NewEndDate   time.Time `json:"new_end_date"`  // Para extend
	NewMaxAmount float64   `json:"new_max_amount"` // Para modify_amount
	ModifiedBy   string    `json:"modified_by"`   // Usuario que hace la modificación
	Reason       string    `json:"reason"`        // Razón de la modificación
}

// CancelDelegationSignal señal para cancelar una delegación
type CancelDelegationSignal struct {
	CancelledBy string `json:"cancelled_by"`
	Reason      string `json:"reason"`
}

// DelegationActivities contiene actividades relacionadas con delegaciones
type DelegationActivities struct {
	// En el futuro aquí irían clientes para DB, servicios externos, etc.
}

// ValidateDelegation valida si una delegación es válida antes de activarla
func (a *DelegationActivities) ValidateDelegation(ctx context.Context, delegation models.Delegation) (*DelegationValidationResult, error) {
	fmt.Printf("Validating delegation %s\n", delegation.ID)
	
	result := &DelegationValidationResult{
		IsValid:          true,
		ValidationErrors: []string{},
	}

	// 1. Validar que el usuario origen existe y tiene permisos de delegación
	fromUser, exists := models.GetUser(delegation.FromUserID)
	if !exists {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			fmt.Sprintf("Origin user %s not found", delegation.FromUserID))
		return result, nil
	}

	fromPerms := fromUser.GetPermissions()
	if !fromPerms.CanDelegate {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			fmt.Sprintf("User %s does not have delegation permissions", delegation.FromUserID))
	}

	// 2. Validar que el usuario destino existe
	_, exists = models.GetUser(delegation.ToUserID)
	if !exists {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			fmt.Sprintf("Target user %s not found", delegation.ToUserID))
		return result, nil
	}

	// 3. Validar que no se delega a sí mismo
	if delegation.FromUserID == delegation.ToUserID {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			"Cannot delegate to yourself")
	}

	// 4. Validar fechas
	now := time.Now()
	if delegation.EndDate.Before(delegation.StartDate) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			"End date must be after start date")
	}

	if delegation.EndDate.Before(now) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			"End date must be in the future")
	}

	// 5. Validar límite de monto
	if delegation.MaxAmount <= 0 {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			"Max amount must be greater than 0")
	}

	// 6. Validar que el monto delegado no excede los permisos del usuario origen
	if delegation.MaxAmount > fromPerms.MaxApprovalAmount {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, 
			fmt.Sprintf("Delegation amount %.2f exceeds user's approval limit %.2f", 
				delegation.MaxAmount, fromPerms.MaxApprovalAmount))
	}

	// 7. Verificar que no hay conflictos con delegaciones existentes
	// TODO: Implementar verificación de solapamientos en BD

	if len(result.ValidationErrors) > 0 {
		result.IsValid = false
	}

	fmt.Printf("Delegation validation completed: %s (valid: %v, errors: %d)\n", 
		delegation.ID, result.IsValid, len(result.ValidationErrors))

	return result, nil
}

// ActivateDelegation activa una delegación en el sistema
func (a *DelegationActivities) ActivateDelegation(ctx context.Context, delegationID string) error {
	fmt.Printf("Activating delegation %s\n", delegationID)
	
	// TODO: Implementar activación en base de datos
	// 1. Marcar delegación como activa
	// 2. Actualizar permisos efectivos del usuario destino
	// 3. Registrar evento de activación
	// 4. Enviar notificaciones

	// Mock implementation
	fmt.Printf("Delegation activated successfully: %s\n", delegationID)
	
	return nil
}

// DeactivateDelegation desactiva una delegación (por expiración o cancelación)
func (a *DelegationActivities) DeactivateDelegation(ctx context.Context, delegationID string) error {
	fmt.Printf("Deactivating delegation %s\n", delegationID)
	
	// TODO: Implementar desactivación en base de datos
	// 1. Marcar delegación como inactiva
	// 2. Restaurar permisos originales del usuario destino
	// 3. Registrar evento de desactivación
	// 4. Enviar notificaciones

	// Mock implementation
	fmt.Printf("Delegation deactivated successfully: %s\n", delegationID)
	
	return nil
}

// ExtendDelegation extiende la fecha de finalización de una delegación
func (a *DelegationActivities) ExtendDelegation(ctx context.Context, delegationID string, newEndDate time.Time) error {
	fmt.Printf("Extending delegation %s to %s\n", delegationID, newEndDate.Format("2006-01-02"))
	
	// TODO: Implementar extensión en base de datos
	// 1. Actualizar fecha de finalización
	// 2. Validar que la nueva fecha es válida
	// 3. Registrar evento de modificación
	// 4. Enviar notificaciones

	// Mock implementation
	fmt.Printf("Delegation extended successfully: %s\n", delegationID)
	
	return nil
}

// ModifyDelegationAmount modifica el límite de monto de una delegación activa
func (a *DelegationActivities) ModifyDelegationAmount(ctx context.Context, delegationID string, newMaxAmount float64) error {
	fmt.Printf("Modifying delegation %s amount to %.2f\n", delegationID, newMaxAmount)
	
	// TODO: Implementar modificación en base de datos
	// 1. Actualizar límite de monto
	// 2. Validar que el nuevo monto es válido
	// 3. Registrar evento de modificación
	// 4. Enviar notificaciones

	// Mock implementation
	fmt.Printf("Delegation amount modified successfully: %s\n", delegationID)
	
	return nil
}

// GetDelegationStatus obtiene el estado actual de una delegación
func (a *DelegationActivities) GetDelegationStatus(ctx context.Context, delegationID string) (*models.DelegationStatus, error) {
	fmt.Printf("Getting delegation status: %s\n", delegationID)
	
	// TODO: Implementar consulta en base de datos
	// Mock implementation
	status := &models.DelegationStatus{
		DelegationID: delegationID,
		IsActive:     true,
		CurrentPhase: "active",
		StartedAt:    time.Now().Add(-time.Hour * 24),
		ExpiresAt:    time.Now().Add(time.Hour * 24 * 6),
		LastModified: time.Now().Add(-time.Hour),
	}
	
	return status, nil
}

// CheckDelegatedPermissions verifica si un usuario tiene permisos delegados para una acción
func (a *DelegationActivities) CheckDelegatedPermissions(ctx context.Context, userID string, action string, amount float64) (*models.DelegatedPermissionResult, error) {
	fmt.Printf("Checking delegated permissions for %s: %s %.2f\n", userID, action, amount)
	
	// TODO: Implementar verificación completa
	// 1. Obtener delegaciones activas donde el usuario es delegatario
	// 2. Verificar si alguna delegación cubre la acción y monto solicitado
	// 3. Registrar uso de la delegación
	
	// Mock implementation - siempre permite por ahora
	result := &models.DelegatedPermissionResult{
		IsAllowed:      true,
		DelegationID:   "mock-delegation-id",
		OriginalUserID: "manager@empresa.com",
		UsedAmount:     amount,
		RemainingAmount: 1000.0,
		ExpiresAt:      time.Now().Add(time.Hour * 24 * 5),
	}
	
	return result, nil
}

// NotifyDelegationEvent envía notificaciones sobre eventos de delegación
func (a *DelegationActivities) NotifyDelegationEvent(ctx context.Context, event models.DelegationEvent) error {
	fmt.Printf("Sending delegation notification: %s for %s\n", event.EventType, event.DelegationID)
	
	// TODO: Implementar sistema de notificaciones
	// 1. Email a usuarios involucrados
	// 2. Notificaciones push si están configuradas
	// 3. Registro en sistema de auditoría
	// 4. Integración con sistemas externos (Slack, Teams, etc.)
	
	return nil
}

// Funciones standalone para el worker (igual que en approval.go)

func ValidateDelegation(ctx context.Context, delegation models.Delegation) (*DelegationValidationResult, error) {
	activities := &DelegationActivities{}
	return activities.ValidateDelegation(ctx, delegation)
}

func ActivateDelegation(ctx context.Context, delegationID string) error {
	activities := &DelegationActivities{}
	return activities.ActivateDelegation(ctx, delegationID)
}

func DeactivateDelegation(ctx context.Context, delegationID string) error {
	activities := &DelegationActivities{}
	return activities.DeactivateDelegation(ctx, delegationID)
}

func ExtendDelegation(ctx context.Context, delegationID string, newEndDate time.Time) error {
	activities := &DelegationActivities{}
	return activities.ExtendDelegation(ctx, delegationID, newEndDate)
}

func ModifyDelegationAmount(ctx context.Context, delegationID string, newMaxAmount float64) error {
	activities := &DelegationActivities{}
	return activities.ModifyDelegationAmount(ctx, delegationID, newMaxAmount)
}

func GetDelegationStatus(ctx context.Context, delegationID string) (*models.DelegationStatus, error) {
	activities := &DelegationActivities{}
	return activities.GetDelegationStatus(ctx, delegationID)
}

func CheckDelegatedPermissions(ctx context.Context, userID string, action string, amount float64) (*models.DelegatedPermissionResult, error) {
	activities := &DelegationActivities{}
	return activities.CheckDelegatedPermissions(ctx, userID, action, amount)
}

func NotifyDelegationEvent(ctx context.Context, event models.DelegationEvent) error {
	activities := &DelegationActivities{}
	return activities.NotifyDelegationEvent(ctx, event)
}