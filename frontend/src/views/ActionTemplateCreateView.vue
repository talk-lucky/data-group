<template>
  <v-container fluid>
    <v-btn to="/actiontemplates" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Action Templates
    </v-btn>
    <h1 class="text-h4 mb-4">Create New Action Template</h1>
    <ActionTemplateForm @template-saved="handleTemplateSaved" @cancel-form="navigateToTemplatesList" />
  </v-container>
</template>

<script setup>
import { useRouter } from 'vue-router';
import ActionTemplateForm from '@/components/ActionTemplateForm.vue';

const router = useRouter();

function handleTemplateSaved(savedTemplate) {
  console.log('Action Template saved:', savedTemplate);
  router.push('/actiontemplates');
}

function navigateToTemplatesList() {
  router.push('/actiontemplates');
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
