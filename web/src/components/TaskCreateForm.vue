<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import type { TaskCreateDraft, TaskTargetOption } from "../types/house";
import { TASK_PRIORITIES, TASK_STATUSES, TASK_TYPES, statusLabel } from "../utils/house";

const props = defineProps<{
  open: boolean;
  targetOptions: TaskTargetOption[];
  defaultTarget?: string;
  lockTarget?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  createTask: [draft: TaskCreateDraft];
}>();

const form = reactive<TaskCreateDraft>({
  title: "",
  target: "",
  priority: "important",
  type: "DIY",
  status: "open",
  notes: "",
  roomName: "",
  subtasks: [],
  itemsNeeded: []
});

const selectedTarget = computed(() => props.targetOptions.find((target) => target.value === form.target));
const targetLabel = computed(() => selectedTarget.value?.label ?? "Current target");
const needsRoomName = computed(() => Boolean(selectedTarget.value?.needsRoomName));
const canSubmit = computed(() => Boolean(form.title.trim() && form.target && (!needsRoomName.value || form.roomName?.trim())));

watch(
  () => [props.open, props.defaultTarget, props.lockTarget, props.targetOptions] as const,
  ([open, defaultTarget, lockTarget]) => {
    if (!open) return;
    resetForm(defaultTarget || (lockTarget ? props.targetOptions[0]?.value || "" : ""));
  },
  { immediate: true }
);

function resetForm(target: string) {
  form.title = "";
  form.target = target;
  form.priority = "important";
  form.type = "DIY";
  form.status = "open";
  form.notes = "";
  form.roomName = "";
  form.subtasks = [];
  form.itemsNeeded = [];
}

function addSubtask() {
  form.subtasks.push({ title: "", status: "open" });
}

function removeSubtask(index: number) {
  form.subtasks.splice(index, 1);
}

function addItem() {
  form.itemsNeeded.push({ name: "", status: "needed" });
}

function removeItem(index: number) {
  form.itemsNeeded.splice(index, 1);
}

function submitForm() {
  if (!canSubmit.value) return;
  emit("createTask", {
    title: form.title.trim(),
    target: form.target,
    priority: form.priority,
    type: form.type,
    status: form.status,
    notes: form.notes?.trim() || undefined,
    roomName: form.roomName?.trim() || undefined,
    subtasks: form.subtasks.filter((subtask) => subtask.title.trim()).map((subtask) => ({ title: subtask.title.trim(), status: subtask.status })),
    itemsNeeded: form.itemsNeeded.filter((item) => item.name.trim()).map((item) => ({ name: item.name.trim(), status: item.status?.trim() || "needed" }))
  });
}
</script>

<template>
  <teleport to="body">
    <div v-if="open" class="modal-backdrop" role="presentation">
      <form class="task-create-modal" role="dialog" aria-modal="true" aria-labelledby="task-create-title" @submit.prevent="submitForm">
        <div class="modal-header">
          <div>
            <p class="eyebrow">New task</p>
            <h2 id="task-create-title">Add Task</h2>
          </div>
          <button class="text-button" type="button" @click="$emit('close')">Close</button>
        </div>

        <div class="task-form-grid">
          <label class="field-label title-field">
            <span>Title</span>
            <input v-model="form.title" class="field-control" type="text" placeholder="What needs to happen?" autofocus required />
          </label>

          <label v-if="!lockTarget" class="field-label">
            <span>Target</span>
            <select v-model="form.target" class="field-control" required>
              <option value="">Choose target</option>
              <option v-for="target in targetOptions" :key="target.value" :value="target.value">{{ target.label }}</option>
            </select>
          </label>
          <div v-else class="locked-target" aria-label="Task target">
            <span>Target</span>
            <strong>{{ targetLabel }}</strong>
          </div>

          <label v-if="needsRoomName" class="field-label">
            <span>Room Name</span>
            <input v-model="form.roomName" class="field-control" type="text" placeholder="Room name" required />
          </label>

          <label class="field-label">
            <span>Priority</span>
            <select v-model="form.priority" class="field-control">
              <option v-for="priority in TASK_PRIORITIES" :key="priority" :value="priority">{{ priority }}</option>
            </select>
          </label>

          <label class="field-label">
            <span>Type</span>
            <select v-model="form.type" class="field-control">
              <option v-for="taskType in TASK_TYPES" :key="taskType" :value="taskType">{{ taskType }}</option>
            </select>
          </label>

          <label class="field-label">
            <span>Status</span>
            <select v-model="form.status" class="field-control">
              <option v-for="status in TASK_STATUSES" :key="status" :value="status">{{ statusLabel({ status }) }}</option>
            </select>
          </label>
        </div>

        <label class="field-label notes-field">
          <span>Notes</span>
          <textarea v-model="form.notes" class="field-control" rows="3" placeholder="Context, measurements, concerns, or links"></textarea>
        </label>

        <div class="ai-placeholder">
          <button class="text-button" type="button" disabled>Plan with AI</button>
          <span>Sign in will be required for AI planning.</span>
        </div>

        <div class="form-detail-grid">
          <section class="task-detail-section">
            <div class="detail-title-row">
              <p class="task-detail-title">Subtasks</p>
              <button class="text-button" type="button" @click="addSubtask">Add</button>
            </div>
            <ul class="detail-list create-detail-list">
              <li v-if="!form.subtasks.length" class="empty-note">No subtasks yet.</li>
              <li v-for="(subtask, index) in form.subtasks" :key="`subtask-${index}`">
                <input v-model="subtask.title" class="field-control" type="text" placeholder="Subtask" />
                <select v-model="subtask.status" class="field-control">
                  <option v-for="status in TASK_STATUSES" :key="status" :value="status">{{ statusLabel({ status }) }}</option>
                </select>
                <button class="text-button danger-button" type="button" @click="removeSubtask(index)">Remove</button>
              </li>
            </ul>
          </section>

          <section class="task-detail-section">
            <div class="detail-title-row">
              <p class="task-detail-title">Items Needed</p>
              <button class="text-button" type="button" @click="addItem">Add</button>
            </div>
            <ul class="detail-list create-detail-list">
              <li v-if="!form.itemsNeeded.length" class="empty-note">No items yet.</li>
              <li v-for="(item, index) in form.itemsNeeded" :key="`item-${index}`">
                <input v-model="item.name" class="field-control" type="text" placeholder="Item" />
                <input v-model="item.status" class="field-control" type="text" placeholder="needed" />
                <button class="text-button danger-button" type="button" @click="removeItem(index)">Remove</button>
              </li>
            </ul>
          </section>
        </div>

        <div class="modal-actions">
          <button class="text-button" type="button" @click="$emit('close')">Cancel</button>
          <button class="save-button" type="submit" :disabled="!canSubmit">Create Task</button>
        </div>
      </form>
    </div>
  </teleport>
</template>
