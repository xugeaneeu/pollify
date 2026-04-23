package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformpg "xugeaneeu/pollify/internal/platform/postgres"
	voting "xugeaneeu/pollify/internal/voting/core"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) HasParticipated(ctx context.Context, pollID string, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM poll_participants WHERE poll_id = $1 AND user_id = $2
		)
	`, strings.TrimSpace(pollID), strings.TrimSpace(userID)).Scan(&exists)
	return exists, err
}

func (r *Repository) SaveVote(ctx context.Context, participation voting.Participation, votes []voting.VoteRecord) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO poll_participants (poll_id, user_id, voted_at)
		VALUES ($1, $2, $3)
	`, participation.PollID, participation.UserID, participation.VotedAt); err != nil {
		return mapVoteWriteError(err)
	}

	for _, vote := range votes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO votes (poll_id, option_id, user_id, custom_text, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, vote.PollID, nullableString(vote.OptionID), nullableString(vote.UserID), nullableString(vote.CustomText), vote.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetResults(ctx context.Context, pollID string, includeVoters bool) (voting.PollResults, error) {
	results := voting.PollResults{PollID: strings.TrimSpace(pollID)}

	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM poll_participants
		WHERE poll_id = $1
	`, results.PollID).Scan(&results.ParticipantsCount); err != nil {
		return voting.PollResults{}, err
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM votes
		WHERE poll_id = $1
	`, results.PollID).Scan(&results.TotalVotesCount); err != nil {
		return voting.PollResults{}, err
	}

	optionRows, err := r.db.Query(ctx, `
		WITH vote_totals AS (
			SELECT COUNT(*)::INTEGER AS total_votes
			FROM votes
			WHERE poll_id = $1
		)
		SELECT o.id, o.text, COUNT(v.id)::INTEGER,
			CASE WHEN vt.total_votes = 0 THEN 0::DOUBLE PRECISION ELSE COUNT(v.id)::DOUBLE PRECISION / vt.total_votes::DOUBLE PRECISION * 100 END
		FROM options o
		CROSS JOIN vote_totals vt
		LEFT JOIN votes v ON v.option_id = o.id
		WHERE o.poll_id = $1
		GROUP BY o.id, o.text, vt.total_votes
		ORDER BY o.text, o.id
	`, results.PollID)
	if err != nil {
		return voting.PollResults{}, err
	}
	defer optionRows.Close()

	for optionRows.Next() {
		var item voting.OptionResult
		if err := optionRows.Scan(&item.OptionID, &item.Label, &item.VotesCount, &item.Percentage); err != nil {
			return voting.PollResults{}, err
		}
		results.Options = append(results.Options, item)
	}
	if err := optionRows.Err(); err != nil {
		return voting.PollResults{}, err
	}

	textRows, err := r.db.Query(ctx, `
		SELECT custom_text, COUNT(*)::INTEGER
		FROM votes
		WHERE poll_id = $1 AND custom_text IS NOT NULL
		GROUP BY custom_text
		ORDER BY COUNT(*) DESC, custom_text ASC
	`, results.PollID)
	if err != nil {
		return voting.PollResults{}, err
	}
	defer textRows.Close()

	for textRows.Next() {
		var item voting.TextAnswer
		if err := textRows.Scan(&item.Value, &item.Count); err != nil {
			return voting.PollResults{}, err
		}
		results.CustomAnswers = append(results.CustomAnswers, item)
	}
	if err := textRows.Err(); err != nil {
		return voting.PollResults{}, err
	}

	if !includeVoters {
		return results, nil
	}

	voterRows, err := r.db.Query(ctx, `
		SELECT p.user_id, COALESCE(u.display_name, ''),
			COALESCE(array_remove(array_agg(v.option_id::TEXT ORDER BY v.option_id) FILTER (WHERE v.option_id IS NOT NULL), NULL), '{}'::TEXT[]),
			COALESCE(MAX(v.custom_text) FILTER (WHERE v.custom_text IS NOT NULL), '')
		FROM poll_participants p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN votes v ON v.poll_id = p.poll_id AND v.user_id = p.user_id
		WHERE p.poll_id = $1
		GROUP BY p.user_id, u.display_name
		ORDER BY u.display_name, p.user_id
	`, results.PollID)
	if err != nil {
		return voting.PollResults{}, err
	}
	defer voterRows.Close()

	for voterRows.Next() {
		var detail voting.VoterDetail
		if err := voterRows.Scan(&detail.UserID, &detail.DisplayName, &detail.SelectedOptionIDs, &detail.CustomText); err != nil {
			return voting.PollResults{}, err
		}
		results.VoterDetails = append(results.VoterDetails, detail)
	}

	return results, voterRows.Err()
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return strings.TrimSpace(value)
}

func mapVoteWriteError(err error) error {
	if err == nil {
		return nil
	}
	if platformpg.ErrorCode(err) == "23505" {
		return voting.ErrPollAlreadyVoted
	}

	return err
}
