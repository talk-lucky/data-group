import { defineStore } from 'pinia';
import {
  getEntities,
  getEntityById,
  createEntity,
  updateEntity,
  deleteEntity,
} from '@/services/apiService'; // Adjusted path

export const useEntityStore = defineStore('entity', {
  state: () => ({
    entities: [],
    currentEntity: null,
    loading: false,
    error: null,
  }),
  getters: {
    allEntities: (state) => state.entities,
    isLoading: (state) => state.loading,
    entityOptions: (state) => {
      return state.entities.map(entity => ({
        title: entity.name, // Use 'name' for display
        value: entity.id,   // Use 'id' as the value
      }));
    },
  },
  actions: {
    async fetchEntities() {
      // Fetch all entities if not already loaded or if a force refresh is needed.
      // For simplicity, always fetching, but could add a check like `if (this.entities.length === 0 || forceRefresh)`
      this.loading = true;
      this.error = null;
      try {
        const response = await getEntities();
        this.entities = response.data;
      } catch (error) {
        this.error = 'Failed to fetch entities: ' + (error.response?.data?.error || error.message);
        console.error(this.error, error);
      } finally {
        this.loading = false;
      }
    },
    async fetchEntityById(id) {
      this.loading = true;
      this.error = null;
      try {
        const response = await getEntityById(id);
        this.currentEntity = response.data;
        return response.data; // Return for component use if needed
      } catch (error) {
        this.error = `Failed to fetch entity ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        this.currentEntity = null; // Reset on error
      } finally {
        this.loading = false;
      }
    },
    async addEntity(entityData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createEntity(entityData);
        this.entities.push(response.data); // Add to local state
        // Optionally, re-fetch all entities to ensure consistency if backend does more processing
        // await this.fetchEntities(); 
        return response.data; // Return created entity
      } catch (error) {
        this.error = 'Failed to create entity: ' + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error; // Re-throw to allow components to handle
      } finally {
        this.loading = false;
      }
    },
    async editEntity(id, entityData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateEntity(id, entityData);
        const index = this.entities.findIndex(e => e.id === id);
        if (index !== -1) {
          this.entities[index] = response.data;
        }
        if (this.currentEntity && this.currentEntity.id === id) {
          this.currentEntity = response.data;
        }
        return response.data; // Return updated entity
      } catch (error) {
        this.error = `Failed to update entity ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error; // Re-throw
      } finally {
        this.loading = false;
      }
    },
    async removeEntity(id) {
      this.loading = true;
      this.error = null;
      try {
        await deleteEntity(id);
        this.entities = this.entities.filter(e => e.id !== id);
        if (this.currentEntity && this.currentEntity.id === id) {
          this.currentEntity = null;
        }
      } catch (error) {
        this.error = `Failed to delete entity ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error; // Re-throw
      } finally {
        this.loading = false;
      }
    },
    // Helper to clear current entity, e.g., when navigating away
    clearCurrentEntity() {
        this.currentEntity = null;
        this.error = null; // Also clear error related to current entity
    }
  },
});
