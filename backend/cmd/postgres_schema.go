package main

const postgresSyncSchema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    birth_date TIMESTAMPTZ NOT NULL,
    avatar_local_path TEXT NULL,
    avatar_url TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS boards (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_boards_user_id ON boards(user_id);
CREATE INDEX IF NOT EXISTS idx_boards_user_id_updated_at ON boards(user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'slate',
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ NULL
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
    due_date TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
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
    limit_cents BIGINT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
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
    opening_balance_cents BIGINT NOT NULL DEFAULT 0,
    current_balance_cents BIGINT NOT NULL DEFAULT 0,
    credit_limit_cents BIGINT NULL,
    cutoff_day INTEGER NULL,
    payment_day INTEGER NULL,
    color TEXT NOT NULL DEFAULT 'from-zinc-700 to-zinc-950',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id ON financial_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id_type ON financial_accounts(user_id, type);

INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, created_at, updated_at, deleted_at)
SELECT id, user_id, 'CREDIT_CARD', name, bank, last4, 0, 0, limit_cents, cutoff_day, payment_day, color, created_at, updated_at, deleted_at
FROM credit_cards
WHERE deleted_at IS NULL
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    institution = excluded.institution,
    last4 = excluded.last4,
    credit_limit_cents = excluded.credit_limit_cents,
    cutoff_day = excluded.cutoff_day,
    payment_day = excluded.payment_day,
    color = excluded.color,
    updated_at = excluded.updated_at;

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credit_card_id TEXT REFERENCES credit_cards(id) ON DELETE SET NULL,
    payment_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
    destination_account_id TEXT REFERENCES financial_accounts(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    msi INTEGER NULL,
    installment_number INTEGER NULL,
    installment_count INTEGER NULL,
    recurrence TEXT NOT NULL DEFAULT 'once',
    recurrence_limit INTEGER NULL,
    last_paid_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
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
    amount_cents BIGINT NOT NULL,
    entry_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_id ON ledger_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_account_id ON ledger_entries(account_id);

CREATE TABLE IF NOT EXISTS credit_card_statements (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credit_account_id TEXT NOT NULL REFERENCES financial_accounts(id) ON DELETE CASCADE,
    cycle_start TIMESTAMPTZ NOT NULL,
    cycle_end TIMESTAMPTZ NOT NULL,
    payment_due_date TIMESTAMPTZ NOT NULL,
    statement_amount_cents BIGINT NOT NULL DEFAULT 0,
    paid_amount_cents BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT uq_credit_card_statements_cycle UNIQUE (credit_account_id, cycle_start, cycle_end)
);
CREATE INDEX IF NOT EXISTS idx_credit_card_statements_user_id ON credit_card_statements(user_id);

CREATE TABLE IF NOT EXISTS account_payable_payments (
    id TEXT PRIMARY KEY,
    account_payable_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due_date TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ NOT NULL,
    amount_cents BIGINT NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    created_transaction_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_account_payable_payments_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT uq_account_payable_payments_cycle UNIQUE (account_payable_id, due_date)
);
ALTER TABLE account_payable_payments ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE account_payable_payments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;
CREATE INDEX IF NOT EXISTS idx_account_payable_payments_user_id ON account_payable_payments(user_id);
CREATE INDEX IF NOT EXISTS idx_account_payable_payments_account_payable_id ON account_payable_payments(account_payable_id);

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_notifications_title_not_empty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT chk_notifications_message_not_empty CHECK (char_length(trim(message)) > 0)
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_created_at ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_is_read ON notifications(user_id, is_read);
`
