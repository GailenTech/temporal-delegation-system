package activities

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"temporal-workflow/internal/models"
)

// AmazonActivities contiene todas las actividades relacionadas con Amazon
type AmazonActivities struct {
	// En el futuro aquí iría el cliente de Amazon API
}

// ValidateAmazonProducts valida una lista de URLs de productos de Amazon
func (a *AmazonActivities) ValidateAmazonProducts(ctx context.Context, productURLs []string) (*models.PurchaseValidationResult, error) {
	result := &models.PurchaseValidationResult{
		ValidItems:      []models.CartItem{},
		InvalidItems:    []models.CartItem{},
		ProhibitedItems: []models.CartItem{},
		DuplicatedItems: []models.CartItem{},
		Warnings:        []string{},
	}

	seenProducts := make(map[string]bool)
	totalAmount := 0.0

	for _, productURL := range productURLs {
		// Validar formato de URL
		if !isValidAmazonURL(productURL) {
			result.InvalidItems = append(result.InvalidItems, models.CartItem{
				ProductURL:   productURL,
				IsValid:      false,
				ErrorMessage: "URL de Amazon inválida",
			})
			continue
		}

		// Extraer ID del producto
		productID := extractProductID(productURL)
		if productID == "" {
			result.InvalidItems = append(result.InvalidItems, models.CartItem{
				ProductURL:   productURL,
				IsValid:      false,
				ErrorMessage: "No se pudo extraer ID del producto",
			})
			continue
		}

		// Verificar duplicados
		if seenProducts[productID] {
			result.DuplicatedItems = append(result.DuplicatedItems, models.CartItem{
				ProductURL:   productURL,
				ProductID:    productID,
				IsValid:      true,
				ErrorMessage: "Producto duplicado",
			})
			continue
		}
		seenProducts[productID] = true

		// Simular obtención de datos del producto (en el futuro sería llamada a API real)
		productInfo, err := a.getProductInfo(ctx, productID, productURL)
		if err != nil {
			result.InvalidItems = append(result.InvalidItems, models.CartItem{
				ProductURL:   productURL,
				ProductID:    productID,
				IsValid:      false,
				ErrorMessage: err.Error(),
			})
			continue
		}

		// Verificar si está prohibido
		if a.isProhibitedProduct(productInfo) {
			result.ProhibitedItems = append(result.ProhibitedItems, *productInfo)
			continue
		}

		// Producto válido
		result.ValidItems = append(result.ValidItems, *productInfo)
		totalAmount += productInfo.Price * float64(productInfo.Quantity)
	}

	result.TotalAmount = totalAmount

	// Agregar warnings si es necesario
	if len(result.InvalidItems) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d productos inválidos encontrados", len(result.InvalidItems)))
	}
	if len(result.DuplicatedItems) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d productos duplicados removidos", len(result.DuplicatedItems)))
	}
	if len(result.ProhibitedItems) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d productos prohibidos encontrados", len(result.ProhibitedItems)))
	}

	return result, nil
}

// ExecuteAmazonPurchase ejecuta la compra en Amazon
func (a *AmazonActivities) ExecuteAmazonPurchase(ctx context.Context, order models.PurchaseOrder) (*models.PurchaseOrder, error) {
	// STUB: En el futuro aquí iría la integración real con Amazon
	
	// Simular tiempo de procesamiento
	time.Sleep(time.Second * 2)

	// Simular éxito/fallo (90% éxito para testing)
	if time.Now().UnixNano()%10 < 9 {
		order.AmazonOrderID = fmt.Sprintf("AMZ-%d", time.Now().Unix())
		order.Status = models.StatusCompleted
	} else {
		order.Status = models.StatusFailed
		return &order, fmt.Errorf("simulación de fallo en compra Amazon")
	}

	return &order, nil
}

// Funciones auxiliares

func isValidAmazonURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Verificar que sea una URL de Amazon
	amazonDomains := []string{
		"amazon.com", "amazon.es", "amazon.co.uk", "amazon.de", "amazon.fr", "amazon.it",
		"www.amazon.com", "www.amazon.es", "www.amazon.co.uk", "www.amazon.de", "www.amazon.fr", "www.amazon.it",
	}

	for _, domain := range amazonDomains {
		if strings.Contains(u.Host, domain) {
			return true
		}
	}

	return false
}

func extractProductID(urlStr string) string {
	// Expresiones regulares para diferentes formatos de URLs de Amazon
	patterns := []string{
		`/dp/([A-Z0-9]{10})`,     // /dp/B08N5WRWNW
		`/gp/product/([A-Z0-9]{10})`, // /gp/product/B08N5WRWNW
		`/product-reviews/([A-Z0-9]{10})`, // /product-reviews/B08N5WRWNW
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func (a *AmazonActivities) getProductInfo(ctx context.Context, productID, productURL string) (*models.CartItem, error) {
	// STUB: En el futuro aquí haría scraping o llamada a API de Amazon
	
	// Simular productos de ejemplo
	mockProducts := map[string]models.CartItem{
		"B08N5WRWNW": {
			ProductID:    "B08N5WRWNW",
			ProductURL:   productURL,
			Title:        "Echo Dot (4ª generación) | Altavoz inteligente con Alexa",
			Price:        29.99,
			Quantity:     1,
			ImageURL:     "https://example.com/echo-dot.jpg",
			IsValid:      true,
			IsProhibited: false,
		},
		"B07XJ8C8F5": {
			ProductID:    "B07XJ8C8F5",
			ProductURL:   productURL,
			Title:        "Fire TV Stick con mando por voz Alexa",
			Price:        39.99,
			Quantity:     1,
			ImageURL:     "https://example.com/fire-stick.jpg",
			IsValid:      true,
			IsProhibited: false,
		},
		"PROHIBITED1": {
			ProductID:    "PROHIBITED1",
			ProductURL:   productURL,
			Title:        "Producto Prohibido - Armas",
			Price:        199.99,
			Quantity:     1,
			ImageURL:     "https://example.com/prohibited.jpg",
			IsValid:      true,
			IsProhibited: true,
		},
	}

	if product, exists := mockProducts[productID]; exists {
		product.ProductURL = productURL // Usar la URL real proporcionada
		return &product, nil
	}

	// Si no está en el mock, crear un producto genérico
	return &models.CartItem{
		ProductID:    productID,
		ProductURL:   productURL,
		Title:        fmt.Sprintf("Producto Amazon %s", productID),
		Price:        25.99, // Precio por defecto
		Quantity:     1,
		ImageURL:     "https://example.com/default-product.jpg",
		IsValid:      true,
		IsProhibited: false,
	}, nil
}

func (a *AmazonActivities) isProhibitedProduct(product *models.CartItem) bool {
	// Lista de palabras prohibidas (configurable)
	prohibitedKeywords := []string{
		"armas", "weapon", "gun", "pistol", "rifle",
		"alcohol", "tabaco", "tobacco", "cigarette",
		"adulto", "adult", "xxx",
	}

	titleLower := strings.ToLower(product.Title)
	for _, keyword := range prohibitedKeywords {
		if strings.Contains(titleLower, keyword) {
			product.IsProhibited = true
			product.ErrorMessage = fmt.Sprintf("Producto prohibido: contiene '%s'", keyword)
			return true
		}
	}

	// También verificar por ID específicos
	prohibitedIDs := []string{"PROHIBITED1"}
	for _, id := range prohibitedIDs {
		if product.ProductID == id {
			product.IsProhibited = true
			product.ErrorMessage = "Producto en lista negra"
			return true
		}
	}

	return false
}

// ValidateAmazonProducts función standalone para el worker
func ValidateAmazonProducts(ctx context.Context, productURLs []string) (*models.PurchaseValidationResult, error) {
	activities := &AmazonActivities{}
	return activities.ValidateAmazonProducts(ctx, productURLs)
}

// ExecuteAmazonPurchase función standalone para el worker
func ExecuteAmazonPurchase(ctx context.Context, order models.PurchaseOrder) (*models.PurchaseOrder, error) {
	activities := &AmazonActivities{}
	return activities.ExecuteAmazonPurchase(ctx, order)
}