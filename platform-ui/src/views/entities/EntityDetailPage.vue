<template>
  <v-container fluid>
    <!-- Entity Details Section -->
    <v-card class="mb-6" elevation="2">
      <v-card-title class="bg-primary">
        <span class="text-h5">Entity: {{ entity?.name || 'Loading...' }}</span>
      </v-card-title>
      <v-card-text v-if="entity && !entityStore.isLoading">
        <v-row>
          <v-col cols="12" md="6">
            <p><strong>ID:</strong> {{ entity.id }}</p>
          </v-col>
          <v-col cols="12" md="6">
            <p><strong>Description:</strong> {{ entity.description || 'N/A' }}</p>
          </v-col>
          <v-col cols="12" md="6">
            <p><strong>Created At:</strong> {{ formatDate(entity.created_at) }}</p>
          </v-col>
          <v-col cols="12" md="6">
            <p><strong>Updated At:</strong> {{ formatDate(entity.updated_at) }}</p>
          </v-col>
        </v-row>
      </v-card-text>
      <v-card-text v-else-if="entityStore.isLoading">
        <v-progress-circular indeterminate color="primary"></v-progress-circular>
        Loading entity details...
      </v-card-text>
      <v-card-text v-if="entityStore.error">
         <v-alert type="error" density="compact" class="mb-4">
          Error loading entity details: {{ entityStore.error }}
        </v-alert>
      </v-card-text>
    </v-card>

    <!-- Attributes Management Section -->
    <h2 class="text-h4 mb-2">Attributes</h2>
    <v-row align="center" class="mb-4">
      <v-col>
        <!-- Placeholder for any attribute-specific header controls -->
      </v-col>
      <v-col class="text-right">
        <v-btn color="primary" @click="openCreateAttributeDialog" :disabled="!entityId">
          <v-icon left>mdi-plus</v-icon>
          Add New Attribute
        </v-btn>
      </v-col>
    </v-row>

    <v-progress-linear :active="attributeStore.isLoading" indeterminate color="primary" class="mb-2"></v-progress-linear>

    <v-alert v-if="attributeStore.error" type="error" density="compact" class="mb-4" closable @click:close="clearAttributeError">
      {{ attributeStore.error }}
    </v-alert>

    <v-card>
      <v-card-title>
        <v-text-field
          v-model="searchAttributes"
          append-icon="mdi-magnify"
          label="Search Attributes"
          single-line
          hide-details
          variant="underlined"
        ></v-text-field>
      </v-card-title>

      <v-data-table
        :headers="attributeHeaders"
        :items="attributes"
        :search="searchAttributes"
        :loading="attributeStore.isLoading"
        item-key="id"
        class="elevation-1"
        density="compact"
      >
        <template v-slot:item.is_filterable="{ item }">
          <v-icon :color="item.raw.is_filterable ? 'green' : 'grey'">
            {{ item.raw.is_filterable ? 'mdi-check-circle' : 'mdi-circle-outline' }}
          </v-icon>
        </template>
        <template v-slot:item.is_pii="{ item }">
          <v-icon :color="item.raw.is_pii ? 'orange' : 'grey'">
            {{ item.raw.is_pii ? 'mdi-alert-circle' : 'mdi-circle-outline' }}
          </v-icon>
        </template>
        <template v-slot:item.created_at="{ item }">
          {{ formatDate(item.raw.created_at) }}
        </template>
        <template v-slot:item.updated_at="{ item }">
          {{ formatDate(item.raw.updated_at) }}
        </template>
        <template v-slot:item.actions="{ item }">
          <v-tooltip location="top" text="Edit Attribute">
            <template v-slot:activator="{ props: tooltipProps }">
              <v-icon v-bind="tooltipProps" small class="mr-2" @click="openEditAttributeDialog(item.raw)" color="primary">mdi-pencil</v-icon>
            </template>
          </v-tooltip>
          <v-tooltip location="top" text="Delete Attribute">
            <template v-slot:activator="{ props: tooltipProps }">
              <v-icon v-bind="tooltipProps" small @click="openDeleteAttributeConfirmDialog(item.raw)" color="error">mdi-delete</v-icon>
            </template>
          </v-tooltip>
        </template>
        <template v-slot:no-data>
          <v-alert :value="true" color="info" icon="mdi-information-variant" class="ma-3">
            No attributes found for this entity. Click "Add New Attribute" to create one.
          </v-alert>
        </template>
      </v-data-table>
    </v-card>

    <attribute-form
      v-if="entityId" 
      :attribute="selectedAttribute"
      :entity-id="entityId"
      :dialog-visible="attributeFormDialogVisible"
      @update:dialogVisible="attributeFormDialogVisible = $event"
      @saved="handleAttributeSaved"
      @error="handleAttributeFormError"
    />

    <confirm-dialog
      :show="deleteAttributeConfirmDialogVisible"
      title="Confirm Delete Attribute"
      :message="`Are you sure you want to delete attribute '${attributeToDelete?.name || ''}'? This action cannot be undone.`"
      confirm-text="Delete"
      confirm-color="error"
      @update:show="deleteAttributeConfirmDialogVisible = $event"
      @confirm="confirmDeleteAttribute"
      @cancel="cancelDeleteAttribute"
    />

    <v-snackbar v-model="snackbar.show" :color="snackbar.color" :timeout="snackbar.timeout" location="top right">
      {{ snackbar.message }}
      <template v-slot:actions>
        <v-btn text @click="snackbar.show = false">Close</v-btn>
      </template>
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useEntityStore } from '@/store/entityStore';
import { useAttributeStore } from '@/store/attributeStore';
import AttributeForm from '@/components/attributes/AttributeForm.vue';
import ConfirmDialog from '@/components/common/ConfirmDialog.vue';

const route = useRoute();
const entityStore = useEntityStore();
const attributeStore = useAttributeStore();

const entityId = ref(route.params.id);
const entity = ref(null); // For storing the fetched entity details

const attributes = computed(() => attributeStore.attributes);
// isLoading and error for attributes are accessed via attributeStore.isLoading and attributeStore.error

const searchAttributes = ref('');
const attributeHeaders = ref([
  { title: 'Name', key: 'name', sortable: true, value: 'name' },
  { title: 'Data Type', key: 'data_type', sortable: true, value: 'data_type' },
  { title: 'Description', key: 'description', sortable: true, value: 'description' },
  { title: 'Filterable', key: 'is_filterable', sortable: true, value: 'is_filterable' },
  { title: 'PII', key: 'is_pii', sortable: true, value: 'is_pii' },
  { title: 'Created At', key: 'created_at', sortable: true, value: 'created_at' },
  { title: 'Updated At', key: 'updated_at', sortable: true, value: 'updated_at' },
  { title: 'Actions', key: 'actions', sortable: false, value: 'actions' },
]);

const attributeFormDialogVisible = ref(false);
const selectedAttribute = ref(null);

const deleteAttributeConfirmDialogVisible = ref(false);
const attributeToDelete = ref(null);

const snackbar = ref({
  show: false,
  message: '',
  color: 'success',
  timeout: 3000,
});

async function fetchEntityDetails() {
  if (entityId.value) {
    // Try to find in store first if entities list is populated
    const found = entityStore.entities.find(e => e.id === entityId.value);
    if (found) {
      entity.value = found;
    } else {
      // If not found or store is empty, fetch specifically
      // Assuming entityStore has a method like fetchEntityById(id)
      // For now, let's just show an error or a simplified state
      // This part might need entityStore to have a `currentEntity` state and action
      // Or, we can rely on a direct API call if entityStore is not designed for single fetches.
      // For simplicity, if not in list, we show limited info or an error.
      // A proper implementation would fetch the entity by ID if not available.
      try {
        // This assumes your API client is set up and entityStore has a way to fetch one entity
        // This is a simplified example; you might need a dedicated action in entityStore
        const response = await attributeStore.apiClient.get(`/entities/${entityId.value}`); 
        entity.value = response.data;
      } catch (err) {
        entityStore.error = `Failed to load entity details for ID ${entityId.value}`;
        entity.value = null;
      }
    }
  }
}

onMounted(async () => {
  entityId.value = route.params.id;
  await fetchEntityDetails(); // Fetch entity details
  attributeStore.fetchAttributes(entityId.value); // Fetch attributes for this entity
});

// Watch for route changes if a user navigates from one entity detail to another
watch(() => route.params.id, (newId) => {
  if (newId) {
    entityId.value = newId;
    fetchEntityDetails();
    attributeStore.fetchAttributes(newId);
  } else {
    entity.value = null;
    attributeStore.clearAttributes(); // Clear attributes if no entity ID
  }
});

function clearAttributeError() {
  attributeStore.error = null;
}

function openCreateAttributeDialog() {
  selectedAttribute.value = null;
  attributeFormDialogVisible.value = true;
}

function openEditAttributeDialog(attribute) {
  selectedAttribute.value = { ...attribute };
  attributeFormDialogVisible.value = true;
}

function openDeleteAttributeConfirmDialog(attribute) {
  attributeToDelete.value = attribute;
  deleteAttributeConfirmDialogVisible.value = true;
}

async function confirmDeleteAttribute() {
  if (attributeToDelete.value && entityId.value) {
    try {
      await attributeStore.deleteAttribute(attributeToDelete.value.id, entityId.value);
      showSnackbar('Attribute deleted successfully.', 'success');
    } catch (e) {
      showSnackbar(attributeStore.error || 'Failed to delete attribute.', 'error');
    }
  }
  deleteAttributeConfirmDialogVisible.value = false;
  attributeToDelete.value = null;
}

function cancelDeleteAttribute() {
  deleteAttributeConfirmDialogVisible.value = false;
  attributeToDelete.value = null;
}

function handleAttributeSaved() {
  showSnackbar('Attribute saved successfully.', 'success');
  // Store action already re-fetches attributes for the current entityId
}

function handleAttributeFormError(errorMessage) {
  showSnackbar(errorMessage, 'error');
}

function showSnackbar(message, color = 'success', timeout = 3000) {
  snackbar.value = { show: true, message, color, timeout };
}

function formatDate(dateString) {
  if (!dateString) return 'N/A';
  const options = { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' };
  try {
    return new Date(dateString).toLocaleDateString(undefined, options);
  } catch (e) {
    return dateString; // Fallback
  }
}

</script>

<style scoped>
/* Add any page-specific styles here */
</style>
