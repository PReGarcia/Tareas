# Plan — Task Manager (Self-hosted, low-RAM)

Aplicación web de gestión de tareas auto-alojada vía Docker con consumo de RAM
estrictamente mínimo (<50–100 MB). Clon visual del estilo **Twenty CRM**
(dark mode nativo, minimalista, tipografía Inter, bordes sutiles).

---

## Stack tecnológico

| Capa | Elección | Justificación (memoria) |
|------|----------|--------------------------|
| **Backend** | Go 1.22 + Fiber (fasthttp) | Binario estático único, arranque ~2–5 MB, sin runtime pesado. Fiber es de los frameworks más ligeros. |
| **DB** | SQLite vía `modernc.org/sqlite` (pure-Go, **sin CGO**) | Embebida en el mismo binario, cero contenedores extra. Evita overhead de Postgres/MySQL. |
| **Frontend** | HTMX + Alpine.js + Tailwind CSS (compilado en build-time) | Sin Virtual DOM ni SPA runtime en memoria. Tailwind se compila en build, no en runtime. |
| **DnD** | SortableJS (~15 KB, vanilla) | Drag & drop ligero para mover tarjetas entre columnas. |
| **Embedding** | `//go:embed` del `dist/` en el binario Go | Una sola imagen, un solo proceso, mínimo footprint. |
| **Docker** | Multistage: `node:alpine` (web) → `golang:alpine` (build) → `scratch` (runtime) | Imagen final ~10–15 MB en disco, ~20–40 MB RAM en ejecución. |

**Consumo estimado:** ~25–45 MB RAM en reposo. Por debajo del límite de 50–100 MB.

---

## Árbol de directorios

```
task-manager/
├── docker-compose.yml
├── Dockerfile
├── .dockerignore
├── go.mod
├── main.go
├── internal/
│   ├── db/
│   │   ├── db.go          # init + migraciones
│   │   └── queries.go     # acceso a datos
│   ├── models/
│   │   └── models.go      # Project, Board, Status, Task
│   └── api/
│       └── handlers.go    # endpoints REST Fiber
├── web/
│   ├── package.json
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── build.js           # copia index.html + Inter + js a dist/
│   ├── src/
│   │   ├── input.css      # @tailwind + tokens tema Twenty oscuro
│   │   ├── app.js         # Alpine + SortableJS + HTMX config
│   │   └── index.html     # layout, sidebar, áreas Kanban/Tabla/Master
│   └── public/
│       └── fonts/         # Inter woff2 (self-hosted)
└── data/                  # tasks.db (gitignored)
```

---

## Modelo de datos (SQLite)

- `projects(id, name, created_at)`
- `boards(id, project_id, name, created_at)`
- `statuses(id, board_id, name, position, color)` — columnas del Kanban
- `tasks(id, board_id, title, description, status_id, priority, due_date, position, created_at, updated_at)`

La **Master Table** global cruza `tasks → boards → projects`.

---

## API REST (Fiber)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET / POST | `/api/projects` | listar / crear proyecto |
| GET / POST | `/api/projects/:id/boards` | tableros de un proyecto |
| GET | `/api/boards/:id` | tablero + tareas agrupadas por status |
| POST | `/api/boards/:id/statuses` | crear columna (status) |
| POST | `/api/boards/:id/tasks` | crear tarea |
| PATCH | `/api/tasks/:id` | mover (status/position) o editar campos |
| DELETE | `/api/tasks/:id` | borrar tarea |
| GET | `/api/tasks?status=&priority=&project=&from=&to=&sort=` | Master Table global con filtros + orden |

---

## UI (estilo Twenty, dark nativo)

- **Tokens de tema:** fondo `#0f1115`, paneles `#16181d`, bordes `#262a31`,
  texto `#e6e8eb`, acento sutil (índigo/azul tenue). Fuente **Inter** self-hosted.
- **Sidebar:** proyectos → tableros (colapsable).
- **Vista Board (Kanban):** columnas por status, tarjetas drag & drop
  (SortableJS → `PATCH /api/tasks/:id`). Botón toggle **Kanban ↔ Tabla**.
- **Vista Tabla (por tablero):** mismo board en filas; cambio de status vía
  `<select>` (HTMX).
- **Master Table global:** todas las tareas, filtros combinables (status,
  prioridad, proyecto, fecha) + orden por columna.
- HTMX para navegación/interacciones sin recargas; Alpine para estado local
  (modales, toggles).

---

## Docker

- **Stage 1 (`node:alpine`):** `npm ci && npm run build` → genera `web/dist`.
- **Stage 2 (`golang:alpine`):** `CGO_ENABLED=0 go build` embebiendo `dist`.
- **Stage 3 (`scratch`):** binario + `/etc/ssl/certs/ca-certificates.crt`.
- `docker-compose.yml`: servicio `app`, `build: .`, volumen `./data:/data`,
  `restart: unless-stopped`, `mem_limit: 128m` (consumo real ~30–45 MB).

---

## Fases de entrega (código paso a paso)

1. **Backend + DB + API** (models, db, handlers, main)
2. **Frontend base** (Tailwind tema Twenty, layout, sidebar, Alpine/HTMX)
3. **Kanban + DnD + vista Tabla por board**
4. **Master Table + filtros/orden**
5. **Docker (Dockerfile + compose)** y verificación de RAM
