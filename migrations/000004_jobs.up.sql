CREATE TYPE job_status AS ENUM ('draft', 'open', 'closed');

CREATE TABLE jobs (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    created_by      bigint      NOT NULL REFERENCES users(id),   -- recruiter/admin
    title           text        NOT NULL,
    description     text        NOT NULL,
    department      text,
    location        text,
    employment_type text,
    min_experience  int         NOT NULL DEFAULT 0,
    status          job_status  NOT NULL DEFAULT 'open',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_jobs_updated_at BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_by ON jobs(created_by);
