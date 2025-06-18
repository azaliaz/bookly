package tests

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/server"
	"github.com/azaliaz/bookly/book-service/internal/server/mocks"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.ReleaseMode) // Добавляем в init()
}
func TestServer_allBooks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	s := &server.Server{Storage: mockStorage}

	t.Run("success", func(t *testing.T) {
		books := []models.Book{{Lable: "Book1"}, {Lable: "Book2"}}
		mockStorage.EXPECT().GetBooks().Return(books, nil)

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		s.AllBooks(ctx)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Book1")
		assert.Contains(t, w.Body.String(), "Book2")
	})

	t.Run("empty list error", func(t *testing.T) {
		mockStorage.EXPECT().GetBooks().Return(nil, storerrros.ErrEmptyBooksList)

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		s.AllBooks(ctx)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), storerrros.ErrEmptyBooksList.Error())
	})

	t.Run("internal error", func(t *testing.T) {
		mockStorage.EXPECT().GetBooks().Return(nil, errors.New("db error"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		s.AllBooks(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "db error")
	})
}

func TestServer_bookInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	s := &server.Server{Storage: mockStorage}

	t.Run("success", func(t *testing.T) {
		book := models.Book{Lable: "Book1"}
		mockStorage.EXPECT().GetBook("123").Return(book, nil)

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Params = gin.Params{{Key: "id", Value: "123"}}

		s.BookInfo(ctx)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Book1")
	})

	t.Run("not found", func(t *testing.T) {
		mockStorage.EXPECT().GetBook("123").Return(models.Book{}, storerrros.ErrBookNoExist)

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Params = gin.Params{{Key: "id", Value: "123"}}

		s.BookInfo(ctx)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), storerrros.ErrBookNoExist.Error())
	})

	t.Run("internal error", func(t *testing.T) {
		mockStorage.EXPECT().GetBook("123").Return(models.Book{}, errors.New("db error"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Params = gin.Params{{Key: "id", Value: "123"}}

		s.BookInfo(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "db error")
	})
}

func TestServer_removeBook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	s := &server.Server{Storage: mockStorage}

	createCtxWithUID := func() (*gin.Context, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Set("uid", "user1")
		ctx.Params = gin.Params{{Key: "id", Value: "book123"}}
		return ctx, w
	}

	t.Run("success", func(t *testing.T) {
		mockStorage.EXPECT().DeleteBook("book123").Return(nil).Times(1) // ← Явно указать, что ожидается один вызов

		ctx, w := createCtxWithUID()
		s.RemoveBook(ctx)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "book deleted")
	})

	t.Run("missing uid", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Params = gin.Params{{Key: "id", Value: "book123"}}

		s.RemoveBook(ctx)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "User ID not found")
	})

	t.Run("missing id", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Set("uid", "user1")
		ctx.Params = gin.Params{} // no id

		s.RemoveBook(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing book ID")
	})

	t.Run("not found error", func(t *testing.T) {
		mockStorage.EXPECT().DeleteBook("book123").Return(errors.New("book not found"))

		ctx, w := createCtxWithUID()
		s.RemoveBook(ctx)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "book not found")
	})

	t.Run("internal error", func(t *testing.T) {
		mockStorage.EXPECT().DeleteBook("book123").Return(errors.New("some error"))

		ctx, w := createCtxWithUID()
		s.RemoveBook(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to delete book")
	})
}

func TestServer_addBook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	s := &server.Server{Storage: mockStorage}

	createMultipartRequest := func(t *testing.T, fields map[string]string, coverContent, pdfContent []byte) *http.Request {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		for field, value := range fields {
			err := writer.WriteField(field, value)
			assert.NoError(t, err)
		}

		if coverContent != nil {
			part, err := writer.CreateFormFile("cover", "cover.jpg")
			assert.NoError(t, err)
			_, err = part.Write(coverContent)
			assert.NoError(t, err)
		}

		if pdfContent != nil {
			part, err := writer.CreateFormFile("pdf", "book.pdf")
			assert.NoError(t, err)
			_, err = part.Write(pdfContent)
			assert.NoError(t, err)
		}

		err := writer.Close()
		assert.NoError(t, err)

		req, err := http.NewRequest("POST", "/add-book", body)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		return req
	}

	t.Run("success", func(t *testing.T) {
		mockStorage.EXPECT().SaveBook(gomock.Any()).Return(nil)

		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "book")
		assert.Contains(t, w.Body.String(), "was added")
	})

	t.Run("missing uid", func(t *testing.T) {
		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req

		s.AddBook(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "User ID not found")
	})

	t.Run("missing cover file", func(t *testing.T) {
		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, nil, []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "failed to get cover file")
	})

	t.Run("missing pdf file", func(t *testing.T) {
		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), nil)

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "failed to get pdf file")
	})

	t.Run("save book fails", func(t *testing.T) {
		mockStorage.EXPECT().SaveBook(gomock.Any()).Return(errors.New("save failed"))

		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "save failed")
	})

	t.Run("invalid rating value", func(t *testing.T) {
		fields := map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "invalid",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid rating value")
	})

	t.Run("missing lable value", func(t *testing.T) {
		fields := map[string]string{
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}

		req := createMultipartRequest(t, fields, []byte("fake cover"), []byte("fake pdf"))

		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Set("uid", "user1")

		s.AddBook(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing or empty field: lable")
	})
	t.Run("failed to create cover directory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStorage := mocks.NewMockStorage(ctrl)
		s := &server.Server{Storage: mockStorage}

		mockStorage.EXPECT().SaveBook(gomock.Any()).Return(nil)

		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = createMultipartRequest(t, map[string]string{
			"lable":  "Test Book",
			"author": "Test Author",
			"desc":   "Test Description",
			"genre":  "Test Genre",
			"age":    "10",
			"rating": "5",
		}, []byte("fake cover"), []byte("fake pdf"))

		ctx.Set("uid", "user1")
		s.AddBook(ctx)
	})

}
