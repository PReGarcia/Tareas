function app() {
  return {
    // Estado
    projects: [],
    boards: [],
    board: { board: {}, columns: [] },
    currentProjectId: null,
    currentBoardId: null,
    viewMode: "kanban",
    views: [
      { id: "kanban", label: "Kanban" },
      { id: "table", label: "Tabla" },
      { id: "master", label: "Global" },
    ],
    master: [],
    filters: { status: "", priority: "", project: "", from: "", to: "", sort: "", order: "desc" },

    // Modales
    showProjModal: false,
    showBoardModal: false,
    showTaskModal: false,
    showRenameModal: false,
    renameType: "",
    renameId: null,
    renameName: "",
    projName: "",
    boardName: "",
    taskTitle: "",
    taskDesc: "",
    taskStatus: null,
    taskPriority: "none",
    taskDue: "",

    _sortables: [],

    // ---- Helpers ----
    async api(path, opts) {
      const res = await fetch("/api" + path, {
        headers: { "Content-Type": "application/json" },
        ...opts,
      });
      if (!res.ok) throw new Error("HTTP " + res.status);
      if (res.status === 204) return null;
      return res.json();
    },

    // ---- Init ----
    async init() {
      // Asegura que ningún modal quede abierto al (re)iniciar.
      this.showProjModal = false;
      this.showBoardModal = false;
      this.showTaskModal = false;
      this.showRenameModal = false;
      await this.loadProjects();
      if (this.projects.length) {
        await this.selectProject(this.projects[0].id);
      }
    },

    switchView(id) {
      this.viewMode = id;
      if (id === "kanban") this.$nextTick(() => this.initSortable());
      if (id === "master") this.loadMaster();
    },

    boardsByProject(pid) {
      return this.boards.filter((b) => b.project_id === pid);
    },
    currentBoard() {
      return this.board.board;
    },
    boardTaskCount() {
      return this.board.columns.reduce((n, c) => n + c.tasks.length, 0);
    },
    allBoardTasks() {
      return this.board.columns.flatMap((c) => c.tasks);
    },

    // ---- Carga ----
    async loadProjects() {
      this.projects = await this.api("/projects");
      this.boards = [];
      for (const p of this.projects) {
        const bs = await this.api("/projects/" + p.id + "/boards");
        this.boards.push(...bs);
      }
    },

    async selectProject(pid) {
      this.currentProjectId = pid;
      const bs = await this.api("/projects/" + pid + "/boards");
      // Añade/sincroniza boards
      for (const b of bs) {
        if (!this.boards.find((x) => x.id === b.id)) this.boards.push(b);
      }
      if (bs.length) await this.selectBoard(bs[0].id);
      else {
        this.currentBoardId = null;
        this.board = { board: {}, columns: [] };
      }
    },

    async selectBoard(bid) {
      this.currentBoardId = bid;
      this.board = await this.api("/boards/" + bid);
      if (this.viewMode === "kanban") this.$nextTick(() => this.initSortable());
    },

    // ---- Drag & Drop (SortableJS) ----
    initSortable() {
      // Limpia instancias previas
      this._sortables.forEach((s) => s.destroy());
      this._sortables = [];
      const self = this;
      document.querySelectorAll("[data-status]").forEach((el) => {
        const s = Sortable.create(el, {
          group: "tasks",
          ghostClass: "sortable-ghost",
          dragClass: "sortable-drag",
          animation: 120,
          onEnd(evt) {
            const toStatus = evt.to.getAttribute("data-status");
            const taskEl = evt.item;
            const taskId = taskEl.getAttribute("data-task");
            const newIndex = evt.newDraggableIndex;
            self
              .api("/tasks/" + taskId, {
                method: "PATCH",
                body: JSON.stringify({ status_id: Number(toStatus), position: newIndex }),
              })
              .then(() => self.selectBoard(self.currentBoardId))
              .catch((e) => alert("Error al mover: " + e.message));
          },
        });
        self._sortables.push(s);
      });
    },

    // ---- Mutaciones ----
    async createProject() {
      if (!this.projName.trim()) return;
      const p = await this.api("/projects", {
        method: "POST",
        body: JSON.stringify({ name: this.projName.trim() }),
      });
      this.projName = "";
      this.showProjModal = false;
      await this.loadProjects();
      await this.selectProject(p.id);
    },

    async createBoard() {
      if (!this.boardName.trim() || !this.currentProjectId) return;
      const b = await this.api("/projects/" + this.currentProjectId + "/boards", {
        method: "POST",
        body: JSON.stringify({ name: this.boardName.trim() }),
      });
      this.boardName = "";
      this.showBoardModal = false;
      await this.loadProjects();
      await this.selectBoard(b.id);
    },

    async createTask() {
      if (!this.taskTitle.trim() || !this.currentBoardId) return;
      if (!this.taskStatus && this.board.columns.length)
        this.taskStatus = this.board.columns[0].status.id;
      const body = {
        title: this.taskTitle.trim(),
        description: this.taskDesc,
        status_id: Number(this.taskStatus),
        priority: this.taskPriority,
      };
      if (this.taskDue) body.due_date = this.taskDue;
      await this.api("/boards/" + this.currentBoardId + "/tasks", {
        method: "POST",
        body: JSON.stringify(body),
      });
      this.taskTitle = "";
      this.taskDesc = "";
      this.taskDue = "";
      this.taskPriority = "none";
      this.showTaskModal = false;
      await this.selectBoard(this.currentBoardId);
    },

    async moveTask(taskId, statusId) {
      await this.patchTask(taskId, { status_id: Number(statusId), position: 0 });
      await this.selectBoard(this.currentBoardId);
    },

    async patchTask(taskId, fields) {
      await this.api("/tasks/" + taskId, {
        method: "PATCH",
        body: JSON.stringify(fields),
      });
    },

    async deleteTask(taskId) {
      if (!confirm("¿Eliminar esta tarea?")) return;
      await this.api("/tasks/" + taskId, { method: "DELETE" });
      await this.selectBoard(this.currentBoardId);
    },

    // ---- Renombrar / Eliminar proyectos y tableros ----
    openRename(type, id, name) {
      this.renameType = type;
      this.renameId = id;
      this.renameName = name;
      this.showRenameModal = true;
    },

    async doRename() {
      if (!this.renameName.trim()) return;
      if (this.renameType === "project") {
        await this.api("/projects/" + this.renameId, {
          method: "PATCH",
          body: JSON.stringify({ name: this.renameName.trim() }),
        });
      } else {
        await this.api("/boards/" + this.renameId, {
          method: "PATCH",
          body: JSON.stringify({ name: this.renameName.trim() }),
        });
        if (this.currentBoardId === this.renameId) {
          this.board.board.name = this.renameName.trim();
        }
      }
      this.showRenameModal = false;
      await this.loadProjects();
    },

    async deleteProject(id) {
      if (!confirm("¿Eliminar este proyecto y todos sus tableros/tareas?")) return;
      await this.api("/projects/" + id, { method: "DELETE" });
      if (this.currentProjectId === id) {
        this.currentProjectId = null;
        this.currentBoardId = null;
        this.board = { board: {}, columns: [] };
      }
      await this.loadProjects();
    },

    async deleteBoard(id) {
      if (!confirm("¿Eliminar este tablero y sus tareas?")) return;
      await this.api("/boards/" + id, { method: "DELETE" });
      if (this.currentBoardId === id) {
        this.currentBoardId = null;
        this.board = { board: {}, columns: [] };
      }
      await this.loadProjects();
      if (this.currentProjectId) {
        const bs = this.boardsByProject(this.currentProjectId);
        if (bs.length) await this.selectBoard(bs[0].id);
      }
    },

    closeModals() {
      this.showProjModal = false;
      this.showBoardModal = false;
      this.showTaskModal = false;
      this.showRenameModal = false;
    },

    // ---- Master Table ----
    async loadMaster() {
      const q = new URLSearchParams();
      if (this.filters.status) q.set("status", this.filters.status);
      if (this.filters.priority) q.set("priority", this.filters.priority);
      if (this.filters.project) q.set("project", this.filters.project);
      if (this.filters.from) q.set("from", this.filters.from);
      if (this.filters.to) q.set("to", this.filters.to);
      if (this.filters.sort) q.set("sort", this.filters.sort);
      if (this.filters.order) q.set("order", this.filters.order);
      this.master = await this.api("/tasks?" + q.toString());
    },

    sortBy(col) {
      if (this.filters.sort === col)
        this.filters.order = this.filters.order === "asc" ? "desc" : "asc";
      else {
        this.filters.sort = col;
        this.filters.order = "asc";
      }
      this.loadMaster();
    },

    resetFilters() {
      this.filters = { status: "", priority: "", project: "", from: "", to: "", sort: "", order: "desc" };
      this.loadMaster();
    },

    statusColor(t) {
      const map = {
        Backlog: "#6b7280",
        "To Do": "#3b82f6",
        "In Progress": "#f59e0b",
        Done: "#22c55e",
      };
      return `background:${map[t.status_name] || "#6b7280"}`;
    },
  };
}

window.app = app;
