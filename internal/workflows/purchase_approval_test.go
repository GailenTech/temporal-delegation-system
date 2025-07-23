package workflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"temporal-workflow/internal/activities"
	"temporal-workflow/internal/models"
)

type PurchaseApprovalTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *PurchaseApprovalTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *PurchaseApprovalTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *PurchaseApprovalTestSuite) Test_PurchaseApprovalWorkflow_Success() {
	// Arrange
	request := models.PurchaseRequest{
		ID:          "test-request-1",
		EmployeeID:  "employee@test.com",
		ProductURLs: []string{"https://amazon.es/dp/B08N5WRWNW"},
		Justification: "Need this for work",
		DeliveryOffice: "madrid",
	}

	expectedValidation := &models.PurchaseValidationResult{
		ValidItems: []models.CartItem{
			{
				ProductID:    "B08N5WRWNW",
				ProductURL:   "https://amazon.es/dp/B08N5WRWNW",
				Title:        "Echo Dot (4ª generación)",
				Price:        29.99,
				Quantity:     1,
				IsValid:      true,
				IsProhibited: false,
			},
		},
		TotalAmount: 29.99,
	}

	expectedApprovers := []string{"supervisor@company.com"}

	expectedOrder := &models.PurchaseOrder{
		RequestID:     "test-request-1",
		AmazonOrderID: "AMZ-12345",
		Status:        models.StatusCompleted,
	}

	// Mock activities
	s.env.OnActivity(activities.ValidateAmazonProducts, mock.Anything, mock.Anything).Return(expectedValidation, nil)
	s.env.OnActivity(activities.GetRequiredApprovers, mock.Anything, mock.Anything).Return(expectedApprovers, nil)
	s.env.OnActivity(activities.NotifyResponsible, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.ExecuteAmazonPurchase, mock.Anything, mock.Anything).Return(expectedOrder, nil)
	s.env.OnActivity(activities.NotifyEmployee, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	s.env.ExecuteWorkflow(PurchaseApprovalWorkflow, request)

	// Wait a bit for the workflow to set up
	s.env.RegisterDelayedCallback(func() {
		// Send approval signal
		approval := models.ApprovalResponse{
			RequestID:     "test-request-1",
			ResponsibleID: "supervisor@company.com",
			Approved:      true,
			Reason:        "Approved for testing",
			RespondedAt:   time.Now(),
		}
		s.env.SignalWorkflow("approval", approval)
	}, time.Millisecond*100)

	// Assert
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result models.PurchaseRequest
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(models.StatusCompleted, result.Status)
	s.Equal("test-request-1", result.ID)
}

func (s *PurchaseApprovalTestSuite) Test_PurchaseApprovalWorkflow_Rejection() {
	// Arrange
	request := models.PurchaseRequest{
		ID:          "test-request-2",
		EmployeeID:  "employee@test.com",
		ProductURLs: []string{"https://amazon.es/dp/B08N5WRWNW"},
		Justification: "Want this for personal use",
		DeliveryOffice: "madrid",
	}

	expectedValidation := &models.PurchaseValidationResult{
		ValidItems: []models.CartItem{
			{
				ProductID:    "B08N5WRWNW",
				ProductURL:   "https://amazon.es/dp/B08N5WRWNW",
				Title:        "Echo Dot (4ª generación)",
				Price:        29.99,
				Quantity:     1,
				IsValid:      true,
				IsProhibited: false,
			},
		},
		TotalAmount: 29.99,
	}

	expectedApprovers := []string{"supervisor@company.com"}

	// Mock activities
	s.env.OnActivity(activities.ValidateAmazonProducts, mock.Anything, mock.Anything).Return(expectedValidation, nil)
	s.env.OnActivity(activities.GetRequiredApprovers, mock.Anything, mock.Anything).Return(expectedApprovers, nil)
	s.env.OnActivity(activities.NotifyResponsible, mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity(activities.NotifyEmployee, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	s.env.ExecuteWorkflow(PurchaseApprovalWorkflow, request)

	// Wait a bit for the workflow to set up
	s.env.RegisterDelayedCallback(func() {
		// Send rejection signal
		approval := models.ApprovalResponse{
			RequestID:     "test-request-2",
			ResponsibleID: "supervisor@company.com",
			Approved:      false,
			Reason:        "Personal use not allowed",
			RespondedAt:   time.Now(),
		}
		s.env.SignalWorkflow("approval", approval)
	}, time.Millisecond*100)

	// Assert
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result models.PurchaseRequest
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(models.StatusRejected, result.Status)
	s.Equal("supervisor@company.com", result.ApprovalFlow.RejectedBy)
	s.Equal("Personal use not allowed", result.ApprovalFlow.RejectedReason)
}

func (s *PurchaseApprovalTestSuite) Test_PurchaseApprovalWorkflow_NoValidItems() {
	// Arrange
	request := models.PurchaseRequest{
		ID:          "test-request-3",
		EmployeeID:  "employee@test.com",
		ProductURLs: []string{"https://invalid-url.com/product"},
		Justification: "Need this for work",
		DeliveryOffice: "madrid",
	}

	expectedValidation := &models.PurchaseValidationResult{
		ValidItems:   []models.CartItem{},
		InvalidItems: []models.CartItem{
			{
				ProductURL:   "https://invalid-url.com/product",
				IsValid:      false,
				ErrorMessage: "URL de Amazon inválida",
			},
		},
		TotalAmount: 0,
		Warnings:    []string{"1 productos inválidos encontrados"},
	}

	// Mock activities
	s.env.OnActivity(activities.ValidateAmazonProducts, mock.Anything, mock.Anything).Return(expectedValidation, nil)
	s.env.OnActivity(activities.NotifyEmployee, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Act
	s.env.ExecuteWorkflow(PurchaseApprovalWorkflow, request)

	// Assert
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result models.PurchaseRequest
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(models.StatusRejected, result.Status)
}

func TestPurchaseApprovalTestSuite(t *testing.T) {
	suite.Run(t, new(PurchaseApprovalTestSuite))
}