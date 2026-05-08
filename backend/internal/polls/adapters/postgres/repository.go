package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	polls "xugeaneeu/pollify/internal/polls/core"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, poll polls.Poll) (polls.Poll, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return polls.Poll{}, err
	}
	defer tx.Rollback(ctx)

	created, err := scanPollRow(tx.QueryRow(ctx, `
		INSERT INTO polls (
			title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
	`, poll.Title, poll.Description, poll.Question, poll.Settings.IsAnonymous, poll.Settings.IsMultipleChoice,
		poll.Settings.MaxChoices, poll.Settings.AllowCustomAnswer, poll.IsHidden, poll.CreatedBy,
		poll.Settings.StartAt, poll.Settings.EndAt, poll.CreatedAt))
	if err != nil {
		return polls.Poll{}, err
	}

	if err := upsertOptions(ctx, tx, created.ID, poll.Options); err != nil {
		return polls.Poll{}, err
	}

	created.Options, err = listOptions(ctx, tx, created.ID)
	if err != nil {
		return polls.Poll{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return polls.Poll{}, err
	}

	return created, nil
}

func (r *Repository) Update(ctx context.Context, poll polls.Poll) (polls.Poll, error) {
	updated, err := scanPollRow(r.db.QueryRow(ctx, `
		UPDATE polls
		SET title = $2,
			description = $3,
			question = $4,
			is_anonymous = $5,
			is_multiple_choice = $6,
			max_choices = $7,
			allow_custom_answer = $8,
			is_hidden = $9,
			start_at = $10,
			end_at = $11
		WHERE id = $1
		RETURNING id, title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
	`, poll.ID, poll.Title, poll.Description, poll.Question, poll.Settings.IsAnonymous,
		poll.Settings.IsMultipleChoice, poll.Settings.MaxChoices, poll.Settings.AllowCustomAnswer,
		poll.IsHidden, poll.Settings.StartAt, poll.Settings.EndAt))
	if err != nil {
		return polls.Poll{}, err
	}

	updated.Options, err = listOptions(ctx, r.db, updated.ID)
	if err != nil {
		return polls.Poll{}, err
	}

	return updated, nil
}

func (r *Repository) Hide(ctx context.Context, pollID string) error {
	commandTag, err := r.db.Exec(ctx, `UPDATE polls SET is_hidden = TRUE WHERE id = $1`, strings.TrimSpace(pollID))
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return polls.ErrPollNotFound
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, pollID string) (polls.Poll, error) {
	poll, err := scanPollRow(r.db.QueryRow(ctx, `
		SELECT id, title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
		FROM polls
		WHERE id = $1
	`, strings.TrimSpace(pollID)))
	if err != nil {
		return polls.Poll{}, err
	}

	poll.Options, err = listOptions(ctx, r.db, poll.ID)
	if err != nil {
		return polls.Poll{}, err
	}

	return poll, nil
}

func (r *Repository) List(ctx context.Context, filter polls.ListFilter) ([]polls.Poll, error) {
	args := []any{}
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
		FROM polls
		WHERE 1=1
	`)

	if filter.Status != nil {
		switch *filter.Status {
		case polls.StatusHidden:
			query.WriteString(` AND is_hidden = TRUE`)
		case polls.StatusScheduled:
			query.WriteString(` AND is_hidden = FALSE AND start_at > CURRENT_TIMESTAMP`)
		case polls.StatusActive:
			query.WriteString(` AND is_hidden = FALSE AND start_at <= CURRENT_TIMESTAMP AND end_at > CURRENT_TIMESTAMP`)
		case polls.StatusCompleted:
			query.WriteString(` AND is_hidden = FALSE AND end_at <= CURRENT_TIMESTAMP`)
		}
	}
	if filter.CreatorID != "" {
		args = append(args, filter.CreatorID)
		query.WriteString(` AND created_by = $` + ordinal(len(args)))
	}
	if filter.IsAnonymous != nil {
		args = append(args, *filter.IsAnonymous)
		query.WriteString(` AND is_anonymous = $` + ordinal(len(args)))
	}
	if filter.IsMultipleChoice != nil {
		args = append(args, *filter.IsMultipleChoice)
		query.WriteString(` AND is_multiple_choice = $` + ordinal(len(args)))
	}
	if filter.AllowCustomAnswer != nil {
		args = append(args, *filter.AllowCustomAnswer)
		query.WriteString(` AND allow_custom_answer = $` + ordinal(len(args)))
	}
	if filter.AvailableForVoting != nil && *filter.AvailableForVoting {
		query.WriteString(` AND is_hidden = FALSE AND start_at <= CURRENT_TIMESTAMP AND end_at > CURRENT_TIMESTAMP`)
	}

	switch filter.Sort {
	case "created_at_asc":
		query.WriteString(` ORDER BY created_at ASC`)
	case "start_at_asc":
		query.WriteString(` ORDER BY start_at ASC, created_at DESC`)
	case "end_at_asc":
		query.WriteString(` ORDER BY end_at ASC, created_at DESC`)
	default:
		query.WriteString(` ORDER BY created_at DESC`)
	}
	limit := maxValue(filter.Limit, 20)
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}
	args = append(args, limit, offset)
	query.WriteString(` LIMIT $` + ordinal(len(args)-1) + ` OFFSET $` + ordinal(len(args)))

	rows, err := r.db.Query(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]polls.Poll, 0)
	ids := make([]string, 0)
	for rows.Next() {
		poll, err := scanPollRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, poll)
		ids = append(ids, poll.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	optionsByPoll, err := listOptionsByPollIDs(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Options = optionsByPoll[items[i].ID]
	}

	return items, nil
}

func (r *Repository) HasParticipants(ctx context.Context, pollID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM poll_participants WHERE poll_id = $1
		)
	`, strings.TrimSpace(pollID)).Scan(&exists)
	return exists, err
}

func (r *Repository) ParticipationCount(ctx context.Context, pollID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM poll_participants WHERE poll_id = $1
	`, strings.TrimSpace(pollID)).Scan(&count)
	return count, err
}

func (r *Repository) HasUserVoted(ctx context.Context, pollID string, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM poll_participants WHERE poll_id = $1 AND user_id = $2
		)
	`, strings.TrimSpace(pollID), strings.TrimSpace(userID)).Scan(&exists)
	return exists, err
}

func (r *Repository) Count(ctx context.Context, filter polls.ListFilter) (int, error) {
	args := []any{}
	query := strings.Builder{}
	query.WriteString(`SELECT COUNT(*) FROM polls WHERE 1=1`)

	if filter.Status != nil {
		switch *filter.Status {
		case polls.StatusHidden:
			query.WriteString(` AND is_hidden = TRUE`)
		case polls.StatusScheduled:
			query.WriteString(` AND is_hidden = FALSE AND start_at > CURRENT_TIMESTAMP`)
		case polls.StatusActive:
			query.WriteString(` AND is_hidden = FALSE AND start_at <= CURRENT_TIMESTAMP AND end_at > CURRENT_TIMESTAMP`)
		case polls.StatusCompleted:
			query.WriteString(` AND is_hidden = FALSE AND end_at <= CURRENT_TIMESTAMP`)
		}
	}
	if filter.CreatorID != "" {
		args = append(args, filter.CreatorID)
		query.WriteString(` AND created_by = $` + ordinal(len(args)))
	}
	if filter.IsAnonymous != nil {
		args = append(args, *filter.IsAnonymous)
		query.WriteString(` AND is_anonymous = $` + ordinal(len(args)))
	}
	if filter.IsMultipleChoice != nil {
		args = append(args, *filter.IsMultipleChoice)
		query.WriteString(` AND is_multiple_choice = $` + ordinal(len(args)))
	}
	if filter.AllowCustomAnswer != nil {
		args = append(args, *filter.AllowCustomAnswer)
		query.WriteString(` AND allow_custom_answer = $` + ordinal(len(args)))
	}
	if filter.AvailableForVoting != nil && *filter.AvailableForVoting {
		query.WriteString(` AND is_hidden = FALSE AND start_at <= CURRENT_TIMESTAMP AND end_at > CURRENT_TIMESTAMP`)
	}

	var count int
	if err := r.db.QueryRow(ctx, query.String(), args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func scanPollRow(row pgx.Row) (polls.Poll, error) {
	var poll polls.Poll
	err := row.Scan(
		&poll.ID,
		&poll.Title,
		&poll.Description,
		&poll.Question,
		&poll.Settings.IsAnonymous,
		&poll.Settings.IsMultipleChoice,
		&poll.Settings.MaxChoices,
		&poll.Settings.AllowCustomAnswer,
		&poll.IsHidden,
		&poll.CreatedBy,
		&poll.Settings.StartAt,
		&poll.Settings.EndAt,
		&poll.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return polls.Poll{}, polls.ErrPollNotFound
		}
		return polls.Poll{}, err
	}

	return poll, nil
}

func scanPollRows(rows pgx.Rows) (polls.Poll, error) {
	var poll polls.Poll
	err := rows.Scan(
		&poll.ID,
		&poll.Title,
		&poll.Description,
		&poll.Question,
		&poll.Settings.IsAnonymous,
		&poll.Settings.IsMultipleChoice,
		&poll.Settings.MaxChoices,
		&poll.Settings.AllowCustomAnswer,
		&poll.IsHidden,
		&poll.CreatedBy,
		&poll.Settings.StartAt,
		&poll.Settings.EndAt,
		&poll.CreatedAt,
	)
	return poll, err
}

func upsertOptions(ctx context.Context, db queryRower, pollID string, options []polls.PollOption) error {
	for _, option := range options {
		if err := db.QueryRow(ctx, `
			INSERT INTO options (poll_id, text)
			VALUES ($1, $2)
			RETURNING id
		`, pollID, option.Text).Scan(&option.ID); err != nil {
			return err
		}
	}

	return nil
}

func listOptions(ctx context.Context, db queryer, pollID string) ([]polls.PollOption, error) {
	optionsByPoll, err := listOptionsByPollIDs(ctx, db, []string{pollID})
	if err != nil {
		return nil, err
	}

	return optionsByPoll[pollID], nil
}

func listOptionsByPollIDs(ctx context.Context, db queryer, pollIDs []string) (map[string][]polls.PollOption, error) {
	rows, err := db.Query(ctx, `
		SELECT id, poll_id, text
		FROM options
		WHERE poll_id = ANY($1::uuid[])
		ORDER BY poll_id, text, id
	`, pollIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]polls.PollOption, len(pollIDs))
	for _, pollID := range pollIDs {
		result[pollID] = []polls.PollOption{}
	}
	for rows.Next() {
		var option polls.PollOption
		var pollID string
		if err := rows.Scan(&option.ID, &pollID, &option.Text); err != nil {
			return nil, err
		}
		result[pollID] = append(result[pollID], option)
	}

	return result, rows.Err()
}

func ordinal(index int) string {
	return strconv.Itoa(index)
}

func maxValue(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}

	return value
}
