import { defineStore } from 'pinia';
import {
  getGroupDefinitions,
  getGroupDefinitionById,
  createGroupDefinition,
  updateGroupDefinition,
  deleteGroupDefinition,
} from '@/services/apiService'; // Assuming apiService is correctly set up

export const useGroupStore = defineStore('group', {
  state: () => ({
    groups: [],
    currentGroup: null,
    loading: false,
    error: null,
  }),
  actions: {
    async fetchGroupDefinitions() {
      this.loading = true;
      this.error = null;
      try {
        const response = await getGroupDefinitions();
        this.groups = response.data || []; // Assuming response.data contains the array
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch group definitions';
        this.groups = []; // Clear groups on error
        console.error('Error fetching group definitions:', error);
      } finally {
        this.loading = false;
      }
    },
    async fetchGroupDefinitionById(groupId) {
      this.loading = true;
      this.error = null;
      this.currentGroup = null;
      try {
        const response = await getGroupDefinitionById(groupId);
        this.currentGroup = response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch group definition';
        this.currentGroup = null;
        console.error(`Error fetching group definition ${groupId}:`, error);
      } finally {
        this.loading = false;
      }
    },
    async addGroupDefinition(groupData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createGroupDefinition(groupData);
        this.groups.push(response.data); // Add to local cache
        this.currentGroup = response.data; // Set as current
        return response.data; // Return the created group
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to create group definition';
        console.error('Error creating group definition:', error);
        throw error; // Re-throw to allow components to handle it
      } finally {
        this.loading = false;
      }
    },
    async editGroupDefinition(groupId, groupData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateGroupDefinition(groupId, groupData);
        const index = this.groups.findIndex(g => g.id === groupId);
        if (index !== -1) {
          this.groups[index] = response.data;
        }
        if (this.currentGroup && this.currentGroup.id === groupId) {
          this.currentGroup = response.data;
        }
        return response.data; // Return the updated group
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to update group definition';
        console.error(`Error updating group definition ${groupId}:`, error);
        throw error; // Re-throw
      } finally {
        this.loading = false;
      }
    },
    async removeGroupDefinition(groupId) {
      this.loading = true;
      this.error = null;
      try {
        await deleteGroupDefinition(groupId);
        this.groups = this.groups.filter(g => g.id !== groupId);
        if (this.currentGroup && this.currentGroup.id === groupId) {
          this.currentGroup = null;
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to delete group definition';
        console.error(`Error deleting group definition ${groupId}:`, error);
        throw error; // Re-throw
      } finally {
        this.loading = false;
      }
    },
    clearCurrentGroup() {
      this.currentGroup = null;
      this.error = null;
    },
  },
});
