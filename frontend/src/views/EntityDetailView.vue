<template>
  <v-container fluid>
    <v-btn to="/entities" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Entities List
    </v-btn>

    <v-progress-linear v-if="entityStore.loading" indeterminate color="primary" class="my-4"></v-progress-linear>
    
    <v-alert v-if="entityStore.error && !entityStore.loading" type="error" dismissible class="my-4">
      Error fetching entity details: {{ entityStore.error }}
    </v-alert>

    <v-card v-if="entity && !entityStore.loading" class="mb-5">
      <v-card-title class="text-h4">
        Entity: {{ entity.name }}
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ entity.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-list dense>
          <v-list-item>
            <v-list-item-title><strong>Description:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ entity.description || 'N/A' }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Created At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(entity.created_at) }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(entity.updated_at) }}</v-list-item-subtitle>
          </v-list-item>
        </v-list>
      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/entities/${entityId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Entity
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!entity && !entityStore.loading && !entityStore.error" class="text-center my-5">
      <v-alert type="warning">
        Entity with ID "{{ entityId }}" not found.
      </v-alert>
    </div>

    <!-- AttributeList component is already Vuetify-styled internally -->
    <AttributeList v-if="entityId" :entity-id="entityId" />

  </v-container>
</template>

<script setup>
import { computed, onMounted, watch, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import { useEntityStore } from '@/store/entityStore';
import { useAttributeStore } from '@/store/attributeStore'; // Import attribute store
import AttributeList from '@/components/AttributeList.vue';

const route = useRoute();
const entityStore = useEntityStore();
const attributeStore = useAttributeStore(); // Get attribute store instance

// entityId from the route params
const entityId = computed(() => route.params.id);

// Get the current entity from the store
const entity = computed(() => entityStore.currentEntity);

// Fetch entity details when component is mounted or entityId changes
onMounted(() => {
  if (entityId.value) {
    entityStore.fetchEntityById(entityId.value);
    // Attributes are fetched by AttributeList itself, but we can clear them if entity changes
    attributeStore.clearAttributes(); 
  }
});

// Watch for route param changes if navigating between detail views directly
watch(entityId, (newId, oldId) => {
  if (newId && newId !== oldId) {
    entityStore.fetchEntityById(newId);
    attributeStore.clearAttributes(); // Clear attributes of the old entity
  }
});

// Clear current entity and attributes when navigating away from this view
onBeforeUnmount(() => {
  entityStore.clearCurrentEntity();
  attributeStore.clearAttributes();
});


function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try {
    return new Date(dateTimeString).toLocaleString();
  } catch (e) {
    return dateTimeString; // Fallback
  }
}
</script>

<style scoped>
/* Scoped styles if needed */
.text-h4 {
  font-weight: 500; /* Adjust title weight if needed */
}
.v-list-item-title {
  font-weight: bold;
}
</style>
