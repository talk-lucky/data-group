<template>
  <v-container fluid>
    <v-card>
      <v-card-title>
        Attributes
        <v-spacer></v-spacer>
        <v-btn color="primary" @click="openCreateAttributeDialog">
          <v-icon left>mdi-plus</v-icon> Add New Attribute
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-progress-linear v-if="attributeStore.loading && !showAttributeDialog" indeterminate color="primary"></v-progress-linear>
        
        <v-alert v-if="attributeStore.error && !showAttributeDialog" type="error" dismissible>
          Error fetching attributes: {{ attributeStore.error }}
        </v-alert>

        <v-data-table
          v-if="!attributeStore.loading || attributes.length" 
          :headers="headers"
          :items="attributes"
          item-key="id"
          class="elevation-1"
          :loading="attributeStore.loading && !showAttributeDialog"
          no-data-text="No attributes found for this entity. Click 'Add New Attribute' to create one."
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
                    <v-icon small class="mr-2" @click="openEditAttributeDialog(item)" v-bind="props">mdi-pencil</v-icon>
                </template>
                <span>Edit Attribute</span>
            </v-tooltip>
            <v-tooltip top>
                <template v-slot:activator="{ props }">
                    <v-icon small @click="openDeleteAttributeConfirmDialog(item)" v-bind="props">mdi-delete</v-icon>
                </template>
                <span>Delete Attribute</span>
            </v-tooltip>
          </template>
        </v-data-table>
        
        <v-alert v-if="!attributeStore.loading && !attributes.length && !attributeStore.error && !showAttributeDialog" type="info">
          No attributes found for this entity. Click "Add New Attribute" to create one.
        </v-alert>

      </v-card-text>
    </v-card>

    <!-- Attribute Form Dialog (for Create/Edit) -->
    <v-dialog v-model="showAttributeDialog" persistent max-width="600px">
      <AttributeForm
        :entity-id="entityId"
        :attribute-id="attributeToEdit ? attributeToEdit.id : null"
        :initial-data="attributeToEdit"
        @attribute-saved="handleAttributeSaved"
        @cancel-form="closeAttributeDialog"
      />
    </v-dialog>

    <!-- Delete Attribute Confirmation Dialog -->
    <v-dialog v-model="deleteAttributeConfirmDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete Attribute</v-card-title>
        <v-card-text>
          Are you sure you want to delete the attribute "<strong>{{ attributeToDelete?.name }}</strong>"? 
          This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="closeDeleteAttributeConfirmDialog">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="confirmDeleteAttribute" :loading="deletingAttribute">Delete</v-btn>
          <v-spacer></v-spacer>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Snackbar for attribute notifications -->
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" bottom right>
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn color="white" text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useAttributeStore } from '@/store/attributeStore';
import AttributeForm from './AttributeForm.vue'; // Assuming AttributeForm is in the same directory

const props = defineProps({
  entityId: {
    type: String,
    required: true,
  },
});

const attributeStore = useAttributeStore();
const attributes = computed(() => attributeStore.attributes);

const headers = ref([
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Data Type', key: 'data_type', sortable: true },
  { title: 'Description', key: 'description', sortable: true },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const showAttributeDialog = ref(false);
const attributeToEdit = ref(null); // Used to pass data to AttributeForm for editing

const deleteAttributeConfirmDialog = ref(false);
const attributeToDelete = ref(null);
const deletingAttribute = ref(false);

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

// Fetch attributes when the component is mounted or entityId changes
onMounted(() => {
  if (props.entityId) {
    attributeStore.fetchAttributesForEntity(props.entityId);
  }
});

watch(() => props.entityId, (newId, oldId) => {
  if (newId && newId !== oldId) {
    attributeStore.clearAttributes(); // Clear old attributes
    attributeStore.fetchAttributesForEntity(newId);
  } else if (!newId) {
    attributeStore.clearAttributes();
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

function openCreateAttributeDialog() {
  attributeToEdit.value = null; // Ensure it's a create operation
  showAttributeDialog.value = true;
}

function openEditAttributeDialog(attribute) {
  attributeToEdit.value = { ...attribute }; // Pass a copy to avoid mutating store directly
  showAttributeDialog.value = true;
}

function closeAttributeDialog() {
  showAttributeDialog.value = false;
  attributeToEdit.value = null; // Clear editing state
  // No need to manually refresh attributes list if store updates reactively
}

function handleAttributeSaved(savedAttribute) {
  closeAttributeDialog();
  const message = attributeToEdit.value 
    ? `Attribute "${savedAttribute.name}" updated successfully.`
    : `Attribute "${savedAttribute.name}" created successfully.`;
  showSnackbar(message, 'success');
  // Attribute store should be updating the list reactively.
  // If not, might need: attributeStore.fetchAttributesForEntity(props.entityId);
}

function openDeleteAttributeConfirmDialog(attribute) {
  attributeToDelete.value = attribute;
  deleteAttributeConfirmDialog.value = true;
}

function closeDeleteAttributeConfirmDialog() {
  attributeToDelete.value = null;
  deleteAttributeConfirmDialog.value = false;
}

async function confirmDeleteAttribute() {
  if (!attributeToDelete.value) return;
  deletingAttribute.value = true;
  try {
    await attributeStore.removeAttribute(props.entityId, attributeToDelete.value.id);
    showSnackbar(`Attribute "${attributeToDelete.value.name}" deleted successfully.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete attribute: ${error.message || 'Unknown error'}`, 'error');
  } finally {
    deletingAttribute.value = false;
    closeDeleteAttributeConfirmDialog();
  }
}
</script>

<style scoped>
/* Styles for AttributeList */
.v-data-table {
  margin-top: 16px;
}
.v-icon {
  cursor: pointer;
}
</style>
