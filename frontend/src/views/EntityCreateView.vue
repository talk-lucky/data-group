<template>
  <div class="entity-create-view">
    <EntityForm />
  </div>
</template>

<script setup>
import EntityForm from '@/components/EntityForm.vue'; // Adjusted path

// No specific logic here as EntityForm handles creation.
// The EntityForm component will navigate away on successful creation or cancellation.
</script>

<style scoped>
.entity-create-view {
  padding: 20px;
}
</style>
