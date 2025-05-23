import { defineStore } from 'pinia';
import {
  getFieldMappingsForDataSource,
  // getFieldMappingById, // Individual get might be less used by list views
  createFieldMapping,
  updateFieldMapping,
  deleteFieldMapping,
} from '@/services/apiService';

export const useFieldMappingStore = defineStore('fieldMapping', {
  state: () => ({
    // Store mappings in a way that's easy to access per data source
    // e.g., { dataSourceId1: [mapping1, mapping2], dataSourceId2: [...] }
    mappingsBySource: {},
    loading: false,
    error: null,
  }),
  getters: {
    getMappingsForSource: (state) => (sourceId) => {
      return state.mappingsBySource[sourceId] || [];
    },
    isLoading: (state) => state.loading,
  },
  actions: {
    async fetchFieldMappings(sourceId) {
      if (!sourceId) {
        this.error = 'Source ID is required to fetch field mappings.';
        console.error(this.error);
        return;
      }
      this.loading = true;
      this.error = null;
      try {
        const response = await getFieldMappingsForDataSource(sourceId);
        this.mappingsBySource = {
          ...this.mappingsBySource,
          [sourceId]: response.data,
        };
      } catch (error) {
        this.error = `Failed to fetch field mappings for source ${sourceId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        // Ensure the entry for sourceId is at least an empty array on error if it doesn't exist
        if (!this.mappingsBySource[sourceId]) {
           this.mappingsBySource[sourceId] = [];
        }
      } finally {
        this.loading = false;
      }
    },
    async addFieldMapping(sourceId, mappingData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createFieldMapping(sourceId, mappingData);
        const updatedMappings = [...(this.mappingsBySource[sourceId] || []), response.data];
        this.mappingsBySource = {
          ...this.mappingsBySource,
          [sourceId]: updatedMappings,
        };
        return response.data;
      } catch (error) {
        this.error = `Failed to create field mapping for source ${sourceId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async editFieldMapping(sourceId, mappingId, mappingData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateFieldMapping(sourceId, mappingId, mappingData);
        if (this.mappingsBySource[sourceId]) {
          const index = this.mappingsBySource[sourceId].findIndex(m => m.id === mappingId);
          if (index !== -1) {
            const updatedMappings = [...this.mappingsBySource[sourceId]];
            updatedMappings[index] = response.data;
            this.mappingsBySource = {
              ...this.mappingsBySource,
              [sourceId]: updatedMappings,
            };
          }
        }
        return response.data;
      } catch (error) {
        this.error = `Failed to update field mapping ${mappingId} for source ${sourceId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async removeFieldMapping(sourceId, mappingId) {
      this.loading = true;
      this.error = null;
      try {
        await deleteFieldMapping(sourceId, mappingId);
        if (this.mappingsBySource[sourceId]) {
          const updatedMappings = this.mappingsBySource[sourceId].filter(m => m.id !== mappingId);
          this.mappingsBySource = {
            ...this.mappingsBySource,
            [sourceId]: updatedMappings,
          };
        }
      } catch (error) {
        this.error = `Failed to delete field mapping ${mappingId} for source ${sourceId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    // Action to clear mappings for a specific source, e.g., when its details page is left
    clearMappingsForSource(sourceId) {
        if (this.mappingsBySource[sourceId]) {
            const newMappingsBySource = { ...this.mappingsBySource };
            delete newMappingsBySource[sourceId];
            this.mappingsBySource = newMappingsBySource;
        }
    },
    // Action to clear all mappings, e.g., on full store reset or logout
    clearAllMappings() {
        this.mappingsBySource = {};
    }
  },
});
