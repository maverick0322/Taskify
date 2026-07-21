package main

const sqliteSchema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    birth_date DATETIME NOT NULL,
    avatar_local_path TEXT NULL,
    avatar_url TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    is_revoked INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active_user_id ON refresh_tokens(user_id) WHERE is_revoked = 0;

CREATE TABLE IF NOT EXISTS boards (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_boards_name_length CHECK (length(trim(name)) >= 3),
    CONSTRAINT chk_boards_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_boards_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_boards_user_id ON boards(user_id);
CREATE INDEX IF NOT EXISTS idx_boards_user_id_updated_at ON boards(user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'slate',
    position INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_columns_name_length CHECK (length(trim(name)) >= 3),
    CONSTRAINT chk_columns_color_not_empty CHECK (length(trim(color)) > 0),
    CONSTRAINT chk_columns_position_non_negative CHECK (position >= 0),
    CONSTRAINT chk_columns_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_columns_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_columns_board_id ON columns(board_id);
CREATE INDEX IF NOT EXISTS idx_columns_board_id_position ON columns(board_id, position);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id TEXT REFERENCES boards(id) ON DELETE CASCADE,
    column_id TEXT REFERENCES columns(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    due_date DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_tasks_title_length CHECK (length(trim(title)) >= 3),
    CONSTRAINT chk_tasks_status CHECK (status IN ('todo', 'in_progress', 'done')),
    CONSTRAINT chk_tasks_priority CHECK (priority IN ('low', 'medium', 'high'))
);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_board_id ON tasks(user_id, board_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_column_id ON tasks(user_id, column_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_status ON tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_due_date ON tasks(user_id, due_date) WHERE due_date IS NOT NULL;

CREATE TABLE IF NOT EXISTS credit_cards (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    bank TEXT NOT NULL,
    last4 TEXT NOT NULL,
    cutoff_day INTEGER NOT NULL,
    payment_day INTEGER NOT NULL,
    limit_cents INTEGER NOT NULL,
    color TEXT NOT NULL,
    network TEXT NOT NULL DEFAULT 'Visa',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_credit_cards_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_credit_cards_bank_not_empty CHECK (length(trim(bank)) > 0),
    CONSTRAINT chk_credit_cards_last4 CHECK (last4 GLOB '[0-9][0-9][0-9][0-9]'),
    CONSTRAINT chk_credit_cards_cutoff_day CHECK (cutoff_day BETWEEN 1 AND 31),
    CONSTRAINT chk_credit_cards_payment_day CHECK (payment_day BETWEEN 1 AND 31),
    CONSTRAINT chk_credit_cards_limit_positive CHECK (limit_cents > 0),
    CONSTRAINT chk_credit_cards_color_not_empty CHECK (length(trim(color)) > 0),
    CONSTRAINT chk_credit_cards_network CHECK (network IN ('Visa', 'Mastercard', 'American Express')),
    CONSTRAINT chk_credit_cards_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_credit_cards_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id ON credit_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id_bank ON credit_cards(user_id, bank);

CREATE TABLE IF NOT EXISTS financial_accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    institution TEXT NOT NULL DEFAULT '',
    last4 TEXT NULL,
    opening_balance_cents INTEGER NOT NULL DEFAULT 0,
    current_balance_cents INTEGER NOT NULL DEFAULT 0,
    credit_limit_cents INTEGER NULL,
    cutoff_day INTEGER NULL,
    payment_day INTEGER NULL,
    color TEXT NOT NULL DEFAULT 'from-zinc-700 to-zinc-950',
    network TEXT NOT NULL DEFAULT 'Visa',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_financial_accounts_type CHECK (type IN ('CASH', 'DEBIT_CARD', 'CREDIT_CARD')),
    CONSTRAINT chk_financial_accounts_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_financial_accounts_cash_balance CHECK (type = 'CREDIT_CARD' OR current_balance_cents >= 0),
    CONSTRAINT chk_financial_accounts_credit_limit CHECK (type != 'CREDIT_CARD' OR credit_limit_cents > 0),
    CONSTRAINT chk_financial_accounts_cutoff_day CHECK (cutoff_day IS NULL OR cutoff_day BETWEEN 1 AND 31),
    CONSTRAINT chk_financial_accounts_payment_day CHECK (payment_day IS NULL OR payment_day BETWEEN 1 AND 31),
    CONSTRAINT chk_financial_accounts_network CHECK (type = 'CASH' OR network IN ('Visa', 'Mastercard', 'American Express'))
);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id ON financial_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id_type ON financial_accounts(user_id, type);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credit_card_id TEXT REFERENCES credit_cards(id) ON DELETE SET NULL,
    payment_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
    destination_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    date DATETIME NOT NULL,
    status TEXT NOT NULL,
    msi INTEGER NULL,
    installment_number INTEGER NULL,
    installment_count INTEGER NULL,
    is_historical BOOLEAN NOT NULL DEFAULT 0,
    recurrence TEXT NOT NULL DEFAULT 'once',
    recurrence_limit INTEGER NULL,
    last_paid_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE', 'DEBT_PAYMENT', 'TRANSFER')),
    CONSTRAINT chk_transactions_concept_not_empty CHECK (length(trim(concept)) > 0),
    CONSTRAINT chk_transactions_category_not_empty CHECK (length(trim(category)) > 0),
    CONSTRAINT chk_transactions_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT chk_transactions_date_not_zero CHECK (date > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING', 'COMPLETED')),
    CONSTRAINT chk_transactions_msi_positive CHECK (msi IS NULL OR msi >= 1),
    CONSTRAINT chk_transactions_installment_number_positive CHECK (installment_number IS NULL OR installment_number >= 1),
    CONSTRAINT chk_transactions_installment_count_positive CHECK (installment_count IS NULL OR installment_count >= 1),
    CONSTRAINT chk_transactions_recurrence CHECK (recurrence IN ('once', 'monthly', 'quarterly', 'biannual', 'annual')),
    CONSTRAINT chk_transactions_recurrence_limit_non_negative CHECK (recurrence_limit IS NULL OR recurrence_limit >= 0),
    CONSTRAINT chk_transactions_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_transactions_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_credit_card_id ON transactions(user_id, credit_card_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_payment_account_id ON transactions(user_id, payment_account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_date ON transactions(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_status ON transactions(user_id, status);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL,
    entry_type TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_id ON ledger_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_id ON ledger_entries(account_id);

CREATE TABLE IF NOT EXISTS credit_card_statements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credit_account_id TEXT NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    cycle_start DATETIME NOT NULL,
    cycle_end DATETIME NOT NULL,
    payment_due_date DATETIME NOT NULL,
    statement_amount_cents INTEGER NOT NULL DEFAULT 0,
    paid_amount_cents INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'DUE',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT uq_credit_card_statements_cycle UNIQUE (credit_account_id, cycle_start, cycle_end)
);
CREATE INDEX IF NOT EXISTS idx_credit_card_statements_user_id ON credit_card_statements(user_id);
CREATE INDEX IF NOT EXISTS idx_credit_card_statements_account_due ON credit_card_statements(credit_account_id, payment_due_date);

CREATE TABLE IF NOT EXISTS credit_card_statement_items (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    statement_id TEXT NOT NULL REFERENCES credit_card_statements(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_credit_card_statement_items_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT uq_credit_card_statement_items_transaction UNIQUE (statement_id, transaction_id)
);
CREATE INDEX IF NOT EXISTS idx_credit_card_statement_items_statement ON credit_card_statement_items(statement_id);

CREATE TABLE IF NOT EXISTS credit_card_payment_allocations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    statement_id TEXT NOT NULL REFERENCES credit_card_statements(id) ON DELETE CASCADE,
    payment_transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_credit_card_payment_allocations_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT uq_credit_card_payment_allocations_statement_payment UNIQUE (statement_id, payment_transaction_id)
);
CREATE INDEX IF NOT EXISTS idx_credit_card_payment_allocations_statement ON credit_card_payment_allocations(statement_id);
CREATE INDEX IF NOT EXISTS idx_credit_card_payment_allocations_payment ON credit_card_payment_allocations(payment_transaction_id);

CREATE TABLE IF NOT EXISTS account_payable_payments (
    id TEXT PRIMARY KEY,
    account_payable_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date DATETIME NOT NULL,
    paid_at DATETIME NOT NULL,
    amount_cents INTEGER NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    created_transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_account_payable_payments_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT uq_account_payable_payments_cycle UNIQUE (account_payable_id, due_date)
);
CREATE INDEX IF NOT EXISTS idx_account_payable_payments_user_id ON account_payable_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_account_payable_payments_account_payable_id ON account_payable_payments(account_payable_id);

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_notifications_title_not_empty CHECK (length(trim(title)) > 0),
    CONSTRAINT chk_notifications_message_not_empty CHECK (length(trim(message)) > 0)
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_created_at ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_is_read ON notifications(user_id, is_read);

CREATE TABLE IF NOT EXISTS storage_sync_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    local_path TEXT NOT NULL,
    bucket TEXT NOT NULL,
    object_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_storage_sync_jobs_entity UNIQUE (entity_type, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_storage_sync_jobs_status ON storage_sync_jobs(status, updated_at);

CREATE TABLE IF NOT EXISTS sync_runtime_flags (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO sync_runtime_flags (key, value)
VALUES ('suppress_outbox', '0')
ON CONFLICT(key) DO NOTHING;

CREATE TABLE IF NOT EXISTS sync_outbox (
    id TEXT PRIMARY KEY,
    table_name TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    next_attempt_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_sync_outbox_entity UNIQUE (table_name, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_sync_outbox_status_next_attempt ON sync_outbox(status, next_attempt_at, updated_at);

CREATE TABLE IF NOT EXISTS sync_state (
    key TEXT PRIMARY KEY,
    last_successful_sync_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
