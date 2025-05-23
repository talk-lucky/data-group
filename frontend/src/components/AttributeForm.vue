<template>
  <v-card>
    <v-card-title class="text-h5">
      {{ isEditMode ? 'Edit Attribute' : 'Create New Attribute' }}
    </v-card-title>
    <v-card-text>
      <v-form ref="formRefAttr" @submit.prevent="handleSubmit">
        <v-text-field
          v-model="formData.name"
          label="Name*"
          :rules="[rules.required]"
          prepend-icon="mdi-text-short"
          variant="outlined"
          density="compact"
          class="mb-3"
        ></v-text-field>

        <v-select
          v-model="formData.data_type"
          :items="dataTypes"
          label="Data Type*"
          :rules="[rules.required]"
          prepend-icon="mdi-cogs"
          variant="outlined"
          density="compact"
          class="mb-3"
        ></v-select>

        <v-textarea
          v-model="formData.description"
          label="Description"
          prepend-icon="mdi-text-long"
          variant="outlined"
          rows="2"
          auto-grow
          density="compact"
          class="mb-3"
        ></v-textarea>

        <v-checkbox
          v-model="formData.is_filterable"
          label="Filterable"
          density="compact"
          class="mb-1"
        ></v-checkbox>

        <v-checkbox
          v-model="formData.is_pii"
          label="PII (Personally Identifiable Information)"
          density="compact"
          class="mb-1"
        ></v-checkbox>

        <v-checkbox
          v-model="formData.is_indexed"
          label="Indexed"
          density="compact"
          class="mb-3"
        ></v-checkbox>

        <v-alert v-if="formError || attributeStore.error" type="error" dense class="mb-4">
          {{ formError || attributeStore.error }}
        </v-alert>

        <v-progress-linear v-if="attributeStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      </v-form>
    </v-card-text>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-btn color="grey darken-1" text @click="cancelForm" :disabled="attributeStore.loading">
        Cancel
      </v-btn>
      <v-btn color="primary" @click="handleSubmit" :loading="attributeStore.loading" :disabled="!isFormValidAttr">
        {{ isEditMode ? 'Save Changes' : 'Create Attribute' }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useAttributeStore } from '@/store/attributeStore';

const props = defineProps({
  entityId: {
    type: String,
    required: true,
  },
  attributeId: { // For edit mode
    type: String,
    default: null,
  },
  initialData: { // Pre-fill form for edit mode
    type: Object,
    default: () => ({ name: '', data_type: '', description: '', is_filterable: false, is_pii: false, is_indexed: false }),
  },
});

const emit = defineEmits(['attribute-saved', 'cancel-form']);

const attributeStore = useAttributeStore();
const formRefAttr = ref(null); // Reference to the v-form
const isFormValidAttr = ref(false);

const isEditMode = computed(() => !!props.attributeId);
const dataTypes = ref(['String', 'Integer', 'Boolean', 'DateTime', 'Float', 'JSON', 'Text', 'Relationship']);

const formData = ref({
  name: '',
  data_type: '',
  description: '',
  is_filterable: false,
  is_pii: false,
  is_indexed: false,
});
const formError = ref(null);

const rules = {
  required: value => !!value || 'This field is required.',
};

// Populate form when initialData (for edit mode) changes or component is re-used
watch(() => props.initialData, (newData) => {
  attributeStore.error = null; // Clear previous store errors
  formError.value = null;
  if (newData && isEditMode.value) { // Check isEditMode as well
    formData.value.name = newData.name || '';
    formData.value.data_type = newData.data_type || '';
    formData.value.description = newData.description || '';
    formData.value.is_filterable = newData.is_filterable || false;
    formData.value.is_pii = newData.is_pii || false;
    formData.value.is_indexed = newData.is_indexed || false;
  } else {
    resetFormInternal(); // Reset if not in edit mode or no initial data
  }
   validateForm(); // Validate after setting data
}, { immediate: true, deep: true });


onMounted(() => {
  // Initial population if not covered by watch immediate
  if (isEditMode.value && props.initialData) {
    formData.value.name = props.initialData.name || '';
    formData.value.data_type = props.initialData.data_type || '';
    formData.value.description = props.initialData.description || '';
    formData.value.is_filterable = props.initialData.is_filterable || false;
    formData.value.is_pii = props.initialData.is_pii || false;
    formData.value.is_indexed = props.initialData.is_indexed || false;
  } else {
     resetFormInternal();
  }
   validateForm(); // Initial validation
});

function resetFormInternal() {
  formData.value.name = '';
  formData.value.data_type = '';
  formData.value.description = '';
  formData.value.is_filterable = false;
  formData.value.is_pii = false;
  formData.value.is_indexed = false;
  formError.value = null;
  if (formRefAttr.value) {
    formRefAttr.value.resetValidation();
  }
   validateForm();
}

async function validateForm() {
  if (formRefAttr.value) {
    const { valid } = await formRefAttr.value.validate();
    isFormValidAttr.value = valid;
  } else {
    isFormValidAttr.value = false;
  }
}

watch(formData, (). => {
  validateForm();
  formError.value = null; // Clear custom error on input
  attributeStore.error = null; // Clear store error on input
}, { deep: true });


async function handleSubmit() {
  await validateForm();
  if (!isFormValidAttr.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null;
  attributeStore.error = null;

  const attributeDataToSave = {
    name: formData.value.name,
    data_type: formData.value.data_type,
    description: formData.value.description,
    is_filterable: formData.value.is_filterable,
    is_pii: formData.value.is_pii,
    is_indexed: formData.value.is_indexed,
  };

  try {
    let savedAttribute;
    if (isEditMode.value) {
      savedAttribute = await attributeStore.editAttribute(props.entityId, props.attributeId, attributeDataToSave);
    } else {
      savedAttribute = await attributeStore.addAttribute(props.entityId, attributeDataToSave);
    }
    emit('attribute-saved', savedAttribute); // Pass saved attribute data back
    resetFormInternal();
  } catch (error) {
    formError.value = 'Operation failed: ' + (attributeStore.error || error.message || 'Unknown error');
    // Store error is preferred as it might contain more specific backend message
  }
}

function cancelForm() {
  resetFormInternal();
  emit('cancel-form');
}
</script>

<style scoped>
/* Styles for AttributeForm */
.mb-3 {
  margin-bottom: 16px !important; /* Vuetify's default spacing might be too small */
}
/* Ensure density="compact" fields are not too small if needed */
.v-text-field--density-compact, .v-select--density-compact, .v-textarea--density-compact {
  --v-input-control-height: 40px; /* Example: Adjust height */
}
</style>
