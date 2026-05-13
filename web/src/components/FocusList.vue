<script setup lang="ts">
import TaskEditor from "./TaskEditor.vue";
import type { ItemNeeded, Subtask, TaskPriority, TaskStatus, TaskType, TaskWithContext } from "../types/house";

defineProps<{ tasks: TaskWithContext[] }>();

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
  <div class="panel">
    <div class="panel-header">
      <div>
        <h2>This Week / Do First</h2>
        <p>Keep the short list focused on safety, venting, and service assessments.</p>
      </div>
    </div>
    <ul class="focus-list">
      <li v-if="!tasks.length" class="empty-note">No do-first tasks yet.</li>
      <li v-for="task in tasks" :key="task.id">
        <TaskEditor
          :task="task"
          :context="`${task.area} · ${task.floor}`"
          @update-task-title="(taskId, title) => $emit('updateTaskTitle', taskId, title)"
          @update-task-priority="(taskId, priority) => $emit('updateTaskPriority', taskId, priority)"
          @update-task-type="(taskId, type) => $emit('updateTaskType', taskId, type)"
          @update-task-status="(taskId, status) => $emit('updateTaskStatus', taskId, status)"
          @update-task-progress="(taskId, percentComplete) => $emit('updateTaskProgress', taskId, percentComplete)"
          @update-task-completed-on="(taskId, completedOn) => $emit('updateTaskCompletedOn', taskId, completedOn)"
          @update-task-do-first="(taskId, doFirst) => $emit('updateTaskDoFirst', taskId, doFirst)"
          @update-task-do-first-rank="(taskId, rank) => $emit('updateTaskDoFirstRank', taskId, rank)"
          @delete-task="(taskId) => $emit('deleteTask', taskId)"
          @add-subtask="(taskId) => $emit('addSubtask', taskId)"
          @update-subtask="(taskId, subtaskId, update) => $emit('updateSubtask', taskId, subtaskId, update)"
          @delete-subtask="(taskId, subtaskId) => $emit('deleteSubtask', taskId, subtaskId)"
          @add-item="(taskId) => $emit('addItem', taskId)"
          @update-item="(taskId, itemIndex, update) => $emit('updateItem', taskId, itemIndex, update)"
          @delete-item="(taskId, itemIndex) => $emit('deleteItem', taskId, itemIndex)"
        />
      </li>
    </ul>
  </div>
</template>
