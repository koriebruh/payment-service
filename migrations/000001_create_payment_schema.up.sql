-- V1__create_payment_service_tables.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE payment_methods (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            VARCHAR(50) UNIQUE NOT NULL,  -- gopay, qris, bca_va
    name            VARCHAR(100) NOT NULL,
    type            VARCHAR(50) NOT NULL,          -- ewallet, bank_transfer, card, qris
    is_active       BOOLEAN NOT NULL DEFAULT true,
    config          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(200) NOT NULL,
    email           VARCHAR(200),
    phone           VARCHAR(20),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id                  VARCHAR(100) UNIQUE NOT NULL,
    customer_id               UUID NOT NULL REFERENCES customers(id),
    payment_method_id         UUID NOT NULL REFERENCES payment_methods(id),
    amount                    BIGINT NOT NULL,   -- dalam Rupiah, hindari float
    currency                  VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status                    VARCHAR(20) NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending','settlement','cancel','deny','expire','failure')),
    midtrans_transaction_id   VARCHAR(200),
    midtrans_order_id         VARCHAR(200),
    payment_url               TEXT,
    snap_token                TEXT,
    paid_at                   TIMESTAMPTZ,
    expired_at                TIMESTAMPTZ,
    midtrans_response         JSONB,
    correlation_id            VARCHAR(100),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE refunds (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id        UUID NOT NULL REFERENCES transactions(id),
    amount                BIGINT NOT NULL,
    reason                TEXT,
    status                VARCHAR(20) NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending','success','failed')),
    midtrans_refund_key   VARCHAR(200),
    midtrans_response     JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_logs (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id            UUID REFERENCES transactions(id),
    event_type                VARCHAR(100),
    midtrans_transaction_id   VARCHAR(200),
    midtrans_status           VARCHAR(50),
    fraud_status              VARCHAR(50),
    raw_payload               TEXT NOT NULL,
    signature_valid           VARCHAR(10) NOT NULL DEFAULT 'unknown',
    received_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_transactions_customer_id    ON transactions(customer_id);
CREATE INDEX idx_transactions_status         ON transactions(status);
CREATE INDEX idx_transactions_created_at     ON transactions(created_at DESC);
CREATE INDEX idx_refunds_transaction_id      ON refunds(transaction_id);
CREATE INDEX idx_webhook_midtrans_trx_id     ON webhook_logs(midtrans_transaction_id);
