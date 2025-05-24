<template>
  <v-container fluid>
    <v-btn to="/workflows" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Workflows
    </v-btn>

    <v-progress-linear v-if="workflowStore.loading && !workflow" indeterminate color="primary" class="my-4"></v-progress-linear>
    
    <v-alert v-if="workflowStore.error && !workflow" type="error" dismissible class="my-4">
      Error fetching workflow: {{ workflowStore.error }}
    </v-alert>

    <v-card v-if="workflow && !workflowStore.loading" class="mb-5">
      <v-card-title class="text-h4 d-flex justify-space-between align-center">
        <span>Workflow: {{ workflow.name }}</span>
        <v-chip :color="workflow.is_enabled ? 'success' : 'grey'" small class="ml-2">
          {{ workflow.is_enabled ? 'Enabled' : 'Disabled' }}
        </v-chip>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ workflow.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-list dense>
          <v-list-item>
            <v-list-item-title><strong>Description:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ workflow.description || 'N/A' }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Trigger Type:</strong></v-list-item-title>
            <v-list-item-subtitle>
                <v-chip small :color="getTriggerTypeColor(workflow.trigger_type)" class="mr-2">{{ workflow.trigger_type }}</v-chip>
            </v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Trigger Configuration:</strong></v-list-item-title>
            <v-list-item-subtitle>
                <pre class="config-display">{{ formattedTriggerConfig }}</pre>
                <div v-if="triggerTypeIsGroup && groupName" class="text-caption mt-1">
                    (Targets Group: 
                    <router-link :to="`/groups/${parsedTriggerConfig.group_id}`" class="text-decoration-none">
                        {{ groupName }} <v-icon x-small>mdi-link</v-icon>
                    </router-link>
                    )
                </div>
            </v-list-item-subtitle>
          </v-list-item>
           <v-list-item>
            <v-list-item-title><strong>Created At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(workflow.created_at) }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(workflow.updated_at) }}</v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-divider class="my-4"></v-divider>
        
        <h3 class="text-h6 mb-2">Action Sequence:</h3>
         <div v-if="workflow.action_sequence_json && workflow.action_sequence_json !== '[]'">
             <WorkflowActionSequenceBuilder 
                :model-value="workflow.action_sequence_json"
                read-only 
                class="mb-3" 
            />
        </div>
        <v-alert v-else type="info" variant="tonal">
            No actions defined in the sequence.
        </v-alert>

      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/workflows/${workflowId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Workflow
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!workflow && !workflowStore.loading && !workflowStore.error" class="text-center my-5">
      <v-alert type="warning">
        Workflow with ID "{{ workflowId }}" not found.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue';
import { useRoute } from 'vue-router';
import { useWorkflowStore } from '@/store/workflowStore';
import { useGroupStore } from '@/store/groupStore'; // For resolving group_id to name
import WorkflowActionSequenceBuilder from '@/components/WorkflowActionSequenceBuilder.vue';

const route = useRoute();
const workflowStore = useWorkflowStore();
const groupStore = useGroupStore();

const workflowId = computed(() => route.params.id);
const workflow = computed(() => workflowStore.currentWorkflow);

const parsedTriggerConfig = ref({});
const groupName = ref('');

const formattedTriggerConfig = computed(() => {
  if (workflow.value?.trigger_config) {
    try {
      const parsed = JSON.parse(workflow.value.trigger_config);
      parsedTriggerConfig.value = parsed; // Store for other uses
      return JSON.stringify(parsed, null, 2); // Pretty print
    } catch (e) {
      parsedTriggerConfig.value = {};
      return workflow.value.trigger_config; // Return as is if not valid JSON
    }
  }
  parsedTriggerConfig.value = {};
  return 'N/A';
});

const triggerTypeIsGroup = computed(() => {
    return workflow.value?.trigger_type === 'on_group_update' && parsedTriggerConfig.value.group_id;
});

async function fetchGroupName(groupId) {
    if (groupStore.groups.length === 0) {
        await groupStore.fetchGroupDefinitions();
    }
    const group = groupStore.groups.find(g => g.id === groupId);
    groupName.value = group ? group.name : 'Unknown Group';
}

watch(triggerTypeIsGroup, (isGroupTrigger) => {
    if(isGroupTrigger && parsedTriggerConfig.value.group_id) {
        fetchGroupName(parsedTriggerConfig.value.group_id);
    } else {
        groupName.value = '';
    }
}, { immediate: true });

// Re-fetch group name if workflow data (and thus trigger_config) changes
watch(workflow, (newWorkflow) => {
    if (newWorkflow?.trigger_type === 'on_group_update' && newWorkflow.trigger_config) {
        try {
            const config = JSON.parse(newWorkflow.trigger_config);
            if (config.group_id) {
                fetchGroupName(config.group_id);
            }
        } catch(e) { console.error("Error parsing trigger_config for group name fetch:", e); }
    }
}, { deep: true });


onMounted(async () => {
  if (workflowId.value) {
    await workflowStore.fetchWorkflowById(workflowId.value);
    // Initial check for group name after workflow is fetched
    if (triggerTypeIsGroup.value && parsedTriggerConfig.value.group_id) {
        fetchGroupName(parsedTriggerConfig.value.group_id);
    }
  }
});

onBeforeUnmount(() => {
  workflowStore.clearCurrentWorkflow();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try { return new Date(dateTimeString).toLocaleString(); } catch (e) { return dateTimeString; }
}

function getTriggerTypeColor(type) {
  const colors = {
    'on_group_update': 'orange',
    'manual': 'blue-grey',
    'scheduled': 'teal',
  };
  return colors[type] || 'grey';
}
</script>

<style scoped>
.text-h4 { font-weight: 500; }
.v-list-item-title { font-weight: bold; }
pre.config-display {
  background-color: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
