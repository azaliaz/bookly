package server

import (
	// "context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
)

func (s *Server) AllBooksWithSearch(ctx *gin.Context) {
	searchQuery := ctx.DefaultQuery("search", "")
	genres := ctx.QueryArray("genre")
	yearFilter := ctx.DefaultQuery("year", "")
	sortBy := ctx.DefaultQuery("sort_by", "rating")
	ascending := ctx.DefaultQuery("ascending", "true") == "true"

	books, err := s.Storage.GetBooksWithFilters(searchQuery, genres, yearFilter, sortBy, ascending)
	if err != nil {
		if errors.Is(err, storerrros.ErrEmptyBooksList) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, books)
}

// получение списка всех книг из хранилища и возврат их клиенту в формате JSON.
func (s *Server) AllBooks(ctx *gin.Context) {

	books, err := s.Storage.GetBooks() //достает список книг из бд
	if err != nil {                    //если произошла ошибка
		if errors.Is(err, storerrros.ErrEmptyBooksList) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, books)
}

func (s *Server) BookInfo(ctx *gin.Context) {
	id := ctx.Param("id")
	book, err := s.Storage.GetBook(id)
	if err != nil {
		if errors.Is(err, storerrros.ErrBookNoExist) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, book)
}

func (s *Server) AddBook(ctx *gin.Context) {
	log := logger.Get()

	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
		return
	}

	if err := ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	form := ctx.Request.MultipartForm
	requiredFields := []string{"lable", "author", "desc", "genre", "age"}
	for _, field := range requiredFields {
		if len(form.Value[field]) == 0 || form.Value[field][0] == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing or empty field: " + field})
			return
		}
	}

	age, err := strconv.Atoi(form.Value["age"][0])
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid age value"})
		return
	}

	book := models.Book{
		Lable:  form.Value["lable"][0],
		Author: form.Value["author"][0],
		Desc:   form.Value["desc"][0],
		Genre:  form.Value["genre"][0],
		Age:    age,
	}

	age, _ = strconv.Atoi(form.Value["age"][0])
	book.Age = age

	if val, ok := form.Value["rating"]; ok && len(val) > 0 && val[0] != "" {
		rating, err := strconv.Atoi(val[0])
		if err != nil {
			log.Error().Err(err).Msg("invalid rating value")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid rating value"})
			return
		}
		book.Rating = rating
	} else {
		book.Rating = 0
	}

	coverFile, coverHeader, err := ctx.Request.FormFile("cover")
	if err != nil {
		log.Error().Err(err).Msg("failed to get cover file")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to get cover file"})
		return
	}
	defer coverFile.Close()

	coverPath := "uploads/covers/" + uuid.New().String() + "_" + coverHeader.Filename
	if err := os.MkdirAll(filepath.Dir(coverPath), os.ModePerm); err != nil {
		log.Error().Err(err).Msg("failed to create cover directory")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cover directory"})
		return
	}

	out, err := os.Create(coverPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create cover file")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cover file"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, coverFile); err != nil {
		log.Error().Err(err).Msg("failed to write cover file")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write cover file"})
		return
	}
	book.CoverURL = "/" + coverPath

	pdfFile, pdfHeader, err := ctx.Request.FormFile("pdf")
	if err != nil {
		log.Error().Err(err).Msg("failed to get pdf file")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to get pdf file"})
		return
	}
	defer pdfFile.Close()

	pdfPath := "uploads/pdfs/" + uuid.New().String() + "_" + pdfHeader.Filename
	if err := os.MkdirAll(filepath.Dir(pdfPath), os.ModePerm); err != nil {
		log.Error().Err(err).Msg("failed to create pdf directory")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create pdf directory"})
		return
	}

	out, err = os.Create(pdfPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create pdf file")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create pdf file"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, pdfFile); err != nil {
		log.Error().Err(err).Msg("failed to write pdf file")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write pdf file"})
		return
	}
	book.PDFURL = "/" + pdfPath

	if err := s.Storage.SaveBook(book); err != nil {
		log.Error().Err(err).Msg("save book failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.String(http.StatusOK, "book %s %s was added", book.Author, book.Lable)
}

func (s *Server) RemoveBook(ctx *gin.Context) {
	log := logger.Get()

	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing book ID"})
		return
	}

	if err := s.Storage.DeleteBook(id); err != nil {
		if err.Error() == "book not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete book")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete book"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "book deleted"})
}
