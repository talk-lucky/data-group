<template>
  <v-container fluid>
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title>
            Entities
            <v-spacer></v-spacer>
            <v-btn color="primary" @click="goToCreateEntity">
              <v-icon left>mdi-plus</v-icon> Create New Entity
            </v-btn>
          </v-card-title>

          <v-card-text>
            <v-progress-linear v-if="entityStore.loading" indeterminate color="primary"></v-progress-linear>
            
            <v-alert v-if="entityStore.error" type="error" dismissible>
              Error fetching entities: {{ entityStore.error }}
            </v-alert>

            <v-data-table
              v-if="!entityStore.loading && entities.length"
              :headers="headers"
              :items="entities"
              item-key="id"
              class="elevation-1"
            >
              <template v-slot:item.description="{ item }">
                {{ item.description || 'N/A' }}
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
                    <v-icon small class="mr-2" @click="viewEntityDetails(item.id)" v-bind="props">mdi-eye</v-icon>
                  </template>
                  <span>View Details</span>
                </v-tooltip>
                <v-tooltip top>
                  <template v-slot:activator="{ props }">
                    <v-icon small class="mr-2" @click="editEntity(item.id)" v-bind="props">mdi-pencil</v-icon>
                  </template>
                  <span>Edit Entity</span>
                </v-tooltip>
                 <v-tooltip top>
                  <template v-slot:activator="{ props }">
                    <v-icon small @click="openDeleteConfirmDialog(item)" v-bind="props">mdi-delete</v-icon>
                  </template>
                  <span>Delete Entity</span>
                </v-tooltip>
              </template>
            </v-data-table>

            <v-alert v-if="!entityStore.loading && !entities.length && !entityStore.error" type="info">
              No entities found. Click "Create New Entity" to add one.
            </v-alert>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteConfirmDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete</v-card-title>
        <v-card-text>
          Are you sure you want to delete the entity "<strong>{{ entityToDelete?.name }}</strong>"? 
          This action will also delete all its associated attributes and cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="closeDeleteConfirmDialog">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="confirmDeleteEntity" :loading="deleting">Delete</v-btn>
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
import { useEntityStore } from '@/store/entityStore';

const router = useRouter();
const entityStore = useEntityStore();

const entities = computed(() => entityStore.entities);

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Description', key: 'description', sortable: true },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const deleteConfirmDialog = ref(false);
const entityToDelete = ref(null);
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
  entityStore.fetchEntities();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString; // Fallback if date is invalid
  }
}

function goToCreateEntity() {
  router.push('/entities/new');
}

function viewEntityDetails(id) {
  router.push(`/entities/${id}/details`);
}

function editEntity(id) {
  router.push(`/entities/${id}/edit`);
}

function openDeleteConfirmDialog(entity) {
  entityToDelete.value = entity;
  deleteConfirmDialog.value = true;
}

function closeDeleteConfirmDialog() {
  entityToDelete.value = null;
  deleteConfirmDialog.value = false;
}

async function confirmDeleteEntity() {
  if (!entityToDelete.value) return;
  deleting.value = true;
  try {
    await entityStore.removeEntity(entityToDelete.value.id);
    showSnackbar(`Entity "${entityToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete entity: ${error.message || 'Unknown error'}`, 'error');
    // Error is also logged by the store
  } finally {
    deleting.value = false;
    closeDeleteConfirmDialog();
  }
}
</script>

<style scoped>
/* Scoped styles can be added here if needed, but Vuetify handles most styling. */
.v-data-table {
  margin-top: 16px;
}
.v-icon {
  cursor: pointer;
}
</style>
