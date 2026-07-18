CREATE TABLE candidate_profiles (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    phone      text,
    headline   text,
    location   text,
    links      jsonb       NOT NULL DEFAULT '{}'::jsonb,
    skills     text[]      NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_candidate_profiles_updated_at BEFORE UPDATE ON candidate_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- GIN index enables fast skill-based candidate search (Phase 6).
CREATE INDEX idx_candidate_profiles_skills ON candidate_profiles USING gin (skills);

CREATE TABLE resumes (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    candidate_user_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_key       text        NOT NULL,             -- object key in S3 / local store
    file_name         text        NOT NULL,
    content_type      text,
    size_bytes        bigint,
    is_primary        boolean     NOT NULL DEFAULT false,
    parsed_at         timestamptz,                       -- set by the resume-parse worker
    parsed_data       jsonb,                             -- extracted email/phone/skills
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_resumes_candidate ON resumes(candidate_user_id);
