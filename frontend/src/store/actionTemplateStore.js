import { defineStore } from 'pinia';
import {
  getActionTemplates,
  getActionTemplateById,
  createActionTemplate,
  updateActionTemplate,
  deleteActionTemplate,
} from '@/services/apiService';

export const useActionTemplateStore = defineStore('actionTemplate', {
  state: () => ({
    actionTemplates: [],
    currentActionTemplate: null,
    loading: false,
    error: null,
  }),
  getters: {
    actionTemplateOptions: (state) => {
      return state.actionTemplates.map(template => ({
        title: `${template.name} (${template.action_type})`,
        value: template.id,
        // You can also include other properties like description or action_type if needed by components
        description: template.description,
        action_type: template.action_type,
        template_content: template.template_content, // To show details when selected
      }));
    },
  },
  actions: {
    async fetchActionTemplates() {
      this.loading = true;
      this.error = null;
      try {
        const response = await getActionTemplates();
        this.actionTemplates = response.data || [];
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch action templates';
        this.actionTemplates = [];
        console.error('Error fetching action templates:', error);
      } finally {
        this.loading = false;
      }
    },
    async fetchActionTemplateById(templateId) {
      this.loading = true;
      this.error = null;
      this.currentActionTemplate = null;
      try {
        const response = await getActionTemplateById(templateId);
        this.currentActionTemplate = response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to fetch action template';
        this.currentActionTemplate = null;
        console.error(`Error fetching action template ${templateId}:`, error);
      } finally {
        this.loading = false;
      }
    },
    async addActionTemplate(templateData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createActionTemplate(templateData);
        this.actionTemplates.push(response.data);
        this.currentActionTemplate = response.data;
        return response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to create action template';
        console.error('Error creating action template:', error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async editActionTemplate(templateId, templateData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateActionTemplate(templateId, templateData);
        const index = this.actionTemplates.findIndex(t => t.id === templateId);
        if (index !== -1) {
          this.actionTemplates[index] = response.data;
        }
        if (this.currentActionTemplate && this.currentActionTemplate.id === templateId) {
          this.currentActionTemplate = response.data;
        }
        return response.data;
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to update action template';
        console.error(`Error updating action template ${templateId}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async removeActionTemplate(templateId) {
      this.loading = true;
      this.error = null;
      try {
        await deleteActionTemplate(templateId);
        this.actionTemplates = this.actionTemplates.filter(t => t.id !== templateId);
        if (this.currentActionTemplate && this.currentActionTemplate.id === templateId) {
          this.currentActionTemplate = null;
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message || 'Failed to delete action template';
        console.error(`Error deleting action template ${templateId}:`, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    clearCurrentActionTemplate() {
      this.currentActionTemplate = null;
      this.error = null;
    },
  },
});
