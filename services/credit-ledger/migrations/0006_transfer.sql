-- 0006_transfer.sql — credit.yaml v1.4: move credits between a tenant's main balance and its
-- sub-accounts (team budgets). A transfer is two legs with operation 'transfer' in one DB transaction,
-- each extending its own balance's hash chain. Rollback: forward-only in prod.
ALTER TABLE credit_transactions DROP CONSTRAINT IF EXISTS credit_transactions_operation_check;
ALTER TABLE credit_transactions ADD CONSTRAINT credit_transactions_operation_check CHECK (operation IN
    ('purchase','consumption','conversion','mint','burn','refund','trade','transfer'));
