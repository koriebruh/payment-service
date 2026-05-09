package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
	"github.com/sony/gobreaker"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/circuitbreaker"
)

type midtransClient struct {
	snapClient snap.Client
	coreClient coreapi.Client
	breaker    *gobreaker.CircuitBreaker
	timeout    time.Duration
}

func NewMidtransClient(cfg *config.Config) port.PaymentGatewayPort {
	var envType midtrans.EnvironmentType
	if cfg.Midtrans.IsProd {
		envType = midtrans.Production
	} else {
		envType = midtrans.Sandbox
	}

	// 1. Setup HTTP Client with Timeout
	httpClient := &midtrans.HttpClientImplementation{
		HttpClient: &http.Client{
			Timeout: cfg.Midtrans.Timeout,
		},
		Logger: midtrans.GetDefaultLogger(envType),
	}

	// 2. Setup Snap Client
	var snapClient snap.Client
	snapClient.New(cfg.Midtrans.ServerKey, envType)
	snapClient.HttpClient = httpClient

	// 3. Setup CoreAPI Client
	var coreClient coreapi.Client
	coreClient.New(cfg.Midtrans.ServerKey, envType)
	coreClient.HttpClient = httpClient

	// 4. Setup Circuit Breaker
	cbCfg := circuitbreaker.CircuitBreakerConfig{
		Name:        "midtrans-gateway",
		MaxRequests: cfg.Midtrans.CBMaxRequests,
		Interval:    cfg.Midtrans.CBInterval,
		Timeout:     cfg.Midtrans.CBTimeout,
	}
	breaker := circuitbreaker.NewCircuitBreaker(cbCfg)

	return &midtransClient{
		snapClient: snapClient,
		coreClient: coreClient,
		breaker:    breaker,
		timeout:    cfg.Midtrans.Timeout,
	}
}

func (m *midtransClient) CreateTransaction(ctx context.Context, t *domain.Transaction, c *domain.Customer) (*domain.Transaction, error) {
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  t.OrderID,
			GrossAmt: t.Amount,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: c.Name,
			Email: c.Email,
			Phone: c.Phone,
		},
	}

	// Execute via Circuit Breaker
	res, err := m.breaker.Execute(func() (interface{}, error) {
		// Snap API doesn't accept context directly, but we use the global configured httpClient timeout
		return m.snapClient.CreateTransaction(req)
	})

	if err != nil {
		return nil, fmt.Errorf("gateway CreateTransaction failed: %w", err)
	}

	snapResp := res.(*snap.Response)
	t.SnapToken = &snapResp.Token
	t.PaymentURL = &snapResp.RedirectURL

	return t, nil
}

func (m *midtransClient) RefundTransaction(ctx context.Context, orderID string, refund *domain.Refund) (*domain.Refund, error) {
	reason := "Refund request"
	if refund.Reason != nil {
		reason = *refund.Reason
	}

	req := &coreapi.RefundReq{
		RefundKey: refund.ID,
		Amount:    refund.Amount,
		Reason:    reason,
	}

	// Execute via Circuit Breaker
	res, err := m.breaker.Execute(func() (interface{}, error) {
		return m.coreClient.RefundTransaction(orderID, req)
	})

	if err != nil {
		return nil, fmt.Errorf("gateway RefundTransaction failed: %w", err)
	}

	resp := res.(*coreapi.RefundResponse)
	refund.Status = domain.RefundStatusSuccess
	refund.MidtransRefundKey = &req.RefundKey

	rawResp, _ := json.Marshal(resp)
	refund.MidtransResponse = rawResp

	return refund, nil
}

func (m *midtransClient) CancelTransaction(ctx context.Context, orderID string) error {
	// Execute via Circuit Breaker
	_, err := m.breaker.Execute(func() (interface{}, error) {
		return m.coreClient.CancelTransaction(orderID)
	})

	if err != nil {
		return fmt.Errorf("gateway CancelTransaction failed: %w", err)
	}

	return nil
}
