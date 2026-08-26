package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"taskmanager/internal/models"
)

// ---- Projects ----

func CreateProject(name string) (*models.Project, error) {
	res, err := DB.Exec(`INSERT INTO projects(name) VALUES(?)`, name)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetProject(id)
}

func GetProject(id int64) (*models.Project, error) {
	var p models.Project
	err := DB.QueryRow(`SELECT id, name, created_at FROM projects WHERE id=?`, id).
		Scan(&p.ID, &p.Name, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func ListProjects() ([]models.Project, error) {
	rows, err := DB.Query(`SELECT id, name, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Project{}
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// ---- Boards ----

func CreateBoard(projectID int64, name string) (*models.Board, error) {
	res, err := DB.Exec(`INSERT INTO boards(project_id, name) VALUES(?,?)`, projectID, name)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	// Estados por defecto del Kanban.
	defaults := []struct {
		name  string
		color string
	}{
		{"Backlog", "#6b7280"},
		{"To Do", "#3b82f6"},
		{"In Progress", "#f59e0b"},
		{"Done", "#22c55e"},
	}
	for i, d := range defaults {
		if _, err := DB.Exec(`INSERT INTO statuses(board_id, name, position, color) VALUES(?,?,?,?)`,
			id, d.name, i, d.color); err != nil {
			return nil, err
		}
	}
	return GetBoard(id)
}

func GetBoard(id int64) (*models.Board, error) {
	var b models.Board
	err := DB.QueryRow(`SELECT id, project_id, name, created_at FROM boards WHERE id=?`, id).
		Scan(&b.ID, &b.ProjectID, &b.Name, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func ListBoards(projectID int64) ([]models.Board, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, created_at FROM boards WHERE project_id=? ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Board{}
	for rows.Next() {
		var b models.Board
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.Name, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// ---- Statuses ----

func CreateStatus(boardID int64, name, color string) (*models.Status, error) {
	var pos int
	DB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM statuses WHERE board_id=?`, boardID).Scan(&pos)
	pos++
	res, err := DB.Exec(`INSERT INTO statuses(board_id, name, position, color) VALUES(?,?,?,?)`,
		boardID, name, pos, color)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	var s models.Status
	err = DB.QueryRow(`SELECT id, board_id, name, position, color FROM statuses WHERE id=?`, id).
		Scan(&s.ID, &s.BoardID, &s.Name, &s.Position, &s.Color)
	return &s, err
}

// ---- Tasks ----

func CreateTask(boardID, statusID int64, title, description, priority string, dueDate *string) (*models.Task, error) {
	var pos int
	DB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM tasks WHERE status_id=?`, statusID).Scan(&pos)
	pos++
	var dd interface{}
	if dueDate != nil {
		dd = *dueDate
	}
	res, err := DB.Exec(`INSERT INTO tasks(board_id, status_id, title, description, priority, due_date, position)
		VALUES(?,?,?,?,?,?,?)`, boardID, statusID, title, description, priority, dd, pos)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetTask(id)
}

func GetTask(id int64) (*models.Task, error) {
	var t models.Task
	var desc, dd sql.NullString
	err := DB.QueryRow(`SELECT id, board_id, title, description, status_id, priority, due_date, position, created_at, updated_at
		FROM tasks WHERE id=?`, id).
		Scan(&t.ID, &t.BoardID, &t.Title, &desc, &t.StatusID, &t.Priority, &dd, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Description = desc.String
	if dd.Valid {
		t.DueDate = &dd.String
	}
	return &t, nil
}

// UpdateTask aplica solo los campos no-nulos del mapa (movimiento o edición).
func UpdateTask(id int64, fields map[string]interface{}) (*models.Task, error) {
	if len(fields) == 0 {
		return GetTask(id)
	}
	set := []string{}
	args := []interface{}{}
	for col, val := range fields {
		set = append(set, col+" = ?")
		args = append(args, val)
	}
	set = append(set, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	q := fmt.Sprintf(`UPDATE tasks SET %s WHERE id = ?`, strings.Join(set, ", "))
	if _, err := DB.Exec(q, args...); err != nil {
		return nil, err
	}
	return GetTask(id)
}

func DeleteTask(id int64) error {
	_, err := DB.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return err
}

// MoveTask reordena una tarea aplicando "orden fino": inserta la tarea en el
// índice newPos de la columna destino y desplaza las demás para abrir hueco,
// manteniendo el orden exacto y persistente (sin empates de position).
func MoveTask(taskID, statusID int64, newPos int) error {
	if newPos < 0 {
		newPos = 0
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldStatus int64
	var oldPos int
	if err := tx.QueryRow(`SELECT status_id, position FROM tasks WHERE id=?`, taskID).
		Scan(&oldStatus, &oldPos); err != nil {
		return err
	}

	if oldStatus == statusID {
		// Mismo columna: reordenar.
		if newPos == oldPos {
			return nil
		}
		if newPos < oldPos {
			// Desplaza [newPos, oldPos-1] +1.
			if _, err := tx.Exec(`UPDATE tasks SET position = position + 1
				WHERE status_id=? AND position >= ? AND position < ?`,
				statusID, newPos, oldPos); err != nil {
				return err
			}
		} else {
			// Desplaza (oldPos, newPos] -1.
			if _, err := tx.Exec(`UPDATE tasks SET position = position - 1
				WHERE status_id=? AND position > ? AND position <= ?`,
				statusID, oldPos, newPos); err != nil {
				return err
			}
		}
	} else {
		// Cambio de columna: cierra hueco en la origen y abre en la destino.
		if _, err := tx.Exec(`UPDATE tasks SET position = position - 1
			WHERE status_id=? AND position > ?`, oldStatus, oldPos); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE tasks SET position = position + 1
			WHERE status_id=? AND position >= ?`, statusID, newPos); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`UPDATE tasks SET status_id=?, position=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`, statusID, newPos, taskID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---- Proyectos: eliminar / renombrar ----

func DeleteProject(id int64) error {
	_, err := DB.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

func RenameProject(id int64, name string) error {
	_, err := DB.Exec(`UPDATE projects SET name=? WHERE id=?`, name, id)
	return err
}

// ---- Tableros: eliminar / renombrar ----

func DeleteBoard(id int64) error {
	_, err := DB.Exec(`DELETE FROM boards WHERE id=?`, id)
	return err
}

func RenameBoard(id int64, name string) error {
	_, err := DB.Exec(`UPDATE boards SET name=? WHERE id=?`, name, id)
	return err
}

// ---- Vistas ----

func GetBoardView(boardID int64) (*models.BoardView, error) {
	board, err := GetBoard(boardID)
	if err != nil || board == nil {
		return nil, err
	}

	// Leer statuses primero y cerrar el result set antes de consultar tareas,
	// para no mantener dos *sql.Rows abiertos con MaxOpenConns(1).
	statusRows, err := DB.Query(`SELECT id, board_id, name, position, color FROM statuses WHERE board_id=? ORDER BY position`, boardID)
	if err != nil {
		return nil, err
	}
	statuses := []models.Status{}
	for statusRows.Next() {
		var s models.Status
		if err := statusRows.Scan(&s.ID, &s.BoardID, &s.Name, &s.Position, &s.Color); err != nil {
			statusRows.Close()
			return nil, err
		}
		statuses = append(statuses, s)
	}
	statusRows.Close()

	taskRows, err := DB.Query(`SELECT id, board_id, title, description, status_id, priority, due_date, position, created_at, updated_at
		FROM tasks WHERE board_id=? ORDER BY position`, boardID)
	if err != nil {
		return nil, err
	}
	defer taskRows.Close()

	tasksByStatus := map[int64][]models.Task{}
	for taskRows.Next() {
		var t models.Task
		var desc, dd sql.NullString
		if err := taskRows.Scan(&t.ID, &t.BoardID, &t.Title, &desc, &t.StatusID, &t.Priority, &dd, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Description = desc.String
		if dd.Valid {
			t.DueDate = &dd.String
		}
		tasksByStatus[t.StatusID] = append(tasksByStatus[t.StatusID], t)
	}

	view := &models.BoardView{Board: *board, Columns: []models.ColumnView{}}
	for _, s := range statuses {
		tasks := tasksByStatus[s.ID]
		if tasks == nil {
			tasks = []models.Task{}
		}
		view.Columns = append(view.Columns, models.ColumnView{
			Status: s,
			Tasks:  tasks,
		})
	}
	return view, nil
}

// ListTasksGlobal devuelve todas las tareas enriquecidas con filtros y orden.
func ListTasksGlobal(f models.TaskFilters) ([]models.TaskRow, error) {
	where := []string{}
	args := []interface{}{}
	if f.Status != "" {
		where = append(where, "st.name = ?")
		args = append(args, f.Status)
	}
	if f.Priority != "" {
		where = append(where, "t.priority = ?")
		args = append(args, f.Priority)
	}
	if f.Project != "" {
		where = append(where, "p.name = ?")
		args = append(args, f.Project)
	}
	if f.From != "" {
		where = append(where, "t.due_date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "t.due_date <= ?")
		args = append(args, f.To)
	}

	q := `SELECT t.id, t.board_id, t.title, t.description, t.status_id, t.priority, t.due_date,
		t.position, t.created_at, t.updated_at, p.name, b.name, st.name
		FROM tasks t
		JOIN boards b ON b.id = t.board_id
		JOIN projects p ON p.id = b.project_id
		JOIN statuses st ON st.id = t.status_id`

	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}

	allowedSort := map[string]string{
		"title":    "t.title",
		"priority": "t.priority",
		"due_date": "t.due_date",
		"status":   "st.name",
		"project":  "p.name",
		"created":  "t.created_at",
	}
	sortCol := allowedSort[f.Sort]
	if sortCol == "" {
		sortCol = "t.created_at"
	}
	order := "DESC"
	if strings.EqualFold(f.Order, "asc") {
		order = "ASC"
	}
	q += fmt.Sprintf(" ORDER BY %s %s", sortCol, order)

	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.TaskRow{}
	for rows.Next() {
		var r models.TaskRow
		var desc, dd sql.NullString
		if err := rows.Scan(&r.ID, &r.BoardID, &r.Title, &desc, &r.StatusID, &r.Priority, &dd,
			&r.Position, &r.CreatedAt, &r.UpdatedAt, &r.ProjectName, &r.BoardName, &r.StatusName); err != nil {
			return nil, err
		}
		r.Description = desc.String
		if dd.Valid {
			r.DueDate = &dd.String
		}
		out = append(out, r)
	}
	return out, nil
}
