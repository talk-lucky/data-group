<template>
  <v-container fluid>
    <v-btn to="/groups" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Groups List
    </v-btn>
    <h1 class="text-h4 mb-4">Edit Group Definition</h1>
    <v-progress-linear v-if="groupStore.loading && !groupStore.currentGroup" indeterminate color="primary" class="my-4"></v-progress-linear>
    <v-alert v-if="groupStore.error && !groupStore.currentGroup" type="error" dismissible class="my-4">
      Error fetching group definition details: {{ groupStore.error }}
    </v-alert>
    <GroupForm 
      v-if="groupStore.currentGroup"
      :group-id="groupId"
      :initial-data="groupStore.currentGroup"
      @group-saved="handleGroupSaved" 
      @cancel-form="navigateToGroupDetail"
    />
     <div v-if="!groupStore.loading && !groupStore.currentGroup && !groupStore.error" class="text-center my-5">
      <v-alert type="warning">
        Group Definition with ID "{{ groupId }}" not found or could not be loaded.
      </v-alert>
    </div>
  </v-container>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import GroupForm from '@/components/GroupForm.vue';
import { useGroupStore } from '@/store/groupStore';

const route = useRoute();
const router = useRouter();
const groupStore = useGroupStore();

const groupId = computed(() => route.params.id);

onMounted(() => {
  if (groupId.value) {
    groupStore.fetchGroupDefinitionById(groupId.value);
  }
});

onBeforeUnmount(() => {
  groupStore.clearCurrentGroup(); // Clear when navigating away
});

function handleGroupSaved(savedGroup) {
  console.log('Group updated:', savedGroup);
  router.push(`/groups/${savedGroup.id}`); // Navigate to detail view after save
}

function navigateToGroupDetail() {
  if (groupId.value) {
    router.push(`/groups/${groupId.value}`);
  } else {
    router.push('/groups'); // Fallback
  }
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
