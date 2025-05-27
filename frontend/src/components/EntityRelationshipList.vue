<template>
  <v-container>
    <v-card class="mx-auto" outlined>
      <v-card-title class="text-h6 pa-4 d-flex justify-space-between align-center">
        Entity Relationships
        <v-btn color="primary" @click="goToCreateForm">
          <v-icon left>mdi-plus</v-icon>
          Create Relationship
        </v-btn>
      </v-card-title>
      <v-divider></v-divider>

      <v-card-text v-if="relationshipStore.loading && relationshipStore.relationships.length === 0">
        <v-progress-linear indeterminate color="primary"></v-progress-linear>
        <p class="text-center mt-2">Loading relationships...</p>
      </v-card-text>
      
      <v-alert v-if="relationshipStore.error" type="error" dense class="ma-3">
        {{ relationshipStore.error }}
      </v-alert>

      <v-data-table-server
        v-if="!relationshipStore.loading || relationshipStore.relationships.length > 0"
        :headers="headers"
        :items="formattedRelationships"
        :items-length="relationshipStore.pagination.totalItems"
        :loading="relationshipStore.loading"
        :page="relationshipStore.pagination.page"
        :items-per-page="relationshipStore.pagination.itemsPerPage"
        @update:options="loadItems"
        class="elevation-1"
        item-value="id"
      >
        <template v-slot:item.actions="{ item }">
          <v-icon small class="mr-2" @click="editRelationship(item.raw.id)" color="primary">
            mdi-pencil
          </v-icon>
          <v-icon small @click="confirmDelete(item.raw.id)" color="error">
            mdi-delete
          </v-icon>
        </template>
         <template v-slot:loading>
            <v-skeleton-loader type="table-row@10"></v-skeleton-loader>
        </template>
         <template v-slot:no-data>
          <v-alert type="info" variant="tonal" class="ma-3">
            No entity relationships found. Click "Create Relationship" to add one.
          </v-alert>
        </template>
      </v-data-table-server>
    </v-card>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog" persistent max-width="400px">
      <v-card>
        <v-card-title class="text-h5">Confirm Delete</v-card-title>
        <v-card-text>
          Are you sure you want to delete this entity relationship? This action cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" text @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="red darken-1" text @click="deleteConfirmed" :loading="deleting">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useEntityRelationshipStore } from '@/stores/entityRelationshipStore';
import { useEntityStore } from '@/stores/entityStore'; // To get entity names
import { useErrorStore } from '@/stores/errorStore';

const router = useRouter();
const relationshipStore = useEntityRelationshipStore();
const entityStore = useEntityStore();
const errorStore = useErrorStore();

const deleteDialog = ref(false);
const deleting = ref(false);
const itemToDelete = ref(null);

const headers = [
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Source Entity', key: 'sourceEntityName', sortable: true },
  { title: 'Target Entity', key: 'targetEntityName', sortable: true },
  { title: 'Relationship Type', key: 'relationship_type', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' },
];

const formattedRelationships = computed(() => {
  return relationshipStore.relationships.map(rel => ({
    ...rel,
    sourceEntityName: entityStore.getEntityById(rel.source_entity_id)?.name || rel.source_entity_id,
    targetEntityName: entityStore.getEntityById(rel.target_entity_id)?.name || rel.target_entity_id,
  }));
});

async function loadItems({ page, itemsPerPage, sortBy }) {
  // sortBy is an array, handle its structure if server-side sorting is implemented
  // For now, we'll just use page and itemsPerPage
  await relationshipStore.fetchEntityRelationships({ 
    page: page, 
    limit: itemsPerPage,
    // sort_by: sortBy.length ? sortBy[0].key : undefined, // Example for sorting
    // sort_desc: sortBy.length ? sortBy[0].order === 'desc' : undefined, 
  });
}

onMounted(async () => {
  errorStore.clearError();
  // Fetch entities if not already loaded, for mapping IDs to names
  if (entityStore.entities.length === 0) {
    await entityStore.fetchEntities();
  }
  // Initial load for the table is handled by the @update:options event with default options
});

function goToCreateForm() {
  router.push({ name: 'EntityRelationshipCreate' }); // Assuming this route name
}

function editRelationship(id) {
  router.push({ name: 'EntityRelationshipEdit', params: { id } }); // Assuming this route name
}

function confirmDelete(id) {
  itemToDelete.value = id;
  deleteDialog.value = true;
}

async function deleteConfirmed() {
  if (itemToDelete.value) {
    deleting.value = true;
    errorStore.clearError();
    try {
      await relationshipStore.deleteEntityRelationship(itemToDelete.value);
    } catch (error) {
      // error is already set in relationshipStore and displayed by alert
      console.error("Failed to delete relationship:", error);
    } finally {
      deleting.value = false;
      deleteDialog.value = false;
      itemToDelete.value = null;
    }
  }
}
</script>

<style scoped>
.v-card {
  border: 1px solid #e0e0e0;
}
.v-data-table-server {
  margin-top: 1rem;
}
</style>
