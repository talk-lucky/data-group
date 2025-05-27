<template>
  <v-dialog :model-value="show" @update:model-value="$emit('update:show', $event)" max-width="500px" persistent>
    <v-card>
      <v-card-title class="text-h5">{{ title }}</v-card-title>
      <v-card-text>{{ message }}</v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn :color="cancelColor" text @click="$emit('cancel')">{{ cancelText }}</v-btn>
        <v-btn :color="confirmColor" text @click="$emit('confirm')">{{ confirmText }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';

defineProps({
  show: Boolean,
  title: {
    type: String,
    default: 'Confirm Action'
  },
  message: {
    type: String,
    default: 'Are you sure you want to perform this action?'
  },
  confirmText: {
    type: String,
    default: 'Confirm'
  },
  cancelText: {
    type: String,
    default: 'Cancel'
  },
  confirmColor: {
    type: String,
    default: 'primary'
  },
  cancelColor: {
    type: String,
    default: 'grey'
  }
});

defineEmits(['update:show', 'confirm', 'cancel']);
</script>
