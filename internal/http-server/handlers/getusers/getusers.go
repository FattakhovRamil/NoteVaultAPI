package getusers

import (
	"fmt"
	"log/slog"
	"net/http"
	"url-shorter/internal/lib/logger/sl"
	"url-shorter/models/task"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type User struct {
	name string
	role string
	hash string // пока просто так оставим, заменим потом на реальное хэширование
}

type Request struct {
	Name          string `json:"name" validate:"required"`
	Hash          string `json:"hash" validate:"required"`
	GetUserNotsId int    `json:"getusernotsid" validate:"required"`
}

type ErrorResponse struct {
	Message string       `json:"message"`
	Errors  []SpellError `json:"errors"`
}

type NotesResponse struct {
	Status  string       `json:"status"`
	Tasks   []*task.Task `json:"tasks,omitempty"`
	Message string       `json:"message,omitempty"`
}

type GetNotesI interface {
	GetNotes(id int) ([]*task.Task, error)
}

func New(log *slog.Logger, noteGetter GetNotesI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.get.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
		var req Request

		// Декодирование тела запроса
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			http.Error(w, "failed to decode request", http.StatusBadRequest) // Код 400 Bad Request
			return
		}
		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			// Создание списка сообщений об ошибках
			var errorMessages []string
			for _, fieldError := range validateErr {
				errorMessage := fmt.Sprintf("Field '%s' is required", fieldError.Field())
				errorMessages = append(errorMessages, errorMessage)
			}

			// Логирование ошибки
			log.Error("invalid request", sl.Err(err))

			// Формирование и отправка ответа с ошибками
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, map[string]interface{}{
				"status":  "error",
				"message": "validation failed",
				"errors":  errorMessages,
			})

			return
		}

		// Создание задачи
		task := &task.Task{
			UserID: getUserID(req.Name, req.Hash), // Получение UserID на основе имени и пароля
		}

		if getUserRole(req.Name, req.Hash) != "admin" {
			log.Error("There is no access")
			http.Error(w, "There is no access", http.StatusForbidden) // Код 404 Not Found
			return
		}

		if req.GetUserNotsId == 0 {
			log.Error("user not found")
			http.Error(w, "user not found", http.StatusNotFound) // Код 404 Not Found
			return
		}

		if task.UserID == 0 {
			log.Error("user not found")
			http.Error(w, "user not found", http.StatusNotFound) // Код 404 Not Found
			return
		}

		// Получение списка задач
		tasks, err := noteGetter.GetNotes(req.GetUserNotsId)
		if err != nil {
			log.Error("failed to get notes", sl.Err(err))
			http.Error(w, "failed to get notes", http.StatusInternalServerError) // Код 500 Internal Server Error
			return
		}

		// Успешное завершение
		response := NotesResponse{
			Status: "success",
			Tasks:  tasks,
		}

		render.JSON(w, r, response)
		log.Info("Notes sent successfully")
	}
}

func getUserID(name, hash string) int {

	var users = map[int]User{
		1: {name: "Alice", role: "admin", hash: "password123"},
		2: {name: "Bob", role: "user", hash: "qwerty456"},
		3: {name: "Charlie", role: "user", hash: "letmein789"},
		4: {name: "Dave", role: "admin", hash: "adminpass"},
		5: {name: "Eve", role: "user", hash: "evepass321"},
	}

	for id, user := range users { // предполагается, что users - это ваша карта пользователей
		if user.name == name && user.hash == hash {
			return id
		}
	}
	return 0 // возвращаем 0, если пользователь не найден
}

func getUserRole(name, hash string) string {
	var users = map[int]User{
		1: {name: "Alice", role: "admin", hash: "password123"},
		2: {name: "Bob", role: "user", hash: "qwerty456"},
		3: {name: "Charlie", role: "user", hash: "letmein789"},
		4: {name: "Dave", role: "admin", hash: "adminpass"},
		5: {name: "Eve", role: "user", hash: "evepass321"},
	}

	for _, user := range users { // предполагается, что users - это ваша карта пользователей
		if user.name == name && user.hash == hash {
			return user.role
		}
	}
	return "" // возвращаем 0, если пользователь не найден
}

type SpellError struct {
	Code int      `json:"code"`
	Pos  int      `json:"pos"`
	Row  int      `json:"row"`
	Col  int      `json:"col"`
	Len  int      `json:"len"`
	Word string   `json:"word"`
	S    []string `json:"s"`
}
