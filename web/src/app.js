function app() {
  return {
    // Estado
    projects: [],
    boards: [],
    board: { board: {}, columns: [] },
    currentProjectId: null,
    currentBoardId: null,
    viewMode: "kanban",
    sidebarOpen: true,
    sidebarCollapsed: false,
    sidebarHover: false,
    views: [
      { id: "kanban", label: "Kanban" },
      { id: "table", label: "Tabla" },
      { id: "calendar", label: "Calendario" },
      { id: "master", label: "Global" },
    ],
    master: [],
    calendar: [],
    calYear: new Date().getFullYear(),
    calMonth: new Date().getMonth(),
    calFilter: { project: "", board: "", tag: "" },
    filters: { status: "", priority: "", project: "", tag: "", from: "", to: "", sort: "", order: "desc" },
    tags: [],
    statuses: [],

    // Modales
    showProjModal: false,
    showBoardModal: false,
    showTaskModal: false,
    showRenameModal: false,
    renameType: "",
    renameId: null,
    renameName: "",
    showTaskEditModal: false,
    editTask: { id: null, title: "", description: "", priority: "none", dueDate: "", statusId: null, tags: "" },
    projName: "",
    boardName: "",
    taskTitle: "",
    taskDesc: "",
    taskStatus: null,
    taskPriority: "none",
    taskDue: "",
    taskTags: "",

    // Filtro de la vista de tabla por tablero (espejo de los filtros globales)
    boardFilter: { status: "", priority: "", tag: "", from: "", to: "", sort: "", order: "desc" },

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
      // Recupera el estado colapsado del sidebar (persistido en localStorage).
      try {
        if (localStorage.getItem("sidebarCollapsed") === "1") this.sidebarCollapsed = true;
      } catch (e) {}
      await this.loadProjects();
      await this.loadTags();
      await this.loadStatuses();
      if (this.projects.length) {
        await this.selectProject(this.projects[0].id);
      }
    },

    async loadTags() {
      this.tags = await this.api("/tags");
    },

    async loadStatuses() {
      this.statuses = await this.api("/statuses");
    },

    statusesForBoard(bid) {
      return this.statuses.filter((s) => s.board_id === bid);
    },

    // Al cambiar de vista se recarga SIEMPRE la fuente de datos correspondiente
    // para garantizar que el estado (columnas, estado de las tareas, etc.) refleje
    // la realidad y no quede desactualizado tras cambios hechos en otra vista.
    async switchView(id) {
      this.viewMode = id;
      if (id === "kanban" || id === "table") {
        // El tablero alimenta tanto al Kanban como a la vista de Tabla.
        await this.reloadBoard();
      } else if (id === "master") {
        await this.loadMaster();
      } else if (id === "calendar") {
        await this.loadCalendar();
      }
    },

    // ---- Calendario ----
    async loadCalendar() {
      const q = new URLSearchParams();
      if (this.calFilter.project) q.set("project", this.calFilter.project);
      if (this.calFilter.board) q.set("board", this.calFilter.board);
      if (this.calFilter.tag) q.set("tag", this.calFilter.tag);
      this.calendar = await this.api("/tasks?" + q.toString());
    },

    // Tableros disponibles para el filtro: los del proyecto elegido (o todos).
    calBoardOptions() {
      if (!this.calFilter.project) return this.boards;
      const p = this.projects.find((x) => x.name === this.calFilter.project);
      if (!p) return this.boards;
      return this.boardsByProject(p.id);
    },

    onCalProjectChange() {
      // Al cambiar de proyecto se reinicia el filtro de tablero para evitar
      // seleccionar un tablero que no pertenece al proyecto activo.
      this.calFilter.board = "";
      this.loadCalendar();
    },

    resetCalFilters() {
      this.calFilter = { project: "", board: "", tag: "" };
      this.loadCalendar();
    },

    calStatusColor(sid) {
      const s = this.statuses.find((x) => x.id === sid);
      return s ? s.color : "#6b7280";
    },

    calMonthLabel() {
      const meses = [
        "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
        "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
      ];
      return meses[this.calMonth] + " " + this.calYear;
    },

    prevMonth() {
      if (this.calMonth === 0) {
        this.calMonth = 11;
        this.calYear--;
      } else {
        this.calMonth--;
      }
    },

    nextMonth() {
      if (this.calMonth === 11) {
        this.calMonth = 0;
        this.calYear++;
      } else {
        this.calMonth++;
      }
    },

    goToday() {
      const d = new Date();
      this.calYear = d.getFullYear();
      this.calMonth = d.getMonth();
    },

    isoDate(d) {
      const y = d.getFullYear();
      const m = String(d.getMonth() + 1).padStart(2, "0");
      const day = String(d.getDate()).padStart(2, "0");
      return y + "-" + m + "-" + day;
    },

    calWeeks() {
      const year = this.calYear;
      const month = this.calMonth;
      const first = new Date(year, month, 1);
      const startDow = (first.getDay() + 6) % 7; // 0 = lunes
      const cursor = new Date(year, month, 1 - startDow);
      const weeks = [];
      const todayIso = this.isoDate(new Date());
      for (let w = 0; w < 6; w++) {
        const days = [];
        for (let d = 0; d < 7; d++) {
          const date = new Date(cursor);
          const iso = this.isoDate(date);
          days.push({
            iso,
            day: date.getDate(),
            inMonth: date.getMonth() === month,
            isToday: iso === todayIso,
            tasks: this.tasksForDate(iso),
          });
          cursor.setDate(cursor.getDate() + 1);
        }
        weeks.push(days);
      }
      return weeks;
    },

    tasksForDate(iso) {
      return this.calendar.filter(
        (t) => t.due_date && (t.due_date || "").slice(0, 10) === iso
      );
    },

    calTasksCount() {
      return this.calendar.filter((t) => t.due_date).length;
    },

    boardsByProject(pid) {
      return this.boards.filter((b) => b.project_id === pid);
    },

    // Colapsa/despliega el sidebar (escritorio). El estado se persiste.
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
      this.sidebarHover = false;
      try {
        localStorage.setItem("sidebarCollapsed", this.sidebarCollapsed ? "1" : "0");
      } catch (e) {}
    },

    sidebarClasses() {
      const mobile = this.sidebarOpen ? "translate-x-0" : "-translate-x-full";
      if (this.sidebarCollapsed) {
        const reveal = this.sidebarHover ? "md:translate-x-0" : "md:-translate-x-full";
        return mobile + " md:fixed md:top-14 " + reveal;
      }
      return mobile + " md:static md:translate-x-0";
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
    filteredBoardTasks() {
      let tasks = this.allBoardTasks();
      const f = this.boardFilter;
      if (f.tag) tasks = tasks.filter((t) => (t.tags || []).includes(f.tag));
      if (f.status) tasks = tasks.filter((t) => this.boardStatusName(t.status_id) === f.status);
      if (f.priority) tasks = tasks.filter((t) => t.priority === f.priority);
      if (f.from) tasks = tasks.filter((t) => t.due_date && t.due_date >= f.from);
      if (f.to) tasks = tasks.filter((t) => t.due_date && t.due_date <= f.to);

        const col = f.sort;
      const dir = f.order === "asc" ? 1 : -1;
      if (col) {
        const prio = { none: 0, low: 1, medium: 2, high: 3 };
        tasks = tasks.slice().sort((a, b) => {
          let av, bv;
          switch (col) {
            case "title":
              av = a.title; bv = b.title; break;
            case "priority":
              av = prio[a.priority] || 0; bv = prio[b.priority] || 0; break;
            case "due_date": {
              // Las tareas sin fecha van siempre al final, asc o desc.
              const an = a.due_date ? 1 : 0;
              const bn = b.due_date ? 1 : 0;
              if (an !== bn) return an - bn;
              av = a.due_date; bv = b.due_date; break;
            }
            case "status":
              av = this.boardStatusName(a.status_id); bv = this.boardStatusName(b.status_id); break;
            default:
              return 0;
          }
          if (av < bv) return -1 * dir;
          if (av > bv) return 1 * dir;
          return 0;
        });
      }
      return tasks;
    },
    boardStatusName(id) {
      const col = this.board.columns.find((c) => c.status.id === id);
      return col ? col.status.name : "";
    },
    boardSortBy(col) {
      if (this.boardFilter.sort === col)
        this.boardFilter.order = this.boardFilter.order === "asc" ? "desc" : "asc";
      else {
        this.boardFilter.sort = col;
        this.boardFilter.order = "asc";
      }
    },
    resetBoardFilters() {
      this.boardFilter = { status: "", priority: "", tag: "", from: "", to: "", sort: "", order: "desc" };
    },
    boardTags() {
      const set = new Set();
      this.allBoardTasks().forEach((t) => (t.tags || []).forEach((tag) => set.add(tag)));
      return Array.from(set).sort();
    },
    parseTags(str) {
      if (!str) return [];
      return Array.from(
        new Set(
          str
            .split(",")
            .map((s) => s.trim())
            .filter((s) => s)
        )
      );
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
      await this.reloadBoard();
      this.sidebarOpen = false;
    },

    // Recarga el tablero actual (columnas y tareas) desde el servidor y
    // reinicializa el drag & drop si estamos en la vista Kanban.
    async reloadBoard() {
      if (!this.currentBoardId) {
        this.board = { board: {}, columns: [] };
        return;
      }
      this.board = await this.api("/boards/" + this.currentBoardId);
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
        tags: this.parseTags(this.taskTags),
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
      this.taskTags = "";
      this.showTaskModal = false;
      await this.selectBoard(this.currentBoardId);
      await this.loadTags();
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
      this.showTaskEditModal = false;
    },

    // ---- Editar tarea ----
    openTaskEdit(task) {
      this.editTask = {
        id: task.id,
        title: task.title,
        description: task.description,
        priority: task.priority,
        dueDate: task.due_date || "",
        statusId: task.status_id,
        tags: (task.tags || []).join(", "),
      };
      this.showTaskEditModal = true;
    },

    async saveTaskEdit() {
      if (!this.editTask.title || !this.editTask.title.trim()) return;
      const body = {
        title: this.editTask.title.trim(),
        description: this.editTask.description,
        priority: this.editTask.priority,
        due_date: this.editTask.dueDate ? this.editTask.dueDate : null,
        tags: this.parseTags(this.editTask.tags),
      };
      if (this.viewMode !== "master") body.status_id = Number(this.editTask.statusId);
      await this.api("/tasks/" + this.editTask.id, {
        method: "PATCH",
        body: JSON.stringify(body),
      });
      this.showTaskEditModal = false;
      if (this.viewMode === "master") await this.loadMaster();
      else if (this.viewMode === "calendar") await this.loadCalendar();
      else await this.selectBoard(this.currentBoardId);
      // Mantiene el Kanban sincronizado aunque se edite desde otra vista.
      await this.reloadBoard();
      await this.loadTags();
    },

    // ---- Master Table ----
    async loadMaster() {
      const q = new URLSearchParams();
      if (this.filters.status) q.set("status", this.filters.status);
      if (this.filters.priority) q.set("priority", this.filters.priority);
      if (this.filters.project) q.set("project", this.filters.project);
      if (this.filters.tag) q.set("tag", this.filters.tag);
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
      this.filters = { status: "", priority: "", project: "", tag: "", from: "", to: "", sort: "", order: "desc" };
      this.loadMaster();
    },

    // Edita un campo de una tarea en la Master Table y recarga la lista.
    async patchTaskGlobal(id, fields) {
      await this.patchTask(id, fields);
      await this.loadMaster();
      // Sincroniza el Kanban por si se cambió el estado de una tarea.
      await this.reloadBoard();
    },

    async deleteMasterTask(id) {
      if (!confirm("¿Eliminar esta tarea?")) return;
      await this.api("/tasks/" + id, { method: "DELETE" });
      await this.loadMaster();
      await this.reloadBoard();
    },
  };
}

window.app = app;
