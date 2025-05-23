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
    // Store attributes per entity to avoid conflicts if multiple entity details are cached or open
    // attributesByEntity: { entityId1: [attr1, attr2], entityId2: [...] }
    // For simplicity with current usage (usually one entity's attributes viewed at a time):
    attributes: [], // Holds attributes for the currently active/selected entity
    currentEntityIdForAttributes: null, // Tracks which entity the 'attributes' array belongs to
    loading: false,
    error: null,
  }),
  getters: {
    getAttributesForCurrentEntity: (state) => state.attributes,
    isLoading: (state) => state.loading,
    // Getter to provide attributes in a format suitable for v-select
    // It will return options for the currently loaded attributes.
    // Assumes attributes for the relevant entity have been fetched.
    attributeOptions: (state) => {
      return state.attributes.map(attr => ({
        title: `${attr.name} (${attr.data_type})`, // Display name and type
        value: attr.id, // Use ID as the value
      }));
    },
  },
  actions: {
    async fetchAttributesForEntity(entityId) {
      if (!entityId) {
        this.error = 'Entity ID is required to fetch attributes.';
        console.error(this.error);
        this.attributes = [];
        this.currentEntityIdForAttributes = null;
        return;
      }
      this.loading = true;
      this.error = null;
      try {
        const response = await getAttributesForEntity(entityId);
        this.attributes = response.data;
        this.currentEntityIdForAttributes = entityId;
      } catch (error) {
        this.error = `Failed to fetch attributes for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        this.attributes = []; // Reset on error
        this.currentEntityIdForAttributes = null;
      } finally {
        this.loading = false;
      }
    },
    async addAttribute(entityId, attributeData) {
      // Ensure that the attributes being added are for the currently focused entity
      if (this.currentEntityIdForAttributes !== entityId && entityId) {
         // If attributes for a different entity are loaded, this might lead to inconsistencies.
         // Consider fetching attributes for entityId first or clearing if that's the desired behavior.
         console.warn(`Adding attribute to entity ${entityId}, but currently loaded attributes are for ${this.currentEntityIdForAttributes}. Refetching attributes for ${entityId}.`);
         await this.fetchAttributesForEntity(entityId);
      }
      this.loading = true;
      this.error = null;
      try {
        const response = await createAttribute(entityId, attributeData);
        // If the new attribute belongs to the currently loaded set, add it
        if (this.currentEntityIdForAttributes === entityId) {
            this.attributes.push(response.data);
        }
        // If it belongs to another entity, the list for that entity will be updated next time it's fetched.
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
        // If the updated attribute belongs to the currently loaded set, update it
        if (this.currentEntityIdForAttributes === entityId) {
            const index = this.attributes.findIndex(a => a.id === attributeId);
            if (index !== -1) {
              this.attributes[index] = response.data;
            }
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
        // If the removed attribute belongs to the currently loaded set, remove it
        if (this.currentEntityIdForAttributes === entityId) {
            this.attributes = this.attributes.filter(a => a.id !== attributeId);
        }
      } catch (error) {
        this.error = `Failed to delete attribute ${attributeId} for entity ${entityId}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    // Helper to clear attributes, e.g., when navigating away from an entity's detail page or changing entity context
    clearAttributes() {
        this.attributes = [];
        this.currentEntityIdForAttributes = null;
        this.error = null; // Also clear any errors related to attributes
    }
  },
});
