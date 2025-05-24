<template>
  <v-container fluid>
    <v-btn to="/workflows" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Workflows
    </v-btn>
    <h1 class="text-h4 mb-4">Create New Workflow</h1>
    <WorkflowForm @workflow-saved="handleWorkflowSaved" @cancel-form="navigateToWorkflowsList" />
  </v-container>
</template>

<script setup>
import { useRouter } from 'vue-router';
import WorkflowForm from '@/components/WorkflowForm.vue';

const router = useRouter();

function handleWorkflowSaved(savedWorkflow) {
  console.log('Workflow saved:', savedWorkflow);
  router.push('/workflows');
}

function navigateToWorkflowsList() {
  router.push('/workflows');
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
