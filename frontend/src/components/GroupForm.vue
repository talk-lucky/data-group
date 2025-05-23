<template>
  <v-card>
    <v-card-title class="text-h5">
      {{ isEditMode ? 'Edit Group Definition' : 'Create New Group Definition' }}
    </v-card-title>
    <v-card-text>
      <v-form ref="formRefGroup" @submit.prevent="handleSubmit">
        <v-text-field
          v-model="formData.name"
          label="Name*"
          :rules="[rules.required]"
          prepend-icon="mdi-label"
          variant="outlined"
          density="compact"
          class="mb-3"
        ></v-text-field>

        <v-select
          v-model="formData.entity_id"
          :items="entityItems"
          label="Entity*"
          :rules="[rules.required]"
          prepend-icon="mdi-shape-outline"
          variant="outlined"
          density="compact"
          class="mb-3"
          :disabled="isEditMode" 
          placeholder="Select the entity this group applies to"
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

        <GroupRuleBuilder 
          v-model="formData.rules_json"
          :entity-id="formData.entity_id"
          class="mb-3"
        />
        
        <!-- For debugging rules_json -->
        <!-- 
        <v-textarea
          v-model="formData.rules_json"
          label="Rules JSON (Debug)"
          readonly
          auto-grow
          rows="3"
          variant="outlined"
          density="compact"
          class="mt-2"
        ></v-textarea>
        -->

        <v-alert v-if="formError || groupStore.error" type="error" dense class="mb-4">
          {{ formError || groupStore.error }}
        </v-alert>

        <v-progress-linear v-if="groupStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      </v-form>
    </v-card-text>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-btn color="grey darken-1" text @click="cancelForm" :disabled="groupStore.loading">
        Cancel
      </v-btn>
      <v-btn color="primary" @click="handleSubmit" :loading="groupStore.loading" :disabled="!isFormValid">
        {{ isEditMode ? 'Save Changes' : 'Create Group' }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useGroupStore } from '@/store/groupStore';
import { useEntityStore } from '@/store/entityStore';
import GroupRuleBuilder from './GroupRuleBuilder.vue';

const props = defineProps({
  groupId: { // For edit mode
    type: String,
    default: null,
  },
  initialData: { // Pre-fill form for edit mode
    type: Object,
    default: () => ({ name: '', entity_id: null, description: '', rules_json: '' }),
  },
});

const emit = defineEmits(['group-saved', 'cancel-form']);

const groupStore = useGroupStore();
const entityStore = useEntityStore();

const formRefGroup = ref(null);
const isFormValid = ref(false);

const isEditMode = computed(() => !!props.groupId);

const formData = ref({
  name: '',
  entity_id: null,
  description: '',
  rules_json: '',
});
const formError = ref(null);

const rules = {
  required: value => !!value || 'This field is required.',
  // Basic JSON validation for rules_json if needed, though builder should ensure valid JSON
  // json: value => {
  //   if (!value) return true; // Allow empty rules if that's valid
  //   try {
  //     const parsed = JSON.parse(value);
  //     return (typeof parsed === 'object' && parsed !== null) || 'Must be a valid JSON object.';
  //   } catch (e) {
  //     return 'Must be a valid JSON string.';
  //   }
  // }
};

const entityItems = computed(() => {
  return entityStore.entities.map(entity => ({
    title: entity.name,
    value: entity.id,
  }));
});

// Populate form when initialData (for edit mode) changes or component is re-used
watch(() => props.initialData, (newData) => {
  groupStore.error = null; // Clear previous store errors
  formError.value = null;
  if (newData && isEditMode.value) {
    formData.value.name = newData.name || '';
    formData.value.entity_id = newData.entity_id || null;
    formData.value.description = newData.description || '';
    formData.value.rules_json = newData.rules_json || '';
  } else {
    resetFormInternal();
  }
  validateForm();
}, { immediate: true, deep: true });


onMounted(() => {
  entityStore.fetchEntities(); // Fetch entities for the dropdown
  // Initial population if not covered by watch immediate
  if (isEditMode.value && props.initialData) {
    formData.value.name = props.initialData.name || '';
    formData.value.entity_id = props.initialData.entity_id || null;
    formData.value.description = props.initialData.description || '';
    formData.value.rules_json = props.initialData.rules_json || '';
  } else {
     resetFormInternal();
  }
  validateForm();
});

function resetFormInternal() {
  formData.value.name = '';
  formData.value.entity_id = null;
  formData.value.description = '';
  formData.value.rules_json = ''; // Reset rules JSON
  formError.value = null;
  if (formRefGroup.value) {
    formRefGroup.value.resetValidation();
  }
  validateForm();
}

async function validateForm() {
  if (formRefGroup.value) {
    const { valid } = await formRefGroup.value.validate();
    isFormValid.value = valid;
  } else {
    isFormValid.value = false;
  }
}

watch(formData, () => {
  validateForm();
  formError.value = null; // Clear custom error on input
  groupStore.error = null; // Clear store error on input
}, { deep: true });


async function handleSubmit() {
  await validateForm();
  if (!isFormValid.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  // Ensure rules_json is not empty if rules are defined, or provide a default empty structure
  if (formData.value.rules_json === '' && formData.value.entity_id) {
     // If entity_id is set, rules_json should ideally be an empty structure if no rules are defined
     // For example: JSON.stringify({ logical_operator: "AND", conditions: [] })
     // However, the backend might handle empty string for rules_json gracefully.
     // For now, allow empty string.
  } else if (formData.value.rules_json !== '') {
    try {
      JSON.parse(formData.value.rules_json); // Validate JSON before submitting
    } catch (e) {
      formError.value = 'Rules JSON is invalid. Please check the rule builder.';
      return;
    }
  }


  formError.value = null;
  groupStore.error = null;

  const groupDataToSave = {
    name: formData.value.name,
    entity_id: formData.value.entity_id,
    description: formData.value.description,
    rules_json: formData.value.rules_json || '', // Send empty string if null/undefined
  };

  try {
    let savedGroup;
    if (isEditMode.value) {
      savedGroup = await groupStore.editGroupDefinition(props.groupId, groupDataToSave);
    } else {
      savedGroup = await groupStore.addGroupDefinition(groupDataToSave);
    }
    emit('group-saved', savedGroup);
    // resetFormInternal(); // Usually called by parent view after successful save and navigation
  } catch (error) {
    // Error is already set in store, formError can show additional frontend specific messages
    formError.value = 'Operation failed: ' + (groupStore.error || error.message || 'Unknown error');
  }
}

function cancelForm() {
  // resetFormInternal(); // Parent view might handle reset or navigation
  emit('cancel-form');
}
</script>

<style scoped>
.mb-3 {
  margin-bottom: 16px !important;
}
</style>
