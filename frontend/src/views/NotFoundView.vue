<template>
  <v-container class="fill-height" fluid>
    <v-row align="center" justify="center">
      <v-col cols="12" sm="8" md="6" lg="4" class="text-center">
        <v-icon size="100" color="grey lighten-1" class="mb-4">mdi-alert-circle-outline</v-icon>
        <h1 class="text-h3 font-weight-bold mb-3">404</h1>
        <h2 class="text-h5 mb-4">Page Not Found</h2>
        <p class="body-1 mb-6">
          Oops! The page you are looking for does not exist or may have been moved.
        </p>
        <v-btn color="primary" to="/" large>
          <v-icon left>mdi-home</v-icon>
          Go to Homepage
        </v-btn>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup>
// No specific script needed for a simple 404 page
</script>

<style scoped>
.fill-height {
  min-height: 80vh; /* Adjust as needed to fill viewport reasonably */
}
.text-h3 {
  color: #555;
}
.text-h5 {
  color: #777;
}
.body-1 {
  color: #888;
}
</style>
