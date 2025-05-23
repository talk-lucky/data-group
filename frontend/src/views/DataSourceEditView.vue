<template>
  <v-container fluid>
    <h1 class="text-h4 mb-4">Edit Data Source</h1>
    <DataSourceForm :data-source-id="dataSourceId" />
  </v-container>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import DataSourceForm from '@/components/DataSourceForm.vue';

const route = useRoute();
const dataSourceId = computed(() => route.params.id);

// DataSourceForm will use the dataSourceId prop to fetch and display
// the data source for editing. It also handles navigation on save or cancel.
</script>

<style scoped>
.text-h4 {
  font-weight: 500;
}
</style>
