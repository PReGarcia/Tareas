package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB es la conexión global a SQLite.
var DB *sql.DB

// Init abre (o crea) la base de datos y aplica las migraciones.
// El path por defecto es ./data/tasks.db, configurable con DATABASE_PATH.
func Init() error {
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "data/tasks.db"
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("crear dir db: %w", err)
		}
	}

	// modernc.org/sqlite no requiere CGO.
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("abrir db: %w", err)
	}

	// Una sola conexión evita overhead de pool y es suficiente para SQLite.
	DB.SetMaxOpenConns(1)

	if err := migrate(); err != nil {
		return err
	}
	return nil
}

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS boards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS statuses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			color TEXT NOT NULL DEFAULT '#3b82f6'
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status_id INTEGER NOT NULL REFERENCES statuses(id) ON DELETE CASCADE,
			priority TEXT NOT NULL DEFAULT 'none',
			due_date TEXT,
			position INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_board ON tasks(board_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status_id)`,
		`CREATE INDEX IF NOT EXISTS idx_boards_project ON boards(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_statuses_board ON statuses(board_id)`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("migración: %w", err)
		}
	}
	return nil
}

// Close cierra la conexión.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
