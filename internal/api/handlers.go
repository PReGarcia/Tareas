package api

import (
	"taskmanager/internal/db"
	"taskmanager/internal/models"

	"github.com/gofiber/fiber/v2"
)

// Register monta todas las rutas de la API.
func Register(app *fiber.App) {
	api := app.Group("/api")

	api.Get("/projects", listProjects)
	api.Post("/projects", createProject)
	api.Patch("/projects/:id", updateProject)
	api.Delete("/projects/:id", deleteProject)
	api.Get("/projects/:id/boards", listBoards)
	api.Post("/projects/:id/boards", createBoard)

	api.Get("/boards/:id", getBoardView)
	api.Patch("/boards/:id", updateBoard)
	api.Delete("/boards/:id", deleteBoard)
	api.Post("/boards/:id/statuses", createStatus)
	api.Post("/boards/:id/tasks", createTask)

	api.Patch("/tasks/:id", updateTask)
	api.Delete("/tasks/:id", deleteTask)

	api.Get("/tasks", listTasksGlobal)
	api.Get("/tags", listTags)
	api.Get("/statuses", listStatuses)
}

func listProjects(c *fiber.Ctx) error {
	projects, err := db.ListProjects()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(projects)
}

func createProject(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name requerido")
	}
	p, err := db.CreateProject(body.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func updateProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name requerido")
	}
	if err := db.RenameProject(int64(id), body.Name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	p, _ := db.GetProject(int64(id))
	return c.JSON(p)
}

func deleteProject(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := db.DeleteProject(int64(id)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func listBoards(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	boards, err := db.ListBoards(int64(id))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(boards)
}

func updateBoard(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name requerido")
	}
	if err := db.RenameBoard(int64(id), body.Name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	b, _ := db.GetBoard(int64(id))
	return c.JSON(b)
}

func deleteBoard(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := db.DeleteBoard(int64(id)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func createBoard(c *fiber.Ctx) error {
	pid, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name requerido")
	}
	b, err := db.CreateBoard(int64(pid), body.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(b)
}

func getBoardView(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	view, err := db.GetBoardView(int64(id))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if view == nil {
		return fiber.NewError(fiber.StatusNotFound, "tablero no encontrado")
	}
	return c.JSON(view)
}

func createStatus(c *fiber.Ctx) error {
	bid, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name requerido")
	}
	if body.Color == "" {
		body.Color = "#3b82f6"
	}
	s, err := db.CreateStatus(int64(bid), body.Name, body.Color)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(s)
}

func createTask(c *fiber.Ctx) error {
	bid, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		StatusID    int64    `json:"status_id"`
		Priority    string   `json:"priority"`
		DueDate     *string  `json:"due_date"`
		Tags        []string `json:"tags"`
	}
	if err := c.BodyParser(&body); err != nil || body.Title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "title requerido")
	}
	if body.StatusID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "status_id requerido")
	}
	if body.Priority == "" {
		body.Priority = "none"
	}
	t, err := db.CreateTask(int64(bid), body.StatusID, body.Title, body.Description, body.Priority, body.DueDate, body.Tags)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

func updateTask(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json inválido")
	}

	// Si cambia de columna, aplica el reordenamiento fino (status_id + position).
	if vStatus, ok := body["status_id"]; ok {
		statusID := toInt64(vStatus)
		newPos := 0
		if vPos, ok2 := body["position"]; ok2 {
			newPos = toInt(vPos)
		}
		if err := db.MoveTask(int64(id), statusID, newPos); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	// Aplica el resto de campos editables en la misma petición.
	fields := map[string]interface{}{}
	allowed := map[string]string{
		"title":       "title",
		"description": "description",
		"priority":    "priority",
		"due_date":    "due_date",
		"position":    "position",
		"tags":        "tags",
	}
	for k, col := range allowed {
		v, ok := body[k]
		if !ok {
			continue
		}
		if k == "tags" {
			// El body trae las etiquetas como array JSON; se serializa a texto.
			arr, ok := v.([]interface{})
			if !ok {
				continue
			}
			tags := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok && s != "" {
					tags = append(tags, s)
				}
			}
			fields[col] = db.MarshalTags(tags)
		} else {
			fields[col] = v
		}
	}
	if len(fields) > 0 {
		if _, err := db.UpdateTask(int64(id), fields); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	t, _ := db.GetTask(int64(id))
	if t == nil {
		return fiber.NewError(fiber.StatusNotFound, "tarea no encontrada")
	}
	return c.JSON(t)
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func deleteTask(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := db.DeleteTask(int64(id)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func listTasksGlobal(c *fiber.Ctx) error {
	f := models.TaskFilters{
		Status:   c.Query("status"),
		Priority: c.Query("priority"),
		Project:  c.Query("project"),
		Tag:      c.Query("tag"),
		From:     c.Query("from"),
		To:       c.Query("to"),
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
	}
	rows, err := db.ListTasksGlobal(f)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(rows)
}

func listTags(c *fiber.Ctx) error {
	tags, err := db.ListAllTags()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(tags)
}

func listStatuses(c *fiber.Ctx) error {
	statuses, err := db.ListAllStatuses()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(statuses)
}
