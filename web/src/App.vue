<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import FloorPlan from "./components/FloorPlan.vue";
import FocusList from "./components/FocusList.vue";
import ProgressPanel from "./components/ProgressPanel.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TaskPanel from "./components/TaskPanel.vue";
import TaskTable from "./components/TaskTable.vue";
import { loadCurrentHouse, saveCurrentHouse } from "./services/api";
import type { HouseState, ItemNeeded, Room, Subtask, Task, TaskPriority, TaskStatus, TaskType, TaskWithContext } from "./types/house";
import { allTasks, clampTaskPercent, deleteHouseTask, getRoomTasks, sortedTasks, taskPercent, updateHouseTask, type SortDirection, type SortKey } from "./utils/house";

type ViewMode = "plan" | "table";
type SaveState = "idle" | "dirty" | "saving" | "saved" | "error";
type RoomSize = "small" | "medium" | "large";
type RoomShape = "square" | "wide" | "tall";

interface HouseDraft {
  house: HouseState;
  updatedAt: number;
  dirty: boolean;
}

const DRAFT_KEY = "homeplan.houseDraft";
const ROOM_SIZE_PRESETS: Record<RoomSize, Record<RoomShape, Pick<Room["layout"], "w" | "h">>> = {
  small: {
    square: { w: 18, h: 18 },
    wide: { w: 26, h: 16 },
    tall: { w: 16, h: 26 }
  },
  medium: {
    square: { w: 28, h: 28 },
    wide: { w: 38, h: 24 },
    tall: { w: 24, h: 38 }
  },
  large: {
    square: { w: 40, h: 40 },
    wide: { w: 52, h: 34 },
    tall: { w: 34, h: 52 }
  }
};

const house = ref<HouseState | null>(null);
const activeFloor = ref("top");
const selectedRoomId = ref("top-stairs");
const taskFilter = ref("rooms");
const viewMode = ref<ViewMode>("plan");
const sortKey = ref<SortKey>("priority");
const sortDirection = ref<SortDirection>("asc");
const dataSource = ref<"api" | "empty">("empty");
const saveState = ref<SaveState>("idle");
const setupMode = ref(false);
const setupHouseName = ref("");
const setupFloorName = ref("");
const setupRoomName = ref("");

const currentFloor = computed(() => house.value?.floors[activeFloor.value]);
const tasks = computed(() => (house.value ? allTasks(house.value) : []));
const taskTableRows = computed(() => sortedTasks(tasks.value, sortKey.value, sortDirection.value));
const unplacedRooms = computed(() => house.value?.unplacedRooms ?? []);
const hasUnsavedChanges = computed(() => Boolean(house.value && (saveState.value === "dirty" || saveState.value === "error")));
const syncLabel = computed(() => {
  if (!house.value) return "No house";
  if (saveState.value === "dirty") return "Unsaved changes";
  if (saveState.value === "error") return "Save failed - local draft kept";
  if (dataSource.value === "api") return "Cloud session";
  return "Local draft";
});
const saveButtonLabel = computed(() => {
  if (saveState.value === "saving") return "Saving";
  if (saveState.value === "saved") return "Saved";
  if (saveState.value === "error") return "Save";
  return "Save";
});
const taskTargetOptions = computed(() => {
  if (!house.value) return [];
  const roomTargets = Object.entries(house.value.floors).flatMap(([floorKey, floor]) =>
    floor.rooms.map((room) => ({ value: `room:${floorKey}:${room.id}`, label: `${floor.label} / ${room.name}` }))
  );
  const groupTargets = Object.entries(house.value.taskGroups).map(([groupKey, group]) => ({
    value: `group:${groupKey}`,
    label: `Project List / ${group.label}`
  }));
  const unplacedTargets = (house.value.unplacedRooms ?? []).map((room) => ({ value: `unplaced:${room.id}`, label: `Unplaced / ${room.name}` }));
  return [
    ...roomTargets,
    ...unplacedTargets,
    ...groupTargets,
    { value: `new-room:${activeFloor.value}`, label: `Create new room on ${currentFloor.value?.label ?? "current floor"}` },
    { value: "new-unplaced-room", label: "Create unplaced room" }
  ];
});

const selectedRoom = computed(() => {
  const floor = currentFloor.value;
  if (!floor) return null;
  return floor.rooms.find((room) => room.id === selectedRoomId.value) ?? floor.rooms[0] ?? null;
});
const selectedRoomSize = computed<RoomSize>(() => {
  const room = selectedRoom.value;
  if (!room) return "medium";
  const area = room.layout.w * room.layout.h;
  if (area < 500) return "small";
  if (area > 1500) return "large";
  return "medium";
});
const selectedRoomShape = computed<RoomShape>(() => {
  const room = selectedRoom.value;
  if (!room) return "wide";
  const ratio = room.layout.w / Math.max(1, room.layout.h);
  if (ratio > 1.25) return "wide";
  if (ratio < 0.8) return "tall";
  return "square";
});

const panelTitle = computed(() => {
  if (!house.value) return "";
  if (taskFilter.value !== "rooms") return house.value.taskGroups[taskFilter.value]?.label ?? "";
  return selectedRoom.value?.name ?? "";
});

const panelSubtitle = computed(() => {
  if (!house.value) return "";
  if (taskFilter.value !== "rooms") return house.value.taskGroups[taskFilter.value]?.description ?? "";
  return currentFloor.value?.label ?? "";
});

const panelTasks = computed(() => {
  if (!house.value) return [];
  if (taskFilter.value !== "rooms") return house.value.taskGroups[taskFilter.value]?.tasks ?? [];
  return selectedRoom.value ? getRoomTasks(house.value, selectedRoom.value) : [];
});

const focusTasks = computed(() =>
  tasks.value
    .filter((task) => task.doFirst && task.status !== "done")
    .sort((a, b) => (a.doFirstRank || 999) - (b.doFirstRank || 999))
);

const progress = computed(() => {
  const all = tasks.value;
  const done = all.filter((task) => task.status === "done").length;
  const total = all.length;
  return {
    done,
    total,
    critical: all.filter((task) => task.priority === "critical" && task.status !== "done").length,
    contractor: all.filter((task) => task.type === "contractor" && task.status !== "done").length,
    later: all.filter((task) => task.priority === "later" && task.status !== "done").length,
    open: all.filter((task) => task.status !== "done").length,
    percent: total ? Math.round((done / total) * 100) : 0
  };
});

const nextFocus = computed(() => focusTasks.value[0]?.area || "Stairs");

onMounted(async () => {
  window.addEventListener("beforeunload", warnBeforeUnload);
  const draft = readDraft();
  try {
    const response = await loadCurrentHouse();
    if (draft?.dirty) {
      loadHouse(draft.house, "empty", "dirty");
      return;
    }

    if (response.house) {
      loadHouse(response.house, response.source, "idle");
      return;
    }

    if (draft) {
      loadHouse(draft.house, "empty", draft.dirty ? "dirty" : "idle");
      return;
    }

    house.value = null;
    dataSource.value = "empty";
    saveState.value = "idle";
  } catch {
    if (draft) {
      loadHouse(draft.house, "empty", "error");
      return;
    }
    house.value = null;
    dataSource.value = "empty";
    saveState.value = "error";
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", warnBeforeUnload);
});

function warnBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasUnsavedChanges.value) return;
  event.preventDefault();
  event.returnValue = "";
}

function readDraft(): HouseDraft | null {
  try {
    const rawDraft = window.localStorage.getItem(DRAFT_KEY);
    if (!rawDraft) return null;
    const draft = JSON.parse(rawDraft) as HouseDraft;
    if (!draft?.house || !Number.isFinite(draft.updatedAt)) return null;
    return draft;
  } catch {
    return null;
  }
}

function writeDraft(nextHouse: HouseState, dirty = true) {
  const draft: HouseDraft = { house: nextHouse, updatedAt: Date.now(), dirty };
  window.localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
}

function clearDraft() {
  window.localStorage.removeItem(DRAFT_KEY);
}

function loadHouse(nextHouse: HouseState, source: "api" | "empty", state: SaveState) {
  house.value = nextHouse;
  dataSource.value = source;
  saveState.value = state;
  activeFloor.value = nextHouse.floors.top ? "top" : Object.keys(nextHouse.floors)[0] ?? "";
  selectedRoomId.value = nextHouse.floors[activeFloor.value]?.defaultRoom ?? Object.values(nextHouse.floors)[0]?.defaultRoom ?? "";
  taskFilter.value = "rooms";
}

function selectFloor(floorKey: string) {
  const floor = house.value?.floors[floorKey];
  if (!floor) return;
  activeFloor.value = floorKey;
  selectedRoomId.value = floor.defaultRoom;
  taskFilter.value = "rooms";
}

function selectRoom(roomId: string) {
  selectedRoomId.value = roomId;
  taskFilter.value = "rooms";
}

function changeSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDirection.value = sortDirection.value === "asc" ? "desc" : "asc";
    return;
  }
  sortKey.value = key;
  sortDirection.value = "asc";
}

function selectTask(task: TaskWithContext) {
  if (task.isUnplaced) {
    viewMode.value = "table";
    return;
  }
  if (task.groupKey) {
    taskFilter.value = task.groupKey;
  } else {
    activeFloor.value = task.floorKey;
    selectedRoomId.value = task.roomId;
    taskFilter.value = "rooms";
  }
  viewMode.value = "plan";
}

function slugify(value: string): string {
  const slug = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
  return slug || "item";
}

function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function starterHouse(houseName = "My Home", floorName = "Main Floor", roomName = "Main Room"): HouseState {
  return {
    schemaVersion: 1,
    id: uniqueId("house"),
    name: houseName.trim() || "My Home",
    taskGroups: {
      wholeHouse: {
        label: "Whole House",
        description: "Projects that cut across multiple rooms.",
        tasks: []
      }
    },
    roomTaskSets: {},
    floors: {
      main: {
        label: floorName.trim() || "Main Floor",
        defaultRoom: "main-room",
        grid: { columns: 100, rows: 100 },
        rooms: [
          {
            id: "main-room",
            name: roomName.trim() || "Main Room",
            layout: { x: 10, y: 10, ...ROOM_SIZE_PRESETS.medium.wide },
            tasks: []
          }
        ]
      }
    },
    unplacedRooms: []
  };
}

function newTask(title = "New task"): Task {
  return {
    id: uniqueId("task"),
    title,
    priority: "important",
    type: "DIY",
    status: "open",
    completedOn: ""
  };
}

function localDateString(date = new Date()): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function markEdited() {
  if (!house.value) return;
  writeDraft(house.value, true);
  saveState.value = "dirty";
}

async function createAndSaveHouse(nextHouse: HouseState) {
  loadHouse(nextHouse, "empty", "dirty");
  writeDraft(nextHouse, true);
  saveState.value = "saving";
  try {
    await saveCurrentHouse(nextHouse);
    clearDraft();
    dataSource.value = "api";
    saveState.value = "saved";
  } catch {
    saveState.value = "error";
  }
}

function createBlankHouse() {
  void createAndSaveHouse(starterHouse());
}

function startGuidedSetup() {
  setupMode.value = true;
}

function finishGuidedSetup() {
  void createAndSaveHouse(starterHouse(setupHouseName.value, setupFloorName.value, setupRoomName.value));
}

function updateTaskTitle(taskId: string, title: string) {
  if (!house.value || !title.trim()) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({ ...task, title: title.trim() }));
  markEdited();
}

function updateTaskPriority(taskId: string, priority: TaskPriority) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({ ...task, priority }));
  markEdited();
}

function updateTaskType(taskId: string, type: TaskType) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({ ...task, type }));
  markEdited();
}

function updateTaskStatus(taskId: string, status: TaskStatus) {
  if (!house.value) return;
  const completedOn = localDateString();
  house.value = updateHouseTask(house.value, taskId, (task) => {
    if (status === "done") {
      return { ...task, status, percentComplete: undefined, completedOn: task.completedOn || completedOn };
    }

    const nextTask = { ...task, status, completedOn: undefined };
    if (status === "in-progress" && taskPercent(task) === 0) return { ...nextTask, percentComplete: 25 };
    if (status === "open" || status === "blocked") return { ...nextTask, percentComplete: undefined };
    return nextTask;
  });
  markEdited();
}

function updateTaskProgress(taskId: string, percentValue: number) {
  if (!house.value) return;
  const percentComplete = clampTaskPercent(percentValue);
  const completedOn = localDateString();
  house.value = updateHouseTask(house.value, taskId, (task) => {
    if (percentComplete === 100) {
      return { ...task, status: "done", percentComplete: undefined, completedOn: task.completedOn || completedOn };
    }

    const status = task.status === "done" ? (percentComplete > 0 ? "in-progress" : "open") : task.status === "open" && percentComplete > 0 ? "in-progress" : task.status;
    return { ...task, status, percentComplete, completedOn: undefined };
  });
  markEdited();
}

function updateTaskCompletedOn(taskId: string, completedOn: string) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => {
    if (!completedOn) return { ...task, completedOn: undefined };
    return { ...task, status: "done", percentComplete: undefined, completedOn };
  });
  markEdited();
}

function updateTaskDoFirst(taskId: string, doFirst: boolean) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    doFirst,
    doFirstRank: doFirst ? task.doFirstRank ?? focusTasks.value.length + 1 : undefined
  }));
  markEdited();
}

function updateTaskDoFirstRank(taskId: string, rank: number | undefined) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    doFirstRank: rank && Number.isFinite(rank) ? Math.max(1, Math.round(rank)) : undefined
  }));
  markEdited();
}

function deleteTask(taskId: string) {
  if (!house.value) return;
  house.value = deleteHouseTask(house.value, taskId);
  markEdited();
}

function addTaskToRoom(floorKey: string, roomId: string, task: Task) {
  if (!house.value) return;
  const floor = house.value.floors[floorKey];
  const room = floor?.rooms.find((candidate) => candidate.id === roomId);
  if (!floor || !room) return;

  if (room.sharedTaskSet) {
    house.value = {
      ...house.value,
      roomTaskSets: {
        ...house.value.roomTaskSets,
        [room.sharedTaskSet]: [...(house.value.roomTaskSets[room.sharedTaskSet] ?? []), task]
      }
    };
    markEdited();
    return;
  }

  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [floorKey]: {
        ...floor,
        rooms: floor.rooms.map((candidate) => (candidate.id === roomId ? { ...candidate, tasks: [...(candidate.tasks ?? []), task] } : candidate))
      }
    }
  };
  markEdited();
}

function addTask() {
  if (!house.value) return;
  const task = newTask();

  if (taskFilter.value !== "rooms") {
    const group = house.value.taskGroups[taskFilter.value];
    if (!group) return;
    house.value = {
      ...house.value,
      taskGroups: {
        ...house.value.taskGroups,
        [taskFilter.value]: {
          ...group,
          tasks: [...group.tasks, task]
        }
      }
    };
    markEdited();
    return;
  }

  const floor = currentFloor.value;
  const room = selectedRoom.value;
  if (!floor || !room) return;

  addTaskToRoom(activeFloor.value, room.id, task);
}

function addTaskToTarget(target: string) {
  if (!house.value) return;
  const task = newTask();
  const [kind, first, second] = target.split(":");

  if (kind === "group") {
    const group = house.value.taskGroups[first];
    if (!group) return;
    house.value = {
      ...house.value,
      taskGroups: {
        ...house.value.taskGroups,
        [first]: { ...group, tasks: [...group.tasks, task] }
      }
    };
    markEdited();
    return;
  }

  if (kind === "room") {
    addTaskToRoom(first, second, task);
    return;
  }

  if (kind === "unplaced") {
    house.value = {
      ...house.value,
      unplacedRooms: (house.value.unplacedRooms ?? []).map((room) => (room.id === first ? { ...room, tasks: [...(room.tasks ?? []), task] } : room))
    };
    markEdited();
    return;
  }

  if (kind === "new-room") {
    const room = buildRoom(`New Room ${Object.values(house.value.floors[first]?.rooms ?? []).length + 1}`, task);
    addRoomToFloor(first, room);
    selectedRoomId.value = room.id;
    activeFloor.value = first;
    taskFilter.value = "rooms";
    return;
  }

  if (target === "new-unplaced-room") {
    const room = buildRoom(`Unplaced Room ${(house.value.unplacedRooms ?? []).length + 1}`, task);
    house.value = { ...house.value, unplacedRooms: [...(house.value.unplacedRooms ?? []), room] };
    markEdited();
  }
}

function updateSubtask(taskId: string, subtaskId: string, update: Partial<Subtask>) {
  if (!house.value) return;
  const completedOn = localDateString();
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    subtasks: (task.subtasks ?? []).map((subtask) => {
      if (subtask.id !== subtaskId) return subtask;
      const percentComplete = update.percentComplete === undefined ? subtask.percentComplete : clampTaskPercent(update.percentComplete);
      const status = update.status ?? (percentComplete === 100 ? "done" : subtask.status === "done" && percentComplete !== 100 ? "in-progress" : subtask.status);
      return {
        ...subtask,
        ...update,
        percentComplete: status === "in-progress" ? percentComplete : undefined,
        status,
        completedOn: status === "done" ? update.completedOn || subtask.completedOn || completedOn : undefined
      };
    })
  }));
  markEdited();
}

function addSubtask(taskId: string) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    subtasks: [...(task.subtasks ?? []), { id: uniqueId("subtask"), title: "New subtask", status: "open" }]
  }));
  markEdited();
}

function deleteSubtask(taskId: string, subtaskId: string) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    subtasks: (task.subtasks ?? []).filter((subtask) => subtask.id !== subtaskId)
  }));
  markEdited();
}

function addItem(taskId: string) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    itemsNeeded: [...(task.itemsNeeded ?? []), { name: "New item", status: "needed" }]
  }));
  markEdited();
}

function updateItem(taskId: string, itemIndex: number, update: Partial<ItemNeeded>) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    itemsNeeded: (task.itemsNeeded ?? []).map((item, index) => (index === itemIndex ? { ...item, ...update } : item))
  }));
  markEdited();
}

function deleteItem(taskId: string, itemIndex: number) {
  if (!house.value) return;
  house.value = updateHouseTask(house.value, taskId, (task) => ({
    ...task,
    itemsNeeded: (task.itemsNeeded ?? []).filter((_, index) => index !== itemIndex)
  }));
  markEdited();
}

function buildRoom(name: string, task?: Task): Room {
  return {
    id: uniqueId("room"),
    name,
    layout: {
      x: 10,
      y: 10,
      ...ROOM_SIZE_PRESETS.medium.wide
    },
    tasks: task ? [task] : []
  };
}

function addRoomToFloor(floorKey: string, room: Room) {
  if (!house.value) return;
  const floor = house.value.floors[floorKey];
  if (!floor) return;
  const roomNumber = floor.rooms.length + 1;
  const placedRoom = {
    ...room,
    layout: {
      ...room.layout,
      x: 6 + ((roomNumber * 7) % 42),
      y: 8 + ((roomNumber * 9) % 42)
    }
  };
  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [floorKey]: {
        ...floor,
        rooms: [...floor.rooms, placedRoom]
      }
    }
  };
  markEdited();
}

function addRoom() {
  if (!house.value || !currentFloor.value) return;
  const room = buildRoom(`New Room ${currentFloor.value.rooms.length + 1}`);
  addRoomToFloor(activeFloor.value, room);
  selectedRoomId.value = room.id;
  taskFilter.value = "rooms";
}

function updateRoomLayout(roomId: string, layout: Room["layout"]) {
  if (!house.value || !currentFloor.value) return;
  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [activeFloor.value]: {
        ...currentFloor.value,
        rooms: currentFloor.value.rooms.map((room) => (room.id === roomId ? { ...room, layout } : room))
      }
    }
  };
  markEdited();
}

function applySelectedRoomPreset(size: RoomSize, shape: RoomShape = selectedRoomShape.value) {
  if (!selectedRoom.value) return;
  const preset = ROOM_SIZE_PRESETS[size][shape];
  const current = selectedRoom.value.layout;
  updateRoomLayout(selectedRoom.value.id, {
    ...current,
    ...preset,
    x: Math.min(current.x, 100 - preset.w),
    y: Math.min(current.y, 100 - preset.h)
  });
}

function updateSelectedRoomName(name: string) {
  if (!house.value || !currentFloor.value || !selectedRoom.value || !name.trim()) return;
  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [activeFloor.value]: {
        ...currentFloor.value,
        rooms: currentFloor.value.rooms.map((room) => (room.id === selectedRoom.value?.id ? { ...room, name: name.trim() } : room))
      }
    }
  };
  markEdited();
}

function addFloor() {
  if (!house.value) return;
  const floorNumber = Object.keys(house.value.floors).length + 1;
  const floorKey = uniqueId(slugify(`floor-${floorNumber}`));
  const roomId = `${floorKey}-main-room`;
  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [floorKey]: {
        label: `Floor ${floorNumber}`,
        defaultRoom: roomId,
        grid: { columns: 100, rows: 100 },
        rooms: [
          {
            id: roomId,
            name: "Main Room",
            layout: { x: 8, y: 8, ...ROOM_SIZE_PRESETS.medium.wide },
            tasks: []
          }
        ]
      }
    }
  };
  activeFloor.value = floorKey;
  selectedRoomId.value = roomId;
  taskFilter.value = "rooms";
  markEdited();
}

function placeUnplacedRoom(roomId: string) {
  if (!house.value || !currentFloor.value) return;
  const room = (house.value.unplacedRooms ?? []).find((candidate) => candidate.id === roomId);
  if (!room) return;
  addRoomToFloor(activeFloor.value, room);
  house.value = {
    ...house.value,
    unplacedRooms: (house.value.unplacedRooms ?? []).filter((candidate) => candidate.id !== roomId)
  };
  selectedRoomId.value = room.id;
  taskFilter.value = "rooms";
  markEdited();
}

function updateCurrentFloorLabel(label: string) {
  if (!house.value || !currentFloor.value || !label.trim()) return;
  house.value = {
    ...house.value,
    floors: {
      ...house.value.floors,
      [activeFloor.value]: {
        ...currentFloor.value,
        label: label.trim()
      }
    }
  };
  markEdited();
}

async function saveHouse() {
  if (!house.value) return;
  const savedHouse = house.value;
  saveState.value = "saving";
  try {
    await saveCurrentHouse(savedHouse);
    dataSource.value = "api";
    if (house.value === savedHouse) {
      clearDraft();
      saveState.value = "saved";
    } else {
      writeDraft(house.value, true);
      saveState.value = "dirty";
    }
  } catch {
    writeDraft(house.value, true);
    saveState.value = "error";
  }
}
</script>

<template>
  <main class="app">
    <section class="hero" aria-labelledby="page-title">
      <div>
        <p class="eyebrow">Home project tracker</p>
        <h1 id="page-title">Room-by-room repair dashboard</h1>
      </div>
      <div class="hero-side">
        <p class="hero-copy">A glanceable plan for the urgent fixes, contractor calls, DIY projects, and slower room refresh work ahead.</p>
        <div class="sync-row">
          <span class="sync-pill">{{ syncLabel }}</span>
          <button class="save-button" type="button" :disabled="!house || saveState === 'saving'" @click="saveHouse">
            {{ saveButtonLabel }}
          </button>
        </div>
      </div>
    </section>

    <section v-if="!house" class="panel empty-state" aria-labelledby="empty-title">
      <div class="panel-header">
        <div>
          <h2 id="empty-title">No house loaded</h2>
          <p>Start with a blank editable plan, or add a few starter names first.</p>
        </div>
        <div class="panel-actions">
          <button class="text-button" type="button" @click="createBlankHouse">Create Blank House</button>
          <button class="text-button" type="button" @click="startGuidedSetup">Start Guided Setup</button>
        </div>
      </div>
      <div v-if="setupMode" class="setup-grid">
        <label class="field-label">
          <span>House Name</span>
          <input v-model="setupHouseName" class="field-control" type="text" placeholder="My Home" />
        </label>
        <label class="field-label">
          <span>First Floor</span>
          <input v-model="setupFloorName" class="field-control" type="text" placeholder="Main Floor" />
        </label>
        <label class="field-label">
          <span>First Room</span>
          <input v-model="setupRoomName" class="field-control" type="text" placeholder="Main Room" />
        </label>
        <div class="setup-actions">
          <button class="text-button" type="button" @click="finishGuidedSetup">Create House</button>
          <button class="text-button" type="button" @click="finishGuidedSetup">Exit Setup And Create</button>
        </div>
      </div>
      <div class="progress-grid">
        <div class="progress-stat"><strong>0</strong><span>Open</span></div>
        <div class="progress-stat"><strong>0</strong><span>Critical</span></div>
        <div class="progress-stat"><strong>0</strong><span>Contractor</span></div>
        <div class="progress-stat"><strong>0</strong><span>Total</span></div>
      </div>
    </section>

    <template v-else>
      <SummaryCards :done-count="progress.done" :next-focus="nextFocus" />

      <div class="view-toggle" aria-label="Dashboard view">
        <button class="view-button" type="button" :aria-pressed="viewMode === 'plan'" @click="viewMode = 'plan'">Interactive Floor Plan</button>
        <button class="view-button" type="button" :aria-pressed="viewMode === 'table'" @click="viewMode = 'table'">List All Tasks</button>
      </div>

      <section v-if="viewMode === 'plan' && currentFloor" class="workbench">
        <div class="panel">
          <div class="panel-header">
            <div>
              <h2>Interactive Floor Plan</h2>
              <p>Tap a room to see its repair list.</p>
            </div>
            <div class="panel-actions">
              <button class="text-button" type="button" @click="addFloor">Add Floor</button>
              <button class="text-button" type="button" @click="addRoom">Add Room</button>
            </div>
          </div>

          <div class="structure-edit-grid">
            <label class="field-label">
              <span>Floor Name</span>
              <input class="field-control" type="text" :value="currentFloor.label" @change="updateCurrentFloorLabel(($event.target as HTMLInputElement).value)" />
            </label>
            <label v-if="selectedRoom" class="field-label">
              <span>Room Name</span>
              <input class="field-control" type="text" :value="selectedRoom.name" @change="updateSelectedRoomName(($event.target as HTMLInputElement).value)" />
            </label>
            <div v-if="selectedRoom" class="preset-field" aria-label="Room size">
              <span>Size</span>
              <div class="segmented-control">
                <button class="segment-button" type="button" :aria-pressed="selectedRoomSize === 'small'" @click="applySelectedRoomPreset('small')">Small</button>
                <button class="segment-button" type="button" :aria-pressed="selectedRoomSize === 'medium'" @click="applySelectedRoomPreset('medium')">Medium</button>
                <button class="segment-button" type="button" :aria-pressed="selectedRoomSize === 'large'" @click="applySelectedRoomPreset('large')">Large</button>
              </div>
            </div>
            <div v-if="selectedRoom" class="preset-field" aria-label="Room shape">
              <span>Shape</span>
              <div class="segmented-control">
                <button class="segment-button" type="button" :aria-pressed="selectedRoomShape === 'square'" @click="applySelectedRoomPreset(selectedRoomSize, 'square')">Square</button>
                <button class="segment-button" type="button" :aria-pressed="selectedRoomShape === 'wide'" @click="applySelectedRoomPreset(selectedRoomSize, 'wide')">Wide</button>
                <button class="segment-button" type="button" :aria-pressed="selectedRoomShape === 'tall'" @click="applySelectedRoomPreset(selectedRoomSize, 'tall')">Tall</button>
              </div>
            </div>
          </div>

          <div class="tabs" role="tablist" aria-label="Floor tabs">
            <button
              v-for="(floor, floorKey) in house.floors"
              :key="floorKey"
              class="tab"
              type="button"
              role="tab"
              :aria-selected="activeFloor === floorKey"
              @click="selectFloor(String(floorKey))"
            >
              {{ floor.label }}
            </button>
          </div>

          <div class="filter-row" aria-label="Task filters">
            <button class="filter-chip" type="button" :aria-pressed="taskFilter === 'rooms'" @click="taskFilter = 'rooms'">Rooms</button>
            <button
              v-for="(group, groupKey) in house.taskGroups"
              :key="groupKey"
              class="filter-chip"
              type="button"
              :aria-pressed="taskFilter === groupKey"
              @click="taskFilter = String(groupKey)"
            >
              {{ group.label }}
            </button>
          </div>

          <FloorPlan
            :house="house"
            :floor-key="activeFloor"
            :floor="currentFloor"
            :selected-room-id="selectedRoomId"
            @select-room="selectRoom"
            @update-room-layout="updateRoomLayout"
          />
        </div>

        <TaskPanel
          :title="panelTitle"
          :subtitle="panelSubtitle"
          :tasks="panelTasks"
          @add-task="addTask"
          @update-task-title="updateTaskTitle"
          @update-task-priority="updateTaskPriority"
          @update-task-type="updateTaskType"
          @update-task-status="updateTaskStatus"
          @update-task-progress="updateTaskProgress"
          @update-task-completed-on="updateTaskCompletedOn"
          @update-task-do-first="updateTaskDoFirst"
          @update-task-do-first-rank="updateTaskDoFirstRank"
          @delete-task="deleteTask"
          @add-subtask="addSubtask"
          @update-subtask="updateSubtask"
          @delete-subtask="deleteSubtask"
          @add-item="addItem"
          @update-item="updateItem"
          @delete-item="deleteItem"
        />
      </section>

      <section class="workbench secondary-workbench">
        <FocusList
          :tasks="focusTasks"
          @update-task-title="updateTaskTitle"
          @update-task-priority="updateTaskPriority"
          @update-task-type="updateTaskType"
          @update-task-status="updateTaskStatus"
          @update-task-progress="updateTaskProgress"
          @update-task-completed-on="updateTaskCompletedOn"
          @update-task-do-first="updateTaskDoFirst"
          @update-task-do-first-rank="updateTaskDoFirstRank"
          @delete-task="deleteTask"
          @add-subtask="addSubtask"
          @update-subtask="updateSubtask"
          @delete-subtask="deleteSubtask"
          @add-item="addItem"
          @update-item="updateItem"
          @delete-item="deleteItem"
        />
        <ProgressPanel v-bind="progress" />
      </section>

      <section v-if="unplacedRooms.length" class="panel unplaced-panel" aria-labelledby="unplaced-title">
        <div class="panel-header">
          <div>
            <h2 id="unplaced-title">Unplaced Rooms</h2>
            <p>Rooms created from task planning can be placed onto the active floor later.</p>
          </div>
        </div>
        <ul class="unplaced-list">
          <li v-for="room in unplacedRooms" :key="room.id" class="unplaced-room">
            <div>
              <strong>{{ room.name }}</strong>
              <span>{{ room.tasks?.length ?? 0 }} tasks</span>
            </div>
            <button class="text-button" type="button" @click="placeUnplacedRoom(room.id)">Place on {{ currentFloor?.label ?? "floor" }}</button>
          </li>
        </ul>
      </section>

      <TaskTable
        v-if="viewMode === 'table'"
        :tasks="taskTableRows"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :task-targets="taskTargetOptions"
        @sort="changeSort"
        @select-task="selectTask"
        @add-task="addTaskToTarget"
        @update-task-title="updateTaskTitle"
        @update-task-priority="updateTaskPriority"
        @update-task-type="updateTaskType"
        @update-task-status="updateTaskStatus"
        @update-task-progress="updateTaskProgress"
        @update-task-completed-on="updateTaskCompletedOn"
        @delete-task="deleteTask"
      />

      <p v-if="saveState === 'error'" class="error-note">Could not save to the API. Your local draft is still available.</p>
    </template>
  </main>
</template>
