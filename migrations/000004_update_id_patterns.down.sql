-- Rollback migration 000004: revert VARCHAR(100) back to UUID

ALTER TABLE refunds DROP CONSTRAINT IF EXISTS refunds_transaction_id_fkey;
ALTER TABLE webhook_logs DROP CONSTRAINT IF EXISTS webhook_logs_transaction_id_fkey;

ALTER TABLE transactions ALTER COLUMN id TYPE UUID USING id::uuid;
ALTER TABLE transactions ALTER COLUMN id SET DEFAULT uuid_generate_v4();

ALTER TABLE refunds ALTER COLUMN id TYPE UUID USING id::uuid;
ALTER TABLE refunds ALTER COLUMN id SET DEFAULT uuid_generate_v4();

ALTER TABLE refunds ALTER COLUMN transaction_id TYPE UUID USING transaction_id::uuid;
ALTER TABLE refunds ADD CONSTRAINT refunds_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES transactions(id);

ALTER TABLE webhook_logs ALTER COLUMN transaction_id TYPE UUID USING transaction_id::uuid;
ALTER TABLE webhook_logs ADD CONSTRAINT webhook_logs_transaction_id_fkey
    FOREIGN KEY (transaction_id) REFERENCES transactions(id);
