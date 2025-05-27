import { mount } from '@vue/test-utils';
import { createTestingPinia } from '@pinia/testing';
import { vi } from 'vitest'; // Vitest's mocking utilities

// Components
import EntityRelationshipList from '@/components/EntityRelationshipList.vue';
import EntityRelationshipForm from '@/components/EntityRelationshipForm.vue';

// Stores
import { useEntityRelationshipStore } from '@/stores/entityRelationshipStore';
import { useEntityStore } from '@/store/entityStore';
import { useAttributeStore } from '@/store/attributeStore';
import { useErrorStore } from '@/stores/errorStore';

// Vuetify setup
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';

// Router mock
const mockRouter = {
  push: vi.fn(),
};

const vuetify = createVuetify({
  components,
  directives,
});

// Global stubs for Vuetify components
const globalStubs = {
  'v-container': true,
  'v-card': true,
  'v-card-title': true,
  'v-card-text': true,
  'v-card-actions': true,
  'v-btn': true,
  'v-icon': true,
  'v-divider': true,
  'v-alert': true,
  'v-progress-linear': true,
  'v-data-table-server': { // More specific stub for v-data-table-server
    template: '<div><slot name="item.actions" :item="{raw:{id:\'test-id\'}}"></slot><slot name="no-data"></slot><slot name="loading"></slot></div>',
    props: ['headers', 'items', 'itemsLength', 'loading', 'page', 'itemsPerPage'],
    emits: ['update:options'],
  },
  'v-dialog': true,
  'v-spacer': true,
  'v-form': {
    template: '<div><slot></slot></div>',
    methods: {
        validate: () => ({ valid: true }) // Mock validate to always be true for simplicity
    }
  },
  'v-text-field': true,
  'v-textarea': true,
  'v-select': true,
  'v-row': true,
  'v-col': true,
  'v-skeleton-loader': true,

};

describe('EntityRelationshipList.vue', () => {
  let wrapper;
  let relationshipStore;
  let entityStore;

  const mockRelationships = [
    { id: 'rel1', name: 'User Orders', source_entity_id: 'user1', target_entity_id: 'order1', relationship_type: 'ONE_TO_MANY' },
    { id: 'rel2', name: 'Product Category', source_entity_id: 'prod1', target_entity_id: 'cat1', relationship_type: 'MANY_TO_ONE' },
  ];
  const mockEntities = [
    { id: 'user1', name: 'User' },
    { id: 'order1', name: 'Order' },
    { id: 'prod1', name: 'Product' },
    { id: 'cat1', name: 'Category' },
  ];

  const setupListWrapper = (initialPiniaState = {}) => {
    const pinia = createTestingPinia({
      initialState: {
        entityRelationship: {
          relationships: mockRelationships,
          pagination: { page: 1, itemsPerPage: 10, totalItems: mockRelationships.length },
          loading: false,
          error: null,
          ...initialPiniaState.entityRelationship,
        },
        entity: {
          entities: mockEntities,
          ...initialPiniaState.entity,
        },
        error: { error: null, ...initialPiniaState.error }
      },
      stubActions: false,
    });

    relationshipStore = useEntityRelationshipStore(pinia);
    entityStore = useEntityStore(pinia);
    useErrorStore(pinia); // Initialize error store

    // Mock fetch actions
    relationshipStore.fetchEntityRelationships = vi.fn().mockResolvedValue();
    entityStore.fetchEntities = vi.fn().mockResolvedValue();
    relationshipStore.deleteEntityRelationship = vi.fn().mockResolvedValue();


    wrapper = mount(EntityRelationshipList, {
      global: {
        plugins: [pinia, vuetify],
        stubs: globalStubs,
        mocks: {
          $router: mockRouter,
        },
      },
    });
  };

  beforeEach(() => {
    vi.clearAllMocks(); // Clear mocks before each test
    setupListWrapper();
  });

  test('renders the component and displays relationships', async () => {
    expect(wrapper.findComponent(components.VDataTableServer).exists()).toBe(true);
    // formattedRelationships resolves entity names
    const formatted = wrapper.vm.formattedRelationships;
    expect(formatted.length).toBe(mockRelationships.length);
    expect(formatted[0].sourceEntityName).toBe('User');
    expect(formatted[0].targetEntityName).toBe('Order');
  });

  test('calls fetchEntityRelationships on mount via @update:options', async () => {
     // v-data-table-server emits @update:options on mount
    expect(relationshipStore.fetchEntityRelationships).toHaveBeenCalled();
  });

  test('navigates to create form when "Create Relationship" is clicked', async () => {
    // Find the button - this might need a more specific selector if there are multiple buttons
    const createButton = wrapper.findAllComponents(components.VBtn).find(btn => btn.text().includes('Create Relationship'));
    await createButton.trigger('click');
    expect(mockRouter.push).toHaveBeenCalledWith({ name: 'EntityRelationshipCreate' });
  });

  test('navigates to edit form when edit icon is clicked', async () => {
    // This test is a bit tricky due to how v-data-table-server slots work with stubs.
    // We assume the slot provides the item, and the click handler works.
    // If RuleNodeRenderer was fully rendered, we could find the icon and click it.
    // For now, we can call the method directly as if the icon was clicked.
    wrapper.vm.editRelationship('rel1');
    expect(mockRouter.push).toHaveBeenCalledWith({ name: 'EntityRelationshipEdit', params: { id: 'rel1' } });
  });
  
  test('opens delete confirmation and calls delete action', async () => {
    wrapper.vm.confirmDelete('rel1');
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.deleteDialog).toBe(true);
    expect(wrapper.vm.itemToDelete).toBe('rel1');

    await wrapper.vm.deleteConfirmed();
    expect(relationshipStore.deleteEntityRelationship).toHaveBeenCalledWith('rel1');
    expect(wrapper.vm.deleteDialog).toBe(false);
  });
});


describe('EntityRelationshipForm.vue', () => {
  let wrapper;
  let relationshipStore;
  let entityStore;
  let attributeStore;
  let errorStore;

  const mockEntities = [
    { id: 'user_ent', name: 'User' },
    { id: 'order_ent', name: 'Order' },
  ];
  const mockUserAttributes = [
    { id: 'user_pk', name: 'ID', entity_id: 'user_ent', data_type: 'uuid' },
    { id: 'user_email', name: 'Email', entity_id: 'user_ent', data_type: 'string' },
  ];
  const mockOrderAttributes = [
    { id: 'order_pk', name: 'ID', entity_id: 'order_ent', data_type: 'uuid' },
    { id: 'order_user_fk', name: 'UserID', entity_id: 'order_ent', data_type: 'uuid' },
  ];

  const setupFormWrapper = (props = {}, initialPiniaState = {}, routeParams = {}) => {
    const pinia = createTestingPinia({
      initialState: {
        entityRelationship: { currentRelationship: null, loading: false, error: null, ...initialPiniaState.entityRelationship },
        entity: { entities: mockEntities, ...initialPiniaState.entity },
        attribute: { attributes: [], ...initialPiniaState.attribute }, // Attributes loaded dynamically
        error: { error: null, ...initialPiniaState.error },
      },
      stubActions: false,
    });

    relationshipStore = useEntityRelationshipStore(pinia);
    entityStore = useEntityStore(pinia);
    attributeStore = useAttributeStore(pinia);
    errorStore = useErrorStore(pinia);
    
    // Mock fetch actions
    entityStore.fetchEntities = vi.fn().mockResolvedValue();
    attributeStore.fetchAttributesForEntity = vi.fn().mockImplementation(async (entityId) => {
      if (entityId === 'user_ent') attributeStore.attributes = mockUserAttributes;
      else if (entityId === 'order_ent') attributeStore.attributes = mockOrderAttributes;
      else attributeStore.attributes = [];
      return Promise.resolve();
    });
    attributeStore.getAttributesByEntityId = vi.fn().mockImplementation((entityId) => {
        if (entityId === 'user_ent') return mockUserAttributes;
        if (entityId === 'order_ent') return mockOrderAttributes;
        return [];
    });


    relationshipStore.createEntityRelationship = vi.fn().mockResolvedValue({ id: 'new_rel_id' });
    relationshipStore.updateEntityRelationship = vi.fn().mockResolvedValue({});
    relationshipStore.fetchEntityRelationship = vi.fn().mockImplementation(async (id) => {
        if (id === 'edit_rel_id') {
            const rel = { 
                id: 'edit_rel_id', name: 'Edit Name', description: 'Edit Desc', 
                source_entity_id: 'user_ent', source_attribute_id: 'user_pk',
                target_entity_id: 'order_ent', target_attribute_id: 'order_user_fk',
                relationship_type: 'ONE_TO_MANY'
            };
            relationshipStore.currentRelationship = rel; // Set in store
            return Promise.resolve(rel);
        }
        return Promise.reject(new Error('Not found'));
    });


    wrapper = mount(EntityRelationshipForm, {
      props: { ...props },
      global: {
        plugins: [pinia, vuetify],
        stubs: globalStubs,
        mocks: {
          $router: mockRouter,
          $route: { params: routeParams, path: routeParams.id ? `/edit/${routeParams.id}` : '/create' }, // Mock route
        },
      },
    });
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders in create mode', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick(); // Allow onMounted to run
    expect(wrapper.find('.v-card-title').text()).toContain('Create Entity Relationship');
    expect(wrapper.vm.isEditing).toBe(false);
  });

  test('renders in edit mode and loads data', async () => {
    setupFormWrapper({}, {}, { id: 'edit_rel_id' });
    await wrapper.vm.$nextTick(); // onMounted
    await wrapper.vm.$nextTick(); // watchers for currentRelationship
    await wrapper.vm.$nextTick(); // after attribute fetches

    expect(wrapper.find('.v-card-title').text()).toContain('Edit Entity Relationship');
    expect(wrapper.vm.isEditing).toBe(true);
    expect(relationshipStore.fetchEntityRelationship).toHaveBeenCalledWith('edit_rel_id');
    
    // Check if form data is populated
    expect(wrapper.vm.formData.name).toBe('Edit Name');
    expect(wrapper.vm.formData.source_entity_id).toBe('user_ent');
    expect(wrapper.vm.formData.source_attribute_id).toBe('user_pk'); // This depends on attributes being loaded
  });

  test('populates entity dropdowns on mount', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick();
    expect(entityStore.fetchEntities).toHaveBeenCalled();
    // Assuming v-select is rendered, its items would be based on entityItems
  });

  test('dynamically loads attributes when source entity changes', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick(); // Mount
    
    // Simulate selecting a source entity
    // Directly call the method as interacting with v-select stub is complex
    await wrapper.vm.onSourceEntityChange('user_ent');
    await wrapper.vm.$nextTick();
    
    expect(attributeStore.fetchAttributesForEntity).toHaveBeenCalledWith('user_ent');
    expect(wrapper.vm.sourceAttributeItems.length).toBe(mockUserAttributes.length);
  });

  test('submits form for creating a new relationship', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick();

    wrapper.vm.formData = {
        name: 'New Test Rel', description: 'Desc',
        source_entity_id: 'user_ent', source_attribute_id: 'user_pk',
        target_entity_id: 'order_ent', target_attribute_id: 'order_user_fk',
        relationship_type: 'ONE_TO_MANY',
    };
    await wrapper.vm.submitForm();
    expect(relationshipStore.createEntityRelationship).toHaveBeenCalledWith(wrapper.vm.formData);
    expect(mockRouter.push).toHaveBeenCalledWith({ name: 'EntityRelationshipList' });
  });

  test('submits form for updating an existing relationship', async () => {
    setupFormWrapper({}, {}, { id: 'edit_rel_id' });
    await wrapper.vm.$nextTick(); 
    await wrapper.vm.$nextTick(); 
    await wrapper.vm.$nextTick(); 

    wrapper.vm.formData.name = "Super Updated Name"; // Change some data
    await wrapper.vm.submitForm();
    
    expect(relationshipStore.updateEntityRelationship).toHaveBeenCalledWith('edit_rel_id', wrapper.vm.formData);
    expect(mockRouter.push).toHaveBeenCalledWith({ name: 'EntityRelationshipList' });
  });
  
  test('validation rules prevent submission', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick();
    
    // Mock form.value.validate to return false for this test
    wrapper.vm.$refs.form.validate = vi.fn().mockResolvedValue({ valid: false });

    await wrapper.vm.submitForm();
    expect(relationshipStore.createEntityRelationship).not.toHaveBeenCalled();
    expect(mockRouter.push).not.toHaveBeenCalled(); // Should not navigate
  });

  test('cancel button navigates back to list', async () => {
    setupFormWrapper();
    await wrapper.vm.$nextTick();
    wrapper.vm.cancel();
    expect(mockRouter.push).toHaveBeenCalledWith({ name: 'EntityRelationshipList' });
  });
});

[end of frontend/tests/unit/EntityRelationship.spec.js]
