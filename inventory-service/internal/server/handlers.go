package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/azaliaz/bookly/book-service/internal/domain/models"
	"github.com/azaliaz/bookly/book-service/internal/logger"
	storerrros "github.com/azaliaz/bookly/book-service/internal/storage/errors"
)

//получение списка всех книг из хранилища и возврат их клиенту в формате JSON.
func (s *Server) allBooks(ctx *gin.Context) {
	log := logger.Get()
	//проверка наличия идентификатора пользователя
	_, exist := ctx.Get("uid") //проверяет есть ли uid в контексте запроса
	if !exist { //если не найден
		log.Error().Msg("user ID not found") //логируется ошиька
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"}) 
		//ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	//получение списка книг
	books, err := s.storage.GetBooks() //достает список книг из бд
	if err != nil { //если произошла ошибка
		if errors.Is(err, storerrros.ErrEmptyBooksList) {
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//возвращение списка книг
	ctx.JSON(http.StatusOK, books)
}
//получение инфорамции о конкретной книге
func (s *Server) bookInfo(ctx *gin.Context) {
	log := logger.Get()
	//проверка налия uid
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
		return
	}
	id := ctx.Param("id") //получение id книги из параметров запроса
	book, err := s.storage.GetBook(id)//получение книги из хранилища
	if err != nil {
		if errors.Is(err, storerrros.ErrBookNoExist) { //Если книги нет
			ctx.String(http.StatusNotFound, err.Error())
			return
		}//Иначе
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusFound, book)
}

//обработчик API позволяет пользователю добавить новую книгу в хранилище.
func (s *Server) addBook(ctx *gin.Context) {
	log := logger.Get()

	//проверка налия идентификатора пользователя
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
		return
	}

	//извлечение данных о книге из тела запроса 
	var book models.Book
	if err := ctx.ShouldBindBodyWithJSON(&book); err != nil {
		log.Error().Err(err).Msg("unmarshal body failed")
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	//установка начального количества книг
	if book.Count == 0 {
		book.Count = 1
	}
	//сохранение книги
	if err := s.storage.SaveBook(book); err != nil {
		log.Error().Err(err).Msg("save user failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.String(http.StatusOK, "book %s %s was added", book.Author, book.Lable)
}

//обработчик API позволяет пользователю добавить несколько книг в хранилище
func (s *Server) addBooks(ctx *gin.Context) {
	log := logger.Get()
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
		return
	}
	var books []models.Book
	if err := ctx.ShouldBindBodyWithJSON(&books); err != nil {
		log.Error().Err(err).Msg("unmarshal body failed")
		ctx.String(http.StatusBadRequest, "incorrectly entered data")
		return
	}
	if err := s.storage.SaveBooks(books); err != nil {
		log.Error().Err(err).Msg("save user failed")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.String(http.StatusOK, "%s books was added", len(books))
}

// func (s *Server) removeBook(ctx *gin.Context) {
// 	log := logger.Get()
// 	_, exist := ctx.Get("uid")
// 	if !exist {
// 		log.Error().Msg("user ID not found")
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
// 		return
// 	}
// 	id := ctx.Param("id")
// 	if err := s.storage.SetDeleteStatus(id); err != nil {
// 		log.Error().Msg("user ID not found")
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found"})
// 		return
// 	}
// 	s.delChan <- struct{}{}
// 	log.Debug().Int("chan len", len(s.delChan)).Msg("book send into delChan")
// 	ctx.String(http.StatusOK, "book "+id+" was deleted")
// }
func (s *Server) removeBook(ctx *gin.Context) {
	log := logger.Get()

	// Проверка наличия uid
	_, exist := ctx.Get("uid")
	if !exist {
		log.Error().Msg("user ID not found")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	// Получаем идентификатор книги из параметра запроса
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing book ID"})
		return
	}

	// Вызываем метод удаления книги
	if err := s.storage.DeleteBook(id); err != nil {
		if err.Error() == "book not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "book not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete book")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete book"})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusOK, gin.H{"message": "book deleted"})
}

//работает в фоне и выполняет удаление книг, когда канал s.delChan заполняется.
func (s *Server) deleter(ctx context.Context) {
	log := logger.Get()

	//в конце работы выводит
	defer log.Debug().Msg("deleter was ended")
	for { 
		//проверка на заверщшение контекста
		select {
		case <-ctx.Done(): //если завершен то логируем
			log.Debug().Msg("deleter context done")
			return //выходим
		default:
			if len(s.delChan) == cap(s.delChan) { //проверяем заполенен ли канал
				log.Debug().Int("cap", cap(s.delChan)).Int("len", cap(s.delChan)).Msg("start deleting") //логируем начало удаления
				for i := 0; i < cap(s.delChan); i++ { //очищаем канал чтоб подготовиться к следующему удалению
					<-s.delChan
				}
				if err := s.storage.DeleteBooks(); err != nil { //вызываем удаления книг из хранилища
					log.Error().Err(err).Msg("deleting books failed") //обработка ошибок
					s.ErrChan <- err
					return
				}
			}
		}
	}
}


func (s *Server) bookReturn(_ *gin.Context) {}
