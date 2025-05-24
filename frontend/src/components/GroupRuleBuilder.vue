<template>
  <v-container>
    <v-card variant="outlined" class="pa-3">
      <v-card-title class="text-subtitle-1 pb-1">
        Grouping Rules
      </v-card-title>
      <v-card-subtitle class="pb-2">
        Define rules to group entities. All rules are currently combined with AND.
      </v-card-subtitle>
      <v-divider class="mb-3"></v-divider>

      <div v-if="!entityId" class="text-center pa-3">
        <v-alert type="info" variant="tonal">
          Please select an Entity first to define rules based on its attributes.
        </v-alert>
      </div>

      <div v-else>
        <v-row v-for="(rule, index) in localRules" :key="index" dense align="center" class="mb-2">
          <v-col cols="12" md="4">
            <v-select
              v-model="rule.attributeId"
              :items="attributeItems"
              label="Attribute"
              placeholder="Select Attribute"
              variant="outlined"
              density="compact"
              hide-details
              @update:modelValue="onRuleChange"
            ></v-select>
          </v-col>

          <v-col cols="12" md="3">
            <v-select
              v-model="rule.operator"
              :items="getOperatorsForAttribute(rule.attributeId)"
              label="Operator"
              placeholder="Select Operator"
              variant="outlined"
              density="compact"
              hide-details
              @update:modelValue="onRuleChange"
            ></v-select>
          </v-col>

          <v-col cols="12" md="4">
            <template v-if="!isUnaryOperator(rule.operator)">
              <v-text-field
                v-if="getInputTypeForAttribute(rule.attributeId) === 'text'"
                v-model="rule.value"
                label="Value"
                placeholder="Enter value"
                variant="outlined"
                density="compact"
                hide-details
                @input="onRuleChange"
              ></v-text-field>
              <v-checkbox
                v-else-if="getInputTypeForAttribute(rule.attributeId) === 'boolean'"
                v-model="rule.value"
                label="Is True"
                density="compact"
                hide-details
                @change="onRuleChange"
              ></v-checkbox>
               <v-text-field
                v-else-if="getInputTypeForAttribute(rule.attributeId) === 'number'"
                v-model.number="rule.value"
                label="Value"
                type="number"
                placeholder="Enter number"
                variant="outlined"
                density="compact"
                hide-details
                @input="onRuleChange"
              ></v-text-field>
            </template>
             <span v-else class="text-caption grey--text">(No value needed)</span>
          </v-col>

          <v-col cols="12" md="1" class="text-right">
            <v-btn icon="mdi-delete" variant="text" color="error" @click="removeRule(index)" density="compact"></v-btn>
          </v-col>
        </v-row>

        <v-btn color="primary" @click="addRule" :disabled="!entityId" class="mt-2">
          <v-icon left>mdi-plus</v-icon> Add Rule
        </v-btn>
      </div>
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
        Error parsing existing rules: {{ parsingError }}. Rules have been reset.
      </v-alert>
    </v-card>
  </v-container>
</template>

<script setup>
import { ref, watch, computed, onMounted } from 'vue';
import { useAttributeStore } from '@/store/attributeStore';

const props = defineProps({
  modelValue: { // JSON string of rules
    type: String,
    default: '',
  },
  entityId: {
    type: String,
    default: null,
  },
});

const emit = defineEmits(['update:modelValue']);

const attributeStore = useAttributeStore();
const localRules = ref([]); // Array of { attributeId: '', operator: '', value: '' }
const parsingError = ref(null);
const debugMode = ref(false); // Set to true to see JSON output for debugging

const entityAttributes = computed(() => attributeStore.attributes);

const attributeItems = computed(() => {
  return entityAttributes.value.map(attr => ({
    title: attr.name,
    value: attr.id,
    dataType: attr.data_type, // Store data type for operator/input logic
  }));
});

const currentJsonOutput = computed(() => {
  if (localRules.value.length === 0) return '';
  const conditions = localRules.value
    .filter(rule => rule.attributeId && rule.operator) // Only include valid rules
    .map(rule => {
      const attr = entityAttributes.value.find(a => a.id === rule.attributeId);
      const baseRule = {
        field: attr ? attr.name : rule.attributeId, // Fallback to ID if name not found
        operator: rule.operator,
      };
      if (!isUnaryOperator(rule.operator)) {
        baseRule.value = rule.value;
      }
      return baseRule;
    });

  if (conditions.length === 0) return '';
  return JSON.stringify({ logical_operator: "AND", conditions }, null, 2);
});


// --- Operator and Input Type Logic ---
const baseOperators = [
  { title: 'Equals', value: 'equals' },
  { title: 'Does Not Equal', value: 'not_equals' },
  { title: 'Contains', value: 'contains' }, // String specific
  { title: 'Does Not Contain', value: 'not_contains' }, // String specific
  { title: 'Is Null', value: 'is_null', unary: true },
  { title: 'Is Not Null', value: 'is_not_null', unary: true },
];
const numericOperators = [
  ...baseOperators.filter(op => op.value === 'equals' || op.value === 'not_equals' || op.value === 'is_null' || op.value === 'is_not_null'),
  { title: 'Greater Than', value: 'greater_than' },
  { title: 'Less Than', value: 'less_than' },
  { title: 'Greater Than or Equal To', value: 'greater_than_or_equal_to' },
  { title: 'Less Than or Equal To', value: 'less_than_or_equal_to' },
];
const booleanOperators = [
  { title: 'Is True', value: 'is_true', unary: true, impliedValue: true }, // Or 'equals' with value true
  { title: 'Is False', value: 'is_false', unary: true, impliedValue: false }, // Or 'equals' with value false
  { title: 'Is Null', value: 'is_null', unary: true },
  { title: 'Is Not Null', value: 'is_not_null', unary: true },
];
 // Default to string/text operators
const stringOperators = baseOperators;

function getOperatorsForAttribute(attributeId) {
  const attribute = attributeItems.value.find(attr => attr.value === attributeId);
  if (!attribute) return stringOperators;
  switch (attribute.dataType?.toLowerCase()) {
    case 'integer':
    case 'float':
    case 'datetime': // DateTime can be compared with greater/less than
      return numericOperators;
    case 'boolean':
      return booleanOperators;
    case 'string':
    case 'text':
    case 'json': // JSON could be stringified for 'contains' or 'equals'
    default:
      return stringOperators;
  }
}

function getInputTypeForAttribute(attributeId) {
  const attribute = attributeItems.value.find(attr => attr.value === attributeId);
  if (!attribute) return 'text';
  switch (attribute.dataType?.toLowerCase()) {
    case 'integer':
    case 'float':
      return 'number';
    case 'boolean':
      return 'boolean';
    default:
      return 'text';
  }
}

function isUnaryOperator(operatorValue) {
  const allOperators = [...stringOperators, ...numericOperators, ...booleanOperators];
  const operator = allOperators.find(op => op.value === operatorValue);
  return operator?.unary || false;
}
// --- End Operator and Input Type Logic ---


watch(() => props.entityId, (newEntityId) => {
  localRules.value = []; // Reset rules when entity changes
  parsingError.value = null;
  if (newEntityId) {
    attributeStore.fetchAttributesForEntity(newEntityId);
  } else {
    attributeStore.clearAttributes();
  }
  onRuleChange(); // Emit empty rules
}, { immediate: true });


watch(() => props.modelValue, (newJsonRules) => {
  if (newJsonRules === currentJsonOutput.value) {
    // Avoid re-parsing if the change came from internal update
    return;
  }
  parseAndSetRules(newJsonRules);
}, { immediate: true }); // Parse initial modelValue if provided

function parseAndSetRules(jsonRules) {
  parsingError.value = null;
  if (!jsonRules || typeof jsonRules !== 'string') {
    localRules.value = [];
    return;
  }
  try {
    const parsed = JSON.parse(jsonRules);
    if (parsed && Array.isArray(parsed.conditions)) {
      localRules.value = parsed.conditions.map(condition => {
        const attr = entityAttributes.value.find(a => a.name === condition.field);
        return {
          attributeId: attr ? attr.id : condition.field, // Store ID if found, else original field name
          operator: condition.operator,
          value: condition.value,
        };
      });
    } else {
      localRules.value = [];
      if (jsonRules.trim() !== "") { // Don't error for empty initial string
        parsingError.value = "Invalid rule structure. Expected 'conditions' array.";
      }
    }
  } catch (e) {
    localRules.value = [];
    if (jsonRules.trim() !== "") { // Don't error for empty initial string
       parsingError.value = e.message;
    }
    console.error("Error parsing rules JSON:", e);
  }
}


function addRule() {
  localRules.value.push({ attributeId: '', operator: '', value: '' });
  onRuleChange(); // To potentially update JSON if this empty rule should be part of it (it won't by current logic)
}

function removeRule(index) {
  localRules.value.splice(index, 1);
  onRuleChange();
}

function onRuleChange() {
  const jsonToEmit = currentJsonOutput.value;
  emit('update:modelValue', jsonToEmit);
}

onMounted(() => {
  // Initial fetch if entityId is already set and attributes aren't loaded
  if (props.entityId && entityAttributes.value.length === 0) {
    attributeStore.fetchAttributesForEntity(props.entityId);
  }
  // Parse initial modelValue if provided and not already parsed by watch
  if (props.modelValue && localRules.value.length === 0) {
      parseAndSetRules(props.modelValue);
  }
});

</script>

<style scoped>
.v-card--variant-outlined {
  border-color: rgba(0,0,0,0.12);
}
</style>
