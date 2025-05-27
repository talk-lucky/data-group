import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../views/HomePage.vue'
import EntityListPage from '../views/entities/EntityListPage.vue'
import EntityDetailPage from '../views/entities/EntityDetailPage.vue' // Added this line

const routes = [
  {
    path: '/',
    name: 'Home',
    component: HomePage
  },
  {
    path: '/entities',
    name: 'Entities',
    component: EntityListPage
  },
  {
    path: '/entities/:id', // Route for entity details and attribute management
    name: 'EntityDetail',
    component: EntityDetailPage,
    props: true // Pass route params as props to the component
  }
  // Other routes can be added here
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes
})

export default router
