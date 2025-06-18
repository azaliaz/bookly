package server

import (
	"github.com/azaliaz/bookly/feedback-service/internal/domain/models"
	"github.com/azaliaz/bookly/feedback-service/internal/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s *Server) saveFeedback(ctx *gin.Context) {
	var feedback models.Feedback
	log := logger.Get()
	if err := ctx.ShouldBindJSON(&feedback); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := s.storage.SaveFeedback(feedback); err != nil {
		log.Printf("failed to save feedback: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save feedback"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "feedback saved successfully"})
}
func (s *Server) getFeedbacksByBookAsc(ctx *gin.Context) {
	bookID := ctx.Param("book_id")

	feedbacks, err := s.storage.GetFeedbacksByBookAsc(bookID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedbacks"})
		return
	}

	ctx.JSON(http.StatusOK, feedbacks)
}
func (s *Server) getFeedbacksByBookDesc(ctx *gin.Context) {
	bookID := ctx.Param("book_id")

	feedbacks, err := s.storage.GetFeedbacksByBookDesc(bookID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedbacks"})
		return
	}

	ctx.JSON(http.StatusOK, feedbacks)
}
func (s *Server) getFeedbacksByUserAsc(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	feedbacks, err := s.storage.GetFeedbacksByUserAsc(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedbacks"})
		return
	}

	ctx.JSON(http.StatusOK, feedbacks)
}
func (s *Server) getFeedbacksByUserDesc(ctx *gin.Context) {
	userID := ctx.Param("user_id")

	feedbacks, err := s.storage.GetFeedbacksByUserDesc(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedbacks"})
		return
	}

	ctx.JSON(http.StatusOK, feedbacks)
}
func (s *Server) deleteFeedback(ctx *gin.Context) {
	feedbackID := ctx.Param("feedback_id")

	err := s.storage.DeleteFeedback(feedbackID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete feedback"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "feedback deleted successfully"})
}
