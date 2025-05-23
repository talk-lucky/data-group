import { defineStore } from 'pinia';
import {
  getWorkflows,
  getWorkflowById,
  createWorkflow,
  updateWorkflow,
  deleteWorkflow,
} from '@/services/apiService';

export const useWorkflowStore = defineStore('workflow', {
  state: () => ({
    workflows: [],
    currentWorkflow: null,
    loading: false,
    error: null,
  }),
  actions: {
    async fetchWorkflows() {
      this.loading = true;
      this.error = null;
      try {
        const response = await getWorkflows();
        this.workflows = response.data || [];
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch workflows';
        this.workflows = [];
        console.error('Error fetching workflows:', error);
      } finally {
        this.loading = false;
      }
    },
    async fetchWorkflowById(workflowId) {
      this.loading = true;
      this.error = null;
      this.currentWorkflow = null;
      try {
        const response = await getWorkflowById(workflowId);
        this.currentWorkflow = response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch workflow';
        this.currentWorkflow = null;
        console.error(`Error fetching workflow ${workflowId}:`, error);
      } finally {
        this.loading = false;
      }
    },
    async addWorkflow(workflowData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createWorkflow(workflowData);
        this.workflows.push(response.data);
        this.currentWorkflow = response.data;
        return response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to create workflow';
        console.error('Error creating workflow:', error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async editWorkflow(workflowId, workflowData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateWorkflow(workflowId, workflowData);
        const index = this.workflows.findIndex(w => w.id === workflowId);
        if (index !== -1) {
          this.workflows[index] = response.data;
        }
        if (this.currentWorkflow && this.currentWorkflow.id === workflowId) {
          this.currentWorkflow = response.data;
        }
        return response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to update workflow';
        console.error(`Error updating workflow ${workflowId}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async removeWorkflow(workflowId) {
      this.loading = true;
      this.error = null;
      try {
        await deleteWorkflow(workflowId);
        this.workflows = this.workflows.filter(w => w.id !== workflowId);
        if (this.currentWorkflow && this.currentWorkflow.id === workflowId) {
          this.currentWorkflow = null;
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to delete workflow';
        console.error(`Error deleting workflow ${workflowId}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    clearCurrentWorkflow() {
      this.currentWorkflow = null;
      this.error = null;
    },
  },
});
