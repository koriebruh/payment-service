package gateway

import (
	"context"

	"encoding/json"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type midtransClient struct {
	snapClient snap.Client
	coreClient coreapi.Client
}

func NewMidtransClient(cfg *config.Config) port.PaymentGatewayPort {
	var envType midtrans.EnvironmentType
	if cfg.Midtrans.IsProd {
		envType = midtrans.Production
	} else {
		envType = midtrans.Sandbox
	}

	var snapClient snap.Client
	snapClient.New(cfg.Midtrans.ServerKey, envType)

	var coreClient coreapi.Client
	coreClient.New(cfg.Midtrans.ServerKey, envType)

	return &midtransClient{
		snapClient: snapClient,
		coreClient: coreClient,
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

	snapResp, err := m.snapClient.CreateTransaction(req)
	if err != nil {
		return nil, err
	}

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

	resp, midtransErr := m.coreClient.RefundTransaction(orderID, req)
	if midtransErr != nil {
		return nil, midtransErr
	}

	refund.Status = domain.RefundStatusSuccess
	refund.MidtransRefundKey = &req.RefundKey

	rawResp, _ := json.Marshal(resp)
	refund.MidtransResponse = rawResp

	return refund, nil
}

func (m *midtransClient) CancelTransaction(ctx context.Context, orderID string) error {
	_, midtransErr := m.coreClient.CancelTransaction(orderID)
	if midtransErr != nil {
		return midtransErr
	}
	return nil
}
