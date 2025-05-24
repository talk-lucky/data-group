<template>
  <v-container fluid>
    <v-card>
      <v-card-title>
        Field Mappings for Data Source: {{ sourceId }}
        <v-spacer></v-spacer>
        <v-btn color="primary" @click="openCreateMappingDialog">
          <v-icon left>mdi-plus</v-icon> Add New Mapping
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-progress-linear 
          v-if="fieldMappingStore.isLoading || entityStore.isLoading || attributeStore.isLoading && !showMappingDialog" 
          indeterminate 
          color="primary"
        ></v-progress-linear>
        
        <v-alert v-if="fieldMappingStore.error && !showMappingDialog" type="error" dismissible>
          Error fetching field mappings: {{ fieldMappingStore.error }}
        </v-alert>
         <v-alert v-if="entityStore.error && !showMappingDialog" type="error" dismissible>
          Error fetching entities for mapping: {{ entityStore.error }}
        </v-alert>
         <v-alert v-if="attributeStore.error && !showMappingDialog" type="error" dismissible>
          Error fetching attributes for mapping: {{ attributeStore.error }}
        </v-alert>

        <v-data-table
          v-if="!fieldMappingStore.isLoading || mappings.length"
          :headers="headers"
          :items="processedMappings"
          item-key="id"
          class="elevation-1"
          :loading="fieldMappingStore.isLoading && !showMappingDialog"
          no-data-text="No field mappings found for this data source."
        >
          <template v-slot:item.transformation_rule="{ item }">
            {{ item.transformation_rule || 'N/A' }}
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
                    <v-icon small class="mr-2" @click="openEditMappingDialog(item.originalItem)" v-bind="props">mdi-pencil</v-icon>
                </template>
                <span>Edit Mapping</span>
            </v-tooltip>
            <v-tooltip top>
                <template v-slot:activator="{ props }">
                    <v-icon small @click="openDeleteMappingDialog(item.originalItem)" v-bind="props">mdi-delete</v-icon>
                </template>
                <span>Delete Mapping</span>
            </v-tooltip>
          </template>
        </v-data-table>
        
        <v-alert v-if="!fieldMappingStore.isLoading && !mappings.length && !fieldMappingStore.error && !showMappingDialog" type="info">
          No field mappings found. Click "Add New Mapping" to create one.
        </v-alert>

      </v-card-text>
    </v-card>

    <!-- FieldMappingForm Dialog (for Create/Edit) -->
    <v-dialog v-model="showMappingDialog" persistent max-width="700px">
      <FieldMappingForm
        :source-id="sourceId"
        :mapping-id="mappingToEdit ? mappingToEdit.id : null"
        :initial-data="mappingToEdit"
        @mapping-saved="handleMappingSaved"
        @cancel-form="closeMappingDialog"
      />
    </v-dialog>

    <!-- Delete Mapping Confirmation Dialog -->
    <v-dialog v-model="deleteMappingDialog" persistent max-width="500px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete Mapping</v-card-title>
        <v-card-text>
          Are you sure you want to delete the mapping for "<strong>{{ mappingToDelete?.source_field_name }}</strong>"? 
          This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="closeDeleteMappingDialog">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="confirmDeleteMapping" :loading="deletingMapping">Delete</v-btn>
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
import { ref, computed, onMounted, watch } from 'vue';
import { useFieldMappingStore } from '@/store/fieldMappingStore';
import { useEntityStore } from '@/store/entityStore';
import { useAttributeStore } from '@/store/attributeStore';
import FieldMappingForm from './FieldMappingForm.vue';

const props = defineProps({
  sourceId: {
    type: String,
    required: true,
  },
});

const fieldMappingStore = useFieldMappingStore();
const entityStore = useEntityStore();
const attributeStore = useAttributeStore(); // For resolving names

const mappings = computed(() => fieldMappingStore.getMappingsForSource(props.sourceId));

const headers = ref([
  { title: 'Source Field Name', key: 'source_field_name', sortable: true },
  { title: 'Target Entity', key: 'entityName', sortable: true },
  { title: 'Target Attribute', key: 'attributeName', sortable: true },
  { title: 'Transformation Rule', key: 'transformation_rule', sortable: true },
  { title: 'Created At', key: 'created_at', sortable: true },
  { title: 'Updated At', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
]);

const showMappingDialog = ref(false);
const mappingToEdit = ref(null); 

const deleteMappingDialog = ref(false);
const mappingToDelete = ref(null);
const deletingMapping = ref(false);

const snackbar = ref({ show: false, message: '', color: '', timeout: 3000 });

function showSnackbar(message, color = 'success') {
  snackbar.value = { show: true, message, color, timeout: 3000 };
}

const processedMappings = computed(() => {
  return mappings.value.map(mapping => {
    const entity = entityStore.entities.find(e => e.id === mapping.entity_id);
    // Note: Attributes are fetched on demand in FieldMappingForm, or need a more robust way
    // to ensure they are loaded for the list view if names are required here directly.
    // For now, assuming attributeStore might have some attributes loaded, or this needs enhancement.
    // A simple approach: find attribute if its entity's attributes are loaded in attributeStore.
    let attributeName = 'N/A (Load Entity Attributes)';
    if (attributeStore.currentEntityIdForAttributes === mapping.entity_id) {
        const attribute = attributeStore.attributes.find(a => a.id === mapping.attribute_id);
        attributeName = attribute ? attribute.name : 'Attribute Not Found';
    } else if (entity) {
        // Potentially trigger a fetch for this entity's attributes if not showing names is an issue
        // console.warn(`Attributes for entity ${entity.name} not pre-loaded for mapping list.`);
        attributeName = `Attribute ID: ${mapping.attribute_id.substring(0,8)}...`;
    }


    return {
      ...mapping,
      entityName: entity ? entity.name : 'Entity Not Found',
      attributeName: attributeName,
      originalItem: mapping, // Keep a reference to the original item for actions
    };
  });
});


onMounted(async () => {
  if (props.sourceId) {
    fieldMappingStore.fetchFieldMappings(props.sourceId);
  }
  // Ensure entities are available for name resolution.
  // Only fetch if not already populated to avoid redundant calls.
  if (entityStore.entities.length === 0) {
    await entityStore.fetchEntities();
  }
  // Attributes are more complex as they are per-entity. The form handles fetching.
  // For the list, we might only show IDs or implement a more complex pre-loading strategy.
});

watch(() => props.sourceId, (newId) => {
  if (newId) {
    fieldMappingStore.fetchFieldMappings(newId);
  } else {
    fieldMappingStore.clearMappingsForSource(props.sourceId); // Clear for old ID if new one is null
  }
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try { return new Date(dateTimeString).toLocaleString(); } catch (e) { return dateTimeString; }
}

function openCreateMappingDialog() {
  mappingToEdit.value = null; 
  showMappingDialog.value = true;
}

function openEditMappingDialog(mapping) {
  mappingToEdit.value = { ...mapping }; 
  showMappingDialog.value = true;
}

function closeMappingDialog() {
  showMappingDialog.value = false;
  mappingToEdit.value = null; 
}

function handleMappingSaved(savedMapping) {
  closeMappingDialog();
  const message = mappingToEdit.value 
    ? `Mapping for "${savedMapping.source_field_name}" updated.`
    : `Mapping for "${savedMapping.source_field_name}" created.`;
  showSnackbar(message, 'success');
}

function openDeleteMappingDialog(mapping) {
  mappingToDelete.value = mapping;
  deleteMappingDialog.value = true;
}

function closeDeleteMappingDialog() {
  mappingToDelete.value = null;
  deleteMappingDialog.value = false;
}

async function confirmDeleteMapping() {
  if (!mappingToDelete.value) return;
  deletingMapping.value = true;
  try {
    await fieldMappingStore.removeFieldMapping(props.sourceId, mappingToDelete.value.id);
    showSnackbar(`Mapping for "${mappingToDelete.value.source_field_name}" deleted.`, 'success');
  } catch (error) {
    showSnackbar(`Failed to delete mapping: ${error.message || 'Unknown error'}`, 'error');
  } finally {
    deletingMapping.value = false;
    closeDeleteMappingDialog();
  }
}
</script>

<style scoped>
.v-data-table { margin-top: 16px; }
.v-icon { cursor: pointer; }
</style>
