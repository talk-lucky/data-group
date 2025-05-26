// src/store/entityStore.js
import { defineStore } from 'pinia'
import apiClient from '../services/api'

export const useEntityStore = defineStore('entities', {
  state: () => ({
    entities: [],
    isLoading: false,
    error: null,
  }),
  actions: {
    async fetchEntities() {
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.get('/entities')
        this.entities = response.data
      } catch (error) {
        this.error = error.message || 'Failed to fetch entities'
        console.error('fetchEntities error:', error)
      } finally {
        this.isLoading = false
      }
    },

    async createEntity(entityData) {
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.post('/entities', entityData)
        // Add to store or re-fetch? For simplicity, re-fetch on mutations for now.
        // This ensures the list is always up-to-date with server state including server-generated fields.
        await this.fetchEntities() // Re-fetch to get the new entity with ID and timestamps
        return response.data // Return the created entity data if needed by component
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to create entity'
        console.error('createEntity error:', error)
        throw error // Re-throw to allow component to handle it
      }
      // No finally isLoading = false here, as fetchEntities will handle it.
    },

    async updateEntity(entityData) {
      this.isLoading = true
      this.error = null
      try {
        const response = await apiClient.put(`/entities/${entityData.id}`, entityData)
        await this.fetchEntities() // Re-fetch for consistency
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to update entity'
        console.error('updateEntity error:', error)
        throw error // Re-throw
      }
    },

    async deleteEntity(entityId) {
      this.isLoading = true
      this.error = null
      try {
        await apiClient.delete(`/entities/${entityId}`)
        // Remove from local store or re-fetch? Re-fetch for simplicity.
        await this.fetchEntities()
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to delete entity'
        console.error('deleteEntity error:', error)
        throw error // Re-throw
      }
    },
  },
})
