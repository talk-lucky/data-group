<template>
  <v-card>
    <v-card-title class="text-h5">
      {{ isEditMode ? 'Edit Action Template' : 'Create New Action Template' }}
    </v-card-title>
    <v-card-text>
      <v-form ref="formRefActionTemplate" @submit.prevent="handleSubmit">
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

        <v-select
          v-model="formData.action_type"
          :items="actionTypes"
          label="Action Type*"
          :rules="[rules.required]"
          prepend-icon="mdi-cogs"
          variant="outlined"
          density="compact"
          class="mb-3"
          placeholder="Select an action type"
          @update:modelValue="handleActionTypeChange"
        ></v-select>

        <!-- Dynamic fields based on action_type -->
        <div v-if="formData.action_type === 'webhook'" class="mb-3">
          <v-text-field
            v-model="webhookConfig.url_template"
            label="Webhook URL Template*"
            :rules="[rules.required, rules.url]"
            prepend-icon="mdi-web"
            variant="outlined"
            density="compact"
            class="mb-2"
            hint="Use {{param_name}} for placeholders. E.g., https://api.example.com/notify?id={{entity_id}}"
          ></v-text-field>
           <v-select
            v-model="webhookConfig.method"
            :items="['POST', 'GET', 'PUT']"
            label="HTTP Method*"
            :rules="[rules.required]"
            prepend-icon="mdi-transfer"
            variant="outlined"
            density="compact"
            class="mb-2"
          ></v-select>
          <v-textarea
            v-model="webhookConfig.payload_template"
            label="Payload Template (JSON)"
            prepend-icon="mdi-code-json"
            variant="outlined"
            rows="3"
            auto-grow
            density="compact"
            :rules="[rules.jsonOrEmpty]"
            hint="JSON template for the request body. Use {{param_name}}. Leave empty for GET requests if not needed."
          ></v-textarea>
        </div>

        <div v-if="formData.action_type === 'email'" class="mb-3">
          <v-text-field
            v-model="emailConfig.subject_template"
            label="Email Subject Template*"
            :rules="[rules.required]"
            prepend-icon="mdi-format-title"
            variant="outlined"
            density="compact"
            class="mb-2"
            hint="Use {{param_name}} for placeholders."
          ></v-text-field>
          <v-textarea
            v-model="emailConfig.body_template"
            label="Email Body Template*"
            :rules="[rules.required]"
            prepend-icon="mdi-text-account"
            variant="outlined"
            rows="5"
            auto-grow
            density="compact"
            hint="HTML or plain text. Use {{param_name}} for placeholders."
          ></v-textarea>
        </div>
        
        <!-- Fallback generic template_content for other/custom types -->
         <v-textarea
            v-if="!['webhook', 'email'].includes(formData.action_type) && formData.action_type"
            v-model="formData.template_content"
            label="Template Content (JSON)*"
            :rules="[rules.required, rules.json]"
            prepend-icon="mdi-code-braces"
            variant="outlined"
            rows="5"
            auto-grow
            density="compact"
            class="mb-3"
            hint="Define the structure and placeholders for this action type as a JSON string."
        ></v-textarea>


        <v-alert v-if="formError || actionTemplateStore.error" type="error" dense class="mb-4">
          {{ formError || actionTemplateStore.error }}
        </v-alert>

        <v-progress-linear v-if="actionTemplateStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>
      </v-form>
    </v-card-text>
    <v-card-actions>
      <v-spacer></v-spacer>
      <v-btn color="grey darken-1" text @click="cancelForm" :disabled="actionTemplateStore.loading">
        Cancel
      </v-btn>
      <v-btn color="primary" @click="handleSubmit" :loading="actionTemplateStore.loading" :disabled="!isFormValid">
        {{ isEditMode ? 'Save Changes' : 'Create Template' }}
      </v-btn>
    </v-card-actions>
  </v-card>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useActionTemplateStore } from '@/store/actionTemplateStore';

const props = defineProps({
  templateId: { // For edit mode
    type: String,
    default: null,
  },
  initialData: { // Pre-fill form for edit mode
    type: Object,
    default: () => ({ name: '', description: '', action_type: '', template_content: '{}' }),
  },
});

const emit = defineEmits(['template-saved', 'cancel-form']);

const actionTemplateStore = useActionTemplateStore();
const formRefActionTemplate = ref(null);
const isFormValid = ref(false);

const isEditMode = computed(() => !!props.templateId);
const actionTypes = ref(['webhook', 'email', 'custom_api_call']); // Add more as needed

const formData = ref({
  name: '',
  description: '',
  action_type: '',
  template_content: '{}', // Stores the stringified JSON of type-specific config
});

// Type-specific config objects
const webhookConfig = ref({ url_template: '', method: 'POST', payload_template: '{}' });
const emailConfig = ref({ subject_template: '', body_template: '' });

const formError = ref(null);

const rules = {
  required: value => !!value || 'This field is required.',
  json: value => {
    try {
      JSON.parse(value);
      return true;
    } catch (e) {
      return 'Must be a valid JSON string.';
    }
  },
  jsonOrEmpty: value => {
    if (!value || value.trim() === '') return true;
    try {
      JSON.parse(value);
      return true;
    } catch (e) {
      return 'Must be a valid JSON string or empty.';
    }
  },
  url: value => {
    try {
      new URL(value.replace(/\{\{.*?\}\}/g, 'placeholder')); // Replace placeholders for validation
      return true;
    } catch (e) {
      if (value.includes('{{') && value.includes('}}')) { // Basic check for template usage
         // More lenient check for URLs with placeholders
         const pattern = /^(https?|ftp):\/\/[^\s/$.?#].[^\s]*$/i;
         return pattern.test(value.replace(/\{\{.*?\}\}/g, 'placeholder.com')) || 'Invalid URL format (even with placeholders).';
      }
      return 'Invalid URL format.';
    }
  }
};

function stringifyTemplateContent() {
  if (formData.value.action_type === 'webhook') {
    formData.value.template_content = JSON.stringify(webhookConfig.value);
  } else if (formData.value.action_type === 'email') {
    formData.value.template_content = JSON.stringify(emailConfig.value);
  }
  // For custom types, template_content is bound directly
}

function parseTemplateContent(contentString, actionType) {
  try {
    const parsed = JSON.parse(contentString || '{}');
    if (actionType === 'webhook') {
      webhookConfig.value.url_template = parsed.url_template || '';
      webhookConfig.value.method = parsed.method || 'POST';
      webhookConfig.value.payload_template = parsed.payload_template || '{}';
    } else if (actionType === 'email') {
      emailConfig.value.subject_template = parsed.subject_template || '';
      emailConfig.value.body_template = parsed.body_template || '';
    }
  } catch (e) {
    console.error("Error parsing template_content:", e);
    formError.value = "Failed to parse existing template content. Please verify.";
    // Reset to defaults if parsing fails for specific types
    if (actionType === 'webhook') webhookConfig.value = { url_template: '', method: 'POST', payload_template: '{}' };
    if (actionType === 'email') emailConfig.value = { subject_template: '', body_template: '' };
  }
}

function handleActionTypeChange(newActionType) {
    // When action type changes, reset specific configs and the generic template_content
    webhookConfig.value = { url_template: '', method: 'POST', payload_template: '{}' };
    emailConfig.value = { subject_template: '', body_template: '' };
    if (newActionType !== 'webhook' && newActionType !== 'email') {
        formData.value.template_content = '{}'; // Default for custom types
    } else {
        formData.value.template_content = ''; // Will be stringified from specific config
    }
    // Trigger validation for the new fields if any
    validateForm();
}

// Populate form when initialData changes
watch(() => props.initialData, (newData) => {
  actionTemplateStore.error = null;
  formError.value = null;
  if (newData) {
    formData.value.name = newData.name || '';
    formData.value.description = newData.description || '';
    formData.value.action_type = newData.action_type || '';
    formData.value.template_content = newData.template_content || '{}';
    if (isEditMode.value) {
      parseTemplateContent(formData.value.template_content, formData.value.action_type);
    }
  } else {
    resetFormInternal();
  }
  validateForm();
}, { immediate: true, deep: true });

onMounted(() => {
  // Initial population if not covered by watch
  if (isEditMode.value && props.initialData) {
    formData.value.name = props.initialData.name || '';
    formData.value.description = props.initialData.description || '';
    formData.value.action_type = props.initialData.action_type || '';
    formData.value.template_content = props.initialData.template_content || '{}';
    parseTemplateContent(formData.value.template_content, formData.value.action_type);
  } else {
     resetFormInternal();
  }
  validateForm();
});

function resetFormInternal() {
  formData.value.name = '';
  formData.value.description = '';
  formData.value.action_type = '';
  formData.value.template_content = '{}';
  webhookConfig.value = { url_template: '', method: 'POST', payload_template: '{}' };
  emailConfig.value = { subject_template: '', body_template: '' };
  formError.value = null;
  if (formRefActionTemplate.value) {
    formRefActionTemplate.value.resetValidation();
  }
  validateForm();
}

async function validateForm() {
  if (formRefActionTemplate.value) {
    const { valid } = await formRefActionTemplate.value.validate();
    isFormValid.value = valid;
  } else {
    isFormValid.value = false;
  }
}

watch([formData, webhookConfig, emailConfig], () => {
  validateForm();
  formError.value = null;
  actionTemplateStore.error = null;
  // Update template_content before validation might be needed if specific fields are complex
  // stringifyTemplateContent(); // This might cause feedback loops if not handled carefully
}, { deep: true });


async function handleSubmit() {
  stringifyTemplateContent(); // Ensure template_content is up-to-date
  await validateForm();
  if (!isFormValid.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null;
  actionTemplateStore.error = null;

  const templateDataToSave = {
    name: formData.value.name,
    description: formData.value.description,
    action_type: formData.value.action_type,
    template_content: formData.value.template_content,
  };

  try {
    let savedTemplate;
    if (isEditMode.value) {
      savedTemplate = await actionTemplateStore.editActionTemplate(props.templateId, templateDataToSave);
    } else {
      savedTemplate = await actionTemplateStore.addActionTemplate(templateDataToSave);
    }
    emit('template-saved', savedTemplate);
  } catch (error) {
    formError.value = 'Operation failed: ' + (actionTemplateStore.error || error.message || 'Unknown error');
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
.mb-2 {
  margin-bottom: 8px !important;
}
</style>
