<template>
  <v-container fluid>
    <v-btn to="/actiontemplates" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Action Templates
    </v-btn>

    <v-progress-linear v-if="actionTemplateStore.loading && !template" indeterminate color="primary" class="my-4"></v-progress-linear>
    
    <v-alert v-if="actionTemplateStore.error && !template" type="error" dismissible class="my-4">
      Error fetching action template: {{ actionTemplateStore.error }}
    </v-alert>

    <v-card v-if="template && !actionTemplateStore.loading" class="mb-5">
      <v-card-title class="text-h4 d-flex justify-space-between align-center">
        <span>Action Template: {{ template.name }}</span>
        <v-chip small :color="getActionTypeColor(template.action_type)" class="ml-2">{{ template.action_type }}</v-chip>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ template.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-list dense>
          <v-list-item>
            <v-list-item-title><strong>Description:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ template.description || 'N/A' }}</v-list-item-subtitle>
          </v-list-item>
           <v-list-item>
            <v-list-item-title><strong>Created At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(template.created_at) }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(template.updated_at) }}</v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-divider class="my-4"></v-divider>
        
        <h3 class="text-h6 mb-2">Template Content:</h3>
        <v-sheet elevation="1" rounded class="pa-3 mt-1" style="background-color: #f5f5f5;">
          <pre style="white-space: pre-wrap; word-break: break-all;">{{ formattedTemplateContent }}</pre>
        </v-sheet>

      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/actiontemplates/${templateId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Template
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!template && !actionTemplateStore.loading && !actionTemplateStore.error" class="text-center my-5">
      <v-alert type="warning">
        Action Template with ID "{{ templateId }}" not found.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import { useActionTemplateStore } from '@/store/actionTemplateStore';

const route = useRoute();
const actionTemplateStore = useActionTemplateStore();

const templateId = computed(() => route.params.id);
const template = computed(() => actionTemplateStore.currentActionTemplate);

const formattedTemplateContent = computed(() => {
  if (template.value?.template_content) {
    try {
      const parsed = JSON.parse(template.value.template_content);
      return JSON.stringify(parsed, null, 2); // Pretty print
    } catch (e) {
      // If it's not valid JSON, return as is (though it should be)
      return template.value.template_content;
    }
  }
  return 'N/A';
});

onMounted(async () => {
  if (templateId.value) {
    await actionTemplateStore.fetchActionTemplateById(templateId.value);
  }
});

onBeforeUnmount(() => {
  actionTemplateStore.clearCurrentActionTemplate();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try { return new Date(dateTimeString).toLocaleString(); } catch (e) { return dateTimeString; }
}

function getActionTypeColor(type) {
  const colors = {
    'webhook': 'blue',
    'email': 'green',
    'custom_api_call': 'purple',
  };
  return colors[type] || 'grey';
}
</script>

<style scoped>
.text-h4 { font-weight: 500; }
.v-list-item-title { font-weight: bold; }
pre {
  background-color: #f0f0f0;
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto;
}
</style>
