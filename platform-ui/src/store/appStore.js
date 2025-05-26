// src/store/appStore.js
import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', {
  state: () => ({
    // Example state
    appName: 'Platform UI',
    user: null, // Placeholder for user information
    isLoading: false,
  }),
  getters: {
    // Example getter
    isUserLoggedIn: (state) => !!state.user,
  },
  actions: {
    // Example actions
    setLoading(isLoading) {
      this.isLoading = isLoading
    },
    setUser(userData) {
      this.user = userData
    },
    logout() {
      this.user = null
      // Potentially also clear tokens, redirect, etc.
    },
  },
})
