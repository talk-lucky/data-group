<template>
  <v-container fluid>
    <v-card>
      <v-card-title class="d-flex justify-space-between align-center">
        Workflow Definitions
        <v-btn color="primary" to="/workflows/new">
          <v-icon left>mdi-plus-flowchart</v-icon> Add New Workflow
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-progress-linear v-if="workflowStore.loading" indeterminate color="primary"></v-progress-linear>
        
        <v-alert v-if="workflowStore.error" type="error" dismissible class="mb-4">
          Error fetching workflows: {{ workflowStore.error }}
        </v-alert>

        <v-data-table
          :headers="headers"
          :items="workflowStore.workflows"
          item-key="id"
          class="elevation-1"
          :loading="workflowStore.loading"
          no-data-text="No workflows found. Click 'Add New Workflow' to create one."
        >
          <template v-slot:item.description="{ item }">
            {{ item.description || 'N/A' }}
          </template>
          <template v-slot:item.is_enabled="{ item }">
            <v-chip :color="item.is_enabled ? 'success' : 'grey'" small>
              {{ item.is_enabled ? 'Enabled' : 'Disabled' }}
            </v-chip>
          </template>
           <template v-slot:item.trigger_type="{ item }">
            <v-chip small :color="getTriggerTypeColor(item.trigger_type)">{{ item.trigger_type }}</v-chip>
          </template>
          <template v-slot:item.created_at="{ item }">
            {{ formatDate(item.created_at) }}
          </template>
          <template v-slot:item.updated_at="{ item }">
            {{ formatDate(item.updated_at) }}
          </template>
          <template v-slot:item.actions="{ item }">
            <v-tooltip text="View Details">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small class="mr-2" @click="navigateToDetail(item.id)">mdi-eye</v-icon>
              </template>
            </v-tooltip>
            <v-tooltip text="Edit Workflow">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small class="mr-2" @click="navigateToEdit(item.id)">mdi-pencil</v-icon>
              </template>
            </v-tooltip>
            <v-tooltip text="Delete Workflow">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small @click="openDeleteConfirmDialog(item)">mdi-delete</v-icon>
              </template>
            </v-tooltip>
          </template>
        </v-data-table>
        
        <v-alert v-if="!workflowStore.loading && workflowStore.workflows.length === 0 && !workflowStore.error" type="info" class="mt-4">
          No workflows found. Click "Add New Workflow" to create one.
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Delete Workflow Confirmation Dialog -->
    <v-dialog v-model="deleteConfirmDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete Workflow</v-card-title>
        <v-card-text>
          Are you sure you want to delete the workflow "<strong>{{ workflowToDelete?.name }}</strong>"? 
          This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="closeDeleteConfirmDialog">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="confirmDelete" :loading="deleting">Delete</v-btn>
          <v-spacer></v-spacer>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" bottom right>
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn color="white" text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>
  </v-container>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useWorkflowStore } from '@/store/workflowStore';

const router = useRouter();
const workflowStore = useWorkflowStore();

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Enabled', key: 'is_enabled', sortable: true },
  { title: 'Trigger Type', key: 'trigger_type', sortable: true },
  { title: 'Description', key: 'description', sortable: true, width: '30%' },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const deleteConfirmDialog = ref(false);
const workflowToDelete = ref(null);
const deleting = ref(false);

const snackbar = ref({
  show: false,
  message: '',
  color: '',
  timeout: 3000,
});

onMounted(() => {
  workflowStore.fetchWorkflows();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString;
  }
}

function getTriggerTypeColor(type) {
  const colors = {
    'on_group_update': 'orange',
    'manual': 'blue-grey',
    'scheduled': 'teal',
  };
  return colors[type] || 'grey';
}

function showSnackbar(message, color = 'success') {
  snackbar.value.message = message;
  snackbar.value.color = color;
  snackbar.value.show = true;
}

function navigateToDetail(workflowId) {
  router.push(`/workflows/${workflowId}`);
}

function navigateToEdit(workflowId) {
  router.push(`/workflows/${workflowId}/edit`);
}

function openDeleteConfirmDialog(workflow) {
  workflowToDelete.value = workflow;
  deleteConfirmDialog.value = true;
}

function closeDeleteConfirmDialog() {
  workflowToDelete.value = null;
  deleteConfirmDialog.value = false;
}

async function confirmDelete() {
  if (!workflowToDelete.value) return;
  deleting.value = true;
  try {
    await workflowStore.removeWorkflow(workflowToDelete.value.id);
    showSnackbar(`Workflow "${workflowToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete workflow: ${error.message || 'Unknown error'}`, 'error');
  } finally {
    deleting.value = false;
    closeDeleteConfirmDialog();
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
