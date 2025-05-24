<template>
  <v-container fluid>
    <v-btn to="/actiontemplates" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Action Templates
    </v-btn>
    <h1 class="text-h4 mb-4">Edit Action Template</h1>
    <v-progress-linear v-if="actionTemplateStore.loading && !actionTemplateStore.currentActionTemplate" indeterminate color="primary" class="my-4"></v-progress-linear>
    <v-alert v-if="actionTemplateStore.error && !actionTemplateStore.currentActionTemplate" type="error" dismissible class="my-4">
      Error fetching action template details: {{ actionTemplateStore.error }}
    </v-alert>
    <ActionTemplateForm 
      v-if="actionTemplateStore.currentActionTemplate"
      :template-id="templateId"
      :initial-data="actionTemplateStore.currentActionTemplate"
      @template-saved="handleTemplateSaved" 
      @cancel-form="navigateToTemplateDetail"
    />
     <div v-if="!actionTemplateStore.loading && !actionTemplateStore.currentActionTemplate && !actionTemplateStore.error" class="text-center my-5">
      <v-alert type="warning">
        Action Template with ID "{{ templateId }}" not found or could not be loaded.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import ActionTemplateForm from '@/components/ActionTemplateForm.vue';
import { useActionTemplateStore } from '@/store/actionTemplateStore';

const route = useRoute();
const router = useRouter();
const actionTemplateStore = useActionTemplateStore();

const templateId = computed(() => route.params.id);

onMounted(() => {
  if (templateId.value) {
    actionTemplateStore.fetchActionTemplateById(templateId.value);
  }
});

onBeforeUnmount(() => {
  actionTemplateStore.clearCurrentActionTemplate();
});

function handleTemplateSaved(savedTemplate) {
  console.log('Action Template updated:', savedTemplate);
  router.push(`/actiontemplates/${savedTemplate.id}`);
}

function navigateToTemplateDetail() {
  if (templateId.value) {
    router.push(`/actiontemplates/${templateId.value}`);
  } else {
    router.push('/actiontemplates');
  }
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
