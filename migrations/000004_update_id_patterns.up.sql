-- Migration 000004: Update ID columns for transactions and refunds to VARCHAR(100)
-- to support human-readable patterns: INV/YYYYMMDD/TRX-XXXXXX and REF/YYYYMMDD/XXXX

-- transactions.id: UUID → VARCHAR(100)
ALTER TABLE refunds DROP CONSTRAINT IF EXISTS refunds_transaction_id_fkey;
ALTER TABLE webhook_logs DROP CONSTRAINT IF EXISTS webhook_logs_transaction_id_fkey;

ALTER TABLE transactions ALTER COLUMN id TYPE VARCHAR(100);
ALTER TABLE transactions ALTER COLUMN id DROP DEFAULT;

ALTER TABLE refunds ALTER COLUMN id TYPE VARCHAR(100);
ALTER TABLE refunds ALTER COLUMN id DROP DEFAULT;

-- restore FK constraints
ALTER TABLE refunds ADD CONSTRAINT refunds_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES transactions(id);

ALTER TABLE webhook_logs ALTER COLUMN transaction_id TYPE VARCHAR(100);
ALTER TABLE webhook_logs ADD CONSTRAINT webhook_logs_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES transactions(id);

ALTER TABLE refunds ALTER COLUMN transaction_id TYPE VARCHAR(100);
