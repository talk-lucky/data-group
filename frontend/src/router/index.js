import { createRouter, createWebHistory } from 'vue-router';
import EntitiesView from '@/views/EntitiesView.vue';
import EntityCreateView from '@/views/EntityCreateView.vue';
import EntityEditView from '@/views/EntityEditView.vue';
import EntityDetailView from '@/views/EntityDetailView.vue';

import DataSourcesView from '@/views/DataSourcesView.vue';
import DataSourceCreateView from '@/views/DataSourceCreateView.vue';
import DataSourceEditView from '@/views/DataSourceEditView.vue';
import DataSourceDetailView from '@/views/DataSourceDetailView.vue';

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
  // Entity Routes
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
    props: true, 
  },
  {
    path: '/entities/:id/details',
    name: 'EntityDetailView',
    component: EntityDetailView,
    props: true, 
  },
  // Data Source Routes
  {
    path: '/datasources',
    name: 'DataSourcesView',
    component: DataSourcesView,
  },
  {
    path: '/datasources/new',
    name: 'DataSourceCreateView',
    component: DataSourceCreateView,
  },
  {
    path: '/datasources/:id/edit',
    name: 'DataSourceEditView',
    component: DataSourceEditView,
    props: true,
  },
  {
    path: '/datasources/:id/details',
    name: 'DataSourceDetailView',
    component: DataSourceDetailView,
    props: true,
  },
  // Group Definition Routes
  {
    path: '/groups',
    name: 'GroupsView',
    component: () => import('@/views/GroupsView.vue')
  },
  {
    path: '/groups/new',
    name: 'GroupCreate',
    component: () => import('@/views/GroupCreateView.vue')
  },
  {
    path: '/groups/:id/edit',
    name: 'GroupEdit',
    component: () => import('@/views/GroupEditView.vue'),
    props: true
  },
  {
    path: '/groups/:id', // Changed from /details to make it consistent
    name: 'GroupDetail',
    component: () => import('@/views/GroupDetailView.vue'),
    props: true
  },
  // Action Template Routes
  {
    path: '/actiontemplates',
    name: 'ActionTemplatesView',
    component: () => import('@/views/ActionTemplatesView.vue')
  },
  {
    path: '/actiontemplates/new',
    name: 'ActionTemplateCreate',
    component: () => import('@/views/ActionTemplateCreateView.vue')
  },
  {
    path: '/actiontemplates/:id/edit',
    name: 'ActionTemplateEdit',
    component: () => import('@/views/ActionTemplateEditView.vue'),
    props: true
  },
  {
    path: '/actiontemplates/:id',
    name: 'ActionTemplateDetail',
    component: () => import('@/views/ActionTemplateDetailView.vue'),
    props: true
  },
  // Workflow Routes
  {
    path: '/workflows',
    name: 'WorkflowsView',
    component: () => import('@/views/WorkflowsView.vue')
  },
  {
    path: '/workflows/new',
    name: 'WorkflowCreate',
    component: () => import('@/views/WorkflowCreateView.vue')
  },
  {
    path: '/workflows/:id/edit',
    name: 'WorkflowEdit',
    component: () => import('@/views/WorkflowEditView.vue'),
    props: true
  },
  {
    path: '/workflows/:id',
    name: 'WorkflowDetail',
    component: () => import('@/views/WorkflowDetailView.vue'),
    props: true
  },
  // Entity Relationship Routes
  {
    path: '/entity-relationships',
    name: 'EntityRelationshipList',
    component: () => import('@/views/EntityRelationshipListView.vue') 
  },
  {
    path: '/entity-relationships/new',
    name: 'EntityRelationshipCreate',
    component: () => import('@/views/EntityRelationshipCreateView.vue')
  },
  {
    path: '/entity-relationships/:id/edit',
    name: 'EntityRelationshipEdit',
    component: () => import('@/views/EntityRelationshipEditView.vue'),
    props: true
  },
  // Fallback route for 404
  {
    path: '/:catchAll(.*)',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'), // Lazy load for non-critical view
  }
];

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL || '/'), 
  routes,
});

export default router;
