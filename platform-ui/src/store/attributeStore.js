// src/store/attributeStore.js
import { defineStore } from 'pinia'
import apiClient from '../services/api'

export const useAttributeStore = defineStore('attributes', {
  state: () => ({
    attributes: [], // Attributes for the currently selected entity
    isLoading: false,
    error: null,
  }),
  actions: {
    async fetchAttributes(entityId) {
      if (!entityId) {
        this.attributes = []
        return
      }
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.get(`/entities/${entityId}/attributes`)
        this.attributes = response.data
      } catch (error) {
        this.attributes = [] // Clear attributes on error
        this.error = error.response?.data?.error || error.message || 'Failed to fetch attributes'
        console.error(`fetchAttributes for entity ${entityId} error:`, error)
      } finally {
        this.isLoading = false
      }
    },

    async createAttribute(entityId, attributeData) {
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.post(`/entities/${entityId}/attributes`, attributeData)
        // Re-fetch attributes for the current entity to include the new one
        await this.fetchAttributes(entityId)
        return response.data // Return the created attribute data
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to create attribute'
        console.error('createAttribute error:', error)
        throw error // Re-throw for component-level handling
      }
      // isLoading will be reset by the fetchAttributes call
    },

    async updateAttribute(attributeId, attributeData, entityIdToRefresh) {
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.put(`/attributes/${attributeId}`, attributeData)
        // Re-fetch attributes for the relevant entity to reflect changes
        if (entityIdToRefresh) {
          await this.fetchAttributes(entityIdToRefresh)
        }
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to update attribute'
        console.error('updateAttribute error:', error)
        throw error // Re-throw
      }
      // isLoading will be reset by the fetchAttributes call if entityIdToRefresh is provided
      // otherwise, ensure it is reset if no refresh happens
      if (!entityIdToRefresh) this.isLoading = false;
    },

    async deleteAttribute(attributeId, entityIdToRefresh) {
      this.isLoading = true
      this.error = null
      try {
        await apiClient.delete(`/attributes/${attributeId}`)
        // Re-fetch attributes for the relevant entity
        if (entityIdToRefresh) {
          await this.fetchAttributes(entityIdToRefresh)
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to delete attribute'
        console.error('deleteAttribute error:', error)
        throw error // Re-throw
      }
      // isLoading will be reset by the fetchAttributes call if entityIdToRefresh is provided
      if (!entityIdToRefresh) this.isLoading = false; 
    },

    // Helper to clear attributes, e.g., when navigating away from entity detail page
    clearAttributes() {
      this.attributes = []
      this.error = null
    }
  },
})
