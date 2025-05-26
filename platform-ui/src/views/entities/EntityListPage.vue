<template>
  <v-container fluid>
    <v-row align="center" class="mb-4">
      <v-col>
        <h1 class="text-h4">Entity Definitions</h1>
      </v-col>
      <v-col class="text-right">
        <v-btn color="primary" @click="openCreateDialog">
          <v-icon left>mdi-plus</v-icon>
          Add New Entity
        </v-btn>
      </v-col>
    </v-row>

    <v-progress-linear :active="isLoading" indeterminate color="primary" class="mb-2"></v-progress-linear>

    <v-alert v-if="error" type="error" density="compact" class="mb-4" closable @click:close="clearError">
      {{ error }}
    </v-alert>

    <v-card>
      <v-card-title>
        <v-text-field
          v-model="search"
          append-icon="mdi-magnify"
          label="Search Entities"
          single-line
          hide-details
          variant="underlined"
        ></v-text-field>
      </v-card-title>

      <v-data-table
        :headers="headers"
        :items="entities"
        :search="search"
        :loading="isLoading"
        item-key="id"
        class="elevation-1"
        density="compact"
      >
        <template v-slot:item.name="{ item }">
          <router-link :to="{ name: 'EntityDetail', params: { id: item.raw.id } }" class="entity-link">
            {{ item.raw.name }}
          </router-link>
        </template>
        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.raw.created_at) }}
        </template>
        <template v-slot:item.updated_at="{ item }">
          {{ formatDate(item.raw.updated_at) }}
        </template>
        <template v-slot:item.actions="{ item }">
          <v-tooltip location="top" text="Edit Entity">
            <template v-slot:activator="{ props: tooltipProps }">
              <v-icon v-bind="tooltipProps" small class="mr-2" @click="openEditDialog(item.raw)" color="primary">mdi-pencil</v-icon>
            </template>
          </v-tooltip>
          <v-tooltip location="top" text="Manage Attributes">
            <template v-slot:activator="{ props: tooltipProps }">
              <v-icon v-bind="tooltipProps" small class="mr-2" @click="navigateToEntityDetail(item.raw.id)" color="teal">mdi-format-list-bulleted-type</v-icon>
            </template>
          </v-tooltip>
          <v-tooltip location="top" text="Delete Entity">
            <template v-slot:activator="{ props: tooltipProps }">
              <v-icon v-bind="tooltipProps" small @click="openDeleteConfirmDialog(item.raw)" color="error">mdi-delete</v-icon>
            </template>
          </v-tooltip>
        </template>
         <template v-slot:no-data>
          <v-alert :value="true" color="info" icon="mdi-information-variant" class="ma-3">
            No entities found. Click "Add New Entity" to create one.
          </v-alert>
        </template>
      </v-data-table>
    </v-card>

    <entity-form
      :entity="selectedEntity"
      :dialog-visible="entityFormDialogVisible"
      @update:dialogVisible="entityFormDialogVisible = $event"
      @saved="handleEntitySaved"
      @error="handleEntityFormError"
    />

    <confirm-dialog
      :show="deleteConfirmDialogVisible"
      title="Confirm Delete"
      :message="`Are you sure you want to delete entity '${entityToDelete?.name || ''}'? This action cannot be undone.`"
      confirm-text="Delete"
      confirm-color="error"
      @update:show="deleteConfirmDialogVisible = $event"
      @confirm="confirmDelete"
      @cancel="cancelDelete"
    />

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" location="top right">
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router'; // Import useRouter
import { useEntityStore } from '@/store/entityStore';
import EntityForm from '@/components/entities/EntityForm.vue';
import ConfirmDialog from '@/components/common/ConfirmDialog.vue';

const entityStore = useEntityStore();
const router = useRouter(); // Initialize router

const entities = computed(() => entityStore.entities);
const isLoading = computed(() => entityStore.isLoading);
const error = computed({
  get: () => entityStore.error,
  set: (value) => { entityStore.error = value; }
});

const search = ref('');
const headers = ref([
  { title: 'Name', key: 'name', sortable: true, value: 'name' }, // Slot used for this column
  { title: 'Description', key: 'description', sortable: true, value: 'description' },
  { title: 'Created At', key: 'created_at', sortable: true, value: 'created_at' },
  { title: 'Updated At', key: 'updated_at', sortable: true, value: 'updated_at' },
  { title: 'Actions', key: 'actions', sortable: false, value: 'actions' },
]);

const entityFormDialogVisible = ref(false);
const selectedEntity = ref(null);

const deleteConfirmDialogVisible = ref(false);
const entityToDelete = ref(null);

const snackbar = ref({
  show: false,
  message: '',
  color: 'success',
  timeout: 3000,
});

onMounted(() => {
  entityStore.fetchEntities();
});

function clearError() {
  error.value = null;
}

function openCreateDialog() {
  selectedEntity.value = null;
  entityFormDialogVisible.value = true;
}

function openEditDialog(entity) {
  selectedEntity.value = { ...entity };
  entityFormDialogVisible.value = true;
}

function navigateToEntityDetail(entityId) {
  router.push({ name: 'EntityDetail', params: { id: entityId } });
}

function openDeleteConfirmDialog(entity) {
  entityToDelete.value = entity;
  deleteConfirmDialogVisible.value = true;
}

async function confirmDelete() {
  if (entityToDelete.value) {
    try {
      await entityStore.deleteEntity(entityToDelete.value.id);
      showSnackbar('Entity deleted successfully.', 'success');
    } catch (e) {
      showSnackbar(entityStore.error || 'Failed to delete entity.', 'error');
    }
  }
  deleteConfirmDialogVisible.value = false;
  entityToDelete.value = null;
}

function cancelDelete() {
  deleteConfirmDialogVisible.value = false;
  entityToDelete.value = null;
}

function handleEntitySaved() {
  showSnackbar('Entity saved successfully.', 'success');
}

function handleEntityFormError(errorMessage) {
  showSnackbar(errorMessage, 'error');
}

function showSnackbar(message, color = 'success', timeout = 3000) {
  snackbar.value = { show: true, message, color, timeout };
}

function formatDate(dateString) {
  if (!dateString) return 'N/A';
  const options = { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' };
  try {
    return new Date(dateString).toLocaleDateString(undefined, options);
  } catch (e) {
    return dateString;
  }
}

</script>

<style scoped>
/* Add any page-specific styles here */
.v-data-table {
  margin-top: 16px;
}
.entity-link {
  color: inherit; /* Or use Vuetify's theme colors if you prefer */
  text-decoration: none;
  font-weight: 500; /* Optional: make it slightly bolder */
}
.entity-link:hover {
  text-decoration: underline;
  color: rgb(var(--v-theme-primary)); /* Optional: use primary color on hover */
}
</style>
EOF
