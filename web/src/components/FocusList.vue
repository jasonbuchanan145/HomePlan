<script setup lang="ts">
import type { TaskWithContext } from "../types/house";
import { taskBadges } from "../utils/house";

defineProps<{ tasks: TaskWithContext[] }>();
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <div>
        <h2>This Week / Do First</h2>
        <p>Keep the short list focused on safety, venting, and service assessments.</p>
      </div>
    </div>
    <ul class="focus-list">
      <li v-for="task in tasks" :key="task.id" class="focus-card">
        <p class="task-title">{{ task.title }}</p>
        <p class="task-meta">{{ task.area }} · {{ task.floor }}</p>
        <div class="task-badges task-badges-spaced">
          <span v-for="badge in taskBadges(task)" :key="`${task.id}-${badge.className}`" class="badge" :class="badge.className">
            {{ badge.label }}
          </span>
        </div>
      </li>
    </ul>
  </div>
</template>
