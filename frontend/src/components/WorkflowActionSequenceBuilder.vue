<template>
  <v-container>
    <v-card variant="outlined" class="pa-3">
      <v-card-title class="text-subtitle-1 pb-1">
        Action Sequence
      </v-card-title>
      <v-card-subtitle class="pb-2">
        Define the sequence of actions for this workflow.
      </v-card-subtitle>
      <v-divider class="mb-3"></v-divider>

      <div v-if="localActionSequence.length === 0" class="text-center pa-3">
        <v-alert type="info" variant="tonal">
          No actions defined yet. Click "Add Action Step" to begin.
        </v-alert>
      </div>

      <v-list v-else dense class="pa-0">
        <draggable 
          v-model="localActionSequence" 
          item-key="id" 
          handle=".drag-handle"
          @end="onSequenceChange"
        >
          <template #item="{element: step, index}">
            <v-list-item class="mb-3 pa-0">
              <v-card variant="tonal">
                <v-card-text class="pb-2">
                  <v-row align="center" dense>
                    <v-col cols="auto" class="drag-handle" style="cursor: move;">
                      <v-icon>mdi-drag-vertical</v-icon>
                    </v-col>
                    <v-col>
                      <div class="text-subtitle-2">Step {{ index + 1 }}: {{ getTemplateName(step.action_template_id) }}</div>
                       <div class="text-caption grey--text">
                        Type: {{ getTemplateType(step.action_template_id) }}
                      </div>
                    </v-col>
                    <v-col cols="auto">
                      <v-btn icon="mdi-delete" variant="text" color="error" @click="removeActionStep(index)" density="compact"></v-btn>
                    </v-col>
                  </v-row>

                  <v-select
                    v-model="step.action_template_id"
                    :items="actionTemplateOptions"
                    label="Action Template"
                    placeholder="Select Action Template"
                    variant="outlined"
                    density="compact"
                    hide-details
                    class="mt-2 mb-2"
                    @update:modelValue="onSequenceChange"
                  ></v-select>

                  <v-textarea
                    v-model="step.parameters_json"
                    label="Parameters (JSON)"
                    placeholder='{"key": "value"}'
                    variant="outlined"
                    density="compact"
                    rows="2"
                    auto-grow
                    hide-details
                    :rules="[rules.jsonOrEmptyObject]"
                    class="mb-1"
                    @input="onSequenceChange"
                  ></v-textarea>
                  <div v-if="getTemplateDescription(step.action_template_id)" class="text-caption pa-1" style="background-color: #f9f9f9; border-radius: 4px;">
                     <strong>Template Description:</strong> {{ getTemplateDescription(step.action_template_id) }}
                  </div>
                </v-card-text>
              </v-card>
            </v-list-item>
          </template>
        </draggable>
      </v-list>

      <v-btn color="primary" @click="addActionStep" class="mt-3">
        <v-icon left>mdi-plus-box-multiple</v-icon> Add Action Step
      </v-btn>

      <v-textarea
        v-if="debugMode"
        v-model="currentJsonOutput"
        label="Current JSON Output (Debug)"
        readonly
        auto-grow
        rows="3"
        class="mt-4"
        variant="outlined"
        density="compact"
      ></v-textarea>
      <v-alert v-if="parsingError" type="error" dense class="mt-3">
        Error parsing existing action sequence: {{ parsingError }}. Sequence has been reset.
      </v-alert>
    </v-card>
  </v-container>
</template>

<script setup>
import { ref, watch, computed, onMounted } from 'vue';
import { useActionTemplateStore } from '@/store/actionTemplateStore'; // To get options
import draggable from 'vuedraggable'; // For reordering

// Simple unique ID generator for list items (not for backend)
const generateLocalId = () => `step_${Math.random().toString(36).substr(2, 9)}`;

const props = defineProps({
  modelValue: { // JSON string of the action sequence
    type: String,
    default: '[]', // Default to an empty array JSON string
  },
  // actionTemplates prop is removed, will use store directly
});

const emit = defineEmits(['update:modelValue']);

const actionTemplateStore = useActionTemplateStore();
const localActionSequence = ref([]); // Array of { id (local), action_template_id: '', parameters_json: '{}' }
const parsingError = ref(null);
const debugMode = ref(false); // Set to true for debugging

const actionTemplateOptions = computed(() => actionTemplateStore.actionTemplateOptions);

const rules = {
  jsonOrEmptyObject: value => {
    if (!value || value.trim() === '' || value.trim() === '{}') return true;
    try {
      const parsed = JSON.parse(value);
      return (typeof parsed === 'object' && parsed !== null) || 'Must be a valid JSON object or empty object {}.';
    } catch (e) {
      return 'Must be a valid JSON string or empty object {}.';
    }
  }
};

const currentJsonOutput = computed(() => {
  const sequenceToEmit = localActionSequence.value.map(step => ({
    action_template_id: step.action_template_id,
    parameters_json: step.parameters_json || '{}', // Ensure it's always a string
  }));
  return JSON.stringify(sequenceToEmit, null, 2);
});

function getTemplateName(templateId) {
  const template = actionTemplateOptions.value.find(t => t.value === templateId);
  return template ? template.title.split(' (')[0] : 'Unknown Template';
}
function getTemplateType(templateId) {
  const template = actionTemplateOptions.value.find(t => t.value === templateId);
  return template ? template.action_type : 'N/A';
}
function getTemplateDescription(templateId) {
  const template = actionTemplateOptions.value.find(t => t.value === templateId);
  return template ? template.description : '';
}


watch(() => props.modelValue, (newJsonSequence) => {
  if (newJsonSequence === currentJsonOutput.value) {
    return; // Avoid re-parsing if the change came from internal update
  }
  parseAndSetSequence(newJsonSequence);
}, { immediate: true });

function parseAndSetSequence(jsonSequence) {
  parsingError.value = null;
  if (!jsonSequence || typeof jsonSequence !== 'string') {
    localActionSequence.value = [];
    return;
  }
  try {
    const parsed = JSON.parse(jsonSequence);
    if (Array.isArray(parsed)) {
      localActionSequence.value = parsed.map(step => ({
        id: generateLocalId(), // Assign a local unique ID for draggable
        action_template_id: step.action_template_id,
        parameters_json: step.parameters_json || '{}',
      }));
    } else {
      localActionSequence.value = [];
      if (jsonSequence.trim() !== "" && jsonSequence.trim() !== "[]") {
        parsingError.value = "Invalid action sequence. Expected a JSON array.";
      }
    }
  } catch (e) {
    localActionSequence.value = [];
     if (jsonSequence.trim() !== "" && jsonSequence.trim() !== "[]") {
        parsingError.value = e.message;
     }
    console.error("Error parsing action sequence JSON:", e);
  }
}

function addActionStep() {
  localActionSequence.value.push({ 
    id: generateLocalId(), 
    action_template_id: '', 
    parameters_json: '{}' 
  });
  onSequenceChange();
}

function removeActionStep(index) {
  localActionSequence.value.splice(index, 1);
  onSequenceChange();
}

function onSequenceChange() {
  const jsonToEmit = currentJsonOutput.value;
  emit('update:modelValue', jsonToEmit);
}

onMounted(() => {
  if (actionTemplateStore.actionTemplates.length === 0) {
    actionTemplateStore.fetchActionTemplates();
  }
  // Parse initial modelValue if not already parsed by watch
  if (props.modelValue && localActionSequence.value.length === 0) {
      parseAndSetSequence(props.modelValue);
  } else if (!props.modelValue && localActionSequence.value.length === 0) {
      // Ensure initial empty array is processed if modelValue is empty string
      parseAndSetSequence("[]");
  }
});

</script>

<style scoped>
.v-card--variant-outlined {
  border-color: rgba(0,0,0,0.12);
}
.drag-handle {
  cursor: move;
  padding-right: 8px;
}
.v-list-item { /* Ensure list items don't have their own padding issues */
  padding: 0 !important;
}
.v-card-text { /* Reduce default padding if needed */
  padding: 12px; 
}
</style>
