<template>
  <v-container fluid>
    <v-btn to="/groups" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Groups List
    </v-btn>

    <v-progress-linear v-if="groupStore.loading && !group" indeterminate color="primary" class="my-4"></v-progress-linear>
    
    <v-alert v-if="groupStore.error && !group" type="error" dismissible class="my-4">
      Error fetching group definition: {{ groupStore.error }}
    </v-alert>

    <v-card v-if="groupDetails && !groupStore.loading" class="mb-5">
      <v-card-title class="text-h4 d-flex justify-space-between align-center">
        <span>Group: {{ groupDetails.name }}</span>
         <v-chip v-if="entityName" color="primary" class="ml-2">{{ entityName }}</v-chip>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ groupDetails.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-list dense>
          <v-list-item>
            <v-list-item-title><strong>Description:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ groupDetails.description || 'N/A' }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item v-if="entityName">
            <v-list-item-title><strong>Associated Entity:</strong></v-list-item-title>
            <v-list-item-subtitle>
              <router-link :to="`/entities/${groupDetails.entity_id}`" class="text-decoration-none">
                {{ entityName }} (ID: {{ groupDetails.entity_id }})
                <v-icon small class="ml-1">mdi-link-variant</v-icon>
              </router-link>
            </v-list-item-subtitle>
          </v-list-item>
           <v-list-item>
            <v-list-item-title><strong>Created At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(groupDetails.created_at) }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(groupDetails.updated_at) }}</v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-divider class="my-4"></v-divider>
        
        <h3 class="text-h6 mb-2">Grouping Rules:</h3>
        <div v-if="groupDetails.rules_json && groupDetails.rules_json !== '{}' && groupDetails.rules_json !== '[]'">
            <GroupRuleBuilder 
                :model-value="groupDetails.rules_json" 
                :entity-id="groupDetails.entity_id" 
                read-only 
                class="mb-3" 
            />
        </div>
        <v-alert v-else type="info" variant="tonal" class="mb-3">
            No specific rules defined for this group, or rules are empty.
        </v-alert>

        <v-divider class="my-4"></v-divider>

        <!-- Calculate Group Section -->
        <h3 class="text-h6 mb-2">Calculate Group Membership</h3>
        <v-btn 
          color="info" 
          @click="triggerCalculation" 
          :loading="isCalculating" 
          :disabled="isCalculating"
          class="mr-3"
        >
          <v-icon left>mdi-calculator</v-icon>
          {{ groupResults && groupResults.calculated_at ? 'Recalculate Group' : 'Calculate Group Now' }}
        </v-btn>
         <v-alert v-if="calculationStatus.message && !calculationStatus.error" type="info" dense class="mt-3 mb-2" closable>
          {{ calculationStatus.message }}
        </v-alert>
        <v-alert v-if="calculationStatus.error" type="error" dense class="mt-3 mb-2" closable>
          Calculation Error: {{ calculationStatus.error }}
        </v-alert>
        
        <v-divider class="my-4"></v-divider>

        <!-- Results Section -->
        <h3 class="text-h6 mb-2">Group Results</h3>
        <v-btn 
            color="success" 
            @click="refreshResults" 
            :loading="resultsLoading"
            :disabled="resultsLoading"
            class="mb-3"
        >
          <v-icon left>mdi-refresh</v-icon>
          View/Refresh Results
        </v-btn>

        <v-progress-linear v-if="resultsLoading" indeterminate color="secondary" class="my-2"></v-progress-linear>
        
        <v-alert v-if="groupResults.error" type="error" dense class="mb-3" closable>
          Error fetching results: {{ groupResults.error }}
        </v-alert>

        <div v-if="!resultsLoading && !groupResults.error">
          <p v-if="groupResults.calculated_at">
            <strong>Last Calculated At:</strong> {{ formatDate(groupResults.calculated_at) }}
          </p>
          <p v-else class="text-grey">No calculation results available yet. Click "Calculate" or "Refresh".</p>
          
          <p><strong>Member Count:</strong> {{ groupResults.member_count || 0 }}</p>

          <div v-if="groupResults.member_ids && groupResults.member_ids.length > 0" class="mt-3">
            <h4 class="text-subtitle-1">Member IDs (Sample - first {{ displayLimit }} shown):</h4>
            <v-list dense lines="one" class="pa-0">
              <v-list-item 
                v-for="(id, index) in limitedMemberIds" 
                :key="index"
                class="pl-0"
              >
                <v-list-item-title>{{ id }}</v-list-item-title>
              </v-list-item>
            </v-list>
            <p v-if="groupResults.member_ids.length > displayLimit" class="text-caption text-grey">
              ...and {{ groupResults.member_ids.length - displayLimit }} more.
            </p>
          </div>
           <p v-else-if="groupResults.calculated_at && groupResults.member_count === 0" class="text-grey mt-2">
            This group has no members based on the last calculation.
          </p>
        </div>
      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/groups/${groupId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Group
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!groupDetails && !groupStore.loading && !groupStore.error" class="text-center my-5">
      <v-alert type="warning">
        Group with ID "{{ groupId }}" not found.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue';
import { useRoute } from 'vue-router';
import { useGroupStore } from '@/store/groupStore';
import { useEntityStore } from '@/store/entityStore';
import GroupRuleBuilder from '@/components/GroupRuleBuilder.vue'; // Assuming it can handle a read-only state

const route = useRoute();
const groupStore = useGroupStore();
const entityStore = useEntityStore();

const groupId = computed(() => route.params.id);
const groupDetails = computed(() => groupStore.currentGroup); // Renamed for clarity
const groupResults = computed(() => groupStore.currentGroupResults);
const calculationStatus = computed(() => groupStore.calculationStatus);
const isCalculating = computed(() => groupStore.calculationStatus.isLoading);
const resultsLoading = computed(() => groupStore.currentGroupResults.isLoading);

const displayLimit = 50; // Max number of member IDs to display

const entityName = computed(() => {
  if (groupDetails.value?.entity_id && entityStore.entities.length > 0) {
    const entity = entityStore.entities.find(e => e.id === groupDetails.value.entity_id);
    return entity ? entity.name : 'Unknown Entity';
  }
  return 'Loading Entity...';
});

const limitedMemberIds = computed(() => {
  return groupResults.value?.member_ids?.slice(0, displayLimit) || [];
});


onMounted(async () => {
  if (groupId.value) {
    await groupStore.fetchGroupDefinitionById(groupId.value);
    if (groupDetails.value?.entity_id && entityStore.entities.length === 0) {
      await entityStore.fetchEntities();
    }
    // Also fetch initial results
    await groupStore.fetchGroupResults(groupId.value);
  }
});

function triggerCalculation() {
  if (groupId.value) {
    groupStore.triggerGroupCalculation(groupId.value);
  }
}

function refreshResults() {
  if (groupId.value) {
    groupStore.fetchGroupResults(groupId.value);
  }
}

onBeforeUnmount(() => {
  groupStore.clearCurrentGroup();
});

function formatDate(dateTimeString) {
  if (!dateTimeString) return 'N/A';
  try { return new Date(dateTimeString).toLocaleString(); } catch (e) { return dateTimeString; }
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
