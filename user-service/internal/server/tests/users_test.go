package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azaliaz/bookly/user-service/internal/config"
	"github.com/azaliaz/bookly/user-service/internal/domain/models"
	"github.com/azaliaz/bookly/user-service/internal/server"
	"github.com/azaliaz/bookly/user-service/internal/server/mocks"
	storerrros "github.com/azaliaz/bookly/user-service/internal/storage/errors"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupRouter(s *server.Server) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/register", s.Register)
	r.POST("/login", s.Login)
	r.GET("/users/info", s.UserInfo)
	return r
}

func TestRegister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)

	cfg := config.Config{
		Addr: ":8080",
	}
	s := server.New(cfg, mockStorage)

	mockStorage.EXPECT().
		SaveUser(gomock.Any(), gomock.Any()).
		Return("some_id", nil).
		Times(1)

	mockStorage.EXPECT().
		GetUser("some_id").
		Return(models.User{
			UID: "some_id",
		}, nil).
		Times(1)

	body := `{"username":"testuser","password":"testpass"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	s.Register(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", w.Code)
	}

}

func TestRegister_BadRequest(t *testing.T) {
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, nil)

	router := setupRouter(s)
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "incorrectly entered data", w.Body.String())
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, mockStorage)

	reqBody := `{
		"email":"exists@example.com",
		"pass":"password123",
		"age":20,
		"role":"",
		"adminKey":""
	}`
	mockStorage.EXPECT().SaveUser(gomock.Any(), "").Return("", storerrros.ErrUserNoExist)

	router := setupRouter(s)
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "User already exists", w.Body.String())
}

func TestLogin_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, mockStorage)

	reqBody := `{
		"email":"test@example.com",
		"pass":"password123",
		"age":20,
		"role":"",
		"adminKey":""
	}`
	mockStorage.EXPECT().ValidUser(gomock.Any()).Return("user-uuid-1", nil)
	mockStorage.EXPECT().GetUser("user-uuid-1").Return(models.User{
		UID:    "user-uuid-1",
		Email:  "test@example.com",
		Role:   "user",
		CartID: "cart-uuid",
	}, nil)

	router := setupRouter(s)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	token := w.Body.String()
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, w.Header().Get("Authorization"))
}

func TestLogin_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, mockStorage)

	reqBody := `{
		"email":"notfound@example.com",
		"pass":"password123",
		"age":20,
		"role":"",
		"adminKey":""
	}`
	mockStorage.EXPECT().ValidUser(gomock.Any()).Return("", storerrros.ErrUserNoExist)

	router := setupRouter(s)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserInfo_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, mockStorage)

	mockStorage.EXPECT().GetUser("user-uuid-1").Return(models.User{
		UID:   "user-uuid-1",
		Email: "test@example.com",
		Role:  "user",
	}, nil)

	router := gin.New()
	router.GET("/users/info", func(c *gin.Context) {
		c.Set("uid", "user-uuid-1")
		s.UserInfo(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test@example.com")
}

func TestUserInfo_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockStorage(ctrl)
	cfg := config.Config{
		Addr: ":8080",
	}

	s := server.New(cfg, mockStorage)

	mockStorage.EXPECT().GetUser("nonexistent-uuid").Return(models.User{}, storerrros.ErrUserNotFound)

	router := gin.New()
	router.GET("/users/info", func(c *gin.Context) {
		c.Set("uid", "nonexistent-uuid")
		s.UserInfo(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), storerrros.ErrUserNotFound.Error())
}
