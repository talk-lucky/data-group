<template>
  <v-container fluid>
    <v-btn to="/workflows" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Workflows
    </v-btn>
    <h1 class="text-h4 mb-4">Edit Workflow</h1>
    <v-progress-linear v-if="workflowStore.loading && !workflowStore.currentWorkflow" indeterminate color="primary" class="my-4"></v-progress-linear>
    <v-alert v-if="workflowStore.error && !workflowStore.currentWorkflow" type="error" dismissible class="my-4">
      Error fetching workflow details: {{ workflowStore.error }}
    </v-alert>
    <WorkflowForm 
      v-if="workflowStore.currentWorkflow"
      :workflow-id="workflowId"
      :initial-data="workflowStore.currentWorkflow"
      @workflow-saved="handleWorkflowSaved" 
      @cancel-form="navigateToWorkflowDetail"
    />
     <div v-if="!workflowStore.loading && !workflowStore.currentWorkflow && !workflowStore.error" class="text-center my-5">
      <v-alert type="warning">
        Workflow with ID "{{ workflowId }}" not found or could not be loaded.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import WorkflowForm from '@/components/WorkflowForm.vue';
import { useWorkflowStore } from '@/store/workflowStore';

const route = useRoute();
const router = useRouter();
const workflowStore = useWorkflowStore();

const workflowId = computed(() => route.params.id);

onMounted(() => {
  if (workflowId.value) {
    workflowStore.fetchWorkflowById(workflowId.value);
  }
});

onBeforeUnmount(() => {
  workflowStore.clearCurrentWorkflow();
});

function handleWorkflowSaved(savedWorkflow) {
  console.log('Workflow updated:', savedWorkflow);
  router.push(`/workflows/${savedWorkflow.id}`);
}

function navigateToWorkflowDetail() {
  if (workflowId.value) {
    router.push(`/workflows/${workflowId.value}`);
  } else {
    router.push('/workflows');
  }
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
