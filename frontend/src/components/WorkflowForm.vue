<template>
  <v-card>
    <v-card-title class="text-h5">
      {{ isEditMode ? 'Edit Workflow Definition' : 'Create New Workflow Definition' }}
    </v-card-title>
    <v-card-text>
      <v-form ref="formRefWorkflow" @submit.prevent="handleSubmit">
        <v-text-field
          v-model="formData.name"
          label="Name*"
          :rules="[rules.required]"
          prepend-icon="mdi-label"
          variant="outlined"
          density="compact"
          class="mb-3"
        ></v-text-field>

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
          v-model="formData.is_enabled"
          label="Is Enabled"
          density="compact"
          class="mb-3"
        ></v-checkbox>

        <v-select
          v-model="formData.trigger_type"
          :items="triggerTypes"
          label="Trigger Type*"
          :rules="[rules.required]"
          prepend-icon="mdi-play-box-outline"
          variant="outlined"
          density="compact"
          class="mb-3"
          placeholder="Select a trigger type"
          @update:modelValue="handleTriggerTypeChange"
        ></v-select>

        <!-- Trigger Configuration -->
        <div v-if="formData.trigger_type === 'on_group_update'" class="mb-3 ml-4 pa-3" style="border: 1px solid #e0e0e0; border-radius: 4px;">
          <h4 class="text-subtitle-1 mb-2">Trigger Configuration: On Group Update</h4>
          <v-select
            v-model="triggerConfigData.group_id"
            :items="groupOptions"
            label="Group ID*"
            :rules="[rules.required]"
            prepend-icon="mdi-group"
            variant="outlined"
            density="compact"
            placeholder="Select the group that triggers this workflow"
          ></v-select>
        </div>
        <!-- Add more trigger_type configurations here as needed -->
         <div v-if="formData.trigger_type === 'manual'" class="mb-3 ml-4 pa-3" style="border: 1px solid #e0e0e0; border-radius: 4px;">
             <h4 class="text-subtitle-1 mb-1">Trigger Configuration: Manual</h4>
            <p class="text-caption">This workflow must be triggered manually via API or a dedicated UI action.</p>
        </div>


        <!-- Action Sequence Builder -->
        <WorkflowActionSequenceBuilder
          v-model="formData.action_sequence_json"
          class="mb-3"
        />
        
        <!-- For debugging action_sequence_json -->
        <!-- 
        <v-textarea
          v-model="formData.action_sequence_json"
          label="Action Sequence JSON (Debug)"
          readonly
          auto-grow
          rows="3"
          variant="outlined"
          density="compact"
          class="mt-2"
        ></v-textarea>
        -->

        <v-alert v-if="formError || workflowStore.error" type="error" dense class="mb-4">
          {{ formError || workflowStore.error }}
        </v-alert>

        <v-progress-linear v-if="workflowStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      </v-form>
    </v-card-text>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-btn color="grey darken-1" text @click="cancelForm" :disabled="workflowStore.loading">
        Cancel
      </v-btn>
      <v-btn color="primary" @click="handleSubmit" :loading="workflowStore.loading" :disabled="!isFormValid">
        {{ isEditMode ? 'Save Changes' : 'Create Workflow' }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useWorkflowStore } from '@/store/workflowStore';
import { useGroupStore } from '@/store/groupStore'; // For group selection
import { useActionTemplateStore } from '@/store/actionTemplateStore'; // To ensure templates are loaded for builder
import WorkflowActionSequenceBuilder from './WorkflowActionSequenceBuilder.vue';

const props = defineProps({
  workflowId: { // For edit mode
    type: String,
    default: null,
  },
  initialData: { // Pre-fill form for edit mode
    type: Object,
    default: () => ({ 
      name: '', 
      description: '', 
      is_enabled: true, 
      trigger_type: '', 
      trigger_config: '{}', 
      action_sequence_json: '[]' 
    }),
  },
});

const emit = defineEmits(['workflow-saved', 'cancel-form']);

const workflowStore = useWorkflowStore();
const groupStore = useGroupStore();
const actionTemplateStore = useActionTemplateStore(); // To ensure templates are loaded

const formRefWorkflow = ref(null);
const isFormValid = ref(false);

const isEditMode = computed(() => !!props.workflowId);
const triggerTypes = ref(['on_group_update', 'manual', 'scheduled']); // Add more as needed

const formData = ref({
  name: '',
  description: '',
  is_enabled: true,
  trigger_type: '',
  trigger_config: '{}', // Stores the stringified JSON of type-specific trigger config
  action_sequence_json: '[]', // Stores the stringified JSON of action sequence
});

// Specific config data object for "on_group_update"
const triggerConfigData = ref({ group_id: null });

const formError = ref(null);

const rules = {
  required: value => !!value || 'This field is required.',
  json: value => { // For direct JSON editing if needed, not for structured config
    try {
      JSON.parse(value);
      return true;
    } catch (e) {
      return 'Must be a valid JSON string.';
    }
  }
};

const groupOptions = computed(() => {
  return groupStore.groups.map(group => ({
    title: group.name,
    value: group.id,
  }));
});

// Stringify trigger_config before saving
function stringifyTriggerConfig() {
  if (formData.value.trigger_type === 'on_group_update') {
    formData.value.trigger_config = JSON.stringify(triggerConfigData.value);
  } else if (formData.value.trigger_type === 'manual') {
    formData.value.trigger_config = JSON.stringify({}); // Empty config for manual
  } else {
    // For other types, ensure trigger_config is a valid JSON string or reset
    try {
      JSON.parse(formData.value.trigger_config);
    } catch(e) {
      formData.value.trigger_config = '{}'; // Default to empty JSON object if invalid
    }
  }
}

// Parse trigger_config when loading data
function parseTriggerConfig(configString, triggerType) {
  try {
    const parsed = JSON.parse(configString || '{}');
    if (triggerType === 'on_group_update') {
      triggerConfigData.value.group_id = parsed.group_id || null;
    }
    // Add parsers for other trigger types if they have structured config
  } catch (e) {
    console.error("Error parsing trigger_config:", e);
    formError.value = "Failed to parse existing trigger configuration.";
    // Reset to defaults if parsing fails
    if (triggerType === 'on_group_update') triggerConfigData.value = { group_id: null };
  }
}

function handleTriggerTypeChange(newTriggerType) {
    // Reset specific configs and the generic trigger_config when type changes
    triggerConfigData.value = { group_id: null }; 
    
    if (newTriggerType === 'on_group_update') {
        formData.value.trigger_config = JSON.stringify(triggerConfigData.value);
    } else if (newTriggerType === 'manual') {
        formData.value.trigger_config = JSON.stringify({});
    } else {
        formData.value.trigger_config = '{}'; // Default for other or custom types
    }
    validateForm(); // Re-validate
}


// Populate form when initialData changes
watch(() => props.initialData, (newData) => {
  workflowStore.error = null;
  formError.value = null;
  if (newData) {
    formData.value.name = newData.name || '';
    formData.value.description = newData.description || '';
    formData.value.is_enabled = typeof newData.is_enabled === 'boolean' ? newData.is_enabled : true;
    formData.value.trigger_type = newData.trigger_type || '';
    formData.value.trigger_config = newData.trigger_config || '{}';
    formData.value.action_sequence_json = newData.action_sequence_json || '[]';

    if (isEditMode.value || formData.value.trigger_type) { // Parse only if type is known or in edit mode
      parseTriggerConfig(formData.value.trigger_config, formData.value.trigger_type);
    }
  } else {
    resetFormInternal();
  }
  validateForm();
}, { immediate: true, deep: true });


onMounted(() => {
  groupStore.fetchGroupDefinitions(); // For group selection
  actionTemplateStore.fetchActionTemplates(); // Ensure templates are available for the builder

  if (isEditMode.value && props.initialData) {
    // Data already set by watch, but ensure parsing happens if type is already set
    if (props.initialData.trigger_type) {
        parseTriggerConfig(props.initialData.trigger_config, props.initialData.trigger_type);
    }
  } else if (!isEditMode.value) {
     resetFormInternal();
  }
  validateForm();
});

function resetFormInternal() {
  formData.value.name = '';
  formData.value.description = '';
  formData.value.is_enabled = true;
  formData.value.trigger_type = '';
  formData.value.trigger_config = '{}';
  formData.value.action_sequence_json = '[]';
  triggerConfigData.value = { group_id: null }; // Reset specific config
  formError.value = null;
  if (formRefWorkflow.value) {
    formRefWorkflow.value.resetValidation();
  }
  validateForm();
}

async function validateForm() {
  if (formRefWorkflow.value) {
    const { valid } = await formRefWorkflow.value.validate();
    isFormValid.value = valid;
  } else {
    isFormValid.value = false;
  }
}

// Watch for changes in formData or specific config data to re-validate
watch([formData, triggerConfigData], () => {
  validateForm();
  formError.value = null;
  workflowStore.error = null;
}, { deep: true });

async function handleSubmit() {
  stringifyTriggerConfig(); // Ensure trigger_config is up-to-date
  
  // Validate action_sequence_json (basic check for non-empty array if needed)
  try {
    const actions = JSON.parse(formData.value.action_sequence_json);
    if (!Array.isArray(actions)) throw new Error("Action sequence must be an array.");
    // Potentially add more validation here if needed, e.g., ensuring each step has an action_template_id
  } catch (e) {
    formError.value = "Action sequence is not valid JSON or is malformed.";
    isFormValid.value = false; // Mark form as invalid
    return;
  }

  await validateForm(); // Final validation call
  if (!isFormValid.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null;
  workflowStore.error = null;

  const workflowDataToSave = { ...formData.value };

  try {
    let savedWorkflow;
    if (isEditMode.value) {
      savedWorkflow = await workflowStore.editWorkflow(props.workflowId, workflowDataToSave);
    } else {
      savedWorkflow = await workflowStore.addWorkflow(workflowDataToSave);
    }
    emit('workflow-saved', savedWorkflow);
  } catch (error) {
    formError.value = 'Operation failed: ' + (workflowStore.error || error.message || 'Unknown error');
  }
}

function cancelForm() {
  emit('cancel-form');
}
</script>

<style scoped>
.mb-3 {
  margin-bottom: 16px !important;
}
</style>
