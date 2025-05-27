import { defineStore } from 'pinia';
import axios from 'axios';

const API_URL = '/api/v1/entity-relationships';

export const useEntityRelationshipStore = defineStore('entityRelationship', {
  state: () => ({
    relationships: [],
    currentRelationship: null,
    loading: false,
    error: null,
    pagination: {
      page: 1,
      itemsPerPage: 10,
      totalItems: 0,
    },
  }),

  actions: {
    async fetchEntityRelationships(params = {}) {
      this.loading = true;
      this.error = null;
      try {
        const response = await axios.get(API_URL, { params });
        // Assuming backend returns { data: [...], total: X }
        this.relationships = response.data.data || []; 
        this.pagination.totalItems = response.data.total || 0;
        // Update page and itemsPerPage if they were part of params and you want to store them
        if (params.page) this.pagination.page = params.page;
        if (params.limit) this.pagination.itemsPerPage = params.limit;

      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch entity relationships';
        this.relationships = [];
        this.pagination.totalItems = 0;
        console.error('Error fetching entity relationships:', error);
      } finally {
        this.loading = false;
      }
    },

    async fetchEntityRelationship(id) {
      this.loading = true;
      this.error = null;
      try {
        const response = await axios.get(`${API_URL}/${id}`);
        this.currentRelationship = response.data;
        return response.data; // Return for component use
      } catch (error) {
        this.error = error.response?.data?.error || `Failed to fetch entity relationship ${id}`;
        this.currentRelationship = null;
        console.error(`Error fetching entity relationship ${id}:`, error);
        throw error; // Re-throw for component to handle
      } finally {
        this.loading = false;
      }
    },

    async createEntityRelationship(relationshipData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await axios.post(API_URL, relationshipData);
        // Optionally, re-fetch list or add to current list
        await this.fetchEntityRelationships({ page: this.pagination.page, limit: this.pagination.itemsPerPage });
        return response.data; 
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to create entity relationship';
        console.error('Error creating entity relationship:', error);
        throw error; 
      } finally {
        this.loading = false;
      }
    },

    async updateEntityRelationship(id, relationshipData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await axios.put(`${API_URL}/${id}`, relationshipData);
         // Optionally, re-fetch list or update in current list
        await this.fetchEntityRelationships({ page: this.pagination.page, limit: this.pagination.itemsPerPage });
        this.currentRelationship = response.data; // Update current if it was the one being edited
        return response.data;
      } catch (error) {
        this.error = error.response?.data?.error || `Failed to update entity relationship ${id}`;
        console.error(`Error updating entity relationship ${id}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },

    async deleteEntityRelationship(id) {
      this.loading = true;
      this.error = null;
      try {
        await axios.delete(`${API_URL}/${id}`);
        // Re-fetch list
        await this.fetchEntityRelationships({ page: this.pagination.page, limit: this.pagination.itemsPerPage });
      } catch (error) {
        this.error = error.response?.data?.error || `Failed to delete entity relationship ${id}`;
        console.error(`Error deleting entity relationship ${id}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },

    // Helper to clear currentRelationship, e.g., when opening form for new
    setCurrentRelationship(relationship) {
        this.currentRelationship = relationship;
    }
  },

  getters: {
    // Example getter
    getRelationshipById: (state) => (id) => {
      return state.relationships.find(r => r.id === id);
    }
  }
});
