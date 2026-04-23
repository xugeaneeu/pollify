CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'report_status') THEN
        CREATE TYPE report_status AS ENUM ('OPEN', 'IN_REVIEW', 'RESOLVED', 'REJECTED');
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'review_decision') THEN
        CREATE TYPE review_decision AS ENUM ('APPROVE', 'REJECT');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL CHECK (role IN ('USER', 'ADMIN')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    question TEXT NOT NULL,
    is_anonymous BOOLEAN NOT NULL,
    is_multiple_choice BOOLEAN NOT NULL,
    max_choices INTEGER NOT NULL DEFAULT 0,
    allow_custom_answer BOOLEAN NOT NULL,
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id),
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT polls_schedule_check CHECK (end_at > start_at),
    CONSTRAINT polls_multiple_choice_check CHECK (
        (NOT is_multiple_choice AND max_choices = 0) OR
        (is_multiple_choice AND max_choices > 1)
    ),
    CONSTRAINT polls_mixed_mode_check CHECK (NOT (is_multiple_choice AND allow_custom_answer))
);

CREATE TABLE IF NOT EXISTS options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS poll_participants (
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    voted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (poll_id, user_id)
);

CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id UUID NULL REFERENCES options(id) ON DELETE CASCADE,
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    custom_text TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT votes_payload_check CHECK (
        (option_id IS NOT NULL AND custom_text IS NULL) OR
        (option_id IS NULL AND custom_text IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    status report_status NOT NULL,
    approval_count INTEGER NOT NULL DEFAULT 0,
    rejection_count INTEGER NOT NULL DEFAULT 0,
    resolution TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS report_reviews (
    report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    decision review_decision NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (report_id, admin_id)
);

CREATE INDEX IF NOT EXISTS polls_created_by_idx ON polls(created_by);
CREATE INDEX IF NOT EXISTS polls_visibility_idx ON polls(is_hidden, start_at, end_at);
CREATE INDEX IF NOT EXISTS options_poll_id_idx ON options(poll_id);
CREATE INDEX IF NOT EXISTS votes_poll_id_idx ON votes(poll_id);
CREATE INDEX IF NOT EXISTS votes_user_id_idx ON votes(user_id);
CREATE INDEX IF NOT EXISTS reports_poll_id_idx ON reports(poll_id);
CREATE INDEX IF NOT EXISTS reports_created_by_idx ON reports(created_by);
CREATE INDEX IF NOT EXISTS reports_status_idx ON reports(status);

CREATE OR REPLACE FUNCTION ensure_report_review_admin() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM users
        WHERE id = NEW.admin_id AND role = 'ADMIN'
    ) THEN
        RAISE EXCEPTION 'report review requires ADMIN role';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION ensure_report_review_not_self() RETURNS trigger AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM reports
        WHERE id = NEW.report_id AND created_by = NEW.admin_id
    ) THEN
        RAISE EXCEPTION 'self review is forbidden';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION ensure_vote_option_matches_poll() RETURNS trigger AS $$
BEGIN
    IF NEW.option_id IS NOT NULL AND NOT EXISTS (
        SELECT 1
        FROM options
        WHERE id = NEW.option_id AND poll_id = NEW.poll_id
    ) THEN
        RAISE EXCEPTION 'vote option does not belong to poll';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS report_reviews_admin_only ON report_reviews;
CREATE TRIGGER report_reviews_admin_only
BEFORE INSERT OR UPDATE ON report_reviews
FOR EACH ROW
EXECUTE FUNCTION ensure_report_review_admin();

DROP TRIGGER IF EXISTS report_reviews_no_self_review ON report_reviews;
CREATE TRIGGER report_reviews_no_self_review
BEFORE INSERT OR UPDATE ON report_reviews
FOR EACH ROW
EXECUTE FUNCTION ensure_report_review_not_self();

DROP TRIGGER IF EXISTS votes_option_matches_poll ON votes;
CREATE TRIGGER votes_option_matches_poll
BEFORE INSERT OR UPDATE ON votes
FOR EACH ROW
EXECUTE FUNCTION ensure_vote_option_matches_poll();
