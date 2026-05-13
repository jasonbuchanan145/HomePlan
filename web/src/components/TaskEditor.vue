<script setup lang="ts">
import type { ItemNeeded, Subtask, Task, TaskPriority, TaskStatus, TaskType } from "../types/house";
import { TASK_PRIORITIES, TASK_STATUSES, TASK_TYPES, formatDate, statusLabel, taskBadges, taskPercent } from "../utils/house";

defineProps<{
  task: Task;
  context?: string;
}>();

defineEmits<{
  updateTaskTitle: [taskId: string, title: string];
  updateTaskPriority: [taskId: string, priority: TaskPriority];
  updateTaskType: [taskId: string, type: TaskType];
  updateTaskStatus: [taskId: string, status: TaskStatus];
  updateTaskProgress: [taskId: string, percentComplete: number];
  updateTaskCompletedOn: [taskId: string, completedOn: string];
  updateTaskDoFirst: [taskId: string, doFirst: boolean];
  updateTaskDoFirstRank: [taskId: string, rank: number | undefined];
  deleteTask: [taskId: string];
  addSubtask: [taskId: string];
  updateSubtask: [taskId: string, subtaskId: string, update: Partial<Subtask>];
  deleteSubtask: [taskId: string, subtaskId: string];
  addItem: [taskId: string];
  updateItem: [taskId: string, itemIndex: number, update: Partial<ItemNeeded>];
  deleteItem: [taskId: string, itemIndex: number];
}>();
</script>

<template>
  <article class="task-item">
    <div class="task-title-row">
      <label class="field-label title-field">
        <span>Task</span>
        <input class="field-control" type="text" :value="task.title" @change="$emit('updateTaskTitle', task.id, ($event.target as HTMLInputElement).value)" />
      </label>
      <button class="text-button danger-button" type="button" @click="$emit('deleteTask', task.id)">Delete</button>
    </div>
    <p v-if="context" class="task-meta">{{ context }}</p>
    <div class="task-badges task-badges-spaced">
      <span v-for="badge in taskBadges(task)" :key="`${task.id}-${badge.className}`" class="badge" :class="badge.className">
        {{ badge.label }}
      </span>
    </div>

    <div class="task-edit-grid task-edit-grid-wide">
      <label class="field-label">
        <span>Status</span>
        <select class="field-control" :value="task.status" @change="$emit('updateTaskStatus', task.id, ($event.target as HTMLSelectElement).value as TaskStatus)">
          <option v-for="status in TASK_STATUSES" :key="status" :value="status">{{ statusLabel({ status }) }}</option>
        </select>
      </label>
      <label class="field-label">
        <span>Priority</span>
        <select class="field-control" :value="task.priority" @change="$emit('updateTaskPriority', task.id, ($event.target as HTMLSelectElement).value as TaskPriority)">
          <option v-for="priority in TASK_PRIORITIES" :key="priority" :value="priority">{{ priority }}</option>
        </select>
      </label>
      <label class="field-label">
        <span>Type</span>
        <select class="field-control" :value="task.type" @change="$emit('updateTaskType', task.id, ($event.target as HTMLSelectElement).value as TaskType)">
          <option v-for="type in TASK_TYPES" :key="type" :value="type">{{ type }}</option>
        </select>
      </label>
      <label v-if="task.status === 'in-progress'" class="field-label">
        <span>Progress</span>
        <input
          class="field-control"
          type="number"
          min="0"
          max="100"
          step="5"
          :value="taskPercent(task)"
          @change="$emit('updateTaskProgress', task.id, Number(($event.target as HTMLInputElement).value))"
        />
      </label>
    </div>

    <label v-if="task.status === 'in-progress'" class="range-label">
      <span class="visually-hidden">Progress slider</span>
      <input
        type="range"
        min="0"
        max="100"
        step="5"
        :value="taskPercent(task)"
        @input="$emit('updateTaskProgress', task.id, Number(($event.target as HTMLInputElement).value))"
      />
    </label>

    <div class="task-edit-grid">
      <label class="checkbox-field">
        <input type="checkbox" :checked="Boolean(task.doFirst)" @change="$emit('updateTaskDoFirst', task.id, ($event.target as HTMLInputElement).checked)" />
        <span>Do First</span>
      </label>
      <label v-if="task.doFirst" class="field-label">
        <span>Rank</span>
        <input
          class="field-control"
          type="number"
          min="1"
          :value="task.doFirstRank ?? ''"
          @change="$emit('updateTaskDoFirstRank', task.id, ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : undefined)"
        />
      </label>
    </div>

    <label v-if="task.status === 'done'" class="field-label completed-field">
      <span>Completed</span>
      <input
        class="field-control"
        type="date"
        :value="task.completedOn || ''"
        @change="$emit('updateTaskCompletedOn', task.id, ($event.target as HTMLInputElement).value)"
      />
    </label>

    <p v-if="task.dateStarted" class="task-meta task-date">Started {{ formatDate(task.dateStarted) }}</p>
    <p v-if="task.completedOn" class="task-meta task-date">Completed {{ formatDate(task.completedOn) }}</p>
    <div v-if="task.status === 'in-progress'" class="task-progress">
      <div class="task-progress-label">
        <span>Progress</span>
        <span>{{ taskPercent(task) }}%</span>
      </div>
      <div class="task-progress-track" aria-hidden="true">
        <div class="task-progress-bar" :style="{ width: `${taskPercent(task)}%` }"></div>
      </div>
    </div>

    <div class="task-detail-section">
      <div class="detail-title-row">
        <p class="task-detail-title">Subtasks</p>
        <button class="text-button" type="button" @click="$emit('addSubtask', task.id)">Add</button>
      </div>
      <ul class="detail-list editable-detail-list">
        <li v-if="!task.subtasks?.length" class="empty-note">No subtasks yet.</li>
        <template v-else>
          <li v-for="subtask in task.subtasks" :key="subtask.id">
            <input class="field-control" type="text" :value="subtask.title" @change="$emit('updateSubtask', task.id, subtask.id, { title: ($event.target as HTMLInputElement).value })" />
            <select class="field-control" :value="subtask.status" @change="$emit('updateSubtask', task.id, subtask.id, { status: ($event.target as HTMLSelectElement).value as TaskStatus })">
              <option v-for="status in TASK_STATUSES" :key="status" :value="status">{{ statusLabel({ status }) }}</option>
            </select>
            <input
              v-if="subtask.status === 'in-progress'"
              class="field-control"
              type="number"
              min="0"
              max="100"
              step="5"
              :value="subtask.percentComplete ?? 0"
              @change="$emit('updateSubtask', task.id, subtask.id, { percentComplete: Number(($event.target as HTMLInputElement).value) })"
            />
            <span v-else class="muted-cell">-</span>
            <input
              v-if="subtask.status === 'done'"
              class="field-control"
              type="date"
              :value="subtask.completedOn || ''"
              @change="$emit('updateSubtask', task.id, subtask.id, { completedOn: ($event.target as HTMLInputElement).value })"
            />
            <button class="text-button danger-button" type="button" @click="$emit('deleteSubtask', task.id, subtask.id)">Delete</button>
          </li>
        </template>
      </ul>
    </div>

    <div class="task-detail-section">
      <div class="detail-title-row">
        <p class="task-detail-title">Items Needed</p>
        <button class="text-button" type="button" @click="$emit('addItem', task.id)">Add</button>
      </div>
      <ul class="detail-list editable-item-list">
        <li v-if="!task.itemsNeeded?.length" class="empty-note">No items yet.</li>
        <template v-else>
          <li v-for="(item, itemIndex) in task.itemsNeeded" :key="`${item.name}-${itemIndex}`">
            <input class="field-control" type="text" :value="item.name" @change="$emit('updateItem', task.id, itemIndex, { name: ($event.target as HTMLInputElement).value })" />
            <input class="field-control" type="text" :value="item.status || ''" placeholder="status" @change="$emit('updateItem', task.id, itemIndex, { status: ($event.target as HTMLInputElement).value })" />
            <button class="text-button danger-button" type="button" @click="$emit('deleteItem', task.id, itemIndex)">Delete</button>
          </li>
        </template>
      </ul>
    </div>
  </article>
</template>
