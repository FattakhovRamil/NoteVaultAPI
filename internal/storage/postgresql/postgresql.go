package postgresql

import (
	"database/sql"
	"fmt"
	"os"
	task "url-shorter/models/task"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB // поле конекта к бд
}

func New(storagePath string) (*Storage, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	const op = "storage.postgresql.New"

	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", dbHost, dbPort, dbUser, dbName, dbPassword)
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}
	err = db.Ping()

	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	createTableQuery := `
    CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		text VARCHAR(255) NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		user_id INT NOT NULL
    )`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		db: db,
	}, nil

}

func (s *Storage) SaveNote(task *task.Task) error {
	const op = "storage.postgresql.SaveNote"

	query := "INSERT INTO tasks (text, user_id) VALUES ($1, $2) RETURNING id"
	err := s.db.QueryRow(query, task.Text, task.UserID).Scan(&task.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) GetNotes(id int) ([]*task.Task, error) {
	const op = "storage.postgresql.GetNotes"

	tasks := []*task.Task{}
	query := `SELECT id, text, user_id FROM tasks WHERE user_id = $1`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	for rows.Next() {
		task := &task.Task{}
		err = rows.Scan(&task.ID, &task.Text, &task.UserID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return tasks, nil

}

func (s *Storage) Close() error {
	return s.db.Close()
}
