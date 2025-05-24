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
    loading: false, // For general group list/detail fetching
    error: null,    // For general group list/detail errors
    currentGroupResults: { 
      member_ids: [], 
      calculated_at: null, 
      member_count: 0, 
      isLoading: false, 
      error: null 
    },
    calculationStatus: { 
      isLoading: false, 
      error: null, 
      message: '' 
    },
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
      this.error = null; // General error
      // Reset results and calculation status when current group is cleared
      this.currentGroupResults = { member_ids: [], calculated_at: null, member_count: 0, isLoading: false, error: null };
      this.calculationStatus = { isLoading: false, error: null, message: '' };
    },
    async triggerGroupCalculation(groupId) {
      this.calculationStatus.isLoading = true;
      this.calculationStatus.error = null;
      this.calculationStatus.message = '';
      try {
        const response = await calculateGroup(groupId); // from apiService
        this.calculationStatus.message = response.data?.message || 'Group calculation triggered successfully. Results are being processed.';
        // Optionally, immediately fetch results or update group info if response contains it
        // For now, we'll let the user click "View Results"
        // await this.fetchGroupResults(groupId); // Or, if the calculation is synchronous for results
        if (response.data && response.data.calculated_at && this.currentGroup && this.currentGroup.id === groupId) {
           // If the calculate endpoint immediately returns results or a new calculation time
           this.currentGroupResults.calculated_at = response.data.calculated_at;
           this.currentGroupResults.member_count = response.data.member_count || 0;
           // If it returns member_ids directly:
           // this.currentGroupResults.member_ids = response.data.member_ids || [];
        }
      } catch (error) {
        this.calculationStatus.error = error.response?.data?.error || error.message || 'Failed to trigger group calculation';
        console.error(`Error triggering calculation for group ${groupId}:`, error);
      } finally {
        this.calculationStatus.isLoading = false;
      }
    },
    async fetchGroupResults(groupId) {
      this.currentGroupResults.isLoading = true;
      this.currentGroupResults.error = null;
      try {
        const response = await getGroupResults(groupId); // from apiService
        this.currentGroupResults.member_ids = response.data?.member_ids || [];
        this.currentGroupResults.calculated_at = response.data?.calculated_at || null;
        this.currentGroupResults.member_count = response.data?.member_count || 0;
      } catch (error) {
        this.currentGroupResults.error = error.response?.data?.error || error.message || 'Failed to fetch group results';
        // Clear data on error
        this.currentGroupResults.member_ids = [];
        this.currentGroupResults.calculated_at = null;
        this.currentGroupResults.member_count = 0;
        console.error(`Error fetching results for group ${groupId}:`, error);
      } finally {
        this.currentGroupResults.isLoading = false;
      }
    },
  },
});
