-- +goose Up
-- +goose StatementBegin

-- ── Account registry ──────────────────────────────────────────────────────────
-- Represents a financial account referenced in investigations.
-- Accounts are org-scoped and reusable across cases.
CREATE TABLE accounts (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID        NOT NULL,
  account_number  TEXT        NOT NULL,
  account_type    TEXT        NOT NULL DEFAULT 'bank',   -- bank, wallet, crypto, card, loan, upi
  bank_name       TEXT        NOT NULL DEFAULT '',
  bank_ifsc       TEXT        NOT NULL DEFAULT '',
  holder_name     TEXT        NOT NULL DEFAULT '',
  holder_type     TEXT        NOT NULL DEFAULT 'individual', -- individual, company, unknown
  currency        TEXT        NOT NULL DEFAULT 'INR',
  country         TEXT        NOT NULL DEFAULT 'IN',
  subject_id      UUID        REFERENCES subjects(id) ON DELETE SET NULL,
  metadata        JSONB       NOT NULL DEFAULT '{}',
  created_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (org_id, account_number)
);

CREATE INDEX idx_accounts_org       ON accounts(org_id);
CREATE INDEX idx_accounts_subject   ON accounts(subject_id) WHERE subject_id IS NOT NULL;
CREATE INDEX idx_accounts_number    ON accounts(org_id, account_number);

-- ── Transaction ledger ────────────────────────────────────────────────────────
-- Each row is one financial transaction. Debit/credit are from the perspective
-- of the originating account. Transactions can exist without being linked to a
-- case; they become case evidence when added to case_transactions.
CREATE TABLE transactions (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id              UUID        NOT NULL,
  txn_ref             TEXT        NOT NULL,           -- bank/payment reference number
  txn_date            DATE        NOT NULL,
  txn_time            TIME,                           -- nullable — not always available
  amount              NUMERIC(20,4) NOT NULL,
  currency            TEXT        NOT NULL DEFAULT 'INR',
  txn_type            TEXT        NOT NULL DEFAULT 'debit',  -- debit, credit, reversal, fee, interest
  channel             TEXT        NOT NULL DEFAULT 'unknown', -- neft, rtgs, imps, upi, cash, card, swift, internal
  status              TEXT        NOT NULL DEFAULT 'completed', -- completed, pending, failed, reversed
  description         TEXT        NOT NULL DEFAULT '',
  narration           TEXT        NOT NULL DEFAULT '',       -- raw bank narration
  -- Parties
  from_account_id     UUID        REFERENCES accounts(id) ON DELETE SET NULL,
  to_account_id       UUID        REFERENCES accounts(id) ON DELETE SET NULL,
  from_account_raw    TEXT        NOT NULL DEFAULT '',  -- raw string when account not resolved
  to_account_raw      TEXT        NOT NULL DEFAULT '',
  from_name_raw       TEXT        NOT NULL DEFAULT '',
  to_name_raw         TEXT        NOT NULL DEFAULT '',
  -- Risk signals
  is_flagged          BOOLEAN     NOT NULL DEFAULT false,
  flag_reason         TEXT        NOT NULL DEFAULT '',
  risk_score          INT         NOT NULL DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
  -- Counterparty geographic signals
  counterparty_bank   TEXT        NOT NULL DEFAULT '',
  counterparty_country TEXT       NOT NULL DEFAULT '',
  -- Metadata from source system
  source_system       TEXT        NOT NULL DEFAULT 'manual', -- manual, csv_import, api_push, alert
  external_id         TEXT        NOT NULL DEFAULT '',       -- ID in source TMS/CBS
  metadata            JSONB       NOT NULL DEFAULT '{}',
  created_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (org_id, txn_ref)
);

CREATE INDEX idx_txn_org         ON transactions(org_id);
CREATE INDEX idx_txn_date        ON transactions(org_id, txn_date DESC);
CREATE INDEX idx_txn_from        ON transactions(from_account_id) WHERE from_account_id IS NOT NULL;
CREATE INDEX idx_txn_to          ON transactions(to_account_id) WHERE to_account_id IS NOT NULL;
CREATE INDEX idx_txn_flagged     ON transactions(org_id, is_flagged) WHERE is_flagged = true;
CREATE INDEX idx_txn_external    ON transactions(org_id, external_id) WHERE external_id <> '';

-- ── Case ↔ Transaction link ───────────────────────────────────────────────────
-- Many-to-many: a transaction can be evidence in multiple cases,
-- a case can have many transactions.
CREATE TABLE case_transactions (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id         UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  transaction_id  UUID        NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  relevance_note  TEXT        NOT NULL DEFAULT '',   -- why this txn matters to this case
  added_by        UUID        REFERENCES users(id) ON DELETE SET NULL,
  added_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (case_id, transaction_id)
);

CREATE INDEX idx_case_txn_case ON case_transactions(case_id);
CREATE INDEX idx_case_txn_txn  ON case_transactions(transaction_id);

-- ── Timeline event types (ENUM values) ───────────────────────────────────────
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'transaction_added';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'transaction_removed';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'transaction_flagged';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS case_transactions;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
