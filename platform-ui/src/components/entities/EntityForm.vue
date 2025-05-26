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
              <v-col cols="12">
                <v-text-field
                  v-model="editableEntity.name"
                  label="Name*"
                  :rules="[rules.required, rules.nameLength(50)]"
                  required
                  variant="outlined"
                  density="compact"
                ></v-text-field>
              </v-col>
              <v-col cols="12">
                <v-textarea
                  v-model="editableEntity.description"
                  label="Description"
                  variant="outlined"
                  density="compact"
                  rows="3"
                  :rules="[rules.descriptionLength(255)]"
                ></v-textarea>
              </v-col>
            </v-row>
          </v-container>
          <small>*indicates required field</small>
        </v-form>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey" text @click="closeDialog">Cancel</v-btn>
        <v-btn color="primary" :disabled="!valid || isLoading" :loading="isLoading" text @click="saveEntity">Save</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import { ref, watch, computed, defineProps, defineEmits, nextTick } from 'vue';
import { useEntityStore } from '@/store/entityStore'; // Using @ alias for src

const props = defineProps({
  entity: Object, // The entity to edit, null for new entity
  dialogVisible: Boolean,
});

const emit = defineEmits(['update:dialogVisible', 'saved', 'error']);

const entityStore = useEntityStore();
const isLoading = computed(() => entityStore.isLoading);

const form = ref(null); // Template ref for v-form
const valid = ref(true); // Form validity state

const defaultEntity = {
  id: null,
  name: '',
  description: ''
};

const editableEntity = ref({ ...defaultEntity });

const formTitle = computed(() => {
  return editableEntity.value.id ? 'Edit Entity' : 'Create New Entity';
});

watch(() => props.dialogVisible, async (newVal) => {
  if (newVal) {
    if (props.entity) {
      editableEntity.value = { ...props.entity };
    } else {
      editableEntity.value = { ...defaultEntity };
    }
    // Reset validation state when dialog opens
    await nextTick(); // Ensure form is rendered
    if (form.value) {
      form.value.resetValidation();
    }
    valid.value = true; // Assume valid initially or rely on first validation pass
  } else {
    // Optional: Reset form when dialog is hidden if not already done by visibility change
    // editableEntity.value = { ...defaultEntity };
    // if (form.value) form.value.resetValidation();
  }
});

const rules = {
  required: value => !!value || 'Required.',
  nameLength: (max) => value => (value && value.length <= max) || `Name must be less than ${max} characters.`,
  descriptionLength: (max) => value => (!value || value.length <= max) || `Description must be less than ${max} characters.`,
};

async function saveEntity() {
  if (form.value) {
    const validationResult = await form.value.validate();
    if (!validationResult.valid) {
      valid.value = false;
      return;
    }
    valid.value = true;
  }

  try {
    if (editableEntity.value.id) {
      await entityStore.updateEntity(editableEntity.value);
    } else {
      await entityStore.createEntity(editableEntity.value);
    }
    emit('saved');
    closeDialog();
  } catch (error) {
    // Error is already logged by store, but we can emit it for component-specific handling
    emit('error', entityStore.error || 'Save operation failed');
    // The dialog remains open for the user to see the error or retry
  }
}

function closeDialog() {
  emit('update:dialogVisible', false);
}

</script>

<style scoped>
/* Add any component-specific styles here */
</style>
