// frontend/src/plugins/vuetify.js

import 'vuetify/styles'; // Global Vuetify styles
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';
import { aliases, mdi } from 'vuetify/iconsets/mdi'; // Material Design Icons
import '@mdi/font/css/materialdesignicons.css'; // Ensure you are using css-loader

const vuetify = createVuetify({
  components,
  directives,
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: {
      mdi,
    },
  },
  // You can define a default theme (light/dark) or custom themes here
  theme: {
    defaultTheme: 'light',
    themes: {
      light: {
        dark: false,
        colors: {
          primary: '#1976D2', // Example primary color (Vue Material Blue)
          secondary: '#424242',
          accent: '#82B1FF',
          error: '#FF5252',
          info: '#2196F3',
          success: '#4CAF50',
          warning: '#FFC107',
        }
      },
      // You can also define a dark theme
      // dark: {
      //   dark: true,
      //   colors: { ... }
      // }
    }
  }
});

export default vuetify;
