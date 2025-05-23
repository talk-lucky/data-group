<template>
  <v-container fluid>
    <v-btn to="/datasources" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Data Sources
    </v-btn>

    <v-progress-linear v-if="dataSourceStore.loading" indeterminate color="primary" class="my-4"></v-progress-linear>
    
    <v-alert v-if="dataSourceStore.error && !dataSourceStore.loading" type="error" dismissible class="my-4">
      Error fetching data source details: {{ dataSourceStore.error }}
    </v-alert>

    <v-card v-if="dataSource && !dataSourceStore.loading" class="mb-5">
      <v-card-title class="text-h4 d-flex justify-space-between align-center">
        <span>Data Source: {{ dataSource.name }}</span>
        <v-chip small :color="getDataSourceTypeColor(dataSource.type)" class="ml-2">{{ dataSource.type }}</v-chip>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ dataSource.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-row>
          <v-col cols="12" md="6">
            <strong class="text-subtitle-1">Connection Details:</strong>
            <v-sheet elevation="1" rounded class="pa-3 mt-1" style="background-color: #f5f5f5;">
              <pre style="white-space: pre-wrap; word-break: break-all;">{{ formattedConnectionDetails }}</pre>
            </v-sheet>
          </v-col>
          <v-col cols="12" md="6">
            <v-list dense>
              <v-list-item>
                <v-list-item-title><strong>Created At:</strong></v-list-item-title>
                <v-list-item-subtitle>{{ formatDate(dataSource.created_at) }}</v-list-item-subtitle>
              </v-list-item>
              <v-list-item>
                <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
                <v-list-item-subtitle>{{ formatDate(dataSource.updated_at) }}</v-list-item-subtitle>
              </v-list-item>
            </v-list>
          </v-col>
        </v-row>
      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/datasources/${sourceId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Data Source
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!dataSource && !dataSourceStore.loading && !dataSourceStore.error" class="text-center my-5">
      <v-alert type="warning">
        Data Source with ID "{{ sourceId }}" not found.
      </v-alert>
    </div>

    <!-- FieldMappingList component -->
    <FieldMappingList v-if="sourceId" :source-id="sourceId" class="mt-5" />

  </v-container>
</template>

<script setup>
import { computed, onMounted, watch, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import { useDataSourceStore } from '@/store/dataSourceStore';
import { useFieldMappingStore } from '@/store/fieldMappingStore';
import FieldMappingList from '@/components/FieldMappingList.vue';

const route = useRoute();
const dataSourceStore = useDataSourceStore();
const fieldMappingStore = useFieldMappingStore();

const sourceId = computed(() => route.params.id);
const dataSource = computed(() => dataSourceStore.currentDataSource);

const formattedConnectionDetails = computed(() => {
  if (dataSource.value?.connection_details) {
    try {
      const parsed = JSON.parse(dataSource.value.connection_details);
      return JSON.stringify(parsed, null, 2); // Pretty print
    } catch (e) {
      return dataSource.value.connection_details; // Return as is if not valid JSON
    }
  }
  return 'N/A';
});

onMounted(() => {
  if (sourceId.value) {
    dataSourceStore.fetchDataSourceById(sourceId.value);
    // Field mappings are fetched by FieldMappingList itself based on sourceId prop
    // However, if we need to clear them when sourceId changes (e.g. navigating between detail views)
    // fieldMappingStore.clearMappingsForSource(sourceId.value); // Or do this in watch
  }
});

watch(sourceId, (newId, oldId) => {
  if (newId && newId !== oldId) {
    dataSourceStore.fetchDataSourceById(newId);
    if (oldId) fieldMappingStore.clearMappingsForSource(oldId); // Clear for the old source
    // FieldMappingList will fetch for newId due to its own watch on sourceId prop
  }
});

onBeforeUnmount(() => {
  dataSourceStore.clearCurrentDataSource();
  if (sourceId.value) {
    fieldMappingStore.clearMappingsForSource(sourceId.value);
  }
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try { return new Date(dateTimeString).toLocaleString(); } catch (e) { return dateTimeString; }
}

function getDataSourceTypeColor(type) {
  const colors = {
    'PostgreSQL': 'blue', 'MySQL': 'orange', 'CSV File': 'teal', 
    'Generic API': 'indigo', 'Other': 'grey'
  };
  return colors[type] || 'grey';
}
</script>

<style scoped>
.text-h4 { font-weight: 500; }
.v-list-item-title { font-weight: bold; }
pre {
  background-color: #f0f0f0; /* Light grey background for pre block */
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto; /* Allow horizontal scrolling for long lines */
}
</style>
