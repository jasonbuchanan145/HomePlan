<script setup lang="ts">
defineProps<{
  doneCount: number;
  nextFocus: string;
}>();
</script>

<template>
  <section class="summary-grid" aria-label="Project summary">
    <article class="summary-card">
      <span class="summary-label">Completed</span>
      <strong class="summary-value">{{ doneCount }}</strong>
      <p class="summary-note">Finished and verified.</p>
    </article>
    <article class="summary-card">
      <span class="summary-label">Next focus</span>
      <strong class="summary-value">{{ nextFocus }}</strong>
      <p class="summary-note">Secure the banister, then line up contractor calls.</p>
    </article>
  </section>
</template>
