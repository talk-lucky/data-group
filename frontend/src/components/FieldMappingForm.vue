<template>
  <v-card>
    <v-card-title class="text-h5">
      {{ isEditMode ? 'Edit Field Mapping' : 'Create New Field Mapping' }}
    </v-card-title>
    <v-card-text>
      <v-form ref="formRefMapping" @submit.prevent="handleSubmit">
        <v-text-field
          v-model="formData.source_field_name"
          label="Source Field Name*"
          :rules="[rules.required]"
          prepend-icon="mdi-table-column"
          variant="outlined"
          density="compact"
          class="mb-3"
        ></v-text-field>

        <v-select
          v-model="selectedEntityId"
          :items="entityStore.entityOptions"
          label="Target Entity*"
          :rules="[rules.required]"
          prepend-icon="mdi-alpha-e-box-outline"
          variant="outlined"
          density="compact"
          class="mb-3"
          :loading="entityStore.isLoading"
          @update:modelValue="handleEntityChange"
        ></v-select>

        <v-select
          v-model="formData.attribute_id"
          :items="attributeOptionsForSelectedEntity"
          label="Target Attribute*"
          :rules="[rules.required]"
          prepend-icon="mdi-alpha-a-box-outline"
          variant="outlined"
          density="compact"
          class="mb-3"
          :loading="attributeStore.isLoading"
          :disabled="!selectedEntityId || attributeStore.isLoading"
          no-data-text="Select an entity first, or no attributes found."
        ></v-select>

        <v-text-field
          v-model="formData.transformation_rule"
          label="Transformation Rule (Optional)"
          prepend-icon="mdi-cogs"
          variant="outlined"
          density="compact"
          class="mb-3"
          hint="e.g., lowercase, trim, date_format:YYYY-MM-DD"
        ></v-text-field>

        <v-alert v-if="formError || fieldMappingStore.error" type="error" dense class="mb-4">
          {{ formError || fieldMappingStore.error }}
        </v-alert>

        <v-progress-linear v-if="fieldMappingStore.isLoading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      </v-form>
    </v-card-text>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-btn color="grey darken-1" text @click="cancelForm" :disabled="fieldMappingStore.isLoading">
        Cancel
      </v-btn>
      <v-btn color="primary" @click="handleSubmit" :loading="fieldMappingStore.isLoading" :disabled="!isFormValidMapping">
        {{ isEditMode ? 'Save Changes' : 'Create Mapping' }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useFieldMappingStore } from '@/store/fieldMappingStore';
import { useEntityStore } from '@/store/entityStore';
import { useAttributeStore } from '@/store/attributeStore';

const props = defineProps({
  sourceId: { type: String, required: true },
  mappingId: { type: String, default: null }, // For edit mode
  initialData: { type: Object, default: () => ({ source_field_name: '', entity_id: '', attribute_id: '', transformation_rule: '' }) },
});

const emit = defineEmits(['mapping-saved', 'cancel-form']);

const fieldMappingStore = useFieldMappingStore();
const entityStore = useEntityStore();
const attributeStore = useAttributeStore();

const formRefMapping = ref(null);
const isFormValidMapping = ref(false);
const isEditMode = computed(() => !!props.mappingId);

const formData = ref({
  source_field_name: '',
  entity_id: '', // This will be bound to selectedEntityId indirectly
  attribute_id: '',
  transformation_rule: '',
});
const selectedEntityId = ref(''); // Separate ref for entity selection to trigger attribute loading
const formError = ref(null);

const rules = {
  required: value => !!value || 'This field is required.',
};

// Fetch entities when component mounts if not already available
onMounted(async () => {
  if (entityStore.entities.length === 0) {
    await entityStore.fetchEntities();
  }
  // If in edit mode, set initial data and fetch attributes for the selected entity
  if (isEditMode.value && props.initialData) {
    formData.value = { ...props.initialData };
    selectedEntityId.value = props.initialData.entity_id; // Trigger attribute load
    if (selectedEntityId.value) {
        await attributeStore.fetchAttributesForEntity(selectedEntityId.value);
        // Ensure the attribute_id from initialData is correctly set after attributes are loaded
        formData.value.attribute_id = props.initialData.attribute_id;
    }
  } else {
    resetFormInternal();
  }
  validateForm();
});

// Watch for changes in initialData to populate form for editing
watch(() => props.initialData, async (newData) => {
  fieldMappingStore.error = null;
  formError.value = null;
  if (newData && isEditMode.value) {
    formData.value.source_field_name = newData.source_field_name || '';
    formData.value.transformation_rule = newData.transformation_rule || '';
    selectedEntityId.value = newData.entity_id || ''; // This will trigger attribute fetch via its own watcher
     if (selectedEntityId.value) {
        // Wait for attributes to be fetched before setting attribute_id
        await attributeStore.fetchAttributesForEntity(selectedEntityId.value);
        formData.value.attribute_id = newData.attribute_id || '';
    } else {
        formData.value.attribute_id = ''; // Clear attribute if entity is cleared
    }
  } else if (!isEditMode.value) {
    resetFormInternal();
  }
  validateForm();
}, { deep: true, immediate: false }); // immediate: false to avoid race with onMounted

// When selectedEntityId changes, fetch its attributes
watch(selectedEntityId, async (newEntityId, oldEntityId) => {
  formData.value.entity_id = newEntityId; // Keep formData.entity_id in sync
  if (newEntityId) {
    // Clear previous attribute selection only if entity actually changes
    if (newEntityId !== oldEntityId) {
        formData.value.attribute_id = ''; // Reset attribute selection
        attributeStore.clearAttributes(); // Clear old attributes from store/state
    }
    await attributeStore.fetchAttributesForEntity(newEntityId);
  } else {
    formData.value.attribute_id = ''; // Clear attribute if no entity selected
    attributeStore.clearAttributes();
  }
   validateForm(); // Re-validate as attribute options change
}, { immediate: false }); // immediate: false to avoid issues on initial load if onMounted handles it


const attributeOptionsForSelectedEntity = computed(() => {
  if (!selectedEntityId.value || attributeStore.currentEntityIdForAttributes !== selectedEntityId.value) {
    return []; // No entity selected or attributes not for this entity
  }
  return attributeStore.attributeOptions; // Use the getter from attributeStore
});

function resetFormInternal() {
  formData.value = { source_field_name: '', entity_id: '', attribute_id: '', transformation_rule: '' };
  selectedEntityId.value = '';
  formError.value = null;
  if (formRefMapping.value) {
    formRefMapping.value.resetValidation();
  }
  attributeStore.clearAttributes();
  validateForm();
}

async function validateForm() {
  if (formRefMapping.value) {
    const { valid } = await formRefMapping.value.validate();
    isFormValidMapping.value = valid;
  } else {
    isFormValidMapping.value = false;
  }
}

watch(formData, () => {
  validateForm();
  formError.value = null;
  fieldMappingStore.error = null;
}, { deep: true });

async function handleSubmit() {
  await validateForm();
  if (!isFormValidMapping.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null;
  fieldMappingStore.error = null;

  const mappingDataToSave = { ...formData.value };

  try {
    let savedMapping;
    if (isEditMode.value) {
      savedMapping = await fieldMappingStore.editFieldMapping(props.sourceId, props.mappingId, mappingDataToSave);
    } else {
      savedMapping = await fieldMappingStore.addFieldMapping(props.sourceId, mappingDataToSave);
    }
    emit('mapping-saved', savedMapping);
    resetFormInternal();
  } catch (error) {
    formError.value = 'Operation failed: ' + (fieldMappingStore.error || error.message || 'Unknown error');
  }
}

function cancelForm() {
  resetFormInternal();
  emit('cancel-form');
}
</script>

<style scoped>
.mb-3 { margin-bottom: 16px !important; }
</style>
