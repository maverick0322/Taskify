-- scripts/init.sql

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active_user_id ON refresh_tokens(user_id) WHERE is_revoked = FALSE;

CREATE TABLE IF NOT EXISTS boards (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_boards_name_length CHECK (char_length(trim(name)) >= 3),
    CONSTRAINT chk_boards_created_at_not_zero CHECK (created_at > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_boards_updated_at_not_zero CHECK (updated_at > TIMESTAMPTZ '0001-01-01 00:00:00+00')
);
CREATE INDEX IF NOT EXISTS idx_boards_user_id ON boards(user_id);
CREATE INDEX IF NOT EXISTS idx_boards_user_id_updated_at ON boards(user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    board_id TEXT REFERENCES boards(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    due_date TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tasks_title_length CHECK (char_length(trim(title)) >= 3),
    CONSTRAINT chk_tasks_status CHECK (status IN ('todo', 'in_progress', 'done')),
    CONSTRAINT chk_tasks_priority CHECK (priority IN ('low', 'medium', 'high'))
);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_board_id ON tasks(user_id, board_id);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_status ON tasks(user_id, status);
ALTER TABLE tasks ALTER COLUMN due_date TYPE TIMESTAMPTZ USING due_date::timestamptz;
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_due_date ON tasks(user_id, due_date) WHERE due_date IS NOT NULL;

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    concept TEXT NOT NULL,
    category TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    msi INTEGER NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE')),
    CONSTRAINT chk_transactions_concept_not_empty CHECK (char_length(trim(concept)) > 0),
    CONSTRAINT chk_transactions_category_not_empty CHECK (char_length(trim(category)) > 0),
    CONSTRAINT chk_transactions_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT chk_transactions_date_not_zero CHECK (date > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING')),
    CONSTRAINT chk_transactions_msi_positive CHECK (msi IS NULL OR msi >= 1),
    CONSTRAINT chk_transactions_created_at_not_zero CHECK (created_at > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_transactions_updated_at_not_zero CHECK (updated_at > TIMESTAMPTZ '0001-01-01 00:00:00+00')
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_date ON transactions(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_status ON transactions(user_id, status);

CREATE TABLE IF NOT EXISTS columns (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_columns_name_length CHECK (char_length(trim(name)) >= 3),
    CONSTRAINT chk_columns_position_non_negative CHECK (position >= 0),
    CONSTRAINT chk_columns_created_at_not_zero CHECK (created_at > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_columns_updated_at_not_zero CHECK (updated_at > TIMESTAMPTZ '0001-01-01 00:00:00+00')
);
CREATE INDEX IF NOT EXISTS idx_columns_board_id ON columns(board_id);
CREATE INDEX IF NOT EXISTS idx_columns_board_id_position ON columns(board_id, position);
