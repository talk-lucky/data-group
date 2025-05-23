<template>
  <div class="entity-edit-view">
    <EntityForm :entity-id="entityId" />
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import EntityForm from '@/components/EntityForm.vue'; // Adjusted path

const route = useRoute();
const entityId = computed(() => route.params.id);

// EntityForm will use the entityId prop to fetch and display the entity for editing.
// It will also handle navigation on save or cancel.
</script>

<style scoped>
.entity-edit-view {
  padding: 20px;
}
</style>
