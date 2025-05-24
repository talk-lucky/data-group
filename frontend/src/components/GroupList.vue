<template>
  <v-container fluid>
    <v-card>
      <v-card-title class="d-flex justify-space-between align-center">
        Group Definitions
        <v-btn color="primary" to="/groups/new">
          <v-icon left>mdi-plus</v-icon> Add New Group
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-progress-linear v-if="groupStore.loading" indeterminate color="primary"></v-progress-linear>
        
        <v-alert v-if="groupStore.error" type="error" dismissible class="mb-4">
          Error fetching group definitions: {{ groupStore.error }}
        </v-alert>

        <v-data-table
          :headers="headers"
          :items="groupsWithEntityNames"
          item-key="id"
          class="elevation-1"
          :loading="groupStore.loading"
          no-data-text="No group definitions found. Click 'Add New Group' to create one."
        >
          <template v-slot:item.description="{ item }">
            {{ item.description || 'N/A' }}
          </template>
          <template v-slot:item.entity_name="{ item }">
             <router-link v-if="item.entity_id" :to="`/entities/${item.entity_id}`" class="text-decoration-none">
              {{ item.entity_name || 'N/A' }}
              <v-icon small class="ml-1">mdi-link-variant</v-icon>
            </router-link>
            <span v-else>{{ item.entity_name || 'N/A' }}</span>
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
            <v-tooltip text="Edit Group">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small class="mr-2" @click="navigateToEdit(item.id)">mdi-pencil</v-icon>
              </template>
            </v-tooltip>
            <v-tooltip text="Delete Group">
              <template v-slot:activator="{ props }">
                <v-icon v-bind="props" small @click="openDeleteConfirmDialog(item)">mdi-delete</v-icon>
              </template>
            </v-tooltip>
          </template>
        </v-data-table>
        
        <v-alert v-if="!groupStore.loading && groupsWithEntityNames.length === 0 && !groupStore.error" type="info" class="mt-4">
          No group definitions found. Click "Add New Group" to create one.
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Delete Group Confirmation Dialog -->
    <v-dialog v-model="deleteConfirmDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete Group Definition</v-card-title>
        <v-card-text>
          Are you sure you want to delete the group "<strong>{{ groupToDelete?.name }}</strong>"? 
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
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useGroupStore } from '@/store/groupStore';
import { useEntityStore } from '@/store/entityStore'; // To get entity names

const router = useRouter();
const groupStore = useGroupStore();
const entityStore = useEntityStore();

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Entity Name', key: 'entity_name', sortable: true },
  { title: 'Description', key: 'description', sortable: true },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const deleteConfirmDialog = ref(false);
const groupToDelete = ref(null);
const deleting = ref(false);

const snackbar = ref({
  show: false,
  message: '',
  color: '',
  timeout: 3000,
});

const groupsWithEntityNames = computed(() => {
  return groupStore.groups.map(group => {
    const entity = entityStore.entities.find(e => e.id === group.entity_id);
    return {
      ...group,
      entity_name: entity ? entity.name : 'Unknown Entity',
    };
  });
});

onMounted(async () => {
  // Fetch groups first
  await groupStore.fetchGroupDefinitions();
  // Then fetch entities if not already loaded, to map names
  if (entityStore.entities.length === 0) {
    await entityStore.fetchEntities();
  }
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString;
  }
}

function showSnackbar(message, color = 'success') {
  snackbar.value.message = message;
  snackbar.value.color = color;
  snackbar.value.show = true;
}

function navigateToDetail(groupId) {
  router.push(`/groups/${groupId}`);
}

function navigateToEdit(groupId) {
  router.push(`/groups/${groupId}/edit`);
}

function openDeleteConfirmDialog(group) {
  groupToDelete.value = group;
  deleteConfirmDialog.value = true;
}

function closeDeleteConfirmDialog() {
  groupToDelete.value = null;
  deleteConfirmDialog.value = false;
}

async function confirmDelete() {
  if (!groupToDelete.value) return;
  deleting.value = true;
  try {
    await groupStore.removeGroupDefinition(groupToDelete.value.id);
    showSnackbar(`Group "${groupToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete group: ${error.message || 'Unknown error'}`, 'error');
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
