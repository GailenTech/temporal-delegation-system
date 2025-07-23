package models

import "time"

// PurchaseRequest representa una solicitud de compra
type PurchaseRequest struct {
	ID          string    `json:"id"`
	EmployeeID  string    `json:"employee_id"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"` // pending, approved, rejected, completed
	TotalAmount float64   `json:"total_amount"`
	
	// Datos del formulario
	ProductURLs    []string `json:"product_urls"`
	Justification  string   `json:"justification"`
	DeliveryOffice string   `json:"delivery_office"`
	
	// Estado del carrito
	Cart Cart `json:"cart"`
	
	// Flujo de aprobación
	ApprovalFlow ApprovalFlow `json:"approval_flow"`
}

// Cart representa el carrito de compras
type Cart struct {
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"total_amount"`
	Currency    string     `json:"currency"`
}

// CartItem representa un producto en el carrito
type CartItem struct {
	ProductURL   string  `json:"product_url"`
	ProductID    string  `json:"product_id"`
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	Quantity     int     `json:"quantity"`
	ImageURL     string  `json:"image_url"`
	IsValid      bool    `json:"is_valid"`
	IsProhibited bool    `json:"is_prohibited"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// ApprovalFlow representa el estado del flujo de aprobación
type ApprovalFlow struct {
	RequiredApprovals []string  `json:"required_approvals"` // Lista de responsables que deben aprobar
	ApprovedBy        []string  `json:"approved_by"`        // Lista de quienes han aprobado
	RejectedBy        string    `json:"rejected_by,omitempty"`
	RejectedReason    string    `json:"rejected_reason,omitempty"`
	ApprovalDeadline  time.Time `json:"approval_deadline"`
	
	// Modificaciones por responsables
	Modifications []CartModification `json:"modifications"`
}

// CartModification representa cambios hechos por un responsable
type CartModification struct {
	ModifiedBy string    `json:"modified_by"`
	ModifiedAt time.Time `json:"modified_at"`
	Changes    string    `json:"changes"` // JSON con los cambios realizados
	Reason     string    `json:"reason"`
}

// PurchaseValidationResult resultado de validación de productos
type PurchaseValidationResult struct {
	ValidItems      []CartItem `json:"valid_items"`
	InvalidItems    []CartItem `json:"invalid_items"`
	ProhibitedItems []CartItem `json:"prohibited_items"`
	DuplicatedItems []CartItem `json:"duplicated_items"`
	TotalAmount     float64    `json:"total_amount"`
	Warnings        []string   `json:"warnings"`
}

// ApprovalRequest solicitud de aprobación enviada a responsable
type ApprovalRequest struct {
	RequestID     string    `json:"request_id"`
	EmployeeID    string    `json:"employee_id"`
	ResponsibleID string    `json:"responsible_id"`
	Cart          Cart      `json:"cart"`
	Justification string    `json:"justification"`
	SentAt        time.Time `json:"sent_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// ApprovalResponse respuesta del responsable
type ApprovalResponse struct {
	RequestID      string `json:"request_id"`
	ResponsibleID  string `json:"responsible_id"`
	Approved       bool   `json:"approved"`
	Reason         string `json:"reason,omitempty"`
	ModifiedCart   *Cart  `json:"modified_cart,omitempty"`
	RespondedAt    time.Time `json:"responded_at"`
}

// PurchaseOrder orden de compra para Amazon
type PurchaseOrder struct {
	RequestID      string    `json:"request_id"`
	Cart           Cart      `json:"cart"`
	DeliveryOffice string    `json:"delivery_office"`
	CreatedAt      time.Time `json:"created_at"`
	AmazonOrderID  string    `json:"amazon_order_id,omitempty"`
	Status         string    `json:"status"` // pending, processing, completed, failed
}

// Constantes para estados
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Constantes para estados de workflow
const (
	RoleResponsible = "responsible" // Mantenemos solo este para compatibilidad
)