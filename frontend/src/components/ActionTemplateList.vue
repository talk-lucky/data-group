<template>
  <v-container fluid>
    <v-card>
      <v-card-title class="d-flex justify-space-between align-center">
        Action Templates
        <v-btn color="primary" to="/actiontemplates/new">
          <v-icon left>mdi-plus</v-icon> Add New Template
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-progress-linear v-if="actionTemplateStore.loading" indeterminate color="primary"></v-progress-linear>
        
        <v-alert v-if="actionTemplateStore.error" type="error" dismissible class="mb-4">
          Error fetching action templates: {{ actionTemplateStore.error }}
        </v-alert>

        <v-data-table
          :headers="headers"
          :items="actionTemplateStore.actionTemplates"
          item-key="id"
          class="elevation-1"
          :loading="actionTemplateStore.loading"
          no-data-text="No action templates found. Click 'Add New Template' to create one."
        >
          <template v-slot:item.description="{ item }">
            {{ item.description || 'N/A' }}
          </template>
          <template v-slot:item.action_type="{ item }">
            <v-chip small :color="getActionTypeColor(item.action_type)">{{ item.action_type }}</v-chip>
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
            <v-tooltip text="Edit Template">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small class="mr-2" @click="navigateToEdit(item.id)">mdi-pencil</v-icon>
              </template>
            </v-tooltip>
            <v-tooltip text="Delete Template">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small @click="openDeleteConfirmDialog(item)">mdi-delete</v-icon>
              </template>
            </v-tooltip>
          </template>
        </v-data-table>
        
        <v-alert v-if="!actionTemplateStore.loading && actionTemplateStore.actionTemplates.length === 0 && !actionTemplateStore.error" type="info" class="mt-4">
          No action templates found. Click "Add New Template" to create one.
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Delete Action Template Confirmation Dialog -->
    <v-dialog v-model="deleteConfirmDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete Action Template</v-card-title>
        <v-card-text>
          Are you sure you want to delete the template "<strong>{{ templateToDelete?.name }}</strong>"? 
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
import { useActionTemplateStore } from '@/store/actionTemplateStore';

const router = useRouter();
const actionTemplateStore = useActionTemplateStore();

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Action Type', key: 'action_type', sortable: true },
  { title: 'Description', key: 'description', sortable: true },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const deleteConfirmDialog = ref(false);
const templateToDelete = ref(null);
const deleting = ref(false);

const snackbar = ref({
  show: false,
  message: '',
  color: '',
  timeout: 3000,
});

onMounted(() => {
  actionTemplateStore.fetchActionTemplates();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString;
  }
}

function getActionTypeColor(type) {
  const colors = {
    'webhook': 'blue',
    'email': 'green',
    'custom_api_call': 'purple',
  };
  return colors[type] || 'grey';
}

function showSnackbar(message, color = 'success') {
  snackbar.value.message = message;
  snackbar.value.color = color;
  snackbar.value.show = true;
}

function navigateToDetail(templateId) {
  router.push(`/actiontemplates/${templateId}`);
}

function navigateToEdit(templateId) {
  router.push(`/actiontemplates/${templateId}/edit`);
}

function openDeleteConfirmDialog(template) {
  templateToDelete.value = template;
  deleteConfirmDialog.value = true;
}

function closeDeleteConfirmDialog() {
  templateToDelete.value = null;
  deleteConfirmDialog.value = false;
}

async function confirmDelete() {
  if (!templateToDelete.value) return;
  deleting.value = true;
  try {
    await actionTemplateStore.removeActionTemplate(templateToDelete.value.id);
    showSnackbar(`Template "${templateToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete template: ${error.message || 'Unknown error'}`, 'error');
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
