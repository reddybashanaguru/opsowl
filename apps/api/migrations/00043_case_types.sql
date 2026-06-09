-- +goose Up
CREATE TABLE case_types (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       TEXT NOT NULL UNIQUE,
    label      TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO case_types (slug, label, description) VALUES
    ('aml',        'AML Investigation',      'Anti-money laundering suspicious transaction investigations'),
    ('fraud',      'Fraud Investigation',    'Payment, identity, or account fraud investigations'),
    ('kyc',        'KYC Review',             'Know Your Customer due diligence reviews'),
    ('sanctions',  'Sanctions Screening',    'Sanctions list hit review and escalation'),
    ('cyber',      'Cyber Incident',         'Cybersecurity incident investigations'),
    ('insider',    'Insider Threat',         'Internal misconduct and insider trading investigations'),
    ('pep',        'PEP Review',             'Politically exposed person enhanced due diligence');

-- +goose Down
DROP TABLE IF EXISTS case_types;
