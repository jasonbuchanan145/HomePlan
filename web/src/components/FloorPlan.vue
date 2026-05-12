<script setup lang="ts">
import type { Floor, HouseState, Room } from "../types/house";
import { getRoomTasks, roomBadges } from "../utils/house";

const props = defineProps<{
  house: HouseState;
  floorKey: string;
  floor: Floor;
  selectedRoomId: string;
}>();

defineEmits<{
  selectRoom: [roomId: string];
}>();

function roomStyle(room: Room) {
  return {
    left: `${room.layout.x}%`,
    top: `${room.layout.y}%`,
    width: `${room.layout.w}%`,
    height: `${room.layout.h}%`
  };
}
</script>

<template>
  <div class="plan-shell" :class="`${floorKey}-plan`" aria-live="polite">
    <button
      v-for="room in floor.rooms"
      :key="room.id"
      type="button"
      class="room"
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
    </button>
  </div>
</template>
