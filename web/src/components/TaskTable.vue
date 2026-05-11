<script setup lang="ts">
import type { SortDirection, SortKey } from "../utils/house";
import type { TaskWithContext } from "../types/house";
import { badgeClass, badgeLabel, formatDate, statusLabel, taskPercent } from "../utils/house";

defineProps<{
  tasks: TaskWithContext[];
  sortKey: SortKey;
  sortDirection: SortDirection;
}>();

defineEmits<{
  sort: [key: SortKey];
  selectTask: [task: TaskWithContext];
}>();

const columns: Array<{ key: SortKey; label: string }> = [
  { key: "title", label: "Task" },
  { key: "area", label: "Area" },
  { key: "priority", label: "Priority" },
  { key: "type", label: "Type" },
  { key: "status", label: "Status" },
  { key: "percentComplete", label: "Progress" },
  { key: "dateStarted", label: "Started" },
  { key: "completedOn", label: "Completed" }
];
</script>

<template>
  <section class="panel table-panel" aria-labelledby="all-tasks-title">
    <div class="panel-header">
      <div>
        <h2 id="all-tasks-title">All Tasks</h2>
        <p>Tap a header to sort. Select a row to open its room or task group.</p>
      </div>
    </div>
    <div class="table-wrap">
      <table class="task-table">
        <thead>
          <tr>
            <th v-for="column in columns" :key="column.key" scope="col">
              <button class="sort-button" type="button" @click="$emit('sort', column.key)">
                {{ column.label }}
                <span class="sort-indicator">{{ sortKey === column.key ? (sortDirection === "asc" ? "↑" : "↓") : "" }}</span>
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="`${task.groupKey || task.roomId}-${task.id}`" @click="$emit('selectTask', task)">
            <td><button class="table-task-button" type="button">{{ task.title }}</button></td>
            <td>{{ task.area }}</td>
            <td><span class="badge" :class="badgeClass(task)">{{ badgeLabel(task) }}</span></td>
            <td>{{ task.type }}</td>
            <td>{{ statusLabel(task) }}</td>
            <td>{{ taskPercent(task) }}%</td>
            <td>{{ formatDate(task.dateStarted) }}</td>
            <td>{{ formatDate(task.completedOn) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
