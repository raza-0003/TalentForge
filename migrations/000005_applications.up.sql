CREATE TYPE application_status AS ENUM (
    'applied', 'screening', 'shortlisted', 'interview',
    'offer', 'hired', 'rejected', 'withdrawn'
);

CREATE TABLE applications (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_id       bigint      NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    candidate_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resume_id    bigint      REFERENCES resumes(id) ON DELETE SET NULL,
    status       application_status NOT NULL DEFAULT 'applied',
    cover_letter text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (job_id, candidate_id)   -- one application per candidate per job
);

CREATE TRIGGER trg_applications_updated_at BEFORE UPDATE ON applications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_applications_job       ON applications(job_id);
CREATE INDEX idx_applications_candidate ON applications(candidate_id);
CREATE INDEX idx_applications_status    ON applications(status);

-- Append-only audit trail that powers the candidate "interview timeline".
CREATE TABLE application_events (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    application_id bigint      NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    actor_id       bigint      REFERENCES users(id) ON DELETE SET NULL,
    event_type     text        NOT NULL,   -- created | status_changed | note | interview_scheduled | ...
    from_status    application_status,
    to_status      application_status,
    note           text,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_application_events_app ON application_events(application_id, created_at);
