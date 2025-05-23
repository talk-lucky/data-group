<template>
  <v-container fluid>
    <h1 class="text-h4 mb-4">Create New Data Source</h1>
    <DataSourceForm />
  </v-container>
</template>

<script setup>
import DataSourceForm from '@/components/DataSourceForm.vue';

// DataSourceForm handles the creation logic and navigation.
// No specific view logic needed here for now.
</script>

<style scoped>
.text-h4 {
  font-weight: 500;
}
</style>
