package data

import (
	"DotaReplays/internal/validator"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"time"
)

type Replay struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Heroes    []string  `json:"heroes,omitempty"`
	Version   int32     `json:"version"`
}

func ValidateReplay(v *validator.Validator, replay *Replay) {
	v.Check(replay.Title != "", "title", "must be provided")
	v.Check(len(replay.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(replay.Year != 0, "year", "must be provided")
	v.Check(replay.Year >= 2011, "year", "must be greater than 2011")
	v.Check(replay.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(replay.Runtime != 0, "runtime", "must be provided")
	v.Check(replay.Runtime > 0, "runtime", "must be a positive integer")
	v.Check(replay.Heroes != nil, "heroes", "must be provided")
	v.Check(len(replay.Heroes) == 10, "heroes", "must contain 10 heroes")
	v.Check(validator.Unique(replay.Heroes), "heroes", "must not contain duplicate values")
}

type ReplayModel struct {
	DB *sql.DB
}

func (m ReplayModel) Insert(replay *Replay) error {
	query := `
INSERT INTO replays (title, year, runtime, heroes)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, version`
	args := []interface{}{replay.Title, replay.Year, replay.Runtime, pq.Array(replay.Heroes)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&replay.ID, &replay.CreatedAt, &replay.Version)
}

func (m ReplayModel) Get(id int64) (*Replay, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	// Remove the pg_sleep(10) clause.
	query := `
SELECT id, created_at, title, year, runtime, heroes, version
FROM replays
WHERE id = $1`
	var replay Replay
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Remove &[]byte{} from the first Scan() destination.
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&replay.ID,
		&replay.CreatedAt,
		&replay.Title,
		&replay.Year,
		&replay.Runtime,
		pq.Array(&replay.Heroes),
		&replay.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &replay, nil
}

func (m ReplayModel) Update(replay *Replay) error {
	query := `
UPDATE replays
SET title = $1, year = $2, runtime = $3, heroes = $4, version = version + 1
WHERE id = $5 AND version = $6
RETURNING version`
	args := []interface{}{
		replay.Title,
		replay.Year,
		replay.Runtime,
		pq.Array(replay.Heroes),
		replay.ID,
		replay.Version,
	}
	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&replay.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m ReplayModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `
DELETE FROM replays
WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (m ReplayModel) GetAll(title string, heroes []string, filters Filters) ([]*Replay, Metadata, error) {
	query := fmt.Sprintf(`
SELECT count(*) OVER(), id, created_at, title, year, runtime, heroes, version
FROM replays
WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (heroes @> $2 OR $2 = '{}')
ORDER BY %s %s, id ASC
LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	args := []any{title, pq.Array(heroes), filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}
	defer rows.Close()
	totalRecords := 0
	replays := []*Replay{}
	for rows.Next() {
		var replay Replay
		err := rows.Scan(
			&totalRecords,
			&replay.ID,
			&replay.CreatedAt,
			&replay.Title,
			&replay.Year,
			&replay.Runtime,
			pq.Array(&replay.Heroes),
			&replay.Version,
		)
		if err != nil {
			return nil, Metadata{}, err // Update this to return an empty Metadata struct.
		}
		replays = append(replays, &replay)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err // Update this to return an empty Metadata struct.
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return replays, metadata, nil
}
