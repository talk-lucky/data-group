import { createRouter, createWebHistory } from 'vue-router';
import EntitiesView from '@/views/EntitiesView.vue';
import EntityCreateView from '@/views/EntityCreateView.vue';
import EntityEditView from '@/views/EntityEditView.vue';
import EntityDetailView from '@/views/EntityDetailView.vue';
// For a default landing page or home, you might create a HomeView
// import HomeView from '@/views/HomeView.vue';

const routes = [
  {
    path: '/',
    redirect: '/entities', // Redirect root to entities list by default
    // For a dedicated home page:
    // name: 'Home',
    // component: HomeView
  },
  {
    path: '/entities',
    name: 'EntitiesView',
    component: EntitiesView,
  },
  {
    path: '/entities/new',
    name: 'EntityCreateView',
    component: EntityCreateView,
  },
  {
    path: '/entities/:id/edit',
    name: 'EntityEditView',
    component: EntityEditView,
    props: true, // Pass route.params to component props
  },
  {
    path: '/entities/:id/details',
    name: 'EntityDetailView',
    component: EntityDetailView,
    props: true, // Pass route.params to component props
  },
  // Fallback route for 404
  {
    path: '/:catchAll(.*)',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'), // Lazy load for non-critical view
  }
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL || '/'), // Ensure correct base for history
  routes,
});

export default router;
