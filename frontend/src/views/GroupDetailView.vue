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

    <v-card v-if="group && !groupStore.loading" class="mb-5">
      <v-card-title class="text-h4 d-flex justify-space-between align-center">
        <span>Group: {{ group.name }}</span>
         <v-chip v-if="entityName" color="primary" class="ml-2">{{ entityName }}</v-chip>
      </v-card-title>
      <v-card-subtitle class="pb-2">
        ID: {{ group.id }}
      </v-card-subtitle>
      <v-divider></v-divider>
      <v-card-text>
        <v-list dense>
          <v-list-item>
            <v-list-item-title><strong>Description:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ group.description || 'N/A' }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item v-if="entityName">
            <v-list-item-title><strong>Associated Entity:</strong></v-list-item-title>
            <v-list-item-subtitle>
              <router-link :to="`/entities/${group.entity_id}`" class="text-decoration-none">
                {{ entityName }} (ID: {{ group.entity_id }})
                <v-icon small class="ml-1">mdi-link-variant</v-icon>
              </router-link>
            </v-list-item-subtitle>
          </v-list-item>
           <v-list-item>
            <v-list-item-title><strong>Created At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(group.created_at) }}</v-list-item-subtitle>
          </v-list-item>
          <v-list-item>
            <v-list-item-title><strong>Last Updated At:</strong></v-list-item-title>
            <v-list-item-subtitle>{{ formatDate(group.updated_at) }}</v-list-item-subtitle>
          </v-list-item>
        </v-list>

        <v-divider class="my-4"></v-divider>
        
        <h3 class="text-h6 mb-2">Grouping Rules:</h3>
        <div v-if="group.rules_json && group.rules_json !== '{}' && group.rules_json !== '[]'">
            <GroupRuleBuilder 
                :model-value="group.rules_json" 
                :entity-id="group.entity_id" 
                read-only 
                class="mb-3" 
            />
            <!-- Fallback for simple JSON display if needed, or for debugging -->
            <!-- 
            <v-sheet elevation="1" rounded class="pa-3 mt-1" style="background-color: #f5f5f5;">
              <pre style="white-space: pre-wrap; word-break: break-all;">{{ formattedRulesJson }}</pre>
            </v-sheet>
            -->
        </div>
        <v-alert v-else type="info" variant="tonal">
            No specific rules defined for this group, or rules are empty.
        </v-alert>

      </v-card-text>
       <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn color="primary" :to="`/groups/${groupId}/edit`">
          <v-icon left>mdi-pencil</v-icon>
          Edit Group
        </v-btn>
      </v-card-actions>
    </v-card>

    <div v-if="!group && !groupStore.loading && !groupStore.error" class="text-center my-5">
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
const group = computed(() => groupStore.currentGroup);

const entityName = computed(() => {
  if (group.value?.entity_id && entityStore.entities.length > 0) {
    const entity = entityStore.entities.find(e => e.id === group.value.entity_id);
    return entity ? entity.name : 'Unknown Entity';
  }
  return 'Loading Entity...';
});

const formattedRulesJson = computed(() => {
  if (group.value?.rules_json) {
    try {
      const parsed = JSON.parse(group.value.rules_json);
      return JSON.stringify(parsed, null, 2); // Pretty print
    } catch (e) {
      return group.value.rules_json; // Return as is if not valid JSON
    }
  }
  return 'No rules defined.';
});

onMounted(async () => {
  if (groupId.value) {
    await groupStore.fetchGroupDefinitionById(groupId.value);
    // If entity_id exists on the fetched group, and entities are not loaded, fetch them
    if (group.value?.entity_id && entityStore.entities.length === 0) {
      await entityStore.fetchEntities();
    }
  }
});

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
