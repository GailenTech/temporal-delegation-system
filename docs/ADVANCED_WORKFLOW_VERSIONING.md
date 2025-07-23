# Estrategias Avanzadas de Versionado de Workflows en Temporal.io

## Caso de Estudio: Sistema de Revisión Automatizada

### Objetivo
Implementar un paso de revisión automatizada en el workflow de aprobación de compras que puede aprobar solicitudes directamente basándose en reglas configurables, reduciendo el tiempo de aprobación para compras de bajo riesgo.

### Arquitectura de la Solución

#### Flujo Original vs Nuevo Flujo

```
FLUJO ORIGINAL:
Solicitud → Validación Amazon → Aprobación Manual → Compra

FLUJO NUEVO:
Solicitud → Validación Amazon → Revisión Automatizada → [Aprobación Directa | Aprobación Manual] → Compra
```

#### Diseño Técnico

```go
// Estructura de la nueva activity
type AutomatedReviewInput struct {
    Request          PurchaseRequest     `json:"request"`
    ValidationResult *ActivityResult    `json:"validation_result"`
    UserContext      UserContext        `json:"user_context"`
}

type AutomatedReviewResult struct {
    AutoApproved    bool      `json:"auto_approved"`
    Confidence      float64   `json:"confidence"`
    Reason          string    `json:"reason"`
    RiskScore       float64   `json:"risk_score"`
    ReviewedBy      string    `json:"reviewed_by"`  // "system" o "human"
    ReviewTime      time.Time `json:"review_time"`
}

// Reglas de auto-aprobación (configurables)
type AutoApprovalRules struct {
    MaxAmount           float64           `json:"max_amount"`
    TrustedCategories   []string          `json:"trusted_categories"`
    DepartmentLimits    map[string]float64 `json:"department_limits"`
    UserHistory         HistoryRules      `json:"user_history"`
    VendorWhitelist     []string          `json:"vendor_whitelist"`
}
```

### Implementación con Temporal Versioning

#### 1. Workflow Principal con GetVersion()

```go
func PurchaseApprovalWorkflow(ctx workflow.Context, input PurchaseWorkflowInput) (*PurchaseResult, error) {
    logger := workflow.GetLogger(ctx)
    
    // 1. Validación Amazon (sin cambios)
    var validationResult *AmazonValidationResult
    err := workflow.ExecuteActivity(ctx, ValidateAmazonProduct, input.Request).Get(ctx, &validationResult)
    if err != nil {
        return nil, fmt.Errorf("amazon validation failed: %w", err)
    }
    
    // 2. PUNTO CRÍTICO: Versioning para nueva funcionalidad
    reviewVersion := workflow.GetVersion(ctx, "automated-review-v1", workflow.DefaultVersion, 1)
    
    var approvalNeeded bool = true
    var reviewResult *AutomatedReviewResult
    
    if reviewVersion == workflow.DefaultVersion {
        // Flujo original: siempre requiere aprobación manual
        logger.Info("Using original approval flow - all requests require manual approval")
        approvalNeeded = true
    } else {
        // Nuevo flujo: revisión automatizada primero  
        logger.Info("Using enhanced flow with automated review")
        
        // Activity de revisión automatizada
        err := workflow.ExecuteActivity(ctx, 
            AutomatedReviewActivity,
            AutomatedReviewInput{
                Request:          input.Request,
                ValidationResult: validationResult,
                UserContext: UserContext{
                    EmployeeID:  input.Request.EmployeeID,
                    Department:  input.Request.Department,
                    ManagerID:   input.Request.ManagerID,
                },
            },
            workflow.ActivityOptions{
                StartToCloseTimeout: time.Minute * 5,
                RetryPolicy: &temporal.RetryPolicy{
                    InitialInterval:    time.Second * 1,
                    BackoffCoefficient: 2.0,
                    MaximumInterval:    time.Second * 100,
                    MaximumAttempts:    3,
                },
            }).Get(ctx, &reviewResult)
            
        if err != nil {
            logger.Error("Automated review failed, falling back to manual approval", "error", err)
            approvalNeeded = true  // Fallback seguro
        } else {
            approvalNeeded = !reviewResult.AutoApproved
            logger.Info("Automated review completed", 
                "auto_approved", reviewResult.AutoApproved,
                "reason", reviewResult.Reason,
                "confidence", reviewResult.Confidence,
                "risk_score", reviewResult.RiskScore)
        }
    }
    
    // 3. Aprobación manual (solo si es necesaria)
    var approvalResult *ApprovalResult
    if approvalNeeded {
        logger.Info("Manual approval required")
        
        // Enviar notificación de aprobación
        err = workflow.ExecuteActivity(ctx, SendApprovalRequest, ApprovalRequestInput{
            Request:      input.Request,
            ReviewResult: reviewResult, // Incluir contexto de revisión automática
        }).Get(ctx, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to send approval request: %w", err)
        }
        
        // Esperar señal de aprobación o timeout
        approvalChannel := workflow.GetSignalChannel(ctx, "approval_decision")
        timeoutTimer := workflow.NewTimer(ctx, time.Hour * 24 * 7) // 7 días
        
        selector := workflow.NewSelector(ctx)
        selector.AddReceive(approvalChannel, func(c workflow.ReceiveChannel, more bool) {
            var decision ApprovalDecision
            c.Receive(ctx, &decision)
            approvalResult = &ApprovalResult{
                Approved:   decision.Approved,
                ApprovedBy: decision.ApprovedBy,
                ApprovedAt: workflow.Now(ctx),
                Comments:   decision.Comments,
            }
        })
        selector.AddFuture(timeoutTimer, func(f workflow.Future) {
            approvalResult = &ApprovalResult{
                Approved:   false,
                ApprovedBy: "system",
                ApprovedAt: workflow.Now(ctx),
                Comments:   "Approval timeout after 7 days",
            }
        })
        
        selector.Select(ctx)
    } else {
        // Auto-aprobación
        logger.Info("Request auto-approved by automated review")
        approvalResult = &ApprovalResult{
            Approved:   true,
            ApprovedBy: "automated_system",
            ApprovedAt: workflow.Now(ctx),
            Comments:   fmt.Sprintf("Auto-approved: %s (confidence: %.2f)", 
                reviewResult.Reason, reviewResult.Confidence),
        }
    }
    
    // 4. Compra (si fue aprobada)
    if !approvalResult.Approved {
        return &PurchaseResult{
            Status:         "rejected",
            ApprovalResult: approvalResult,
            ReviewResult:   reviewResult,
        }, nil
    }
    
    // Ejecutar compra
    var purchaseResult *AmazonPurchaseResult
    err = workflow.ExecuteActivity(ctx, ExecuteAmazonPurchase, input.Request).Get(ctx, &purchaseResult)
    if err != nil {
        return nil, fmt.Errorf("purchase execution failed: %w", err)
    }
    
    // 5. Notificaciones finales
    _ = workflow.ExecuteActivity(ctx, SendPurchaseNotification, NotificationInput{
        Request:        input.Request,
        ApprovalResult: approvalResult,
        ReviewResult:   reviewResult,
        PurchaseResult: purchaseResult,
    })
    
    return &PurchaseResult{
        Status:         "completed",
        ApprovalResult: approvalResult,
        ReviewResult:   reviewResult,
        PurchaseResult: purchaseResult,
    }, nil
}
```

#### 2. Activity de Revisión Automatizada

```go
func AutomatedReviewActivity(ctx context.Context, input AutomatedReviewInput) (*AutomatedReviewResult, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Starting automated review", "request_id", input.Request.ID)
    
    // Cargar reglas de auto-aprobación (desde configuración dinámica)
    rules, err := loadAutoApprovalRules(input.UserContext.Department)
    if err != nil {
        return nil, fmt.Errorf("failed to load approval rules: %w", err)
    }
    
    result := &AutomatedReviewResult{
        AutoApproved: false,
        Confidence:   0.0,
        ReviewedBy:   "automated_system",
        ReviewTime:   time.Now(),
    }
    
    // 1. Verificar límite de monto
    if input.Request.Amount > rules.MaxAmount {
        result.Reason = fmt.Sprintf("Amount %.2f exceeds max limit %.2f", 
            input.Request.Amount, rules.MaxAmount)
        result.RiskScore = 0.8
        return result, nil
    }
    
    // 2. Verificar categoría de producto
    category := extractProductCategory(input.ValidationResult.ProductInfo)
    if !contains(rules.TrustedCategories, category) {
        result.Reason = fmt.Sprintf("Product category '%s' not in trusted list", category)
        result.RiskScore = 0.6
        return result, nil
    }
    
    // 3. Verificar historial del usuario
    userHistory, err := getUserPurchaseHistory(input.UserContext.EmployeeID)
    if err != nil {
        logger.Warn("Could not retrieve user history", "error", err)
        result.Reason = "Unable to verify user purchase history"
        result.RiskScore = 0.7
        return result, nil
    }
    
    if !evaluateUserHistory(userHistory, rules.UserHistory) {
        result.Reason = "User purchase history doesn't meet approval criteria"
        result.RiskScore = 0.5
        return result, nil
    }
    
    // 4. Verificar límites departamentales
    deptLimit, exists := rules.DepartmentLimits[input.UserContext.Department]
    if exists && input.Request.Amount > deptLimit {
        result.Reason = fmt.Sprintf("Amount %.2f exceeds department limit %.2f", 
            input.Request.Amount, deptLimit)
        result.RiskScore = 0.4
        return result, nil
    }
    
    // 5. Verificar vendor whitelist
    vendor := extractVendor(input.ValidationResult.ProductInfo.URL)
    if len(rules.VendorWhitelist) > 0 && !contains(rules.VendorWhitelist, vendor) {
        result.Reason = fmt.Sprintf("Vendor '%s' not in approved whitelist", vendor)
        result.RiskScore = 0.3
        return result, nil
    }
    
    // 6. Todos los checks pasaron - auto-aprobar
    result.AutoApproved = true
    result.Confidence = calculateConfidence(input, rules)
    result.Reason = "All automated approval criteria met"
    result.RiskScore = 0.1
    
    logger.Info("Automated review completed", 
        "auto_approved", result.AutoApproved,
        "confidence", result.Confidence,
        "risk_score", result.RiskScore)
    
    return result, nil
}

// Funciones auxiliares
func loadAutoApprovalRules(department string) (*AutoApprovalRules, error) {
    // En producción, esto vendría de una base de datos o servicio de configuración
    rules := &AutoApprovalRules{
        MaxAmount:         500.00, // €500 máximo para auto-aprobación
        TrustedCategories: []string{"electronics", "books", "office_supplies"},
        DepartmentLimits: map[string]float64{
            "IT":        1000.00,
            "Marketing": 300.00,
            "Finance":   200.00,
        },
        VendorWhitelist: []string{"amazon.es", "amazon.com"},
        UserHistory: HistoryRules{
            MinPreviousPurchases: 3,
            MaxRejectedInLast30Days: 1,
        },
    }
    
    return rules, nil
}

func calculateConfidence(input AutomatedReviewInput, rules *AutoApprovalRules) float64 {
    confidence := 0.5 // Base confidence
    
    // Factor: monto vs límite
    amountRatio := input.Request.Amount / rules.MaxAmount
    if amountRatio < 0.5 {
        confidence += 0.3
    } else if amountRatio < 0.8 {
        confidence += 0.1
    }
    
    // Factor: categoría de producto
    category := extractProductCategory(input.ValidationResult.ProductInfo)
    if category == "office_supplies" {
        confidence += 0.2
    }
    
    // Factor: departamento
    if input.UserContext.Department == "IT" {
        confidence += 0.1
    }
    
    return math.Min(confidence, 1.0)
}
```

### Estrategias de Deployment Selectivo

#### 1. Worker Versioning con Build IDs

```bash
#!/bin/bash
# deploy-selective.sh

NEW_BUILD_ID="$1"
TARGET_DEPARTMENT="$2"
ROLLOUT_PERCENTAGE="${3:-10}"

echo "🚀 Iniciando deployment selectivo..."
echo "Build ID: $NEW_BUILD_ID"
echo "Departamento objetivo: $TARGET_DEPARTMENT"
echo "Porcentaje inicial: $ROLLOUT_PERCENTAGE%"

# 1. Desplegar nueva versión del worker
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: purchase-worker-${NEW_BUILD_ID}
  labels:
    app: purchase-worker
    version: ${NEW_BUILD_ID}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: purchase-worker
      version: ${NEW_BUILD_ID}
  template:
    metadata:
      labels:
        app: purchase-worker
        version: ${NEW_BUILD_ID}
    spec:
      containers:
      - name: worker
        image: gcr.io/temporal-demo-0723/purchase-worker:${NEW_BUILD_ID}
        env:
        - name: BUILD_ID
          value: "${NEW_BUILD_ID}"
        - name: TARGET_DEPARTMENT
          value: "${TARGET_DEPARTMENT}"
        - name: TEMPORAL_ADDRESS
          value: "temporal-server:7233"
EOF

# 2. Esperar a que el deployment esté listo
kubectl rollout status deployment/purchase-worker-${NEW_BUILD_ID}

# 3. Configurar routing selectivo en Temporal
temporal worker deployment add-new-build-id \
    --task-queue "purchase-approval-task-queue" \
    --build-id "$NEW_BUILD_ID"

# 4. Configurar ramping inicial
temporal worker deployment set-build-id-ramping \
    --task-queue "purchase-approval-task-queue" \
    --build-id "$NEW_BUILD_ID" \
    --percentage "$ROLLOUT_PERCENTAGE"

echo "✅ Deployment selectivo completado"
echo "🔍 Monitoreando métricas..."

# 5. Script de monitoreo automático
./scripts/monitor-deployment.sh "$NEW_BUILD_ID" &
```

#### 2. Feature Flags Dinámicas

```go
// Servicio de feature flags integrado con el workflow
type FeatureFlagService struct {
    client *redis.Client
    cache  map[string]interface{}
    mutex  sync.RWMutex
}

func (f *FeatureFlagService) IsAutomatedReviewEnabled(userID, department string) bool {
    // Lógica de feature flags jerárquica:
    // 1. Usuario específico
    // 2. Departamento
    // 3. Global
    
    userKey := fmt.Sprintf("feature:automated-review:user:%s", userID)
    if val, exists := f.getFlag(userKey); exists {
        return val.(bool)
    }
    
    deptKey := fmt.Sprintf("feature:automated-review:dept:%s", department)
    if val, exists := f.getFlag(deptKey); exists {
        return val.(bool)
    }
    
    globalKey := "feature:automated-review:global"
    if val, exists := f.getFlag(globalKey); exists {
        return val.(bool)
    }
    
    return false // Default: disabled
}

// Activity que verifica feature flags
func CheckFeatureFlagsActivity(ctx context.Context, input FeatureFlagInput) (*FeatureFlagResult, error) {
    flagService := GetFeatureFlagService()
    
    result := &FeatureFlagResult{
        Flags: make(map[string]interface{}),
    }
    
    // Check automated review flag
    result.Flags["automated-review-enabled"] = flagService.IsAutomatedReviewEnabled(
        input.UserID, input.Department)
    
    // Check review strictness level
    result.Flags["review-strictness"] = flagService.GetStringFlag(
        "review-strictness", input.UserID, input.Department, "standard")
    
    return result, nil
}
```

#### 3. API de Control Dinámico

```go
// REST API para controlar el rollout en tiempo real
func (h *DeploymentHandler) SetRolloutPercentage(w http.ResponseWriter, r *http.Request) {
    var req struct {
        BuildID    string  `json:"build_id"`
        Percentage float64 `json:"percentage"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Validación
    if req.Percentage < 0 || req.Percentage > 100 {
        http.Error(w, "Percentage must be between 0 and 100", http.StatusBadRequest)
        return
    }
    
    // Actualizar ramping en Temporal
    cmd := exec.Command("temporal", "worker", "deployment", "set-build-id-ramping",
        "--task-queue", "purchase-approval-task-queue",
        "--build-id", req.BuildID,
        "--percentage", fmt.Sprintf("%.1f", req.Percentage))
    
    if err := cmd.Run(); err != nil {
        http.Error(w, "Failed to update ramping", http.StatusInternalServerError)
        return
    }
    
    // Log the change
    log.Printf("Rollout percentage updated: %s -> %.1f%%", req.BuildID, req.Percentage)
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "message": fmt.Sprintf("Rollout updated to %.1f%%", req.Percentage),
    })
}

// Endpoint para rollback de emergencia
func (h *DeploymentHandler) EmergencyRollback(w http.ResponseWriter, r *http.Request) {
    var req struct {
        BuildID string `json:"build_id"`
        Reason  string `json:"reason"`
    }
    
    json.NewDecoder(r.Body).Decode(&req)
    
    // Ejecutar rollback inmediato
    previousBuildID := h.getPreviousBuildID(req.BuildID)
    
    cmd := exec.Command("temporal", "worker", "deployment", "set-current-build-id",
        "--task-queue", "purchase-approval-task-queue", 
        "--build-id", previousBuildID)
    
    if err := cmd.Run(); err != nil {
        http.Error(w, "Rollback failed", http.StatusInternalServerError)
        return
    }
    
    // Notificar equipo
    h.sendSlackAlert(fmt.Sprintf("🚨 Emergency rollback executed: %s -> %s. Reason: %s", 
        req.BuildID, previousBuildID, req.Reason))
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "rolled_back",
        "previous_build_id": previousBuildID,
    })
}
```

### Testing Multi-Versión

```go
// tests/workflow_automated_review_test.go
func TestAutomatedReviewVersioning(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    
    t.Run("OriginalFlow_NoAutomatedReview", func(t *testing.T) {
        env := testSuite.NewTestWorkflowEnvironment()
        
        // Mock activities
        env.OnActivity("ValidateAmazonProduct", mock.Anything).Return(
            &AmazonValidationResult{Valid: true, ProductInfo: mockProductInfo()}, nil)
        env.OnActivity("SendApprovalRequest", mock.Anything).Return(nil)
        
        // No SetWorkflowVersion -> usa DefaultVersion
        env.ExecuteWorkflow(PurchaseApprovalWorkflow, PurchaseWorkflowInput{
            Request: mockPurchaseRequest(),
        })
        
        // Verificar que se llamó aprobación manual
        env.AssertActivityCalled(t, "SendApprovalRequest")
        env.AssertActivityNotCalled(t, "AutomatedReviewActivity")
        
        require.True(t, env.IsWorkflowCompleted())
    })
    
    t.Run("NewFlow_AutoApproved", func(t *testing.T) {
        env := testSuite.NewTestWorkflowEnvironment()
        env.SetWorkflowVersion("automated-review-v1", 1)
        
        // Mock activities
        env.OnActivity("ValidateAmazonProduct", mock.Anything).Return(
            &AmazonValidationResult{Valid: true, ProductInfo: mockProductInfo()}, nil)
        env.OnActivity("AutomatedReviewActivity", mock.Anything).Return(
            &AutomatedReviewResult{
                AutoApproved: true, 
                Confidence: 0.85,
                Reason: "Low risk purchase - under department limit",
                RiskScore: 0.15,
            }, nil)
        env.OnActivity("ExecuteAmazonPurchase", mock.Anything).Return(
            &AmazonPurchaseResult{Success: true}, nil)
        
        env.ExecuteWorkflow(PurchaseApprovalWorkflow, PurchaseWorkflowInput{
            Request: mockPurchaseRequest(),
        })
        
        // Verificar que NO se llamó aprobación manual
        env.AssertActivityNotCalled(t, "SendApprovalRequest")
        env.AssertActivityCalled(t, "AutomatedReviewActivity")
        env.AssertActivityCalled(t, "ExecuteAmazonPurchase")
        
        require.True(t, env.IsWorkflowCompleted())
    })
    
    t.Run("NewFlow_ManualApprovalRequired", func(t *testing.T) {
        env := testSuite.NewTestWorkflowEnvironment()
        env.SetWorkflowVersion("automated-review-v1", 1)
        
        // Mock automated review que rechaza auto-aprobación
        env.OnActivity("AutomatedReviewActivity", mock.Anything).Return(
            &AutomatedReviewResult{
                AutoApproved: false,
                Confidence: 0.3,
                Reason: "Amount exceeds auto-approval limit",
                RiskScore: 0.7,
            }, nil)
        env.OnActivity("SendApprovalRequest", mock.Anything).Return(nil)
        
        env.ExecuteWorkflow(PurchaseApprovalWorkflow, PurchaseWorkflowInput{
            Request: mockHighAmountPurchaseRequest(),
        })
        
        // Verificar que SÍ se llamó aprobación manual
        env.AssertActivityCalled(t, "AutomatedReviewActivity")
        env.AssertActivityCalled(t, "SendApprovalRequest")
    })
}

// Test de replay para verificar compatibilidad
func TestWorkflowReplayCompatibility(t *testing.T) {
    // Este test verifica que workflows existentes puedan ser 
    // "replayados" con la nueva versión del código sin errores
    
    client, err := client.Dial(client.Options{
        HostPort: "localhost:7233",
    })
    require.NoError(t, err)
    defer client.Close()
    
    // Obtener histories de workflows recientes
    resp, err := client.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
        Namespace: "default",
        Query:     "TaskQueue='purchase-approval-task-queue' AND StartTime > '2023-07-01'",
        PageSize:  10,
    })
    require.NoError(t, err)
    
    replayer := worker.NewWorkflowReplayer()
    replayer.RegisterWorkflow(PurchaseApprovalWorkflow)
    
    // Replay cada workflow history
    for _, execution := range resp.Executions {
        history, err := client.GetWorkflowHistory(context.Background(), 
            execution.Execution.WorkflowId, execution.Execution.RunId, false, 0)
        require.NoError(t, err)
        
        err = replayer.ReplayWorkflowHistory(nil, history)
        require.NoError(t, err, "Replay failed for workflow %s", execution.Execution.WorkflowId)
    }
}
```

### Monitoreo y Observabilidad

```go
// monitoring/deployment_monitor.go
type DeploymentMonitor struct {
    client         temporal.Client
    buildID        string
    targetDept     string
    alertThreshold AlertThresholds
}

type AlertThresholds struct {
    MaxErrorRate     float64 `json:"max_error_rate"`     // 5%
    MaxLatencyP99    time.Duration `json:"max_latency_p99"` // 30s
    MinSuccessRate   float64 `json:"min_success_rate"`   // 95%
}

func (m *DeploymentMonitor) MonitorDeployment(ctx context.Context) error {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics, err := m.collectMetrics()
            if err != nil {
                log.Printf("Error collecting metrics: %v", err)
                continue
            }
            
            if m.shouldAlert(metrics) {
                m.sendAlert(metrics)
            }
            
            if m.shouldRollback(metrics) {
                log.Printf("Critical metrics detected, initiating rollback")
                return m.executeEmergencyRollback()
            }
            
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

func (m *DeploymentMonitor) collectMetrics() (*DeploymentMetrics, error) {
    // Query Temporal para obtener métricas específicas del build ID
    query := fmt.Sprintf(
        `WorkerBuildId='%s' AND TaskQueue='purchase-approval-task-queue' AND StartTime > '%s'`,
        m.buildID,
        time.Now().Add(-5*time.Minute).Format(time.RFC3339),
    )
    
    resp, err := m.client.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{
        Namespace: "default",
        Query:     query,
        PageSize:  100,
    })
    if err != nil {
        return nil, err
    }
    
    metrics := &DeploymentMetrics{
        BuildID:    m.buildID,
        Timestamp:  time.Now(),
        Total:      len(resp.Executions),
    }
    
    for _, execution := range resp.Executions {
        switch execution.Status {
        case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
            metrics.Completed++
        case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
            metrics.Failed++
        case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
            metrics.Running++
        }
        
        // Calcular latencia
        if execution.CloseTime != nil {
            duration := execution.CloseTime.AsTime().Sub(execution.StartTime.AsTime())
            metrics.Latencies = append(metrics.Latencies, duration)
        }
    }
    
    metrics.ErrorRate = float64(metrics.Failed) / float64(metrics.Total)
    metrics.SuccessRate = float64(metrics.Completed) / float64(metrics.Total)
    
    return metrics, nil
}
```

### Configuración de Reglas Dinámicas

```go
// config/rules_manager.go
type RulesManager struct {
    storage RulesStorage
    cache   *sync.Map
}

// Interfaz para almacenamiento de reglas (Redis, DB, etc.)
type RulesStorage interface {
    GetRules(department string) (*AutoApprovalRules, error)
    SetRules(department string, rules *AutoApprovalRules) error
    ListRules() (map[string]*AutoApprovalRules, error)
}

// API REST para gestión de reglas
func (r *RulesManager) UpdateRulesHandler(w http.ResponseWriter, req *http.Request) {
    department := req.URL.Query().Get("department")
    
    var rules AutoApprovalRules
    if err := json.NewDecoder(req.Body).Decode(&rules); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Validar reglas
    if err := r.validateRules(&rules); err != nil {
        http.Error(w, fmt.Sprintf("Invalid rules: %v", err), http.StatusBadRequest)
        return
    }
    
    // Guardar en storage
    if err := r.storage.SetRules(department, &rules); err != nil {
        http.Error(w, "Failed to save rules", http.StatusInternalServerError)
        return
    }
    
    // Invalidar cache
    r.cache.Delete(department)
    
    log.Printf("Auto-approval rules updated for department: %s", department)
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "department": department,
    })
}

// Ejemplo de reglas por departamento
func (r *RulesManager) GetDefaultRules() map[string]*AutoApprovalRules {
    return map[string]*AutoApprovalRules{
        "IT": {
            MaxAmount: 1000.00,
            TrustedCategories: []string{"electronics", "software", "books"},
            UserHistory: HistoryRules{
                MinPreviousPurchases: 5,
                MaxRejectedInLast30Days: 0,
            },
        },
        "Marketing": {
            MaxAmount: 300.00,
            TrustedCategories: []string{"books", "office_supplies"},
            UserHistory: HistoryRules{
                MinPreviousPurchases: 3,
                MaxRejectedInLast30Days: 1,
            },
        },
        "Finance": {
            MaxAmount: 200.00,
            TrustedCategories: []string{"office_supplies"},
            UserHistory: HistoryRules{
                MinPreviousPurchases: 10,
                MaxRejectedInLast30Days: 0,
            },
        },
    }
}
```

## Métricas de Éxito

### KPIs del Proyecto
- **Reducción de tiempo de aprobación**: Target 40% para compras auto-aprobadas
- **Tasa de auto-aprobación**: Target 60-70% de solicitudes elegibles
- **Precisión de auto-aprobación**: Target >95% (pocas reversiones posteriores)
- **Zero downtime deployments**: 100% de deployments sin interrupciones
- **Rollback time**: <5 minutos para rollback de emergencia

### Dashboard de Monitoreo
```json
{
  "automated_review_metrics": {
    "auto_approval_rate": "67%",
    "avg_review_time": "2.3s",
    "confidence_score_avg": "0.78",
    "false_positive_rate": "2.1%"
  },
  "deployment_metrics": {
    "current_build_id": "purchase-worker-v2.1.0-abc123",
    "rollout_percentage": "50%",
    "error_rate": "0.8%",
    "avg_latency_p99": "12s"
  },
  "department_breakdown": {
    "IT": {"auto_approval": "85%", "avg_amount": "€456"},
    "Marketing": {"auto_approval": "45%", "avg_amount": "€189"},
    "Finance": {"auto_approval": "30%", "avg_amount": "€134"}
  }
}
```

## Lecciones Aprendidas

1. **GetVersion() es crítico**: Sin esto, cambios de workflow pueden quebrar execuciones existentes
2. **Feature flags complementan versioning**: Permiten control más granular sin redeploys
3. **Monitoreo proactivo esencial**: Detectar problemas antes de que afecten usuarios
4. **Testing de replay obligatorio**: Garantiza compatibilidad con workflows existentes  
5. **Rollback automático salva vidas**: Sistemas complejos requieren recuperación automática
6. **Documentación viva**: Cambios deben documentarse automáticamente

## Próximos Pasos

1. **Machine Learning Integration**: Usar datos históricos para mejorar precisión
2. **Multi-región deployment**: Estrategias para deployments globales
3. **Advanced A/B testing**: Comparar múltiples algoritmos de revisión
4. **Integration APIs**: Permitir que otros sistemas consulten reglas de aprobación
5. **Audit trail completo**: Trazabilidad total de decisiones automatizadas