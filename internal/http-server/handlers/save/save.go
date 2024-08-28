package save

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"url-shorter/internal/lib/api/response"
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
	Name string `json:"name" validate:"required"`
	Hash string `json:"hash" validate:"required"`
	Text string `json:"text" validate:"required"`
}

type ErrorResponse struct {
	Message string       `json:"message"`
	Errors  []SpellError `json:"errors"`
}

type SaveNoteI interface {
	SaveNote(t *task.Task) error
}

func New(log *slog.Logger, noteSaver SaveNoteI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.save.New"
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
			Text:   req.Text,                      // Название задачи берем из текста в запросе
			UserID: getUserID(req.Name, req.Hash), // Получение UserID на основе имени и пароля
		}

		if task.UserID == 0 {
			log.Error("user not found")
			http.Error(w, "user not found", http.StatusNotFound) // Код 404 Not Found
			return
		}

		// Валидация данных задачи
		spellErrors, err := checkSpelling(task.Text)
		if err != nil {
			log.Error("spell check failed", sl.Err(err))
			http.Error(w, "spell check failed", http.StatusInternalServerError) // Код 500 Internal Server Error
			return
		}

		// Если найдены ошибки орфографии, вернуть их пользователю
		if len(spellErrors) > 0 {
			log.Warn("spelling errors found", slog.Any("errors", spellErrors))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, ErrorResponse{
				Message: "Invalid data format: spelling errors found",
				Errors:  spellErrors,
			})
			// Код 400 Bad Request
			return
		}

		// Сохранение заметки
		err = noteSaver.SaveNote(task)
		if err != nil {
			log.Error("failed to save note", sl.Err(err))
			http.Error(w, "failed to save note", http.StatusInternalServerError) // Код 500 Internal Server Error
			return
		}

		// Успешное завершение
		log.Info("Note added successfully")
		render.JSON(w, r, response.OK())
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

type SpellError struct {
	Code int      `json:"code"`
	Pos  int      `json:"pos"`
	Row  int      `json:"row"`
	Col  int      `json:"col"`
	Len  int      `json:"len"`
	Word string   `json:"word"`
	S    []string `json:"s"`
}

// Функция для проверки орфографии текста через Яндекс Спеллер
func checkSpelling(text string) ([]SpellError, error) {
	spellURL := "https://speller.yandex.net/services/spellservice.json/checkText"

	data := url.Values{}
	data.Set("text", text)
	data.Set("lang", "ru,en") // Укажите нужные языки
	data.Set("options", "0")  // Опции проверки, 0 означает использовать стандартные настройки

	resp, err := http.Post(spellURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("ошибка при отправке запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа: %w", err)
	}

	var spellErrors []SpellError
	err = json.Unmarshal(body, &spellErrors)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return spellErrors, nil
}
