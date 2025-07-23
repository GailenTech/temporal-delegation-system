package activities

import (
	"context"
	"fmt"
	"log"
	"time"

	"temporal-workflow/internal/models"
)

// ApprovalActivities contiene actividades relacionadas con el flujo de aprobación
type ApprovalActivities struct {
	// En el futuro aquí irían clientes para email, Slack, etc.
}

// GetRequiredApproversWithDelegation determina responsables considerando delegaciones activas
func (a *ApprovalActivities) GetRequiredApproversWithDelegation(ctx context.Context, request models.PurchaseRequest) ([]string, error) {
	log.Printf("Getting required approvers with delegation support for request %s, amount: %.2f", request.ID, request.Cart.TotalAmount)
	
	// Primero obtener los aprobadores base
	baseApprovers, err := a.GetRequiredApprovers(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// Para cada aprobador base, verificar si hay delegaciones activas
	effectiveApprovers := []string{}
	
	for _, approverID := range baseApprovers {
		// Verificar si el aprobador tiene delegaciones activas que cubran este monto
		delegatedTo := a.findActiveDelegation(ctx, approverID, request.Cart.TotalAmount)
		
		if delegatedTo != "" {
			log.Printf("Found active delegation: %s -> %s for amount %.2f", approverID, delegatedTo, request.Cart.TotalAmount)
			effectiveApprovers = append(effectiveApprovers, delegatedTo)
		} else {
			// No hay delegación, usar el aprobador original
			effectiveApprovers = append(effectiveApprovers, approverID)
		}
	}
	
	log.Printf("Effective approvers with delegation: %v", effectiveApprovers)
	return effectiveApprovers, nil
}

// findActiveDelegation busca una delegación activa que cubra el monto especificado
func (a *ApprovalActivities) findActiveDelegation(ctx context.Context, fromUserID string, amount float64) string {
	// TODO: Implementar búsqueda real en base de datos
	// Por ahora, lógica mock
	
	// Mock: manager@company.com ha delegado a supervisor@company.com hasta 1500€
	if fromUserID == "manager@company.com" && amount <= 1500 {
		return "supervisor@company.com"
	}
	
	// Mock: ceo@company.com ha delegado a manager@company.com hasta 3000€
	if fromUserID == "ceo@company.com" && amount <= 3000 {
		return "manager@company.com"
	}
	
	return "" // No hay delegación activa
}

// ValidateApprovalWithDelegation valida si una aprobación es válida considerando delegaciones
func (a *ApprovalActivities) ValidateApprovalWithDelegation(ctx context.Context, approverID string, amount float64) (bool, error) {
	log.Printf("Validating approval with delegation for approver %s, amount: %.2f", approverID, amount)
	
	// TODO: Implementar validación real con base de datos de usuarios y delegaciones
	// Por ahora, lógica mock simple
	
	// Usuarios con permisos directos (mock)
	directPermissions := map[string]float64{
		"supervisor@company.com": 200,   // Supervisor puede aprobar hasta 200€
		"manager@company.com":    1000,  // Manager puede aprobar hasta 1000€
		"ceo@company.com":        10000, // CEO puede aprobar hasta 10000€
	}
	
	// Verificar permisos directos
	if maxAmount, exists := directPermissions[approverID]; exists && amount <= maxAmount {
		log.Printf("Approval valid with direct permissions: %s can approve %.2f (limit: %.2f)", approverID, amount, maxAmount)
		return true, nil
	}
	
	// TODO: Verificar permisos delegados
	// Por ahora, permitir si el usuario existe en la tabla mock
	if _, exists := directPermissions[approverID]; exists {
		log.Printf("Approval valid for existing user: %s", approverID)
		return true, nil
	}
	
	log.Printf("Approval NOT valid: %s cannot approve %.2f", approverID, amount)
	return false, nil
}

// RecordDelegationUsage registra el uso de una delegación
func (a *ApprovalActivities) RecordDelegationUsage(ctx context.Context, approverID string, amount float64) error {
	log.Printf("Recording delegation usage: approver=%s, amount=%.2f", approverID, amount)
	
	// TODO: Implementar registro real en base de datos
	// 1. Encontrar la delegación activa utilizada
	// 2. Actualizar el monto usado
	// 3. Verificar si se exceden límites
	// 4. Registrar en log de auditoría
	// 5. Enviar notificaciones si es necesario
	
	// Mock implementation - solo logging por ahora
	log.Printf("DELEGATION USAGE RECORDED: Approver=%s used delegation for amount=%.2f at %s", 
		approverID, amount, time.Now().Format("2006-01-02 15:04:05"))
	
	return nil
}

// GetRequiredApprovers determina qué responsables deben aprobar una solicitud
func (a *ApprovalActivities) GetRequiredApprovers(ctx context.Context, request models.PurchaseRequest) ([]string, error) {
	// STUB: En el futuro esto consultaría una base de datos o servicio de configuración
	
	var approvers []string
	
	// Lógica de ejemplo basada en el monto
	if request.Cart.TotalAmount > 500 {
		// Para compras grandes, requiere CEO
		approvers = append(approvers, "ceo@company.com")
	}
	
	if request.Cart.TotalAmount > 100 {
		// Para compras medianas, requiere manager
		approvers = append(approvers, "manager@company.com")
	}
	
	// Siempre requiere supervisor directo
	approvers = append(approvers, "supervisor@company.com")
	
	// Remover duplicados
	seen := make(map[string]bool)
	uniqueApprovers := []string{}
	for _, approver := range approvers {
		if !seen[approver] {
			uniqueApprovers = append(uniqueApprovers, approver)
			seen[approver] = true
		}
	}
	
	return uniqueApprovers, nil
}

// NotifyEmployee envía notificación a un empleado
func (a *ApprovalActivities) NotifyEmployee(ctx context.Context, employeeID, message string) error {
	// STUB: En el futuro esto enviaría email, Slack, push notification, etc.
	log.Printf("NOTIFICATION TO EMPLOYEE %s: %s", employeeID, message)
	
	// Simular envío de notificación
	// Aquí iría la lógica real de notificación:
	// - Email via SendGrid/SES
	// - Slack via webhook
	// - Push notification
	// - SMS
	
	return nil
}

// NotifyResponsible envía notificación a un responsable sobre una solicitud pendiente
func (a *ApprovalActivities) NotifyResponsible(ctx context.Context, approvalRequest models.ApprovalRequest) error {
	// STUB: En el futuro esto enviaría una notificación rica con detalles de la solicitud
	
	message := fmt.Sprintf(`
Nueva solicitud de aprobación pendiente:
- ID: %s
- Empleado: %s
- Monto total: %.2f EUR
- Productos: %d items
- Justificación: %s
- Vence: %s

Para aprobar/rechazar, visite: http://localhost:8081/approval/%s
`, 
		approvalRequest.RequestID,
		approvalRequest.EmployeeID,
		approvalRequest.Cart.TotalAmount,
		len(approvalRequest.Cart.Items),
		approvalRequest.Justification,
		approvalRequest.ExpiresAt.Format("2006-01-02 15:04"),
		approvalRequest.RequestID,
	)
	
	log.Printf("NOTIFICATION TO RESPONSIBLE %s: %s", approvalRequest.ResponsibleID, message)
	
	// En el futuro aquí se enviaría:
	// - Email con template HTML
	// - Slack con botones interactivos
	// - Dashboard notification
	
	return nil
}

// CheckDuplicatePurchases verifica si hay compras duplicadas recientes
func (a *ApprovalActivities) CheckDuplicatePurchases(ctx context.Context, employeeID string, productIDs []string) ([]string, error) {
	// STUB: En el futuro esto consultaría una base de datos de compras históricas
	
	// Simular algunas compras duplicadas
	recentPurchases := map[string]bool{
		"B08N5WRWNW": true, // Echo Dot comprado recientemente
	}
	
	var duplicates []string
	for _, productID := range productIDs {
		if recentPurchases[productID] {
			duplicates = append(duplicates, productID)
		}
	}
	
	return duplicates, nil
}

// LogPurchaseDecision registra la decisión de aprobación/rechazo para auditoría
func (a *ApprovalActivities) LogPurchaseDecision(ctx context.Context, requestID string, decision models.ApprovalResponse) error {
	// STUB: En el futuro esto escribiría a un log de auditoría persistente
	
	log.Printf("AUDIT LOG - Purchase Decision: RequestID=%s, ResponsibleID=%s, Approved=%v, Reason=%s", 
		requestID, decision.ResponsibleID, decision.Approved, decision.Reason)
	
	// Aquí iría:
	// - Inserción en base de datos de auditoría
	// - Log a sistema de monitoreo
	// - Notificación a compliance
	
	return nil
}

// Funciones standalone para el worker

func GetRequiredApprovers(ctx context.Context, request models.PurchaseRequest) ([]string, error) {
	activities := &ApprovalActivities{}
	return activities.GetRequiredApprovers(ctx, request)
}

func GetRequiredApproversWithDelegation(ctx context.Context, request models.PurchaseRequest) ([]string, error) {
	activities := &ApprovalActivities{}
	return activities.GetRequiredApproversWithDelegation(ctx, request)
}

func ValidateApprovalWithDelegation(ctx context.Context, approverID string, amount float64) (bool, error) {
	activities := &ApprovalActivities{}
	return activities.ValidateApprovalWithDelegation(ctx, approverID, amount)
}

func RecordDelegationUsage(ctx context.Context, approverID string, amount float64) error {
	activities := &ApprovalActivities{}
	return activities.RecordDelegationUsage(ctx, approverID, amount)
}

func NotifyEmployee(ctx context.Context, employeeID, message string) error {
	activities := &ApprovalActivities{}
	return activities.NotifyEmployee(ctx, employeeID, message)
}

func NotifyResponsible(ctx context.Context, approvalRequest models.ApprovalRequest) error {
	activities := &ApprovalActivities{}
	return activities.NotifyResponsible(ctx, approvalRequest)
}

func CheckDuplicatePurchases(ctx context.Context, employeeID string, productIDs []string) ([]string, error) {
	activities := &ApprovalActivities{}
	return activities.CheckDuplicatePurchases(ctx, employeeID, productIDs)
}

func LogPurchaseDecision(ctx context.Context, requestID string, decision models.ApprovalResponse) error {
	activities := &ApprovalActivities{}
	return activities.LogPurchaseDecision(ctx, requestID, decision)
}