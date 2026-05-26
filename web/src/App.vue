<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import FloorPlan from "./components/FloorPlan.vue";
import FocusList from "./components/FocusList.vue";
import ProgressPanel from "./components/ProgressPanel.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TaskCreateForm from "./components/TaskCreateForm.vue";
import TaskPanel from "./components/TaskPanel.vue";
import TaskTable from "./components/TaskTable.vue";
import { deleteCurrentHouse, loadCurrentHouse, loadMe, logout as logoutSession, saveCurrentHouse, type MeResponse } from "./services/api";
import type { HouseState, ItemNeeded, Room, Subtask, Task, TaskCreateDraft, TaskPriority, TaskStatus, TaskTargetOption, TaskType, TaskWithContext } from "./types/house";
import { allTasks, clampTaskPercent, deleteHouseTask, getRoomTasks, sortedTasks, taskPercent, updateHouseTask, type SortDirection, type SortKey } from "./utils/house";

type ViewMode = "plan" | "table";
type SaveState = "idle" | "dirty" | "saving" | "saved" | "error";
type ResetState = "idle" | "resetting" | "error";
type RoomSize = "small" | "medium" | "large";
type RoomShape = "square" | "wide" | "tall";
type OnboardingMode = "projects" | "rooms" | null;
type ProjectSetupFloorKey = "main" | "second" | "exterior";
type PageMode = "app" | "privacy" | "cookies";

interface HouseDraft {
  house: HouseState;
  updatedAt: number;
  dirty: boolean;
}

interface ProjectSetupRow {
  id: string;
  floorKey: ProjectSetupFloorKey;
  areaName: string;
  taskTitles: string[];
}

interface FloorSetupRow {
  id: string;
  name: string;
  rooms: string[];
}

const DRAFT_KEY = "homeplan.houseDraft";
const anonymousCookieNotice = "Saving uses an essential session cookie so this browser can reopen this house plan.";
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
const PROJECT_SETUP_FLOORS: Array<{ value: ProjectSetupFloorKey; label: string }> = [
  { value: "main", label: "Main Floor" },
  { value: "second", label: "Floor 2" },
  { value: "exterior", label: "Exterior" }
];
const COMMON_PROJECT_AREAS = [
  "Kitchen",
  "Living Room",
  "Main Bedroom",
  "Bedroom 2",
  "Bedroom 3",
  "Family Room",
  "Dining Room",
  "Main Bathroom 1",
  "Main Bathroom 2",
  "Main Bathroom 3",
  "Laundry Room",
  "Basement",
  "Garage",
  "Hallway",
  "Entry",
  "Office",
  "Exterior"
];

const house = ref<HouseState | null>(null);
const pageMode = ref<PageMode>(pathPageMode());
const me = ref<MeResponse>({ authenticated: false, user: null, apps: { homeplan: { canAccess: false, canUseAI: false } } });
const authState = ref<"loading" | "ready" | "error">("loading");
const activeFloor = ref("top");
const selectedRoomId = ref("top-stairs");
const taskFilter = ref("rooms");
const viewMode = ref<ViewMode>("plan");
const sortKey = ref<SortKey>("priority");
const sortDirection = ref<SortDirection>("asc");
const dataSource = ref<"api" | "empty">("empty");
const saveState = ref<SaveState>("idle");
const resetState = ref<ResetState>("idle");
const onboardingMode = ref<OnboardingMode>(null);
const projectSetupHouseName = ref("My Home");
const projectSetupRows = ref<ProjectSetupRow[]>([
  { id: uniqueId("project-area"), floorKey: "main", areaName: "Kitchen", taskTitles: [""] }
]);
const roomSetupHouseName = ref("My Home");
const roomSetupFloors = ref<FloorSetupRow[]>([
  { id: uniqueId("setup-floor"), name: "Main Floor", rooms: ["Kitchen", "Living Room", "Bathroom"] },
  { id: uniqueId("setup-floor"), name: "Second Floor", rooms: ["Bedroom", "Office"] }
]);
const taskCreateOpen = ref(false);
const taskCreateDefaultTarget = ref("");
const taskCreateLockTarget = ref(false);

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
const accountLabel = computed(() => {
  if (authState.value === "loading") return "Checking account";
  if (me.value.authenticated && me.value.user) return me.value.user.displayName || me.value.user.email;
  return "Not signed in";
});
const startOverLabel = computed(() => (resetState.value === "resetting" ? "Clearing" : "Start Over"));
const taskTargetOptions = computed<TaskTargetOption[]>(() => {
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
    { value: `new-room:${activeFloor.value}`, label: `Create new room on ${currentFloor.value?.label ?? "current floor"}`, needsRoomName: true },
    { value: "new-unplaced-room", label: "Create unplaced room", needsRoomName: true }
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
  window.addEventListener("popstate", syncPageMode);
  await refreshMe();
  if (pageMode.value !== "app") return;
  const draft = readDraft();
  try {
    const response = await loadCurrentHouse();
    if (house.value || saveState.value === "dirty" || saveState.value === "saving") return;
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
    if (house.value || saveState.value === "dirty" || saveState.value === "saving") return;
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
  window.removeEventListener("popstate", syncPageMode);
});

function pathPageMode(): PageMode {
  if (window.location.pathname === "/privacy") return "privacy";
  if (window.location.pathname === "/cookies") return "cookies";
  return "app";
}

function syncPageMode() {
  pageMode.value = pathPageMode();
}

async function refreshMe() {
  authState.value = "loading";
  try {
    me.value = await loadMe();
    authState.value = "ready";
  } catch {
    me.value = { authenticated: false, user: null, apps: { homeplan: { canAccess: false, canUseAI: false } } };
    authState.value = "error";
  }
}

async function signOut() {
  try {
    await logoutSession();
  } finally {
    await refreshMe();
  }
}

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
  viewMode.value = "plan";
  onboardingMode.value = null;
  resetState.value = "idle";
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

function defaultRoomLayout(index: number): Room["layout"] {
  const preset = ROOM_SIZE_PRESETS.medium.wide;
  const column = index % 2;
  const row = Math.floor(index / 2) % 3;
  return {
    x: 6 + column * 44,
    y: 8 + row * 30,
    ...preset
  };
}

function roomFromSetup(name: string, index: number, tasks: Task[] = [], tone?: Room["display"]): Room {
  const roomName = name.trim() || `Room ${index + 1}`;
  return {
    id: uniqueId(slugify(roomName)),
    name: roomName,
    layout: defaultRoomLayout(index),
    display: tone,
    tasks
  };
}

function taskFromSetupTitle(title: string): Task {
  return newTask({
    title: title.trim(),
    priority: "important",
    type: "DIY",
    status: "open",
    subtasks: [],
    itemsNeeded: []
  });
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

function newTask(draft: Pick<TaskCreateDraft, "title" | "priority" | "type" | "status" | "notes" | "subtasks" | "itemsNeeded">): Task {
  const task: Task = {
    id: uniqueId("task"),
    title: draft.title,
    priority: draft.priority,
    type: draft.type,
    status: draft.status,
    notes: draft.notes,
    subtasks: draft.subtasks.map((subtask) => ({ id: uniqueId("subtask"), title: subtask.title, status: subtask.status })),
    itemsNeeded: draft.itemsNeeded.map((item) => ({ name: item.name, status: item.status || "needed" }))
  };
  if (draft.status === "in-progress") task.percentComplete = 25;
  if (draft.status === "done") task.completedOn = localDateString();
  return task;
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
    dataSource.value = "api";
    if (house.value === nextHouse) {
      clearDraft();
      saveState.value = "saved";
    } else if (house.value) {
      writeDraft(house.value, true);
      saveState.value = "dirty";
    }
  } catch {
    saveState.value = "error";
  }
}

function createBlankHouse() {
  void createAndSaveHouse(starterHouse());
}

function selectOnboardingMode(mode: Exclude<OnboardingMode, null>) {
  onboardingMode.value = mode;
}

function addProjectSetupRow() {
  projectSetupRows.value.push({ id: uniqueId("project-area"), floorKey: "main", areaName: "", taskTitles: [""] });
}

function removeProjectSetupRow(rowId: string) {
  projectSetupRows.value = projectSetupRows.value.filter((row) => row.id !== rowId);
  if (!projectSetupRows.value.length) addProjectSetupRow();
}

function addProjectTask(row: ProjectSetupRow) {
  row.taskTitles.push("");
}

function removeProjectTask(row: ProjectSetupRow, taskIndex: number) {
  row.taskTitles.splice(taskIndex, 1);
  if (!row.taskTitles.length) row.taskTitles.push("");
}

function createHouseFromProjects() {
  const rows = projectSetupRows.value
    .map((row) => ({
      floorKey: row.floorKey,
      areaName: row.areaName.trim(),
      taskTitles: row.taskTitles.map((title) => title.trim()).filter(Boolean)
    }))
    .filter((row) => row.areaName || row.taskTitles.length);

  const setupRows = rows.length ? rows : [{ floorKey: "main" as ProjectSetupFloorKey, areaName: "Main Room", taskTitles: [] }];
  const floors = Object.fromEntries(
    PROJECT_SETUP_FLOORS.map((floorOption) => {
      const floorRows = setupRows.filter((row) => row.floorKey === floorOption.value);
      if (!floorRows.length) return null;
      const rooms = floorRows.map((row, index) =>
        roomFromSetup(
          row.areaName || `Project Area ${index + 1}`,
          index,
          row.taskTitles.map(taskFromSetupTitle),
          floorOption.value === "exterior" ? { tone: "outdoor", borderStyle: "dashed" } : undefined
        )
      );
      return [
        floorOption.value,
        {
          label: floorOption.label,
          defaultRoom: rooms[0].id,
          grid: { columns: 100, rows: 100 },
          rooms
        }
      ];
    }).filter(Boolean) as Array<[ProjectSetupFloorKey, HouseState["floors"][string]]>
  );

  void createAndSaveHouse({
    schemaVersion: 1,
    id: uniqueId("house"),
    name: projectSetupHouseName.value.trim() || "My Home",
    taskGroups: {
      wholeHouse: {
        label: "Whole House",
        description: "Projects that cut across multiple rooms.",
        tasks: []
      }
    },
    roomTaskSets: {},
    floors,
    unplacedRooms: []
  });
}

function addFloorSetupRow() {
  const floorNumber = roomSetupFloors.value.length + 1;
  roomSetupFloors.value.push({ id: uniqueId("setup-floor"), name: `Floor ${floorNumber}`, rooms: [""] });
}

function removeFloorSetupRow(floorId: string) {
  roomSetupFloors.value = roomSetupFloors.value.filter((floor) => floor.id !== floorId);
  if (!roomSetupFloors.value.length) addFloorSetupRow();
}

function addSetupRoom(floor: FloorSetupRow) {
  floor.rooms.push("");
}

function removeSetupRoom(floor: FloorSetupRow, roomIndex: number) {
  floor.rooms.splice(roomIndex, 1);
  if (!floor.rooms.length) floor.rooms.push("");
}

function createHouseFromRooms() {
  const floors = roomSetupFloors.value
    .map((floor, floorIndex) => {
      const roomNames = floor.rooms.map((room) => room.trim()).filter(Boolean);
      return {
        label: floor.name.trim() || `Floor ${floorIndex + 1}`,
        roomNames: roomNames.length ? roomNames : ["Main Room"]
      };
    })
    .filter((floor) => floor.label || floor.roomNames.length);

  const setupFloors = floors.length ? floors : [{ label: "Main Floor", roomNames: ["Main Room"] }];
  const floorEntries = setupFloors.map((floor, floorIndex) => {
    const floorKey = floorIndex === 0 ? "main" : uniqueId(slugify(floor.label));
    const rooms = floor.roomNames.map((roomName, roomIndex) => roomFromSetup(roomName, roomIndex));
    return [
      floorKey,
      {
        label: floor.label,
        defaultRoom: rooms[0].id,
        grid: { columns: 100, rows: 100 },
        rooms
      }
    ] as const;
  });

  void createAndSaveHouse({
    schemaVersion: 1,
    id: uniqueId("house"),
    name: roomSetupHouseName.value.trim() || "My Home",
    taskGroups: {
      wholeHouse: {
        label: "Whole House",
        description: "Projects that cut across multiple rooms.",
        tasks: []
      }
    },
    roomTaskSets: {},
    floors: Object.fromEntries(floorEntries),
    unplacedRooms: []
  });
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

function currentPanelTaskTarget(): string {
  if (!house.value) return "";
  if (taskFilter.value !== "rooms" && house.value.taskGroups[taskFilter.value]) return `group:${taskFilter.value}`;
  if (currentFloor.value && selectedRoom.value) return `room:${activeFloor.value}:${selectedRoom.value.id}`;
  return "";
}

function openTaskCreateForm(defaultTarget = "", lockTarget = false) {
  taskCreateDefaultTarget.value = defaultTarget;
  taskCreateLockTarget.value = lockTarget;
  taskCreateOpen.value = true;
}

function closeTaskCreateForm() {
  taskCreateOpen.value = false;
}

function openPanelTaskForm() {
  const target = currentPanelTaskTarget();
  openTaskCreateForm(target, Boolean(target));
}

function openAllTasksTaskForm() {
  openTaskCreateForm("", false);
}

function createTaskFromDraft(draft: TaskCreateDraft) {
  if (!house.value) return;
  const task = newTask(draft);

  if (draft.target === "new-unplaced-room") {
    const room = buildRoom(draft.roomName || `Unplaced Room ${(house.value.unplacedRooms ?? []).length + 1}`, task);
    house.value = { ...house.value, unplacedRooms: [...(house.value.unplacedRooms ?? []), room] };
    markEdited();
    closeTaskCreateForm();
    return;
  }

  const [kind, first, second] = draft.target.split(":");

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
    closeTaskCreateForm();
    return;
  }

  if (kind === "room") {
    addTaskToRoom(first, second, task);
    closeTaskCreateForm();
    return;
  }

  if (kind === "unplaced") {
    house.value = {
      ...house.value,
      unplacedRooms: (house.value.unplacedRooms ?? []).map((room) => (room.id === first ? { ...room, tasks: [...(room.tasks ?? []), task] } : room))
    };
    markEdited();
    closeTaskCreateForm();
    return;
  }

  if (kind === "new-room") {
    const room = buildRoom(draft.roomName || `New Room ${Object.values(house.value.floors[first]?.rooms ?? []).length + 1}`, task);
    addRoomToFloor(first, room);
    selectedRoomId.value = room.id;
    activeFloor.value = first;
    taskFilter.value = "rooms";
    closeTaskCreateForm();
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

async function startOver() {
  if (!house.value || resetState.value === "resetting") return;
  resetState.value = "resetting";
  try {
    await deleteCurrentHouse();
    clearDraft();
    house.value = null;
    dataSource.value = "empty";
    saveState.value = "idle";
    onboardingMode.value = null;
    taskCreateOpen.value = false;
    resetState.value = "idle";
  } catch {
    resetState.value = "error";
    saveState.value = "error";
  }
}
</script>

<template>
  <main class="app">
    <section v-if="pageMode === 'privacy'" class="policy-page" aria-labelledby="privacy-title">
      <a class="text-button policy-back" href="/">Back to HomePlan</a>
      <p class="eyebrow">Privacy</p>
      <h1 id="privacy-title">Privacy Policy</h1>
      <p class="policy-updated">Last updated May 26, 2026.</p>
      <div class="policy-copy">
        <p>HomePlan helps you create and save room-by-room home project plans. We collect only the information needed to run the planner, keep your work available, and support optional account features.</p>
        <h2>Information We Collect</h2>
        <p>Anonymous plans use an essential browser session cookie so this browser can save and reopen a house plan. Saved plans include the rooms, tasks, notes, materials, and progress details you enter.</p>
        <p>If you sign in with Google, we use your Google account ID, email address, display name, and profile image to create your account and session. We do not request Google Drive, Gmail, Calendar, or other Google content scopes.</p>
        <h2>How We Use Information</h2>
        <p>We use your information to load and save your house plan, maintain your signed-in session, show account status, and apply HomePlan entitlements such as future AI access.</p>
        <p>AI planning is not active yet. When added, it will be backend-only, entitlement-gated, and designed to return proposed subtasks or item suggestions without automatically changing your plan. The frontend does not call OpenAI directly.</p>
        <h2>Sharing And Tracking</h2>
        <p>HomePlan does not use advertising cookies, cross-site tracking, marketing pixels, or analytics cookies. We do not sell personal information.</p>
        <h2>Deletion</h2>
        <p>You can clear the current house from the app with Start Over. For account or saved data deletion requests, contact the site operator for your HomePlan deployment.</p>
      </div>
      <nav class="policy-links" aria-label="Policy links">
        <a href="/cookies">Cookie Policy</a>
        <a href="/">HomePlan</a>
      </nav>
    </section>

    <section v-else-if="pageMode === 'cookies'" class="policy-page" aria-labelledby="cookies-title">
      <a class="text-button policy-back" href="/">Back to HomePlan</a>
      <p class="eyebrow">Cookies</p>
      <h1 id="cookies-title">Cookie Policy</h1>
      <p class="policy-updated">Last updated May 26, 2026.</p>
      <div class="policy-copy">
        <p>HomePlan currently uses only essential cookies required for requested app features. There are no analytics, advertising, cross-site tracking, or marketing cookies.</p>
        <h2>Essential Cookies</h2>
        <p><strong>homeplan_session</strong> keeps an anonymous house plan associated with this browser so you can save and reopen it. It is currently configured for a 14-day anonymous session.</p>
        <p><strong>homeplan_auth</strong> keeps you signed in after Google authentication. It is currently configured for a 30-day signed-in session.</p>
        <h2>Why There Is No Cookie Banner</h2>
        <p>These cookies are used only to provide the save, reopen, and sign-in features you choose to use. If HomePlan later adds analytics, ads, A/B testing, or other nonessential tracking, the cookie experience will be updated before those tools are enabled.</p>
      </div>
      <nav class="policy-links" aria-label="Policy links">
        <a href="/privacy">Privacy Policy</a>
        <a href="/">HomePlan</a>
      </nav>
    </section>

    <template v-else>
    <section class="hero" aria-labelledby="page-title">
      <div>
        <p class="eyebrow">Home project tracker</p>
        <h1 id="page-title">Room-by-room repair dashboard</h1>
      </div>
      <div class="hero-side">
        <p class="hero-copy">A glanceable plan for the urgent fixes, contractor calls, DIY projects, and slower room refresh work ahead.</p>
        <div class="account-row" aria-label="Account status">
          <span class="sync-pill">{{ accountLabel }}</span>
          <a v-if="!me.authenticated" class="text-button" href="/api/auth/google/start">Sign in with Google</a>
          <button v-else class="text-button" type="button" @click="signOut">Sign out</button>
        </div>
        <div class="sync-row">
          <span class="sync-pill">{{ syncLabel }}</span>
          <button class="save-button" type="button" :disabled="!house || saveState === 'saving'" @click="saveHouse">
            {{ saveButtonLabel }}
          </button>
          <button class="text-button danger-button" type="button" :disabled="!house || resetState === 'resetting'" @click="startOver">
            {{ startOverLabel }}
          </button>
        </div>
      </div>
    </section>

    <section v-if="!house" class="empty-state" aria-labelledby="empty-title">
      <div class="splash-grid">
        <div class="empty-intro">
          <p class="eyebrow">Set up your home</p>
          <h2 id="empty-title">Plan home projects by room, not just by list.</h2>
          <p>Map the house, attach the work to each area, and keep repairs, materials, and progress in one place.</p>
        </div>

        <div class="splash-preview" role="group" aria-label="Static example of a room-by-room home project plan">
          <div class="preview-plan" aria-hidden="true">
            <div class="preview-room preview-kitchen is-active">
              <strong>Kitchen</strong>
              <span>3 tasks</span>
            </div>
            <div class="preview-room preview-living">
              <strong>Living Room</strong>
              <span>1 task</span>
            </div>
            <div class="preview-room preview-bedroom">
              <strong>Main Bedroom</strong>
              <span>2 tasks</span>
            </div>
            <div class="preview-room preview-bath">
              <strong>Bathroom</strong>
              <span>1 blocked</span>
            </div>
            <div class="preview-room preview-exterior">
              <strong>Exterior</strong>
              <span>seasonal</span>
            </div>
          </div>

          <div class="preview-details">
            <p class="preview-label">Example plan</p>
            <h3>Kitchen • 3 open tasks</h3>
            <ul class="preview-task-list" aria-label="Example kitchen tasks">
              <li><span class="badge urgent">urgent</span> Replace loose outlet cover</li>
              <li><span class="badge contractor">contractor</span> Book electrician</li>
              <li><span class="badge diy">DIY</span> Measure cabinet hinges</li>
            </ul>
            <p class="preview-next">Next: book electrician</p>
          </div>
        </div>
      </div>

      <div class="onboarding-choice-grid" aria-label="Setup paths">
        <button class="setup-card" type="button" aria-label="Start with projects" :aria-pressed="onboardingMode === 'projects'" @click="selectOnboardingMode('projects')">
          <span class="setup-card-kicker">Have a punch list?</span>
          <strong>Start with projects</strong>
          <span>Turn a punch list into rooms with tasks attached.</span>
        </button>
        <button class="setup-card" type="button" aria-label="Start with rooms" :aria-pressed="onboardingMode === 'rooms'" @click="selectOnboardingMode('rooms')">
          <span class="setup-card-kicker">Mapping the house?</span>
          <strong>Start with rooms</strong>
          <span>Map floors and rooms first, then add work as you walk through.</span>
        </button>
      </div>

      <div v-if="onboardingMode === 'projects'" class="panel onboarding-panel" aria-labelledby="projects-setup-title">
        <div class="panel-header">
          <div>
            <h3 id="projects-setup-title">Start from a repair list</h3>
            <p>Pick the floor or exterior area first, then add the tasks you already know about.</p>
          </div>
          <button class="text-button" type="button" @click="addProjectSetupRow">Add More Areas Or Rooms</button>
        </div>
        <label class="field-label onboarding-house-name">
          <span>House Name</span>
          <input v-model="projectSetupHouseName" class="field-control" type="text" placeholder="My Home" />
        </label>
        <div class="project-setup-list">
          <article v-for="row in projectSetupRows" :key="row.id" class="setup-row-card">
            <div class="setup-row-fields">
              <label class="field-label">
                <span>Floor Or Area</span>
                <select v-model="row.floorKey" class="field-control">
                  <option v-for="floorOption in PROJECT_SETUP_FLOORS" :key="floorOption.value" :value="floorOption.value">{{ floorOption.label }}</option>
                </select>
              </label>
              <label class="field-label">
                <span>Area Or Room</span>
                <select v-model="row.areaName" class="field-control">
                  <option value="">Choose an area</option>
                  <option v-for="areaName in COMMON_PROJECT_AREAS" :key="areaName" :value="areaName">{{ areaName }}</option>
                </select>
              </label>
              <button class="text-button danger-button" type="button" @click="removeProjectSetupRow(row.id)">Remove</button>
            </div>
            <div class="setup-task-list">
              <label v-for="(_, taskIndex) in row.taskTitles" :key="`${row.id}-${taskIndex}`" class="field-label">
                <span>Task {{ taskIndex + 1 }}</span>
                <input v-model="row.taskTitles[taskIndex]" class="field-control" type="text" placeholder="What needs to happen?" />
              </label>
            </div>
            <div class="setup-actions">
              <button class="text-button" type="button" @click="addProjectTask(row)">Add Task Line</button>
              <button v-if="row.taskTitles.length > 1" class="text-button danger-button" type="button" @click="removeProjectTask(row, row.taskTitles.length - 1)">Remove Last Task</button>
            </div>
          </article>
        </div>
        <div class="setup-submit-row">
          <button class="save-button" type="button" @click="createHouseFromProjects">Create House From Projects</button>
        </div>
      </div>

      <div v-if="onboardingMode === 'rooms'" class="panel onboarding-panel" aria-labelledby="rooms-setup-title">
        <div class="panel-header">
          <div>
            <h3 id="rooms-setup-title">Map rooms first</h3>
            <p>Name the floors you care about now. You can rename, resize, and add more rooms later.</p>
          </div>
          <button class="text-button" type="button" @click="addFloorSetupRow">Add Floor</button>
        </div>
        <label class="field-label onboarding-house-name">
          <span>House Name</span>
          <input v-model="roomSetupHouseName" class="field-control" type="text" placeholder="My Home" />
        </label>
        <div class="floor-setup-list">
          <article v-for="floor in roomSetupFloors" :key="floor.id" class="setup-row-card">
            <div class="setup-row-header">
              <label class="field-label">
                <span>Floor Name</span>
                <input v-model="floor.name" class="field-control" type="text" placeholder="Main Floor" />
              </label>
              <button class="text-button danger-button" type="button" @click="removeFloorSetupRow(floor.id)">Remove</button>
            </div>
            <div class="setup-room-list">
              <label v-for="(_, roomIndex) in floor.rooms" :key="`${floor.id}-${roomIndex}`" class="field-label">
                <span>Room {{ roomIndex + 1 }}</span>
                <input v-model="floor.rooms[roomIndex]" class="field-control" type="text" placeholder="Kitchen, Bedroom, Hall" />
              </label>
            </div>
            <div class="setup-actions">
              <button class="text-button" type="button" @click="addSetupRoom(floor)">Add Room</button>
              <button v-if="floor.rooms.length > 1" class="text-button danger-button" type="button" @click="removeSetupRoom(floor, floor.rooms.length - 1)">Remove Last Room</button>
            </div>
          </article>
        </div>
        <div class="setup-submit-row">
          <button class="save-button" type="button" @click="createHouseFromRooms">Create House From Rooms</button>
        </div>
      </div>

      <div class="blank-house-row">
        <button class="text-button" type="button" @click="createBlankHouse">Create Blank House</button>
      </div>
      <p class="cookie-save-note">{{ anonymousCookieNotice }}</p>
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
          @add-task="openPanelTaskForm"
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
        @add-task="openAllTasksTaskForm"
        @update-task-title="updateTaskTitle"
        @update-task-priority="updateTaskPriority"
        @update-task-type="updateTaskType"
        @update-task-status="updateTaskStatus"
        @update-task-progress="updateTaskProgress"
        @update-task-completed-on="updateTaskCompletedOn"
        @delete-task="deleteTask"
      />

      <p v-if="saveState === 'error'" class="error-note">Could not save to the API. Your local draft is still available.</p>
      <p v-if="resetState === 'error'" class="error-note">Could not clear the saved house. Try again in a moment.</p>
    </template>

    <TaskCreateForm
      :open="taskCreateOpen"
      :target-options="taskTargetOptions"
      :default-target="taskCreateDefaultTarget"
      :lock-target="taskCreateLockTarget"
      @close="closeTaskCreateForm"
      @create-task="createTaskFromDraft"
    />

    <footer class="app-footer">
      <a href="/privacy">Privacy Policy</a>
      <a href="/cookies">Cookie Policy</a>
    </footer>
    </template>
  </main>
</template>
