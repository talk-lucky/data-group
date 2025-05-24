<template>
  <v-container fluid>
    <h1 class="text-h4 mb-4">Data Sources</h1>
    <DataSourceList />
  </v-container>
</template>

<script setup>
import DataSourceList from '@/components/DataSourceList.vue';
import { useDataSourceStore } from '@/store/dataSourceStore';
import { onMounted } from 'vue';

const dataSourceStore = useDataSourceStore();

// Fetch data sources when the view is mounted.
// DataSourceList also calls this, but calling here ensures data is available
// even if DataSourceList had conditional rendering or was part of a more complex setup.
onMounted(() => {
  // Only fetch if not already populated or if a force refresh is desired.
  // For simplicity, DataSourceList handles its own fetching, but this is a common pattern.
  if (dataSourceStore.dataSources.length === 0) {
    dataSourceStore.fetchDataSources();
  }
});
</script>

<style scoped>
.text-h4 {
  font-weight: 500;
}
</style>
