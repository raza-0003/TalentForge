CREATE TYPE offer_status AS ENUM ('draft', 'sent', 'accepted', 'declined', 'rescinded');

CREATE TABLE offer_letters (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    application_id  bigint        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    created_by      bigint        NOT NULL REFERENCES users(id),
    position_title  text          NOT NULL,
    salary_amount   numeric(12,2),
    salary_currency text          NOT NULL DEFAULT 'USD',
    start_date      date,
    status          offer_status  NOT NULL DEFAULT 'draft',
    storage_key     text,                         -- generated PDF object key
    expires_at      timestamptz,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    updated_at      timestamptz   NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_offer_letters_updated_at BEFORE UPDATE ON offer_letters
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_offers_application ON offer_letters(application_id);
