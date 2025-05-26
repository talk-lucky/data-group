<template>
  <v-app>
    <v-app-bar app color="primary" dark>
      <v-toolbar-title>Platform UI</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn to="/" text class="mx-1">
        <v-icon left>mdi-home</v-icon>
        Home
      </v-btn>
      <v-btn to="/entities" text class="mx-1">
        <v-icon left>mdi-database-outline</v-icon> <!-- Changed icon for entities -->
        Entities
      </v-btn>
      <!-- More navigation items can go here -->
    </v-app-bar>
    <v-main>
      <v-container fluid>
        <router-view />
      </v-container>
    </v-main>
    <v-footer app>
      <span>&copy; {{ new Date().getFullYear() }}</span>
    </v-footer>
  </v-app>
</template>

<script setup>
// No complex script needed for basic layout
</script>

<style scoped>
/* Add any layout-specific styles here */
.v-btn {
  text-transform: none; /* Keep button text casing as is */
}
</style>
