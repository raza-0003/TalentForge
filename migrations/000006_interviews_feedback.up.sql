CREATE TYPE interview_mode           AS ENUM ('onsite', 'phone', 'video');
CREATE TYPE interview_status         AS ENUM ('scheduled', 'completed', 'cancelled', 'no_show');
CREATE TYPE feedback_recommendation  AS ENUM ('strong_yes', 'yes', 'no', 'strong_no');

CREATE TABLE interviews (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    application_id   bigint          NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    interviewer_id   bigint          NOT NULL REFERENCES users(id),
    created_by       bigint          NOT NULL REFERENCES users(id),
    scheduled_at     timestamptz     NOT NULL,
    duration_minutes int             NOT NULL DEFAULT 60,
    mode             interview_mode  NOT NULL DEFAULT 'video',
    location         text,                       -- room name or video link
    status           interview_status NOT NULL DEFAULT 'scheduled',
    created_at       timestamptz     NOT NULL DEFAULT now(),
    updated_at       timestamptz     NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_interviews_updated_at BEFORE UPDATE ON interviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_interviews_application ON interviews(application_id);
CREATE INDEX idx_interviews_interviewer ON interviews(interviewer_id, scheduled_at);

CREATE TABLE interview_feedback (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    interview_id   bigint      NOT NULL UNIQUE REFERENCES interviews(id) ON DELETE CASCADE,
    author_id      bigint      NOT NULL REFERENCES users(id),
    rating         int         NOT NULL CHECK (rating BETWEEN 1 AND 5),
    recommendation feedback_recommendation NOT NULL,
    strengths      text,
    weaknesses     text,
    comments       text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_interview_feedback_updated_at BEFORE UPDATE ON interview_feedback
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
