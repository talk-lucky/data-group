import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import router from './router'; // Import router
import vuetify from './plugins/vuetify'; // Import Vuetify plugin

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router); // Use router
app.use(vuetify); // Use Vuetify

app.mount('#app');
