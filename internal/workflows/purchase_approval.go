package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-workflow/internal/activities"
	"temporal-workflow/internal/models"
)

// PurchaseApprovalWorkflow orquesta todo el proceso de aprobación de compras
func PurchaseApprovalWorkflow(ctx workflow.Context, request models.PurchaseRequest) (*models.PurchaseRequest, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Purchase Approval Workflow", "request_id", request.ID)

	// Configurar opciones para activities
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 10,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Configurar query handlers para consultar el estado
	err := workflow.SetQueryHandler(ctx, "getStatus", func() (models.PurchaseRequest, error) {
		return request, nil
	})
	if err != nil {
		logger.Error("Failed to set query handler", "error", err)
		return nil, err
	}

	// Configurar signal handlers para aprobaciones
	approvalChannel := workflow.GetSignalChannel(ctx, "approval")
	var approvalResponse models.ApprovalResponse
	
	modificationChannel := workflow.GetSignalChannel(ctx, "cart_modification")
	var cartModification models.CartModification

	// Paso 1: Validar productos de Amazon
	logger.Info("Step 1: Validating Amazon products")
	var validationResult models.PurchaseValidationResult
	err = workflow.ExecuteActivity(ctx, activities.ValidateAmazonProducts, request.ProductURLs).Get(ctx, &validationResult)
	if err != nil {
		logger.Error("Failed to validate Amazon products", "error", err)
		request.Status = models.StatusFailed
		return &request, err
	}

	// Actualizar carrito con resultados de validación
	request.Cart = models.Cart{
		Items:       append(validationResult.ValidItems, validationResult.InvalidItems...),
		TotalAmount: validationResult.TotalAmount,
		Currency:    "EUR",
	}

	// Si no hay productos válidos, rechazar automáticamente
	if len(validationResult.ValidItems) == 0 {
		logger.Info("No valid products found, rejecting request")
		request.Status = models.StatusRejected
		
		// Notificar al empleado
		_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
			request.EmployeeID, "Su solicitud ha sido rechazada automáticamente: no hay productos válidos").Get(ctx, nil)
		
		return &request, nil
	}

	// Si hay warnings, notificar pero continuar
	if len(validationResult.Warnings) > 0 {
		logger.Info("Validation warnings found", "warnings", validationResult.Warnings)
		_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
			request.EmployeeID, fmt.Sprintf("Advertencias en su solicitud: %v", validationResult.Warnings)).Get(ctx, nil)
	}

	// Paso 2: Iniciar flujo de aprobación (con soporte para delegaciones)
	logger.Info("Step 2: Starting approval flow with delegation support")
	var responsibles []string
	err = workflow.ExecuteActivity(ctx, activities.GetRequiredApproversWithDelegation, request).Get(ctx, &responsibles)
	if err != nil {
		logger.Error("Failed to get required approvers", "error", err)
		return nil, err
	}

	request.ApprovalFlow.RequiredApprovals = responsibles
	request.ApprovalFlow.ApprovalDeadline = workflow.Now(ctx).Add(time.Hour * 24 * 7) // 7 días para aprobar
	request.Status = models.StatusPending

	// Notificar a responsables
	approvalRequest := models.ApprovalRequest{
		RequestID:     request.ID,
		EmployeeID:    request.EmployeeID,
		Cart:          request.Cart,
		Justification: request.Justification,
		SentAt:        workflow.Now(ctx),
		ExpiresAt:     request.ApprovalFlow.ApprovalDeadline,
	}

	for _, responsibleID := range responsibles {
		approvalRequest.ResponsibleID = responsibleID
		_ = workflow.ExecuteActivity(ctx, activities.NotifyResponsible, approvalRequest).Get(ctx, nil)
	}

	// Paso 3: Esperar aprobaciones o timeout
	logger.Info("Step 3: Waiting for approvals")
	approvalTimeout := workflow.NewTimer(ctx, time.Until(request.ApprovalFlow.ApprovalDeadline))
	
	for len(request.ApprovalFlow.ApprovedBy) < len(request.ApprovalFlow.RequiredApprovals) {
		selector := workflow.NewSelector(ctx)
		
		// Escuchar aprobaciones (con validación de delegaciones)
		selector.AddReceive(approvalChannel, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &approvalResponse)
			logger.Info("Received approval response", "responsible", approvalResponse.ResponsibleID, "approved", approvalResponse.Approved)
			
			// Validar si la aprobación es válida (considerando delegaciones)
			var isValidApproval bool
			err := workflow.ExecuteActivity(ctx, activities.ValidateApprovalWithDelegation, 
				approvalResponse.ResponsibleID, request.Cart.TotalAmount).Get(ctx, &isValidApproval)
			if err != nil {
				logger.Error("Failed to validate approval with delegation", "error", err)
				return
			}
			
			if !isValidApproval {
				logger.Warn("Invalid approval detected, ignoring", "responsible", approvalResponse.ResponsibleID)
				return
			}
			
			if approvalResponse.Approved {
				// Agregar a lista de aprobados
				request.ApprovalFlow.ApprovedBy = append(request.ApprovalFlow.ApprovedBy, approvalResponse.ResponsibleID)
				
				// Registrar uso de delegación si aplica
				_ = workflow.ExecuteActivity(ctx, activities.RecordDelegationUsage, 
					approvalResponse.ResponsibleID, request.Cart.TotalAmount).Get(ctx, nil)
				
				// Si el responsable modificó el carrito, aplicar cambios
				if approvalResponse.ModifiedCart != nil {
					modification := models.CartModification{
						ModifiedBy: approvalResponse.ResponsibleID,
						ModifiedAt: workflow.Now(ctx),
						Changes:    "Cart modified by approver", // En un caso real, aquí iría el diff
						Reason:     approvalResponse.Reason,
					}
					request.ApprovalFlow.Modifications = append(request.ApprovalFlow.Modifications, modification)
					request.Cart = *approvalResponse.ModifiedCart
					
					// Notificar al empleado sobre las modificaciones
					_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
						request.EmployeeID, fmt.Sprintf("Su carrito ha sido modificado por %s: %s", approvalResponse.ResponsibleID, approvalResponse.Reason)).Get(ctx, nil)
				}
			} else {
				// Rechazo - terminar flujo
				request.Status = models.StatusRejected
				request.ApprovalFlow.RejectedBy = approvalResponse.ResponsibleID
				request.ApprovalFlow.RejectedReason = approvalResponse.Reason
				
				// Notificar al empleado
				_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
					request.EmployeeID, fmt.Sprintf("Su solicitud ha sido rechazada por %s: %s", approvalResponse.ResponsibleID, approvalResponse.Reason)).Get(ctx, nil)
			}
		})
		
		// Escuchar modificaciones adicionales del carrito
		selector.AddReceive(modificationChannel, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &cartModification)
			logger.Info("Received cart modification", "modified_by", cartModification.ModifiedBy)
			request.ApprovalFlow.Modifications = append(request.ApprovalFlow.Modifications, cartModification)
		})
		
		// Timeout de aprobación
		selector.AddFuture(approvalTimeout, func(f workflow.Future) {
			logger.Info("Approval timeout reached")
			request.Status = models.StatusRejected
			request.ApprovalFlow.RejectedReason = "Timeout: no approval received within deadline"
			
			// Notificar timeout
			_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
				request.EmployeeID, "Su solicitud ha sido rechazada automáticamente por timeout").Get(ctx, nil)
		})
		
		selector.Select(ctx)
		
		// Si se rechazó o hubo timeout, salir
		if request.Status == models.StatusRejected {
			return &request, nil
		}
	}

	// Paso 4: Todas las aprobaciones obtenidas, proceder con la compra
	logger.Info("Step 4: All approvals received, proceeding with purchase")
	request.Status = models.StatusApproved

	// Crear orden de compra
	purchaseOrder := models.PurchaseOrder{
		RequestID:      request.ID,
		Cart:           request.Cart,
		DeliveryOffice: request.DeliveryOffice,
		CreatedAt:      workflow.Now(ctx),
		Status:         models.StatusPending,
	}

	// Ejecutar compra en Amazon
	err = workflow.ExecuteActivity(ctx, activities.ExecuteAmazonPurchase, purchaseOrder).Get(ctx, &purchaseOrder)
	if err != nil {
		logger.Error("Failed to execute Amazon purchase", "error", err)
		request.Status = models.StatusFailed
		
		// Notificar fallo
		_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
			request.EmployeeID, fmt.Sprintf("Error al procesar su compra en Amazon: %v", err)).Get(ctx, nil)
		
		return &request, err
	}

	// Compra exitosa
	request.Status = models.StatusCompleted
	logger.Info("Purchase completed successfully", "amazon_order_id", purchaseOrder.AmazonOrderID)

	// Notificar éxito
	_ = workflow.ExecuteActivity(ctx, activities.NotifyEmployee, 
		request.EmployeeID, fmt.Sprintf("Su compra ha sido procesada exitosamente. ID de orden Amazon: %s", purchaseOrder.AmazonOrderID)).Get(ctx, nil)

	// Notificar a responsables que aprobaron
	for _, approverID := range request.ApprovalFlow.ApprovedBy {
		_ = workflow.ExecuteActivity(ctx, activities.NotifyResponsible, models.ApprovalRequest{
			RequestID:     request.ID,
			ResponsibleID: approverID,
		}).Get(ctx, nil)
	}

	return &request, nil
}

// Funciones auxiliares para queries
func GetPurchaseStatus(ctx workflow.Context) (models.PurchaseRequest, error) {
	// Esta función será llamada por el query handler
	var request models.PurchaseRequest
	// En un caso real, obtendríamos el estado actual del workflow
	return request, nil
}