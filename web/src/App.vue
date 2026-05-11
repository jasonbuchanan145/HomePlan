<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import FloorPlan from "./components/FloorPlan.vue";
import FocusList from "./components/FocusList.vue";
import ProgressPanel from "./components/ProgressPanel.vue";
import SummaryCards from "./components/SummaryCards.vue";
import TaskPanel from "./components/TaskPanel.vue";
import TaskTable from "./components/TaskTable.vue";
import { loadCurrentHouse, saveCurrentHouse } from "./services/api";
import type { HouseState, TaskWithContext } from "./types/house";
import { allTasks, getRoomTasks, sortedTasks, type SortDirection, type SortKey } from "./utils/house";

type ViewMode = "plan" | "table";

const house = ref<HouseState | null>(null);
const activeFloor = ref("top");
const selectedRoomId = ref("top-stairs");
const taskFilter = ref("rooms");
const viewMode = ref<ViewMode>("plan");
const sortKey = ref<SortKey>("priority");
const sortDirection = ref<SortDirection>("asc");
const dataSource = ref<"api" | "seed">("seed");
const saveState = ref<"idle" | "saving" | "saved" | "error">("idle");

const currentFloor = computed(() => house.value?.floors[activeFloor.value]);
const tasks = computed(() => (house.value ? allTasks(house.value) : []));
const taskTableRows = computed(() => sortedTasks(tasks.value, sortKey.value, sortDirection.value));

const selectedRoom = computed(() => {
  const floor = currentFloor.value;
  if (!floor) return null;
  return floor.rooms.find((room) => room.id === selectedRoomId.value) ?? floor.rooms[0] ?? null;
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
  const response = await loadCurrentHouse();
  house.value = response.house;
  dataSource.value = response.source;
  activeFloor.value = "top";
  selectedRoomId.value = response.house.floors.top?.defaultRoom ?? Object.values(response.house.floors)[0]?.defaultRoom ?? "";
});

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
  if (task.groupKey) {
    taskFilter.value = task.groupKey;
  } else {
    activeFloor.value = task.floorKey;
    selectedRoomId.value = task.roomId;
    taskFilter.value = "rooms";
  }
  viewMode.value = "plan";
}

async function saveHouse() {
  if (!house.value) return;
  saveState.value = "saving";
  try {
    await saveCurrentHouse(house.value);
    dataSource.value = "api";
    saveState.value = "saved";
  } catch {
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
          <span class="sync-pill">{{ dataSource === "api" ? "Cloud session" : "Seed data" }}</span>
          <button class="save-button" type="button" :disabled="!house || saveState === 'saving'" @click="saveHouse">
            {{ saveState === "saving" ? "Saving" : saveState === "saved" ? "Saved" : "Save" }}
          </button>
        </div>
      </div>
    </section>

    <p v-if="!house" class="empty-note">Loading HomePlan...</p>

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

          <FloorPlan :house="house" :floor-key="activeFloor" :floor="currentFloor" :selected-room-id="selectedRoomId" @select-room="selectRoom" />
        </div>

        <TaskPanel :title="panelTitle" :subtitle="panelSubtitle" :tasks="panelTasks" />
      </section>

      <section class="workbench secondary-workbench">
        <FocusList :tasks="focusTasks" />
        <ProgressPanel v-bind="progress" />
      </section>

      <TaskTable
        v-if="viewMode === 'table'"
        :tasks="taskTableRows"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        @sort="changeSort"
        @select-task="selectTask"
      />

      <p v-if="saveState === 'error'" class="error-note">Could not save to the API. The local view is still available.</p>
    </template>
  </main>
</template>
