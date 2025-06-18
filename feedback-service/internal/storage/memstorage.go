package storage

import (
	"github.com/azaliaz/bookly/feedback-service/internal/domain/models"
	"github.com/azaliaz/bookly/feedback-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/feedback-service/internal/storage/errors"
	"github.com/azaliaz/bookly/feedback-service/internal/utils"
	"github.com/google/uuid"
	"sort"
)

type MemStorage struct {
	feedbackStor map[string]models.Feedback
}

func New() *MemStorage {
	return &MemStorage{
		feedbackStor: make(map[string]models.Feedback),
	}
}

// Сохранить отзыв
func (ms *MemStorage) SaveFeedback(feedback models.Feedback) error {
	if _, exists := ms.feedbackStor[feedback.FeedbackID]; exists {

		ms.feedbackStor[feedback.FeedbackID] = feedback
		return nil
	}
	feedback.FeedbackID = uuid.New().String()
	ms.feedbackStor[feedback.FeedbackID] = feedback
	return nil
}

func (ms *MemStorage) SaveFeedbacks(feedbacks []models.Feedback) error {
	for _, feedback := range feedbacks {
		if err := ms.SaveFeedback(feedback); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MemStorage) GetFeedbacksByBookAsc(bookID string) ([]models.Feedback, error) {
	return ms.getFeedbacks("bid", bookID, utils.SortAsc)
}

func (ms *MemStorage) GetFeedbacksByBookDesc(bookID string) ([]models.Feedback, error) {
	return ms.getFeedbacks("bid", bookID, utils.SortDesc)
}
func (ms *MemStorage) GetFeedbacksByUserAsc(userID string) ([]models.Feedback, error) {
	return ms.getFeedbacks("uid", userID, utils.SortAsc)
}

func (ms *MemStorage) GetFeedbacksByUserDesc(userID string) ([]models.Feedback, error) {
	return ms.getFeedbacks("uid", userID, utils.SortDesc)
}

func (ms *MemStorage) getFeedbacks(filterBy string, id string, sortOrder utils.SortOrder) ([]models.Feedback, error) {
	var feedbacks []models.Feedback
	for _, feedback := range ms.feedbackStor {
		if (filterBy == "bid" && feedback.BookID == id) || (filterBy == "uid" && feedback.UserID == id) {
			feedbacks = append(feedbacks, feedback)
		}
	}

	if len(feedbacks) == 0 {
		return nil, storerrros.ErrFeedbackNoExist
	}

	sort.Slice(feedbacks, func(i, j int) bool {
		if sortOrder == utils.SortAsc {
			return feedbacks[i].CreatedAt.Before(feedbacks[j].CreatedAt)
		}
		return feedbacks[i].CreatedAt.After(feedbacks[j].CreatedAt)
	})

	return feedbacks, nil
}

func (ms *MemStorage) DeleteFeedback(feedbackID string) error {
	log := logger.Get()
	if _, exists := ms.feedbackStor[feedbackID]; !exists {
		log.Warn().Str("feedbackID", feedbackID).Msg("feedback not found")
		return storerrros.ErrFeedbackNoExist
	}

	delete(ms.feedbackStor, feedbackID)
	log.Info().Str("feedbackID", feedbackID).Msg("feedback deleted successfully")

	return nil
}
