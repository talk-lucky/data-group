<template>
  <v-dialog :model-value="dialogVisible" @update:model-value="$emit('update:dialogVisible', $event)" max-width="600px" persistent>
    <v-card>
      <v-card-title>
        <span class="text-h5">{{ formTitle }}</span>
      </v-card-title>
      <v-card-text>
        <v-form ref="form" v-model="valid" lazy-validation>
          <v-container>
            <v-row>
              <v-col cols="12" sm="6">
                <v-text-field
                  v-model="editableAttribute.name"
                  label="Name*"
                  :rules="[rules.required, rules.nameLength(50)]"
                  required
                  variant="outlined"
                  density="compact"
                ></v-text-field>
              </v-col>
              <v-col cols="12" sm="6">
                <v-select
                  v-model="editableAttribute.data_type"
                  :items="dataTypes"
                  label="Data Type*"
                  :rules="[rules.required]"
                  required
                  variant="outlined"
                  density="compact"
                ></v-select>
              </v-col>
              <v-col cols="12">
                <v-textarea
                  v-model="editableAttribute.description"
                  label="Description"
                  variant="outlined"
                  density="compact"
                  rows="2"
                  :rules="[rules.descriptionLength(255)]"
                ></v-textarea>
              </v-col>
              <v-col cols="12" sm="6">
                <v-checkbox
                  v-model="editableAttribute.is_filterable"
                  label="Filterable"
                  density="compact"
                ></v-checkbox>
              </v-col>
              <v-col cols="12" sm="6">
                <v-checkbox
                  v-model="editableAttribute.is_pii"
                  label="PII (Personally Identifiable Information)"
                  density="compact"
                ></v-checkbox>
              </v-col>
            </v-row>
          </v-container>
          <small>*indicates required field</small>
        </v-form>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey" text @click="closeDialog">Cancel</v-btn>
        <v-btn color="primary" :disabled="!valid || isLoading" :loading="isLoading" text @click="saveAttribute">Save</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import { ref, watch, computed, defineProps, defineEmits, nextTick } from 'vue';
import { useAttributeStore } from '@/store/attributeStore';

const props = defineProps({
  attribute: Object, // The attribute to edit, null for new attribute
  entityId: { // Required for creating a new attribute, to associate it
    type: String,
    required: true,
  },
  dialogVisible: Boolean,
});

const emit = defineEmits(['update:dialogVisible', 'saved', 'error']);

const attributeStore = useAttributeStore();
const isLoading = computed(() => attributeStore.isLoading);

const form = ref(null); // Template ref for v-form
const valid = ref(true); // Form validity state

const dataTypes = ref(['STRING', 'TEXT', 'INTEGER', 'FLOAT', 'BOOLEAN', 'DATETIME']);

const defaultAttribute = {
  id: null,
  name: '',
  data_type: '',
  description: '',
  is_filterable: false,
  is_pii: false,
  entity_id: props.entityId, // Associate with the current entity
};

const editableAttribute = ref({ ...defaultAttribute });

const formTitle = computed(() => {
  return editableAttribute.value.id ? 'Edit Attribute' : 'Create New Attribute';
});

watch(() => props.dialogVisible, async (newVal) => {
  if (newVal) {
    if (props.attribute) {
      editableAttribute.value = { ...props.attribute, entity_id: props.entityId };
    } else {
      editableAttribute.value = { ...defaultAttribute, entity_id: props.entityId };
    }
    await nextTick();
    if (form.value) {
      form.value.resetValidation();
    }
    valid.value = true;
  }
});

const rules = {
  required: value => !!value || 'Required.',
  nameLength: (max) => value => (value && value.length <= max) || `Name must be less than ${max} characters.`,
  descriptionLength: (max) => value => (!value || value.length <= max) || `Description must be less than ${max} characters.`,
};

async function saveAttribute() {
  if (form.value) {
    const validationResult = await form.value.validate();
    if (!validationResult.valid) {
      valid.value = false;
      return;
    }
    valid.value = true;
  }

  // Prepare payload, ensuring boolean defaults are set if null/undefined from checkbox interaction
  const payload = {
    ...editableAttribute.value,
    is_filterable: !!editableAttribute.value.is_filterable, // Ensure boolean
    is_pii: !!editableAttribute.value.is_pii,             // Ensure boolean
  };

  try {
    if (editableAttribute.value.id) { // Editing existing attribute
      // The updateAttribute action takes attributeId, payload, and entityIdToRefresh
      await attributeStore.updateAttribute(editableAttribute.value.id, payload, props.entityId);
    } else { // Creating new attribute
      await attributeStore.createAttribute(props.entityId, payload);
    }
    emit('saved');
    closeDialog();
  } catch (error) {
    emit('error', attributeStore.error || 'Save operation failed');
  }
}

function closeDialog() {
  emit('update:dialogVisible', false);
}

</script>
