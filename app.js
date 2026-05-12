    const dashboardData = {
      taskGroups: {},
      roomTaskSets: {},
      floors: {
        top: {
          label: "Top Floor",
          defaultRoom: "top-stairs",
          rooms: [
            {
              id: "top-bedroom-1",
              name: "Bedroom 1",
              className: "top-bedroom-1",
              tasks: []
            },
            {
              id: "top-bedroom-2",
              name: "Bedroom 2",
              className: "top-bedroom-2",
              tasks: []
            },
            {
              id: "top-stairs",
              name: "Stairs",
              mapLabel: "Stairs",
              className: "top-stairs",
              compact: true,
              sharedTaskSet: "stairs"
            },
            {
              id: "top-living-room",
              name: "Living Room",
              className: "top-living-room",
              tasks: []
            },
            {
              id: "top-dining-room",
              name: "Dining Room",
              className: "top-dining-room",
              tasks: []
            },
            {
              id: "top-kitchen",
              name: "Kitchen",
              className: "top-kitchen",
              tasks: []
            },
            {
              id: "top-bathroom",
              name: "Bathroom",
              mapLabel: "Bath",
              className: "top-bathroom",
              compact: true,
              tasks: []
            },
            {
              id: "top-bedroom-3",
              name: "Bedroom 3",
              className: "top-bedroom-3",
              tasks: []
            },
            {
              id: "top-hall-closet",
              name: "Hall Closet",
              mapLabel: "Closet",
              className: "top-hall-closet",
              compact: true,
              tasks: []
            },
            {
              id: "top-back-patio",
              name: "Back Patio",
              className: "top-back-patio",
              tasks: []
            }
          ]
        },
        bottom: {
          label: "Bottom Floor",
          defaultRoom: "bottom-laundry-electrical-storage",
          rooms: [
            {
              id: "bottom-garage",
              name: "Garage",
              className: "bottom-garage",
              tasks: []
            },
            {
              id: "bottom-stairs",
              name: "Stairs",
              mapLabel: "Stairs",
              className: "bottom-stairs",
              compact: true,
              sharedTaskSet: "stairs"
            },
            {
              id: "bottom-family-room-office",
              name: "Family Room / Office",
              className: "bottom-family-room-office",
              tasks: []
            },
            {
              id: "bottom-downstairs-bathroom",
              name: "Downstairs Bathroom",
              className: "bottom-downstairs-bathroom",
              tasks: []
            },
            {
              id: "bottom-laundry-electrical-storage",
              name: "Laundry / Electrical / Storage",
              className: "bottom-laundry-electrical-storage",
              tasks: []
            }
          ]
        }
      }
    };

    const state = {
      activeFloor: "top",
      selectedRoomId: "top-stairs",
      taskFilter: "rooms",
      viewMode: "plan",
      sortKey: "priority",
      sortDirection: "asc"
    };

    const floorPlan = document.querySelector("#floor-plan");
    const planView = document.querySelector("#plan-view");
    const tableView = document.querySelector("#table-view");
    const roomTitle = document.querySelector("#selected-room-title");
    const roomSubtitle = document.querySelector("#selected-room-subtitle");
    const roomTasks = document.querySelector("#room-tasks");
    const focusList = document.querySelector("#focus-list");
    const taskTableBody = document.querySelector("#task-table-body");

    function getRoomTasks(room) {
      return room.sharedTaskSet ? dashboardData.roomTaskSets[room.sharedTaskSet] : room.tasks;
    }

    function applyExternalTaskData(taskData) {
      dashboardData.taskGroups = taskData.taskGroups || dashboardData.taskGroups;
      dashboardData.roomTaskSets = taskData.roomTaskSets || dashboardData.roomTaskSets;

      Object.entries(taskData.floors || {}).forEach(([floorKey, externalFloor]) => {
        const localFloor = dashboardData.floors[floorKey];
        if (!localFloor) return;

        externalFloor.rooms.forEach((externalRoom) => {
          const localRoom = localFloor.rooms.find((room) => room.id === externalRoom.id);
          if (!localRoom) return;

          if ("tasks" in externalRoom) localRoom.tasks = externalRoom.tasks;
          if ("sharedTaskSet" in externalRoom) localRoom.sharedTaskSet = externalRoom.sharedTaskSet;
          if ("mapLabel" in externalRoom) localRoom.mapLabel = externalRoom.mapLabel;
          if ("compact" in externalRoom) localRoom.compact = externalRoom.compact;
        });
      });
    }

    function allTasks() {
      const roomTasks = Object.values(dashboardData.floors).flatMap((floor) =>
        floor.rooms.flatMap((room) =>
          getRoomTasks(room).map((task) => ({
            ...task,
            area: room.name,
            floor: floor.label,
            floorKey: Object.keys(dashboardData.floors).find((key) => dashboardData.floors[key] === floor),
            roomId: room.id,
            groupKey: ""
          }))
        )
      );
      const groupTasks = Object.entries(dashboardData.taskGroups).flatMap(([groupKey, group]) =>
        group.tasks.map((task) => ({
          ...task,
          area: group.label,
          floor: "Project List",
          floorKey: "",
          roomId: "",
          groupKey
        }))
      );
      const uniqueTasks = new Map();
      [...roomTasks, ...groupTasks].forEach((task) => {
        if (!uniqueTasks.has(task.id)) {
          uniqueTasks.set(task.id, task);
        } else if (task.id.startsWith("stairs-")) {
          const existing = uniqueTasks.get(task.id);
          uniqueTasks.set(task.id, { ...existing, area: "Stairs", floor: "Top Floor / Bottom Floor" });
        }
      });
      return [...uniqueTasks.values()];
    }

    function formatDate(dateValue) {
      if (!dateValue) return "—";
      const [year, month, day] = dateValue.split("-");
      return `${Number(month)}/${Number(day)}/${year}`;
    }

    function sortValue(task, key) {
      const priorityRank = { critical: 1, important: 2, later: 3, complete: 4 };
      const statusRank = { "in-progress": 1, open: 2, done: 3 };
      const typeRank = { contractor: 1, DIY: 2, done: 3 };

      if (key === "priority") return priorityRank[task.priority] || 99;
      if (key === "status") return statusRank[task.status] || 99;
      if (key === "type") return typeRank[task.type] || 99;
      if (key === "percentComplete") return task.percentComplete ?? (task.status === "done" ? 100 : 0);
      if (key === "dateStarted") return task.dateStarted || "9999-12-31";
      if (key === "completedOn") return task.completedOn || "9999-12-31";
      return String(task[key] || "").toLowerCase();
    }

    function sortedTasks() {
      const direction = state.sortDirection === "asc" ? 1 : -1;
      return [...allTasks()].sort((a, b) => {
        const aValue = sortValue(a, state.sortKey);
        const bValue = sortValue(b, state.sortKey);
        if (aValue < bValue) return -1 * direction;
        if (aValue > bValue) return 1 * direction;
        return a.title.localeCompare(b.title);
      });
    }

    function taskBadges(task) {
      const badges = [{ className: badgeClass(task), label: badgeLabel(task) }];
      if (task.status !== "done" && task.type === "contractor" && badgeClass(task) !== "contractor") {
        badges.push({ className: "contractor", label: "contractor" });
      }
      if (task.status !== "done" && task.type === "DIY" && badgeClass(task) !== "diy") {
        badges.push({ className: "diy", label: "DIY" });
      }
      return badges;
    }

    function badgeClass(task) {
      if (task.status === "done") return "done";
      if (task.status === "in-progress") return "in-progress";
      if (task.priority === "critical") return "urgent";
      if (task.priority === "later") return "later";
      if (task.type === "contractor") return "contractor";
      return "diy";
    }

    function badgeLabel(task) {
      if (task.status === "done") return "done";
      if (task.status === "in-progress") return "in progress";
      if (task.priority === "critical") return "urgent";
      if (task.priority === "later") return "later";
      if (task.type === "contractor") return "contractor";
      return "DIY";
    }

    function statusLabel(task) {
      if (task.status === "done") return "Done";
      if (task.status === "in-progress") return "In Progress";
      return "Open";
    }

    function taskPercent(task) {
      if (Number.isFinite(task.percentComplete)) return task.percentComplete;
      return task.status === "done" ? 100 : 0;
    }

    function progressMarkup(task) {
      const percent = taskPercent(task);
      if (!percent && task.status !== "in-progress") return "";

      return `
        <div class="task-progress">
          <div class="task-progress-label">
            <span>Progress</span>
            <span>${percent}%</span>
          </div>
          <div class="task-progress-track" aria-hidden="true">
            <div class="task-progress-bar" style="width: ${percent}%"></div>
          </div>
        </div>
      `;
    }

    function subtasksMarkup(task) {
      if (!task.subtasks?.length) return "";

      return `
        <div class="task-detail-section">
          <p class="task-detail-title">Subtasks</p>
          <ul class="detail-list">
            ${task.subtasks.map((subtask) => `
              <li>
                <strong>${subtask.title}</strong>
                <span>${statusLabel(subtask)}${Number.isFinite(subtask.percentComplete) ? ` · ${subtask.percentComplete}%` : ""}${subtask.dateStarted ? ` · Started ${formatDate(subtask.dateStarted)}` : ""}</span>
              </li>
            `).join("")}
          </ul>
        </div>
      `;
    }

    function itemsNeededMarkup(task) {
      if (!task.itemsNeeded?.length) return "";

      return `
        <div class="task-detail-section">
          <p class="task-detail-title">Items Needed</p>
          <ul class="detail-list">
            ${task.itemsNeeded.map((item) => `
              <li>
                <strong>${item.name}</strong>
                <span>${item.status || "needed"}</span>
              </li>
            `).join("")}
          </ul>
        </div>
      `;
    }

    function roomBadges(room) {
      const tasks = getRoomTasks(room);
      const badges = [];
      if (tasks.some((task) => task.priority === "critical" && task.status !== "done")) badges.push("urgent");
      if (tasks.some((task) => task.type === "contractor" && task.status !== "done")) badges.push("contractor");
      if (tasks.length && tasks.every((task) => task.status === "done")) badges.push("done");
      return badges;
    }

    function findRoom(roomId = state.selectedRoomId) {
      const floor = dashboardData.floors[state.activeFloor];
      return floor.rooms.find((room) => room.id === roomId) || floor.rooms[0];
    }

    function renderFloor() {
      const floor = dashboardData.floors[state.activeFloor];
      floorPlan.className = `plan-shell ${state.activeFloor}-plan`;
      floorPlan.innerHTML = "";

      floor.rooms.forEach((room) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = `room ${room.className}${room.compact ? " is-compact" : ""}${room.id === state.selectedRoomId ? " is-selected" : ""}`;
        button.setAttribute("aria-pressed", room.id === state.selectedRoomId ? "true" : "false");
        button.dataset.roomId = room.id;

        const openTasks = getRoomTasks(room).filter((task) => task.status !== "done").length;
        const badges = roomBadges(room);
        button.innerHTML = `
          <span class="room-name">${room.mapLabel || room.name}</span>
          <span class="room-count">${openTasks} open</span>
          <span class="room-badges">${badges.map((badge) => `<span class="badge ${badge}">${badge}</span>`).join("")}</span>
        `;

        button.addEventListener("click", () => {
          state.selectedRoomId = room.id;
          state.taskFilter = "rooms";
          updateFilterButtons();
          renderFloor();
          renderRoomPanel();
        });

        floorPlan.appendChild(button);
      });
    }

    function renderRoomPanel() {
      if (state.taskFilter !== "rooms") {
        const group = dashboardData.taskGroups[state.taskFilter];
        roomTitle.textContent = group.label;
        roomSubtitle.textContent = group.description;
        renderTasks(group.tasks);
        return;
      }

      const floor = dashboardData.floors[state.activeFloor];
      const room = findRoom();
      roomTitle.textContent = room.name;
      roomSubtitle.textContent = floor.label;
      renderTasks(getRoomTasks(room));
    }

    function renderTasks(tasks) {
      roomTasks.innerHTML = "";

      if (!tasks.length) {
        roomTasks.innerHTML = `<li class="empty-note">No tasks yet.</li>`;
        return;
      }

      tasks.forEach((task) => {
        const item = document.createElement("li");
        item.className = "task-item";
        item.innerHTML = `
          <p class="task-title">${task.title}</p>
          <div class="task-badges">
            ${taskBadges(task).map((badge) => `<span class="badge ${badge.className}">${badge.label}</span>`).join("")}
          </div>
          ${task.dateStarted ? `<p class="task-meta" style="margin-top: 8px;">Started ${formatDate(task.dateStarted)}</p>` : ""}
          ${task.completedOn ? `<p class="task-meta" style="margin-top: 8px;">Completed ${formatDate(task.completedOn)}</p>` : ""}
          ${progressMarkup(task)}
          ${subtasksMarkup(task)}
          ${itemsNeededMarkup(task)}
        `;
        roomTasks.appendChild(item);
      });
    }

    function renderSummary() {
      const tasks = allTasks();
      const critical = tasks.filter((task) => task.priority === "critical" && task.status !== "done").length;
      const contractor = tasks.filter((task) => task.type === "contractor" && task.status !== "done").length;
      const done = tasks.filter((task) => task.status === "done").length;
      const later = tasks.filter((task) => task.priority === "later" && task.status !== "done").length;
      const open = tasks.filter((task) => task.status !== "done").length;
      const total = tasks.length;
      const percent = total ? Math.round((done / total) * 100) : 0;

      document.querySelector("#critical-count").textContent = critical;
      document.querySelector("#contractor-count").textContent = contractor;
      document.querySelector("#done-count").textContent = done;
      document.querySelector("#open-count").textContent = open;
      document.querySelector("#later-count").textContent = later;
      document.querySelector("#total-count").textContent = total;
      document.querySelector("#progress-copy").textContent = `${done} of ${total} tasks complete.`;
      document.querySelector("#progress-bar").style.width = `${percent}%`;
    }

    function renderFocusList() {
      const focusTasks = allTasks()
        .filter((task) => task.doFirst && task.status !== "done")
        .sort((a, b) => (a.doFirstRank || 999) - (b.doFirstRank || 999));

      focusList.innerHTML = "";
      focusTasks.forEach((task) => {
        const item = document.createElement("li");
        item.className = "focus-card";
        item.innerHTML = `
          <p class="task-title">${task.title}</p>
          <p class="task-meta">${task.area} · ${task.floor}</p>
          <div class="task-badges" style="margin-top: 8px;">
            ${taskBadges(task).map((badge) => `<span class="badge ${badge.className}">${badge.label}</span>`).join("")}
          </div>
        `;
        focusList.appendChild(item);
      });
    }

    function renderTaskTable() {
      const tasks = sortedTasks();
      taskTableBody.innerHTML = "";
      updateSortIndicators();

      tasks.forEach((task) => {
        const row = document.createElement("tr");
        row.dataset.taskId = task.id;
        row.innerHTML = `
          <td><button class="table-task-button" type="button">${task.title}</button></td>
          <td>${task.area}</td>
          <td><span class="badge ${badgeClass(task)}">${badgeLabel(task)}</span></td>
          <td>${task.type}</td>
          <td>${statusLabel(task)}</td>
          <td>${taskPercent(task)}%</td>
          <td>${formatDate(task.dateStarted)}</td>
          <td>${formatDate(task.completedOn)}</td>
        `;

        row.addEventListener("click", () => selectTaskFromTable(task));
        taskTableBody.appendChild(row);
      });
    }

    function selectTaskFromTable(task) {
      taskTableBody.querySelectorAll("tr").forEach((row) => {
        row.classList.toggle("is-selected", row.dataset.taskId === task.id);
      });

      if (task.groupKey) {
        state.taskFilter = task.groupKey;
      } else {
        state.activeFloor = task.floorKey;
        state.selectedRoomId = task.roomId;
        state.taskFilter = "rooms";
        document.querySelectorAll("[data-floor-tab]").forEach((button) => {
          button.setAttribute("aria-selected", button.dataset.floorTab === state.activeFloor ? "true" : "false");
        });
      }

      state.viewMode = "plan";
      updateFilterButtons();
      updateViewMode();
      renderFloor();
      renderRoomPanel();
      roomTitle.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }

    function updateViewMode() {
      planView.hidden = state.viewMode !== "plan";
      tableView.hidden = state.viewMode !== "table";
      document.querySelectorAll("[data-view-mode]").forEach((button) => {
        button.setAttribute("aria-pressed", button.dataset.viewMode === state.viewMode ? "true" : "false");
      });
    }

    function updateSortIndicators() {
      document.querySelectorAll("[data-sort-indicator]").forEach((indicator) => {
        indicator.textContent = indicator.dataset.sortIndicator === state.sortKey
          ? (state.sortDirection === "asc" ? "↑" : "↓")
          : "";
      });
    }

    document.querySelectorAll("[data-floor-tab]").forEach((tab) => {
      tab.addEventListener("click", () => {
        const floorKey = tab.dataset.floorTab;
        state.activeFloor = floorKey;
        state.selectedRoomId = dashboardData.floors[floorKey].defaultRoom;
        state.taskFilter = "rooms";

        document.querySelectorAll("[data-floor-tab]").forEach((button) => {
          button.setAttribute("aria-selected", button.dataset.floorTab === floorKey ? "true" : "false");
        });
        updateFilterButtons();

        renderFloor();
        renderRoomPanel();
      });
    });

    document.querySelectorAll("[data-task-filter]").forEach((button) => {
      button.addEventListener("click", () => {
        state.taskFilter = button.dataset.taskFilter;
        updateFilterButtons();
        renderRoomPanel();
      });
    });

    document.querySelectorAll("[data-view-mode]").forEach((button) => {
      button.addEventListener("click", () => {
        state.viewMode = button.dataset.viewMode;
        updateViewMode();
      });
    });

    document.querySelectorAll("[data-sort-key]").forEach((button) => {
      button.addEventListener("click", () => {
        const sortKey = button.dataset.sortKey;
        if (state.sortKey === sortKey) {
          state.sortDirection = state.sortDirection === "asc" ? "desc" : "asc";
        } else {
          state.sortKey = sortKey;
          state.sortDirection = "asc";
        }
        renderTaskTable();
      });
    });

    function updateFilterButtons() {
      document.querySelectorAll("[data-task-filter]").forEach((button) => {
        button.setAttribute("aria-pressed", button.dataset.taskFilter === state.taskFilter ? "true" : "false");
      });
    }

    function renderDashboard() {
      renderSummary();
      renderFocusList();
      renderTaskTable();
      renderFloor();
      renderRoomPanel();
      updateViewMode();
    }

    async function initDashboard() {
      try {
        const response = await fetch("tasks.json", { cache: "no-store" });
        if (!response.ok) {
          throw new Error("Could not load tasks.json");
        }
        applyExternalTaskData(await response.json());
      } catch (error) {
        roomTasks.innerHTML = `<li class="empty-note">Task data could not be loaded. Use GitHub Pages or a local static server so tasks.json can be fetched.</li>`;
        focusList.innerHTML = "";
        return;
      }

      renderDashboard();
    }

    initDashboard();