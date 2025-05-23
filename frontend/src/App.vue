<template>
  <v-app>
    <v-app-bar app color="primary" dark>
      <v-toolbar-title>Metadata Management</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn text to="/">Home (Entities)</v-btn>
      <v-btn text to="/entities">Entities</v-btn>
      <v-btn text to="/datasources">Data Sources</v-btn>
      <v-btn text to="/groups">Groups</v-btn>
      <!-- Add other navigation items here if needed -->
    </v-app-bar>

    <v-main>
      <v-container fluid>
        <router-view />
      </v-container>
    </v-main>

    <v-footer app color="primary" dark padless>
      <v-col class="text-center" cols="12">
        &copy; {{ new Date().getFullYear() }} — <strong>Metadata Management App</strong>
      </v-col>
    </v-footer>
  </v-app>
</template>

<script setup>
// No specific script logic needed for App.vue with this basic layout.
// Theme (light/dark) could be managed here if dynamic theming is desired.
</script>

<style>
/* Global styles can go here if absolutely necessary, but prefer Vuetify's styling system */
/* For example, to ensure v-main content takes up available space: */
.v-main {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
}
.v-container {
  flex: 1 1 auto;
}

/* Custom global error/loading message styles if not using Vuetify components for them */
/* .global-error-message { ... } */
/* .global-loading-message { ... } */
</style>
