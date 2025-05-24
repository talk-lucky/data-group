<template>
  <v-container fluid>
    <v-btn to="/groups" color="grey darken-2" class="mb-4">
      <v-icon left>mdi-arrow-left</v-icon>
      Back to Groups List
    </v-btn>
    <h1 class="text-h4 mb-4">Create New Group Definition</h1>
    <GroupForm @group-saved="handleGroupSaved" @cancel-form="navigateToGroupsList" />
  </v-container>
</template>

<script setup>
import { useRouter } from 'vue-router';
import GroupForm from '@/components/GroupForm.vue';
import { useGroupStore } from '@/store/groupStore'; // Optional: for snackbar or direct feedback

const router = useRouter();
// const groupStore = useGroupStore(); // If you want to show snackbar from store actions

function handleGroupSaved(savedGroup) {
  console.log('Group saved:', savedGroup);
  // Could show a snackbar here if not handled by the store/form itself
  // groupStore.showSnackbar('Group definition saved successfully!', 'success');
  router.push('/groups');
}

function navigateToGroupsList() {
  router.push('/groups');
}
</script>

<style scoped>
/* Add any view-specific styles here */
</style>
