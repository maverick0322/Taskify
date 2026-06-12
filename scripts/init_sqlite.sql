PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    birth_date DATETIME NOT NULL,
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
    position INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_columns_name_length CHECK (length(trim(name)) >= 3),
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
    CONSTRAINT chk_credit_cards_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_credit_cards_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id ON credit_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id_bank ON credit_cards(user_id, bank);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credit_card_id TEXT REFERENCES credit_cards(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    date DATETIME NOT NULL,
    status TEXT NOT NULL,
    msi INTEGER NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE')),
    CONSTRAINT chk_transactions_concept_not_empty CHECK (length(trim(concept)) > 0),
    CONSTRAINT chk_transactions_category_not_empty CHECK (length(trim(category)) > 0),
    CONSTRAINT chk_transactions_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT chk_transactions_date_not_zero CHECK (date > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING')),
    CONSTRAINT chk_transactions_msi_positive CHECK (msi IS NULL OR msi >= 1),
    CONSTRAINT chk_transactions_created_at_not_zero CHECK (created_at > '0001-01-01 00:00:00+00:00'),
    CONSTRAINT chk_transactions_updated_at_not_zero CHECK (updated_at > '0001-01-01 00:00:00+00:00')
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_credit_card_id ON transactions(user_id, credit_card_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_date ON transactions(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_status ON transactions(user_id, status);

CREATE TABLE IF NOT EXISTS sync_state (
    key TEXT PRIMARY KEY,
    last_successful_sync_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
