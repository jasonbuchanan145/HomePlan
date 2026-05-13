<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import interact from "interactjs";
import type { Floor, HouseState, Room, RoomLayout } from "../types/house";
import { getRoomTasks, roomBadges } from "../utils/house";

const props = defineProps<{
  house: HouseState;
  floorKey: string;
  floor: Floor;
  selectedRoomId: string;
}>();

const emit = defineEmits<{
  selectRoom: [roomId: string];
  updateRoomLayout: [roomId: string, layout: RoomLayout];
}>();

const planShell = ref<HTMLElement | null>(null);
const snapPercent = 5;

function roomStyle(room: Room) {
  return {
    left: `${room.layout.x}%`,
    top: `${room.layout.y}%`,
    width: `${room.layout.w}%`,
    height: `${room.layout.h}%`
  };
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

function snap(value: number): number {
  return Math.round(value / snapPercent) * snapPercent;
}

function unbindInteractables() {
  planShell.value?.querySelectorAll<HTMLElement>("[data-room-id]").forEach((element) => {
    interact(element).unset();
  });
}

async function bindInteractables() {
  await nextTick();
  const shell = planShell.value;
  if (!shell) return;

  unbindInteractables();
  const shellRect = shell.getBoundingClientRect();
  const grid = {
    x: Math.max(8, shellRect.width * (snapPercent / 100)),
    y: Math.max(8, shellRect.height * (snapPercent / 100))
  };

  shell.querySelectorAll<HTMLElement>("[data-room-id]").forEach((element) => {
    interact(element)
      .draggable({
        modifiers: [
          interact.modifiers.restrictRect({ restriction: "parent" }),
          interact.modifiers.snap({
            targets: [interact.snappers.grid(grid)],
            range: Infinity,
            relativePoints: [{ x: 0, y: 0 }]
          })
        ],
        listeners: {
          start(event) {
            const roomId = (event.target as HTMLElement).dataset.roomId;
            if (roomId) emit("selectRoom", roomId);
          },
          move(event) {
            const target = event.target as HTMLElement;
            const x = (Number(target.dataset.dragX) || 0) + event.dx;
            const y = (Number(target.dataset.dragY) || 0) + event.dy;
            target.dataset.dragX = String(x);
            target.dataset.dragY = String(y);
            target.style.transform = `translate(${x}px, ${y}px)`;
          },
          end(event) {
            const target = event.target as HTMLElement;
            const roomId = target.dataset.roomId;
            const room = props.floor.rooms.find((candidate) => candidate.id === roomId);
            const parentRect = planShell.value?.getBoundingClientRect();
            if (!room || !roomId || !parentRect) return;

            const dragX = Number(target.dataset.dragX) || 0;
            const dragY = Number(target.dataset.dragY) || 0;
            const nextX = clamp(snap(room.layout.x + (dragX / parentRect.width) * 100), 0, 100 - room.layout.w);
            const nextY = clamp(snap(room.layout.y + (dragY / parentRect.height) * 100), 0, 100 - room.layout.h);

            target.dataset.dragX = "0";
            target.dataset.dragY = "0";
            target.style.transform = "";
            emit("updateRoomLayout", roomId, { ...room.layout, x: nextX, y: nextY });
          }
        }
      })
      .resizable({
        edges: { right: true, bottom: true },
        listeners: {
          start(event) {
            const roomId = (event.target as HTMLElement).dataset.roomId;
            if (roomId) emit("selectRoom", roomId);
          },
          move(event) {
            const target = event.target as HTMLElement;
            const parentRect = planShell.value?.getBoundingClientRect();
            if (!parentRect) return;
            const snappedWidth = Math.max(40, (snap((event.rect.width / parentRect.width) * 100) / 100) * parentRect.width);
            const snappedHeight = Math.max(40, (snap((event.rect.height / parentRect.height) * 100) / 100) * parentRect.height);
            target.style.width = `${snappedWidth}px`;
            target.style.height = `${snappedHeight}px`;
          },
          end(event) {
            const target = event.target as HTMLElement;
            const roomId = target.dataset.roomId;
            const room = props.floor.rooms.find((candidate) => candidate.id === roomId);
            const parentRect = planShell.value?.getBoundingClientRect();
            if (!room || !roomId || !parentRect) return;

            const nextW = clamp(snap((event.rect.width / parentRect.width) * 100), 5, 100 - room.layout.x);
            const nextH = clamp(snap((event.rect.height / parentRect.height) * 100), 5, 100 - room.layout.y);
            target.style.width = "";
            target.style.height = "";
            emit("updateRoomLayout", roomId, { ...room.layout, w: nextW, h: nextH });
          }
        }
      });
  });
}

onMounted(bindInteractables);
onBeforeUnmount(unbindInteractables);
watch(() => [props.floorKey, props.floor.rooms.map((room) => `${room.id}:${room.layout.x}:${room.layout.y}:${room.layout.w}:${room.layout.h}`).join("|")], bindInteractables);
</script>

<template>
  <div ref="planShell" class="plan-shell" :class="`${floorKey}-plan`" aria-live="polite">
    <button
      v-for="room in floor.rooms"
      :key="room.id"
      type="button"
      class="room"
      :data-room-id="room.id"
      :class="{
        'is-compact': room.display?.compact,
        'is-selected': room.id === selectedRoomId,
        'is-outdoor': room.display?.tone === 'outdoor',
        'is-dashed': room.display?.borderStyle === 'dashed'
      }"
      :style="roomStyle(room)"
      :aria-pressed="room.id === selectedRoomId"
      @click="$emit('selectRoom', room.id)"
    >
      <span class="room-name">{{ room.display?.mapLabel || room.name }}</span>
      <span class="room-count">{{ getRoomTasks(house, room).filter((task) => task.status !== "done").length }} open</span>
      <span class="room-badges">
        <span v-for="badge in roomBadges(house, room)" :key="badge" class="badge" :class="badge">{{ badge }}</span>
      </span>
      <span v-if="room.id === selectedRoomId" class="resize-corner" aria-hidden="true"></span>
    </button>
  </div>
</template>
