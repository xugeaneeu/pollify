DROP TRIGGER IF EXISTS votes_option_matches_poll ON votes;
DROP TRIGGER IF EXISTS report_reviews_no_self_review ON report_reviews;
DROP TRIGGER IF EXISTS report_reviews_admin_only ON report_reviews;

DROP FUNCTION IF EXISTS ensure_vote_option_matches_poll();
DROP FUNCTION IF EXISTS ensure_report_review_not_self();
DROP FUNCTION IF EXISTS ensure_report_review_admin();

DROP TABLE IF EXISTS report_reviews;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS poll_participants;
DROP TABLE IF EXISTS options;
DROP TABLE IF EXISTS polls;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS review_decision;
DROP TYPE IF EXISTS report_status;
