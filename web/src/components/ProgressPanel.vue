<script setup lang="ts">
defineProps<{
  done: number;
  open: number;
  critical: number;
  contractor: number;
  later: number;
  total: number;
  percent: number;
}>();
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <div>
        <h2>Progress</h2>
        <p>{{ done }} of {{ total }} tasks complete.</p>
      </div>
    </div>
    <div class="progress-track" aria-hidden="true">
      <div class="progress-bar" :style="{ width: `${percent}%` }"></div>
    </div>
    <div class="progress-grid">
      <div class="progress-stat"><strong>{{ open }}</strong><span>Open</span></div>
      <div class="progress-stat"><strong>{{ critical }}</strong><span>Critical</span></div>
      <div class="progress-stat"><strong>{{ contractor }}</strong><span>Contractor</span></div>
      <div class="progress-stat"><strong>{{ later }}</strong><span>Later</span></div>
      <div class="progress-stat"><strong>{{ total }}</strong><span>Total</span></div>
    </div>
  </div>
</template>
