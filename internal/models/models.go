package models

import "time"

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Board struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Status struct {
	ID       int64  `json:"id"`
	BoardID  int64  `json:"board_id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Color    string `json:"color"`
}

type Task struct {
	ID          int64     `json:"id"`
	BoardID     int64     `json:"board_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StatusID    int64     `json:"status_id"`
	Priority    string    `json:"priority"`
	DueDate     *string   `json:"due_date"`
	Tags        []string  `json:"tags"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ColumnView agrupa un status con sus tareas para la vista Kanban.
type ColumnView struct {
	Status Status `json:"status"`
	Tasks  []Task `json:"tasks"`
}

// BoardView es la respuesta completa de un tablero.
type BoardView struct {
	Board   Board        `json:"board"`
	Columns []ColumnView `json:"columns"`
}

// TaskRow es una tarea enriquecida para la Master Table global.
type TaskRow struct {
	Task
	ProjectName string `json:"project_name"`
	BoardName   string `json:"board_name"`
	StatusName  string `json:"status_name"`
}

// TaskFilters son los filtros combinables de la Master Table.
type TaskFilters struct {
	Status   string
	Priority string
	Project  string
	Board    string
	Tag      string
	From     string
	To       string
	Sort     string
	Order    string
}
