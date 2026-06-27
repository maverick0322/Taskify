-- scripts/init.sql

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    avatar_local_path TEXT NULL,
    avatar_url TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

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
    network TEXT NOT NULL DEFAULT 'Visa',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_credit_cards_name_not_empty CHECK (char_length(trim(name)) > 0),
    CONSTRAINT chk_credit_cards_bank_not_empty CHECK (char_length(trim(bank)) > 0),
    CONSTRAINT chk_credit_cards_last4 CHECK (last4 ~ '^[0-9]{4}$'),
    CONSTRAINT chk_credit_cards_cutoff_day CHECK (cutoff_day BETWEEN 1 AND 31),
    CONSTRAINT chk_credit_cards_payment_day CHECK (payment_day BETWEEN 1 AND 31),
    CONSTRAINT chk_credit_cards_limit_positive CHECK (limit_cents > 0),
    CONSTRAINT chk_credit_cards_color_not_empty CHECK (char_length(trim(color)) > 0),
    CONSTRAINT chk_credit_cards_created_at_not_zero CHECK (created_at > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_credit_cards_updated_at_not_zero CHECK (updated_at > TIMESTAMPTZ '0001-01-01 00:00:00+00')
);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id ON credit_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_credit_cards_user_id_bank ON credit_cards(user_id, bank);
ALTER TABLE credit_cards ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;
ALTER TABLE credit_cards ADD COLUMN IF NOT EXISTS network TEXT NOT NULL DEFAULT 'Visa';

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
    network TEXT NOT NULL DEFAULT 'Visa',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id ON financial_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_financial_accounts_user_id_type ON financial_accounts(user_id, type);
ALTER TABLE financial_accounts ADD COLUMN IF NOT EXISTS network TEXT NOT NULL DEFAULT 'Visa';

INSERT INTO financial_accounts (id, user_id, type, name, institution, last4, opening_balance_cents, current_balance_cents, credit_limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at, deleted_at)
SELECT id, user_id, 'CREDIT_CARD', name, bank, last4, 0, 0, limit_cents, cutoff_day, payment_day, color, network, created_at, updated_at, deleted_at
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
    network = excluded.network,
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
    is_historical BOOLEAN NOT NULL DEFAULT FALSE,
    recurrence TEXT NOT NULL DEFAULT 'once',
    recurrence_limit INTEGER NULL,
    last_paid_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE', 'DEBT_PAYMENT', 'TRANSFER')),
    CONSTRAINT chk_transactions_concept_not_empty CHECK (char_length(trim(concept)) > 0),
    CONSTRAINT chk_transactions_category_not_empty CHECK (char_length(trim(category)) > 0),
    CONSTRAINT chk_transactions_amount_positive CHECK (amount_cents > 0),
    CONSTRAINT chk_transactions_date_not_zero CHECK (date > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING', 'COMPLETED')),
    CONSTRAINT chk_transactions_msi_positive CHECK (msi IS NULL OR msi >= 1),
    CONSTRAINT chk_transactions_installment_number_positive CHECK (installment_number IS NULL OR installment_number >= 1),
    CONSTRAINT chk_transactions_installment_count_positive CHECK (installment_count IS NULL OR installment_count >= 1),
    CONSTRAINT chk_transactions_recurrence CHECK (recurrence IN ('once', 'monthly', 'quarterly', 'biannual', 'annual')),
    CONSTRAINT chk_transactions_recurrence_limit_non_negative CHECK (recurrence_limit IS NULL OR recurrence_limit >= 0),
    CONSTRAINT chk_transactions_created_at_not_zero CHECK (created_at > TIMESTAMPTZ '0001-01-01 00:00:00+00'),
    CONSTRAINT chk_transactions_updated_at_not_zero CHECK (updated_at > TIMESTAMPTZ '0001-01-01 00:00:00+00')
);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS credit_card_id TEXT NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS payment_account_id TEXT NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS destination_account_id TEXT NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS installment_number INTEGER NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS installment_count INTEGER NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS is_historical BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS recurrence TEXT NOT NULL DEFAULT 'once';
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS recurrence_limit INTEGER NULL;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS last_paid_at TIMESTAMPTZ NULL;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_status;
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_status CHECK (status IN ('PAID', 'PENDING', 'COMPLETED'));
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_type;
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_type CHECK (type IN ('INCOME', 'EXPENSE', 'DEBT_PAYMENT', 'TRANSFER'));
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_recurrence_limit_positive;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_recurrence_limit_non_negative;
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_recurrence_limit_non_negative CHECK (recurrence_limit IS NULL OR recurrence_limit >= 0);
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_transactions_credit_card_id'
    ) THEN
        ALTER TABLE transactions
            ADD CONSTRAINT fk_transactions_credit_card_id
            FOREIGN KEY (credit_card_id) REFERENCES credit_cards(id) ON DELETE SET NULL;
    END IF;
END $$;
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

CREATE OR REPLACE FUNCTION notify_taskify_sync()
RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify(
        'taskify_sync_events',
        json_build_object(
            'table', TG_TABLE_NAME,
            'operation', TG_OP,
            'id', COALESCE(NEW.id, OLD.id)
        )::text
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    sync_table text;
    trigger_name text;
BEGIN
    FOREACH sync_table IN ARRAY ARRAY[
        'users',
        'boards',
        'columns',
        'tasks',
        'financial_accounts',
        'transactions',
        'ledger_entries',
        'credit_card_statements',
        'account_payable_payments',
        'notifications'
    ]
    LOOP
        trigger_name := 'trg_' || sync_table || '_taskify_sync_notify';
        IF NOT EXISTS (
            SELECT 1
            FROM pg_trigger
            WHERE tgname = trigger_name
        ) THEN
            EXECUTE format(
                'CREATE TRIGGER %I AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION notify_taskify_sync()',
                trigger_name,
                sync_table
            );
        END IF;
    END LOOP;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'supabase_realtime') THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_publication_tables
            WHERE pubname = 'supabase_realtime'
              AND schemaname = 'public'
              AND tablename = 'boards'
        ) THEN
            ALTER PUBLICATION supabase_realtime ADD TABLE boards;
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_publication_tables
            WHERE pubname = 'supabase_realtime'
              AND schemaname = 'public'
              AND tablename = 'tasks'
        ) THEN
            ALTER PUBLICATION supabase_realtime ADD TABLE tasks;
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_publication_tables
            WHERE pubname = 'supabase_realtime'
              AND schemaname = 'public'
              AND tablename = 'columns'
        ) THEN
            ALTER PUBLICATION supabase_realtime ADD TABLE columns;
        END IF;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated')
       AND to_regprocedure('auth.uid()') IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_policies
            WHERE schemaname = 'public'
              AND tablename = 'boards'
              AND policyname = 'taskify_boards_realtime_select'
        ) THEN
            CREATE POLICY taskify_boards_realtime_select
            ON boards
            FOR SELECT
            TO authenticated
            USING (user_id = auth.uid()::text);
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_policies
            WHERE schemaname = 'public'
              AND tablename = 'tasks'
              AND policyname = 'taskify_tasks_realtime_select'
        ) THEN
            CREATE POLICY taskify_tasks_realtime_select
            ON tasks
            FOR SELECT
            TO authenticated
            USING (user_id = auth.uid()::text);
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_policies
            WHERE schemaname = 'public'
              AND tablename = 'columns'
              AND policyname = 'taskify_columns_realtime_select'
        ) THEN
            CREATE POLICY taskify_columns_realtime_select
            ON columns
            FOR SELECT
            TO authenticated
            USING (
                EXISTS (
                    SELECT 1
                    FROM boards
                    WHERE boards.id = columns.board_id
                      AND boards.user_id = auth.uid()::text
                )
            );
        END IF;
    END IF;
END $$;
