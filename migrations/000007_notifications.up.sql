CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed');

-- Outbox / email log. The worker reads pending rows and dispatches them.
CREATE TABLE notifications (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipient_user_id bigint      REFERENCES users(id) ON DELETE SET NULL,
    recipient_email   text        NOT NULL,
    type              text        NOT NULL,   -- application_received | status_changed | interview_scheduled | ...
    subject           text        NOT NULL,
    body              text        NOT NULL,
    status            notification_status NOT NULL DEFAULT 'pending',
    error             text,
    sent_at           timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_status ON notifications(status);
