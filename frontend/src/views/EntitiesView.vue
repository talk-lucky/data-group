<template>
  <div class="entities-view">
    <h1>Metadata Entities</h1>
    <EntityList />
  </div>
</template>

<script setup>
import EntityList from '@/components/EntityList.vue'; // Adjusted path

// No specific logic needed here for now as EntityList handles its own data fetching.
// If pre-fetching or other view-specific logic is needed, it can be added here.
</script>

<style scoped>
.entities-view {
  padding: 20px;
}
</style>
