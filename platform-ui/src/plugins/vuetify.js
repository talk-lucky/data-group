// src/plugins/vuetify.js
import 'vuetify/styles' // Global Vuetify styles
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { aliases, mdi } from 'vuetify/iconsets/mdi' // Material Design Icons

// Custom theme definition
const myCustomTheme = {
  dark: false, // You can set this to true for a dark theme by default
  colors: {
    primary: '#1976D2',    // Example: Blue
    secondary: '#424242',  // Example: Grey
    accent: '#82B1FF',     // Example: Light Blue
    error: '#FF5252',      // Example: Red
    info: '#2196F3',       // Example: Bright Blue
    success: '#4CAF50',    // Example: Green
    warning: '#FB8C00',    // Example: Orange
    // You can add more custom colors here
    // surface: '#FFFFFF',
    // background: '#FFFFFF',
  }
}

export default createVuetify({
  components,
  directives,
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: {
      mdi,
    },
  },
  theme: {
    defaultTheme: 'myCustomTheme',
    themes: {
      myCustomTheme,
      // You could define a 'darkTheme' here as well if needed
      // darkTheme: { ...myCustomTheme, dark: true, colors: { ...myCustomTheme.colors, surface: '#000000' } }
    }
  }
})
