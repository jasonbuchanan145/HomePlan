<script setup lang="ts">
import { ref } from "vue";
import type { SortDirection, SortKey } from "../utils/house";
import type { TaskPriority, TaskStatus, TaskType, TaskWithContext } from "../types/house";
import { badgeClass, badgeLabel, formatDate, statusLabel, TASK_PRIORITIES, TASK_STATUSES, TASK_TYPES, taskPercent } from "../utils/house";

defineProps<{
  tasks: TaskWithContext[];
  sortKey: SortKey;
  sortDirection: SortDirection;
  taskTargets: Array<{ value: string; label: string }>;
}>();

defineEmits<{
  sort: [key: SortKey];
  selectTask: [task: TaskWithContext];
  updateTaskTitle: [taskId: string, title: string];
  updateTaskPriority: [taskId: string, priority: TaskPriority];
  updateTaskType: [taskId: string, type: TaskType];
  updateTaskStatus: [taskId: string, status: TaskStatus];
  updateTaskProgress: [taskId: string, percentComplete: number];
  updateTaskCompletedOn: [taskId: string, completedOn: string];
  deleteTask: [taskId: string];
  addTask: [target: string];
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

const selectedTarget = ref("");
</script>

<template>
  <section class="panel table-panel" aria-labelledby="all-tasks-title">
    <div class="panel-header">
      <div>
        <h2 id="all-tasks-title">All Tasks</h2>
        <p>Tap a header to sort. Select a row to open its room or task group.</p>
      </div>
      <div class="table-add-task">
        <select v-model="selectedTarget" class="field-control" aria-label="New task target">
          <option value="">Choose target</option>
          <option v-for="target in taskTargets" :key="target.value" :value="target.value">{{ target.label }}</option>
        </select>
        <button class="text-button" type="button" :disabled="!selectedTarget" @click="$emit('addTask', selectedTarget)">Add Task</button>
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
            <th scope="col">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="`${task.groupKey || task.roomId}-${task.id}`" @click="$emit('selectTask', task)">
            <td>
              <input
                class="table-control table-title-control"
                type="text"
                :value="task.title"
                @click.stop
                @change="$emit('updateTaskTitle', task.id, ($event.target as HTMLInputElement).value)"
              />
            </td>
            <td>{{ task.area }}</td>
            <td>
              <select
                class="table-control"
                :value="task.priority"
                @click.stop
                @change="$emit('updateTaskPriority', task.id, ($event.target as HTMLSelectElement).value as TaskPriority)"
              >
                <option v-for="priority in TASK_PRIORITIES" :key="priority" :value="priority">{{ priority }}</option>
              </select>
              <span class="badge table-badge" :class="badgeClass(task)">{{ badgeLabel(task) }}</span>
            </td>
            <td>
              <select
                class="table-control"
                :value="task.type"
                @click.stop
                @change="$emit('updateTaskType', task.id, ($event.target as HTMLSelectElement).value as TaskType)"
              >
                <option v-for="type in TASK_TYPES" :key="type" :value="type">{{ type }}</option>
              </select>
            </td>
            <td>
              <select
                class="table-control"
                :value="task.status"
                @click.stop
                @change="$emit('updateTaskStatus', task.id, ($event.target as HTMLSelectElement).value as TaskStatus)"
              >
                <option v-for="status in TASK_STATUSES" :key="status" :value="status">{{ statusLabel({ status }) }}</option>
              </select>
            </td>
            <td>
              <div v-if="task.status === 'in-progress'" class="table-progress-edit" @click.stop>
                <input
                  class="table-control"
                  type="number"
                  min="0"
                  max="100"
                  step="5"
                  :value="taskPercent(task)"
                  @change="$emit('updateTaskProgress', task.id, Number(($event.target as HTMLInputElement).value))"
                />
                <input
                  v-if="task.status === 'in-progress'"
                  type="range"
                  min="0"
                  max="100"
                  step="5"
                  :value="taskPercent(task)"
                  @input="$emit('updateTaskProgress', task.id, Number(($event.target as HTMLInputElement).value))"
                />
              </div>
              <span v-else class="muted-cell">-</span>
            </td>
            <td>{{ formatDate(task.dateStarted) }}</td>
            <td>
              <input
                v-if="task.status === 'done'"
                class="table-control"
                type="date"
                :value="task.completedOn || ''"
                @click.stop
                @change="$emit('updateTaskCompletedOn', task.id, ($event.target as HTMLInputElement).value)"
              />
              <span v-else>{{ formatDate(task.completedOn) }}</span>
            </td>
            <td>
              <button class="text-button danger-button" type="button" @click.stop="$emit('deleteTask', task.id)">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
