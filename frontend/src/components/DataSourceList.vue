<template>
  <v-container fluid>
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title>
            Data Sources
            <v-spacer></v-spacer>
            <v-btn color="primary" to="/datasources/new">
              <v-icon left>mdi-plus</v-icon> Create New Data Source
            </v-btn>
          </v-card-title>

          <v-card-text>
            <v-progress-linear v-if="dataSourceStore.loading" indeterminate color="primary"></v-progress-linear>
            
            <v-alert v-if="dataSourceStore.error" type="error" dismissible>
              Error fetching data sources: {{ dataSourceStore.error }}
            </v-alert>

            <v-data-table
              v-if="!dataSourceStore.loading && dataSources.length"
              :headers="headers"
              :items="dataSources"
              item-key="id"
              class="elevation-1"
            >
              <template v-slot:item.type="{ item }">
                <v-chip small :color="getDataSourceTypeColor(item.type)">{{ item.type }}</v-chip>
              </template>
              <template v-slot:item.created_at="{ item }">
                {{ formatDate(item.created_at) }}
              </template>
              <template v-slot:item.updated_at="{ item }">
                {{ formatDate(item.updated_at) }}
              </template>
              <template v-slot:item.actions="{ item }">
                <v-tooltip top>
                  <template v-slot:activator="{ props }">
                    <v-icon small class="mr-2" @click="viewDataSourceDetails(item.id)" v-bind="props">mdi-eye</v-icon>
                  </template>
                  <span>View Details & Mappings</span>
                </v-tooltip>
                <v-tooltip top>
                  <template v-slot:activator="{ props }">
                    <v-icon small class="mr-2" @click="editDataSource(item.id)" v-bind="props">mdi-pencil</v-icon>
                  </template>
                  <span>Edit Data Source</span>
                </v-tooltip>
                 <v-tooltip top>
                  <template v-slot:activator="{ props }">
                    <v-icon small @click="openDeleteDialog(item)" v-bind="props">mdi-delete</v-icon>
                  </template>
                  <span>Delete Data Source</span>
                </v-tooltip>
              </template>
            </v-data-table>

            <v-alert v-if="!dataSourceStore.loading && !dataSources.length && !dataSourceStore.error" type="info">
              No data sources found. Click "Create New Data Source" to add one.
            </v-alert>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete</v-card-title>
        <v-card-text>
          Are you sure you want to delete the data source "<strong>{{ itemToDelete?.name }}</strong>"? 
          This action will also delete all its associated field mappings and cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="closeDeleteDialog">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="confirmDelete" :loading="deleting">Delete</v-btn>
          <v-spacer></v-spacer>
        </v-card-actions>
      </v-card>
    </v-dialog>

     <!-- Snackbar for notifications -->
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" bottom right>
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn color="white" text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useDataSourceStore } from '@/store/dataSourceStore';

const router = useRouter();
const dataSourceStore = useDataSourceStore();

const dataSources = computed(() => dataSourceStore.dataSources);

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Type', key: 'type', sortable: true },
  // { title: 'Connection Details', key: 'connection_details', sortable: false }, // Often too long for table
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const deleteDialog = ref(false);
const itemToDelete = ref(null);
const deleting = ref(false);

const snackbar = ref({
  show: false,
  message: '',
  color: '',
  timeout: 3000,
});

function showSnackbar(message, color = 'success') {
  snackbar.value.message = message;
  snackbar.value.color = color;
  snackbar.value.show = true;
}

onMounted(() => {
  dataSourceStore.fetchDataSources();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString; 
  }
}

function getDataSourceTypeColor(type) {
  const colors = {
    'PostgreSQL': 'blue',
    'MySQL': 'orange',
    'CSV': 'green',
    'API': 'purple',
    'CSV File': 'teal',
    'Generic API': 'indigo',
  };
  return colors[type] || 'grey';
}

function viewDataSourceDetails(id) {
  router.push(`/datasources/${id}/details`);
}

function editDataSource(id) {
  router.push(`/datasources/${id}/edit`);
}

function openDeleteDialog(item) {
  itemToDelete.value = item;
  deleteDialog.value = true;
}

function closeDeleteDialog() {
  itemToDelete.value = null;
  deleteDialog.value = false;
}

async function confirmDelete() {
  if (!itemToDelete.value) return;
  deleting.value = true;
  try {
    await dataSourceStore.removeDataSource(itemToDelete.value.id);
    showSnackbar(`Data source "${itemToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete data source: ${error.message || 'Unknown error'}`, 'error');
  } finally {
    deleting.value = false;
    closeDeleteDialog();
  }
}
</script>

<style scoped>
.v-data-table {
  margin-top: 16px;
}
.v-icon {
  cursor: pointer;
}
</style>
