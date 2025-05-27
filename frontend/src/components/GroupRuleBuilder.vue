<template>
  <v-container>
    <v-card variant="outlined" class="pa-3 group-rule-builder-card">
      <v-card-title class="text-subtitle-1 pb-1">
        Grouping Rules
      </v-card-title>
      <v-card-subtitle class="pb-2">
        Define nested rules and conditions to group entities.
      </v-card-subtitle>
      <v-divider class="mb-3"></v-divider>

      <div v-if="!entityId" class="text-center pa-3">
        <v-alert type="info" variant="tonal">
          Please select an Entity first to define rules based on its attributes.
        </v-alert>
      </div>

      <div v-else>
        <RuleNodeRenderer
          :node="localRules"
          :path="[]" 
          :depth="0"
          :availableAttributes="attributeItemsForRenderer"
          :attributeStore="attributeStore" 
          @update-node="handleUpdateNode"
          @remove-node="handleRemoveNode"
          @add-condition="handleAddCondition"
          @add-group="handleAddGroup"
        />
      </div>
      
      <v-textarea
        v-if="debugMode"
        :model-value="currentJsonOutput" 
        label="Current JSON Output (Debug)"
        readonly
        auto-grow
        rows="5"
        class="mt-4"
        variant="outlined"
        density="compact"
      ></v-textarea>
      <v-alert v-if="parsingError" type="error" dense class="mt-3">
        Error processing rules: {{ parsingError }}. Rules may have been reset or defaulted.
      </v-alert>
    </v-card>
  </v-container>
</template>

<script setup>
import { ref, watch, computed, onMounted } from 'vue';
import { v4 as uuidv4 } from 'uuid';
import { useAttributeStore } from '@/store/attributeStore'; // Ensure path is correct
import RuleNodeRenderer from './RuleNodeRenderer.vue';

const props = defineProps({
  modelValue: { // JSON string of rules
    type: String,
    default: '',
  },
  entityId: { // Selected entity ID for which rules are being defined
    type: String,
    default: null,
  },
});

const emit = defineEmits(['update:modelValue']);

const attributeStore = useAttributeStore();
const localRules = ref(createDefaultRootGroup()); // Root of the rule tree
const parsingError = ref(null);
const debugMode = ref(import.meta.env.DEV); // Show debug output in development

// --- Initialization and Data Structure ---
function createDefaultRootGroup() {
  return {
    id: uuidv4(), // Internal UI ID
    type: 'group',
    logical_operator: 'AND',
    rules: [],
  };
}

function createNewCondition() {
  return {
    id: uuidv4(),
    type: 'condition',
    attributeId: '', // Will be attribute_id in JSON
    attributeName: '',
    entityId: '',    // Will be entity_id in JSON
    operator: '',
    value: '',
    valueType: '',   // Will be value_type in JSON (derived from attribute)
  };
}

function createNewGroup() {
  return {
    id: uuidv4(),
    type: 'group',
    logical_operator: 'AND',
    rules: [],
  };
}

// Attributes to be passed to RuleNodeRenderer
const attributeItemsForRenderer = computed(() => {
  // attributeStore.attributes should be [{id, name, entity_id, data_type, entity_name}, ...]
  // RuleNodeRenderer expects 'availableAttributes' with id, name, entityName
  return attributeStore.attributes.map(attr => ({
    id: attr.id, // Used as value in select
    name: attr.name, // Display name
    entityName: attr.entity_name, // For display in dropdown like "Attribute (Entity)"
    entity_id: attr.entity_id, // Crucial for JSON output
    data_type: attr.data_type, // For operator/input logic
  }));
});


// --- Tree Traversal and Modification ---
function getNodeByPath(path, rootNode = localRules.value) {
  let currentNode = rootNode;
  for (const index of path) {
    if (!currentNode || !currentNode.rules || !currentNode.rules[index]) {
      console.error('Invalid path or node not found at path:', path, 'in node:', rootNode);
      return null;
    }
    currentNode = currentNode.rules[index];
  }
  return currentNode;
}

function getParentNodeByPath(path, rootNode = localRules.value) {
  if (!path || path.length === 0) return null; // Root has no parent
  let parent = rootNode;
  for (let i = 0; i < path.length - 1; i++) {
    const index = path[i];
    if (!parent || !parent.rules || !parent.rules[index]) {
      console.error('Invalid parent path:', path);
      return null;
    }
    parent = parent.rules[index];
  }
  return parent;
}

// --- Event Handlers from RuleNodeRenderer ---
function handleUpdateNode(path, updates) {
  const nodeToUpdate = path.length === 0 ? localRules.value : getNodeByPath(path);
  if (nodeToUpdate) {
    Object.assign(nodeToUpdate, updates);
  } else {
    console.error('Node not found for update at path:', path);
  }
}

function handleRemoveNode(path) {
  if (path.length === 0) { // Cannot remove root node
    localRules.value = createDefaultRootGroup(); // Reset to default
    return;
  }
  const parentNode = getParentNodeByPath(path);
  const nodeIndex = path[path.length - 1];
  if (parentNode && parentNode.rules && parentNode.rules[nodeIndex] !== undefined) {
    parentNode.rules.splice(nodeIndex, 1);
  } else {
    console.error('Node or parent not found for removal at path:', path);
  }
}

function handleAddCondition(groupPath) {
  const parentGroup = groupPath.length === 0 ? localRules.value : getNodeByPath(groupPath);
  if (parentGroup && parentGroup.type === 'group') {
    parentGroup.rules.push(createNewCondition());
  } else {
    console.error('Parent group not found or not a group at path:', groupPath);
  }
}

function handleAddGroup(groupPath) {
  const parentGroup = groupPath.length === 0 ? localRules.value : getNodeByPath(groupPath);
  if (parentGroup && parentGroup.type === 'group') {
    parentGroup.rules.push(createNewGroup());
  } else {
    console.error('Parent group not found or not a group at path:', groupPath);
  }
}

// --- JSON Generation (Internal to Standardized) ---
function buildJsonNode(internalNode) {
  if (!internalNode) return null;

  if (internalNode.type === 'condition') {
    if (!internalNode.attributeId || !internalNode.operator) return null; // Incomplete condition

    const attribute = attributeStore.getAttributeById(internalNode.attributeId);
    // Value coercion happens here or in RuleNodeRenderer; ensure consistency.
    // RuleNodeRenderer currently does some coercion.
    return {
      type: "condition",
      attribute_id: internalNode.attributeId,
      attribute_name: attribute ? attribute.name : internalNode.attributeName,
      entity_id: attribute ? attribute.entity_id : internalNode.entityId,
      operator: internalNode.operator,
      value: internalNode.value, // Assuming value is already correctly typed
      value_type: attribute ? attribute.data_type : internalNode.valueType,
    };
  } else if (internalNode.type === 'group') {
    return {
      type: "group",
      logical_operator: internalNode.logical_operator,
      rules: internalNode.rules.map(rule => buildJsonNode(rule)).filter(r => r !== null),
    };
  }
  return null;
}

const currentJsonOutput = computed(() => {
  const jsonStructure = buildJsonNode(localRules.value);
  if (!jsonStructure) return ""; // Handle case where root is invalid (should not happen)
  return JSON.stringify(jsonStructure, null, 2);
});

// --- Parsing Input JSON (Standardized to Internal) ---
function parseJsonNode(jsonNode) {
  if (!jsonNode || !jsonNode.type) {
    console.warn("Parsing invalid JSON node:", jsonNode);
    return null;
  }
  const id = uuidv4();

  if (jsonNode.type === 'condition') {
    const attribute = attributeStore.getAttributeById(jsonNode.attribute_id);
    return {
      id,
      type: 'condition',
      attributeId: jsonNode.attribute_id,
      attributeName: jsonNode.attribute_name || (attribute ? attribute.name : ''),
      entityId: jsonNode.entity_id || (attribute ? attribute.entity_id : ''),
      operator: jsonNode.operator,
      value: jsonNode.value, // Value type should be handled by input components or on display
      valueType: jsonNode.value_type || (attribute ? attribute.data_type : 'string'),
    };
  } else if (jsonNode.type === 'group') {
    return {
      id,
      type: 'group',
      logical_operator: jsonNode.logical_operator || 'AND',
      rules: Array.isArray(jsonNode.rules) ? jsonNode.rules.map(rule => parseJsonNode(rule)).filter(r => r !== null) : [],
    };
  }
  return null;
}

function parseAndSetRules(jsonString) {
  parsingError.value = null;
  try {
    if (jsonString && jsonString.trim() !== '') {
      const parsedJson = JSON.parse(jsonString);
      if (parsedJson && parsedJson.type === 'group') {
        const newRules = parseJsonNode(parsedJson);
        if (newRules) {
          localRules.value = newRules;
        } else {
          console.error("Failed to parse valid structure from JSON prop. Resetting to default.");
          localRules.value = createDefaultRootGroup();
        }
      } else {
        // If props.modelValue is not a group, or invalid, initialize with a default root.
        // This can happen if an old flat structure is passed.
        console.warn("Input JSON is not a valid group structure or is empty. Initializing with default.", parsedJson);
        localRules.value = createDefaultRootGroup();
        if (jsonString.trim() !== '{}' && jsonString.trim() !== '""' && jsonString.trim() !== '') {
             // Don't show error for empty or default empty object/string
            parsingError.value = "Provided rules were not in the expected nested format and have been reset.";
        }
      }
    } else {
      localRules.value = createDefaultRootGroup(); // Initialize if string is empty
    }
  } catch (error) {
    console.error('Error parsing rule JSON from prop:', error, "\nJSON string:", jsonString);
    localRules.value = createDefaultRootGroup(); // Reset to default on error
    parsingError.value = `Failed to parse rules: ${error.message}. Rules have been reset.`;
  }
}

// --- Watchers ---
watch(() => props.entityId, (newEntityId, oldEntityId) => {
  if (newEntityId !== oldEntityId) {
    localRules.value = createDefaultRootGroup(); // Reset rules
    parsingError.value = null;
    if (newEntityId) {
      attributeStore.fetchAttributesForEntity(newEntityId); // Fetch attributes for the new entity
    } else {
      attributeStore.clearAttributes();
    }
    // No need to emit here, changing entity implies new rule set
  }
}, { immediate: true });

watch(() => props.modelValue, (newValue) => {
  if (newValue !== currentJsonOutput.value) { // Avoid re-parsing if change came from internal update
    parseAndSetRules(newValue);
  }
}, { immediate: true }); // Parse initial value

watch(localRules, (newRules) => {
  const newJson = currentJsonOutput.value;
  // Check if the newJson is actually different from what was received via prop modelValue
  // This is to prevent emitting update if the internal change is just due to initial parsing.
  if (newJson !== props.modelValue) {
      emit('update:modelValue', newJson);
  }
}, { deep: true });


onMounted(async () => {
  // Ensure attributes are loaded if an entityId is present on mount
  if (props.entityId && attributeStore.attributes.length === 0) {
    await attributeStore.fetchAttributesForEntity(props.entityId);
  }
  // Initial parsing is handled by the immediate watcher on props.modelValue
});

</script>

<style scoped>
.group-rule-builder-card {
  border-color: rgba(0,0,0,0.12);
}
/* Add any additional styles for GroupRuleBuilder.vue itself if needed */
</style>
