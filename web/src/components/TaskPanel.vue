<script setup lang="ts">
import type { Task } from "../types/house";
import { formatDate, statusLabel, taskBadges, taskPercent } from "../utils/house";

defineProps<{
  title: string;
  subtitle: string;
  tasks: Task[];
}>();
</script>

<template>
  <aside class="panel" aria-labelledby="selected-room-title">
    <div class="panel-header">
      <div>
        <h2 id="selected-room-title">{{ title }}</h2>
        <p>{{ subtitle }}</p>
      </div>
    </div>
    <ul class="task-list">
      <li v-if="!tasks.length" class="empty-note">No tasks yet.</li>
      <template v-else>
        <li v-for="task in tasks" :key="task.id" class="task-item">
          <p class="task-title">{{ task.title }}</p>
          <div class="task-badges">
            <span v-for="badge in taskBadges(task)" :key="`${task.id}-${badge.className}`" class="badge" :class="badge.className">
              {{ badge.label }}
            </span>
          </div>
          <p v-if="task.dateStarted" class="task-meta task-date">Started {{ formatDate(task.dateStarted) }}</p>
          <p v-if="task.completedOn" class="task-meta task-date">Completed {{ formatDate(task.completedOn) }}</p>
          <div v-if="taskPercent(task) || task.status === 'in-progress'" class="task-progress">
            <div class="task-progress-label">
              <span>Progress</span>
              <span>{{ taskPercent(task) }}%</span>
            </div>
            <div class="task-progress-track" aria-hidden="true">
              <div class="task-progress-bar" :style="{ width: `${taskPercent(task)}%` }"></div>
            </div>
          </div>
          <div v-if="task.subtasks?.length" class="task-detail-section">
            <p class="task-detail-title">Subtasks</p>
            <ul class="detail-list">
              <li v-for="subtask in task.subtasks" :key="subtask.id">
                <strong>{{ subtask.title }}</strong>
                <span>
                  {{ statusLabel(subtask) }}{{ Number.isFinite(subtask.percentComplete) ? ` · ${subtask.percentComplete}%` : "" }}{{ subtask.dateStarted ? ` · Started ${formatDate(subtask.dateStarted)}` : "" }}
                </span>
              </li>
            </ul>
          </div>
          <div v-if="task.itemsNeeded?.length" class="task-detail-section">
            <p class="task-detail-title">Items Needed</p>
            <ul class="detail-list">
              <li v-for="item in task.itemsNeeded" :key="item.name">
                <strong>{{ item.name }}</strong>
                <span>{{ item.status || "needed" }}</span>
              </li>
            </ul>
          </div>
        </li>
      </template>
    </ul>
  </aside>
</template>
