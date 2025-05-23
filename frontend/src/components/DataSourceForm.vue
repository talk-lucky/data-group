<template>
  <v-container>
    <v-card class="mx-auto" max-width="700">
      <v-card-title class="text-h5">
        {{ isEditMode ? 'Edit Data Source' : 'Create New Data Source' }}
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

          <v-select
            v-model="formData.type"
            :items="dataSourceTypes"
            label="Type*"
            :rules="[rules.required]"
            prepend-icon="mdi-cog-outline"
            variant="outlined"
            class="mb-3"
          ></v-select>

          <v-textarea
            v-model="formData.connection_details"
            label="Connection Details (JSON)*"
            :rules="[rules.required, rules.json]"
            prepend-icon="mdi-code-json"
            variant="outlined"
            rows="5"
            auto-grow
            class="mb-3"
            hint="Enter a valid JSON string. For example: {\"host\":\"localhost\", \"port\":5432, \"user\":\"admin\"}"
          ></v-textarea>

          <v-alert v-if="formError || dataSourceStore.error" type="error" dense class="mb-4">
            {{ formError || dataSourceStore.error }}
          </v-alert>
          
          <v-progress-linear v-if="dataSourceStore.loading" indeterminate color="primary" class="mb-3"></v-progress-linear>

        </v-form>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="grey darken-1" text @click="cancelForm" :disabled="dataSourceStore.loading">
          Cancel
        </v-btn>
        <v-btn color="primary" @click="handleSubmit" :loading="dataSourceStore.loading" :disabled="!isFormValid">
          {{ isEditMode ? 'Save Changes' : 'Create Data Source' }}
        </v-btn>
      </v-card-actions>
    </v-card>

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
import { useDataSourceStore } from '@/store/dataSourceStore';

const props = defineProps({
  dataSourceId: { 
    type: String,
    default: null,
  },
});

const router = useRouter();
const route = useRoute(); 
const dataSourceStore = useDataSourceStore();

const formRef = ref(null); 
const isFormValid = ref(false);

const isEditMode = computed(() => !!props.dataSourceId);
const dataSourceTypes = ref(['PostgreSQL', 'MySQL', 'CSV File', 'Generic API', 'Other']);

const formData = ref({
  name: '',
  type: '',
  connection_details: '{}', // Default to an empty JSON object
});
const formError = ref(null);

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
  json: value => {
    try {
      JSON.parse(value);
      return true;
    } catch (e) {
      return 'Must be a valid JSON string.';
    }
  }
};

watch(() => props.dataSourceId, (newId) => {
  dataSourceStore.error = null; 
  formError.value = null; 
  if (newId) {
    loadDataSourceData(newId);
  } else {
    resetForm();
  }
}, { immediate: true });

onMounted(() => {
  const idToLoad = props.dataSourceId || route.params.id;
  if (idToLoad && !isEditMode.value) { 
     console.warn("DataSourceForm: dataSourceId loaded from route params, consider using props for consistency.");
     loadDataSourceData(idToLoad);
  } else if (!idToLoad) {
    resetForm(); 
  }
   validateForm(); 
});

async function loadDataSourceData(id) {
  let dsToEdit = null;
  if (dataSourceStore.currentDataSource && dataSourceStore.currentDataSource.id === id) {
    dsToEdit = dataSourceStore.currentDataSource;
  } else {
    dsToEdit = dataSourceStore.dataSources.find(ds => ds.id === id);
  }
  
  if (!dsToEdit) {
    dsToEdit = await dataSourceStore.fetchDataSourceById(id);
  }

  if (dsToEdit) {
    formData.value.name = dsToEdit.name;
    formData.value.type = dsToEdit.type;
    formData.value.connection_details = dsToEdit.connection_details || '{}';
  } else {
    formError.value = `Data Source with ID ${id} not found.`;
    showSnackbar(formError.value, 'error');
  }
  validateForm();
}

function resetForm() {
  formData.value.name = '';
  formData.value.type = '';
  formData.value.connection_details = '{}';
  formError.value = null;
  dataSourceStore.clearCurrentDataSource();
  if (formRef.value) {
    formRef.value.resetValidation(); 
  }
  validateForm();
}

async function validateForm() {
  if (formRef.value) {
    const { valid } = await formRef.value.validate();
    isFormValid.value = valid;
  } else {
    isFormValid.value = false; 
  }
}

watch(formData, () => {
  validateForm();
  formError.value = null; 
  dataSourceStore.error = null; 
}, { deep: true });

async function handleSubmit() {
  await validateForm(); 
  if (!isFormValid.value) {
    formError.value = 'Please correct the errors in the form.';
    return;
  }
  formError.value = null; 
  dataSourceStore.error = null;

  try {
    let savedDataSource;
    const dataToSave = { ...formData.value };
    // Ensure connection_details is a string, even if it's empty JSON object.
    if (typeof dataToSave.connection_details !== 'string') {
        dataToSave.connection_details = JSON.stringify(dataToSave.connection_details);
    }


    if (isEditMode.value) {
      savedDataSource = await dataSourceStore.editDataSource(props.dataSourceId, dataToSave);
      showSnackbar(`Data Source "${savedDataSource.name}" updated successfully.`, 'success');
    } else {
      savedDataSource = await dataSourceStore.addDataSource(dataToSave);
      showSnackbar(`Data Source "${savedDataSource.name}" created successfully.`, 'success');
    }
    router.push('/datasources'); 
  } catch (error) {
    formError.value = 'Operation failed: ' + (dataSourceStore.error || error.message || 'Unknown error');
    showSnackbar(formError.value, 'error');
  }
}

function cancelForm() {
  resetForm();
  router.push('/datasources');
}
</script>

<style scoped>
.mb-3 {
  margin-bottom: 16px !important;
}
</style>
