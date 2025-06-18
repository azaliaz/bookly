package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/azaliaz/bookly/feedback-service/internal/domain/consts"
	"github.com/azaliaz/bookly/feedback-service/internal/domain/models"
	"github.com/azaliaz/bookly/feedback-service/internal/logger"
	"github.com/azaliaz/bookly/feedback-service/internal/utils"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

type DBStorage struct {
	conn *pgx.Conn
}

func NewDB(ctx context.Context, addr string) (*DBStorage, error) {
	conn, err := pgx.Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		conn: conn,
	}, nil
}

func (dbs *DBStorage) SaveFeedback(feedback models.Feedback) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()
	feedbackID := uuid.New().String()
	createdAt := feedback.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := dbs.conn.Exec(ctx, `
		INSERT INTO feedbacks (feedback_id, user_id, book_id, text, create_at)
		VALUES ($1, $2, $3, $4, $5)
	`, feedbackID, feedback.UserID, feedback.BookID, feedback.Text, createdAt)

	if err != nil {
		log.Error().Err(err).Msg("failed to save feedback")
		return err
	}

	log.Info().Str("feedback_id", feedbackID).Msg("feedback saved successfully")
	return nil
}

func (dbs *DBStorage) GetFeedbacksByBookAsc(bookID string) ([]models.Feedback, error) {
	return dbs.getFeedbacks("book_id", bookID, utils.SortAsc)
}

func (dbs *DBStorage) GetFeedbacksByBookDesc(bookID string) ([]models.Feedback, error) {
	return dbs.getFeedbacks("book_id", bookID, utils.SortDesc)
}

func (dbs *DBStorage) GetFeedbacksByUserAsc(userID string) ([]models.Feedback, error) {
	return dbs.getFeedbacks("user_id", userID, utils.SortAsc)
}

func (dbs *DBStorage) GetFeedbacksByUserDesc(userID string) ([]models.Feedback, error) {
	return dbs.getFeedbacks("user_id", userID, utils.SortDesc)
}

func (dbs *DBStorage) getFeedbacks(columnName, id string, sortOrder utils.SortOrder) ([]models.Feedback, error) {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	query := fmt.Sprintf(`
	SELECT f.feedback_id, f.user_id, f.book_id, f.text, f.create_at,
	       u.name, u.lastname
	FROM feedbacks f
	JOIN users u ON f.user_id = u.uid
	WHERE f.%s = $1
	ORDER BY f.create_at %s`, columnName, string(sortOrder))

	rows, err := dbs.conn.Query(ctx, query, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get feedbacks")
		return nil, err
	}
	defer rows.Close()

	var feedbacks []models.Feedback
	for rows.Next() {
		var fb models.Feedback
		if err := rows.Scan(&fb.FeedbackID, &fb.UserID, &fb.BookID, &fb.Text, &fb.CreatedAt, &fb.Name, &fb.Lastname); err != nil {
			log.Error().Err(err).Msg("failed to scan feedback row")
			return nil, err
		}
		feedbacks = append(feedbacks, fb)
	}

	if rows.Err() != nil {
		log.Error().Err(rows.Err()).Msg("rows iteration error")
		return nil, rows.Err()
	}

	if len(feedbacks) == 0 {
		log.Warn().Str(columnName, id).Msg("no feedbacks found")
		return nil, errors.New("no feedbacks found")
	}

	return feedbacks, nil
}

func (dbs *DBStorage) DeleteFeedback(feedbackID string) error {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), consts.DBCtxTimeout)
	defer cancel()

	res, err := dbs.conn.Exec(ctx, `DELETE FROM feedbacks WHERE feedback_id = $1`, feedbackID)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete feedback")
		return err
	}
	if res.RowsAffected() == 0 {
		log.Warn().Str("fid", feedbackID).Msg("feedback not found")
		return errors.New("feedback not found")
	}
	log.Info().Str("fid", feedbackID).Msg("feedback deleted successfully")
	return nil
}

func Migrations(dbDsn string, migrationsPath string) error {
	log := logger.Get()
	migratePath := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(migratePath, dbDsn)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("no migrations to apply")
			return nil
		}
		return err
	}
	log.Info().Msg("all migrations applied")
	return nil
}
