import { defineStore } from 'pinia';
import {
  getAttributesForEntity,
  // getAttributeById, // Currently not used directly by store, components might use it
  createAttribute,
  updateAttribute,
  deleteAttribute,
} from '@/services/apiService'; // Adjusted path

export const useAttributeStore = defineStore('attribute', {
  state: () => ({
    attributes: [], // Attributes for the currently selected/viewed entity
    loading: false,
    error: null,
  }),
  actions: {
    async fetchAttributesForEntity(entityId) {
      this.loading = true;
      this.error = null;
      try {
        const response = await getAttributesForEntity(entityId);
        this.attributes = response.data;
      } catch (error) {
        this.error = `Failed to fetch attributes for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        this.attributes = []; // Reset on error
      } finally {
        this.loading = false;
      }
    },
    async addAttribute(entityId, attributeData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createAttribute(entityId, attributeData);
        this.attributes.push(response.data);
        return response.data;
      } catch (error) {
        this.error = `Failed to create attribute for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async editAttribute(entityId, attributeId, attributeData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateAttribute(entityId, attributeId, attributeData);
        const index = this.attributes.findIndex(a => a.id === attributeId);
        if (index !== -1) {
          this.attributes[index] = response.data;
        }
        return response.data;
      } catch (error) {
        this.error = `Failed to update attribute ${attributeId} for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async removeAttribute(entityId, attributeId) {
      this.loading = true;
      this.error = null;
      try {
        await deleteAttribute(entityId, attributeId);
        this.attributes = this.attributes.filter(a => a.id !== attributeId);
      } catch (error) {
        this.error = `Failed to delete attribute ${attributeId} for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    // Helper to clear attributes, e.g., when navigating away from an entity's detail page
    clearAttributes() {
        this.attributes = [];
    }
  },
});
