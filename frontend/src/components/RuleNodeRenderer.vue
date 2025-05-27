<template>
  <div class="rule-node" :class="{ 'group-node': isGroup, 'condition-node': !isGroup }" :style="{ 'margin-left': (depth * 20) + 'px' }">
    <div v-if="isGroup" class="group-controls">
      <select :value="node.logical_operator" @change="updateLogicalOperator($event.target.value)">
        <option value="AND">AND</option>
        <option value="OR">OR</option>
      </select>
      <button @click="addCondition">Add Condition</button>
      <button @click="addGroup">Add Group</button>
      <button v-if="depth > 0" @click="removeNode" class="remove-btn">Remove Group</button>
    </div>

    <div v-if="!isGroup" class="condition-controls">
      <select :value="node.attributeId" @change="updateAttribute($event.target.value)">
        <option disabled value="">Select Attribute</option>
        <option v-for="attr in availableAttributes" :key="attr.id" :value="attr.id">
          {{ attr.name }} ({{ attr.entityName }})
        </option>
      </select>

      <select v-if="node.attributeId" :value="node.operator" @change="updateOperator($event.target.value)">
        <option disabled value="">Select Operator</option>
        <option v-for="op in availableOperators" :key="op.value" :value="op.value">
          {{ op.label }}
        </option>
      </select>

      <input
        v-if="node.attributeId && showValueInput"
        :type="valueInputType"
        :value="node.value"
        @input="updateValue($event.target.value)"
        placeholder="Value"
      />
      <button @click="removeNode" class="remove-btn">Remove Condition</button>
    </div>

    <div v-if="isGroup" class="nested-rules">
      <RuleNodeRenderer
        v-for="(childNode, index) in node.rules"
        :key="childNode.id"
        :node="childNode"
        :path="[...path, index]"
        :depth="depth + 1"
        :availableAttributes="availableAttributes"
        :attributeStore="attributeStore"
        @update-node="emitUpdateNode"
        @remove-node="emitRemoveNode"
        @add-condition="emitAddCondition"
        @add-group="emitAddGroup"
      />
    </div>
  </div>
</template>

<script setup>
import { computed, defineProps, defineEmits, inject } from 'vue';
import { getOperatorsByDataType, getInputTypeForDataType } from '../utils/attributeUtils'; // Assuming utils file

const props = defineProps({
  node: {
    type: Object,
    required: true,
  },
  path: { // Array of indices to locate this node in the main rules tree e.g. [0, 'rules', 1]
    type: Array,
    required: true,
  },
  depth: {
    type: Number,
    default: 0,
  },
  availableAttributes: { // Pass down from GroupRuleBuilder
    type: Array,
    required: true,
  },
  attributeStore: { // Pass down from GroupRuleBuilder (or inject if provided at root)
    type: Object,
    required: true,
  }
});

const emit = defineEmits(['update-node', 'remove-node', 'add-condition', 'add-group']);

const isGroup = computed(() => props.node.type === 'group');

const selectedAttributeDetails = computed(() => {
  if (!isGroup.value && props.node.attributeId && props.attributeStore) {
    return props.attributeStore.getAttributeById(props.node.attributeId);
  }
  return null;
});

const availableOperators = computed(()
=> {
  if (selectedAttributeDetails.value) {
    return getOperatorsByDataType(selectedAttributeDetails.value.data_type);
  }
  return [];
});

const valueInputType = computed(() => {
  if (selectedAttributeDetails.value) {
    return getInputTypeForDataType(selectedAttributeDetails.value.data_type);
  }
  return 'text';
});

const showValueInput = computed(() => {
  if (!props.node.operator) return false;
  // Operators like 'is_null', 'is_not_null' might not need a value
  return !['is_null', 'is_not_null', 'is_empty', 'is_not_empty'].includes(props.node.operator?.toLowerCase());
});


// --- Methods to emit changes upwards to GroupRuleBuilder ---

function updateLogicalOperator(newOperator) {
  emit('update-node', props.path, { logical_operator: newOperator });
}

function updateAttribute(attributeId) {
  const attribute = props.attributeStore.getAttributeById(attributeId);
  if (attribute) {
    emit('update-node', props.path, {
      attributeId: attribute.id,
      attributeName: attribute.name, // Store for convenience/display
      entityId: attribute.entity_id,   // Store entity_id
      valueType: attribute.data_type, // Store data_type as valueType
      operator: '', // Reset operator
      value: '',    // Reset value
    });
  }
}

function updateOperator(operator) {
  emit('update-node', props.path, { operator });
}

function updateValue(value) {
  // Basic type conversion based on valueType, can be more sophisticated
  let coercedValue = value;
  if (selectedAttributeDetails.value) {
    const type = selectedAttributeDetails.value.data_type.toLowerCase();
    if (type === 'integer' || type === 'long') {
      coercedValue = parseInt(value, 10);
      if (isNaN(coercedValue)) coercedValue = value; // Or handle error
    } else if (type === 'float' || type === 'double' || type === 'decimal' || type === 'numeric') {
      coercedValue = parseFloat(value);
      if (isNaN(coercedValue)) coercedValue = value; // Or handle error
    } else if (type === 'boolean') {
      if (value.toLowerCase() === 'true') coercedValue = true;
      else if (value.toLowerCase() === 'false') coercedValue = false;
      // else keep as string or handle error
    }
  }
  emit('update-node', props.path, { value: coercedValue });
}

function addCondition() {
  emit('add-condition', props.path); // Emit path of the current group
}

function addGroup() {
  emit('add-group', props.path); // Emit path of the current group
}

function removeNode() {
  emit('remove-node', props.path);
}

// --- Methods to pass through emissions from nested children ---
function emitUpdateNode(path, M) {
  emit('update-node', path, M);
}
function emitRemoveNode(path) {
  emit('remove-node', path);
}
function emitAddCondition(path) {
  emit('add-condition', path);
}
function emitAddGroup(path) {
  emit('add-group', path);
}

</script>

<style scoped>
.rule-node {
  padding: 10px;
  margin-bottom: 10px;
  border: 1px solid #ccc;
  border-radius: 4px;
}

.group-node {
  background-color: #f0f0f0;
  border-left: 5px solid #607d8b; /* Blue-grey */
}

.condition-node {
  background-color: #fafafa;
  border-left: 5px solid #81c784; /* Green */
}

.group-controls, .condition-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.remove-btn {
  background-color: #ef5350; /* Red */
  color: white;
  border: none;
  padding: 5px 10px;
  border-radius: 4px;
  cursor: pointer;
}
.remove-btn:hover {
  background-color: #d32f2f;
}

.nested-rules {
  margin-top: 10px;
}

select, input[type="text"], input[type="number"], input[type="date"] {
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
</style>
