package gateway

import (
	"context"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type midtransClient struct {
	snapClient snap.Client
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

	return &midtransClient{
		snapClient: snapClient,
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
