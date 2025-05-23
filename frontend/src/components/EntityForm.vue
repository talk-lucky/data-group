<template>
  <v-container>
    <v-card class="mx-auto" max-width="700">
      <v-card-title class="text-h5">
        {{ isEditMode ? 'Edit Entity' : 'Create New Entity' }}
      </v-card-title>
      <v-card-text>
        <v-form ref="formRef" @submit.prevent="handleSubmit">
          <v-text-field
            v-model="formData.name"
            label="Name*"
            :rules="[rules.required]"
            prepend-icon="mdi-text-short"
            variant="outlined"
            class="mb-3"
          ></v-text-field>

          <v-textarea
            v-model="formData.description"
            label="Description"
            prepend-icon="mdi-text-long"
            variant="outlined"
            rows="3"
            auto-grow
            class="mb-3"
          ></v-textarea>

          <v-alert v-if="formError || entityStore.error" type="error" dense class="mb-4">
            {{ formError || entityStore.error }}
          </v-alert>
          
          <v-progress-linear v-if="entityStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>

        </v-form>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey darken-1" text @click="cancelForm" :disabled="entityStore.loading">
          Cancel
        </v-btn>
        <v-btn color="primary" @click="handleSubmit" :loading="entityStore.loading" :disabled="!isFormValid">
          {{ isEditMode ? 'Save Changes' : 'Create Entity' }}
        </v-btn>
      </v-card-actions>
    </v-card>

    <!-- Snackbar for notifications -->
    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" bottom right>
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn color="white" text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>
  </v-container>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useEntityStore } from '@/store/entityStore';

const props = defineProps({
  entityId: { // For edit mode, passed as prop from router
    type: String,
    default: null,
  },
});

const router = useRouter();
const route = useRoute(); // Can be used if entityId is not passed as prop
const entityStore = useEntityStore();

const formRef = ref(null); // Reference to the v-form
const isFormValid = ref(false); // Vuetify form validation status

const isEditMode = computed(() => !!props.entityId);

const formData = ref({
  name: '',
  description: '',
});
const formError = ref(null); // For custom form errors not covered by Vuetify rules

const snackbar = ref({
  show: false,
  message: '',
  color: '',
  timeout: 3000,
});

function showSnackbar(message, color = 'success') {
  snackbar.value.message = message;
  snackbar.value.color = color;
  snackbar.value.show = true;
}

const rules = {
  required: value => !!value || 'This field is required.',
  // Add other rules like minLength, email format, etc. if needed
};

// Watch for changes in entityId (e.g., when navigating directly to an edit page)
// or when the prop updates.
watch(() => props.entityId, (newId) => {
  entityStore.error = null; // Clear previous store errors
  formError.value = null; // Clear form errors
  if (newId) {
    loadEntityData(newId);
  } else {
    resetForm();
  }
}, { immediate: true });


onMounted(() => {
  // If entityId is not passed as a prop, it might be in route.params
  // This handles cases where the component might be used not directly by a route with props:true
  const idToLoad = props.entityId || route.params.id;
  if (idToLoad && !isEditMode.value) { // Only if prop wasn't primary source
     // This might indicate a need to reconsider how entityId is passed, props is cleaner
     console.warn("EntityForm: entityId loaded from route params, consider using props for consistency.");
     loadEntityData(idToLoad);
  } else if (!idToLoad) {
    resetForm(); // Ensure form is clean for create mode
  }
   validateForm(); // Initial validation check
});

async function loadEntityData(id) {
  // Attempt to fetch from store's currentEntity if it matches, or entities list
  let entityToEdit = null;
  if (entityStore.currentEntity && entityStore.currentEntity.id === id) {
    entityToEdit = entityStore.currentEntity;
  } else {
    entityToEdit = entityStore.entities.find(e => e.id === id);
  }
  
  if (!entityToEdit) {
    entityToEdit = await entityStore.fetchEntityById(id); // Fetches and sets currentEntity
  }

  if (entityToEdit) {
    formData.value.name = entityToEdit.name;
    formData.value.description = entityToEdit.description || ''; // Ensure description is not null
  } else {
    formError.value = `Entity with ID ${id} not found.`;
    showSnackbar(formError.value, 'error');
    // Optionally redirect or disable form
  }
  validateForm();
}

function resetForm() {
  formData.value.name = '';
  formData.value.description = '';
  formError.value = null;
  entityStore.clearCurrentEntity(); // Clear any existing entity from store
  if (formRef.value) {
    formRef.value.resetValidation(); // Reset Vuetify form validation state
  }
  validateForm();
}

async function validateForm() {
  if (formRef.value) {
    const { valid } = await formRef.value.validate();
    isFormValid.value = valid;
  } else {
    isFormValid.value = false; // Default if formRef not available yet
  }
}


// Watch formData to re-validate when user types
watch(formData, () => {
  validateForm();
  formError.value = null; // Clear custom errors on input
  entityStore.error = null; // Clear store errors on input
}, { deep: true });


async function handleSubmit() {
  await validateForm(); // Ensure form is validated
  if (!isFormValid.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null; // Clear previous custom errors
  entityStore.error = null; // Clear previous store errors

  try {
    let savedEntity;
    if (isEditMode.value) {
      savedEntity = await entityStore.editEntity(props.entityId, { ...formData.value });
      showSnackbar(`Entity "${savedEntity.name}" updated successfully.`, 'success');
    } else {
      savedEntity = await entityStore.addEntity({ ...formData.value });
      showSnackbar(`Entity "${savedEntity.name}" created successfully.`, 'success');
    }
    router.push('/entities'); // Navigate back to the list after successful operation
  } catch (error) {
    // Error should be set in the store, but we can also set formError for display
    formError.value = 'Operation failed: ' + (entityStore.error || error.message || 'Unknown error');
    showSnackbar(formError.value, 'error');
  }
}

function cancelForm() {
  resetForm();
  router.go(-1); // Go back to the previous page
}
</script>

<style scoped>
/* Additional scoped styles if needed */
.mb-3 {
  margin-bottom: 16px !important; /* Vuetify's default spacing might be too small for some layouts */
}
</style>
