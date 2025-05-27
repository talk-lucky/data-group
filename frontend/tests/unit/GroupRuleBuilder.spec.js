import { mount } from '@vue/test-utils';
import { createTestingPinia } from '@pinia/testing';
import { v4 as uuidv4 } from 'uuid'; // To match how GroupRuleBuilder assigns IDs

// Components
import GroupRuleBuilder from '@/components/GroupRuleBuilder.vue';
import RuleNodeRenderer from '@/components/RuleNodeRenderer.vue'; // Will be stubbed generally

// Stores
import { useAttributeStore } from '@/store/attributeStore';
import { useEntityStore } from '@/store/entityStore';
import { useEntityRelationshipStore } from '@/stores/entityRelationshipStore'; // Corrected path

// Vuetify setup
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';

const vuetify = createVuetify({
  components,
  directives,
});

// Global stubs for Vuetify components used deeply if not using global.plugins
const globalStubs = {
  'v-container': true,
  'v-card': true,
  'v-card-title': true,
  'v-card-subtitle': true,
  'v-card-text': true, // if used by GroupRuleBuilder directly
  'v-divider': true,
  'v-alert': true,
  'v-textarea': true,
  // RuleNodeRenderer will be stubbed where its internal logic is not under test
};


describe('GroupRuleBuilder.vue', () => {
  let wrapper;
  let entityStore;
  let attributeStore;
  let relationshipStore;

  // Mock data
  const mockEntityId = 'user_entity_id';
  const mockAttributes = [
    { id: 'attr1', name: 'Age', entity_id: mockEntityId, data_type: 'integer', entity_name: 'User' },
    { id: 'attr2', name: 'Country', entity_id: mockEntityId, data_type: 'string', entity_name: 'User' },
  ];
  const mockRelatedEntityId = 'order_entity_id';
  const mockRelatedAttributes = [
    { id: 'attr3', name: 'Amount', entity_id: mockRelatedEntityId, data_type: 'numeric', entity_name: 'Order' },
  ];
  const mockRelationships = [
    { id: 'rel1', name: 'UserOrders', source_entity_id: mockEntityId, target_entity_id: mockRelatedEntityId, source_attribute_id: 'attr_user_pk', target_attribute_id: 'attr_order_fk_user' },
  ];

  const setupWrapper = (props = {}, initialPiniaState = {}) => {
    const pinia = createTestingPinia({
      initialState: {
        entity: { entities: [{ id: mockEntityId, name: 'User' }, {id: mockRelatedEntityId, name: 'Order'}], ...initialPiniaState.entity },
        attribute: { attributes: [...mockAttributes, ...mockRelatedAttributes], ...initialPiniaState.attribute },
        entityRelationship: { relationships: mockRelationships, ...initialPiniaState.entityRelationship },
      },
      stubActions: false, // We might want to test actions are called
    });

    entityStore = useEntityStore(pinia);
    attributeStore = useAttributeStore(pinia);
    relationshipStore = useEntityRelationshipStore(pinia);
    
    // Mock fetch actions that might be called on mount
    entityStore.fetchEntities = vi.fn().mockResolvedValue(entityStore.entities);
    attributeStore.fetchAttributesForEntity = vi.fn().mockImplementation((entityId) => {
        if (entityId === mockEntityId) attributeStore.attributes = mockAttributes;
        else if (entityId === mockRelatedEntityId) attributeStore.attributes = mockRelatedAttributes;
        else attributeStore.attributes = [];
        return Promise.resolve();
    });
    relationshipStore.fetchEntityRelationships = vi.fn().mockResolvedValue(relationshipStore.relationships);


    wrapper = mount(GroupRuleBuilder, {
      props: {
        entityId: mockEntityId,
        modelValue: '', // Start with empty rules usually
        ...props,
      },
      global: {
        plugins: [pinia, vuetify],
        stubs: {
           // Stubbing RuleNodeRenderer to prevent deep rendering issues in unit tests for GroupRuleBuilder itself.
           // For tests focusing on RuleNodeRenderer's interaction, we might not stub it or use shallowMount.
          RuleNodeRenderer: true, // globally stub it for these tests
        }
      },
    });
  };

  beforeEach(() => {
    // Default setup for most tests
    setupWrapper();
  });

  test('initializes with a default root group', () => {
    expect(wrapper.vm.localRules.type).toBe('group');
    expect(wrapper.vm.localRules.logical_operator).toBe('AND');
    expect(wrapper.vm.localRules.rules).toEqual([]);
    expect(wrapper.vm.localRules.entity_id).toBe(mockEntityId);
  });

  test('adds a condition to the root group', async () => {
    // Simulate event from RuleNodeRenderer (since it's stubbed)
    // We need to call the handler directly
    wrapper.vm.handleAddCondition([]); // Path to root
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.localRules.rules.length).toBe(1);
    const newCondition = wrapper.vm.localRules.rules[0];
    expect(newCondition.type).toBe('condition');
    expect(newCondition.entityId).toBe(mockEntityId); // Should inherit entityId from root group
  });

  test('adds a nested group to the root group', async () => {
    wrapper.vm.handleAddGroup([]); // Path to root
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.localRules.rules.length).toBe(1);
    const newGroup = wrapper.vm.localRules.rules[0];
    expect(newGroup.type).toBe('group');
    expect(newGroup.entity_id).toBe(mockEntityId); // Should inherit entityId
    expect(newGroup.rules).toEqual([]);
  });
  
  test('adds a relationship group to the root group', async () => {
    wrapper.vm.handleAddRelationshipGroup([]); // Path to root
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.localRules.rules.length).toBe(1);
    const newRelGroup = wrapper.vm.localRules.rules[0];
    expect(newRelGroup.type).toBe('relationship_group');
    expect(newRelGroup.relationshipId).toBeNull();
    expect(newRelGroup.relatedEntityRules.type).toBe('group');
    // The entity_id of relatedEntityRules is initially null, will be set when a relationship is selected.
    expect(newRelGroup.relatedEntityRules.entity_id).toBeNull();
  });

  test('removes a node', async () => {
    wrapper.vm.handleAddCondition([]);
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.localRules.rules.length).toBe(1);
    
    wrapper.vm.handleRemoveNode([0]); // Path to the first rule in root
    await wrapper.vm.$nextTick();
    expect(wrapper.vm.localRules.rules.length).toBe(0);
  });

  test('updates a node', async () => {
    wrapper.vm.handleAddCondition([]);
    await wrapper.vm.$nextTick();
    
    const updates = { attributeId: 'attr1', operator: '=', value: 'test' };
    wrapper.vm.handleUpdateNode([0], updates); // Path to the first rule in root
    await wrapper.vm.$nextTick();
    
    const updatedNode = wrapper.vm.localRules.rules[0];
    expect(updatedNode.attributeId).toBe('attr1');
    expect(updatedNode.operator).toBe('=');
    expect(updatedNode.value).toBe('test');
  });

  describe('JSON Generation and Parsing', () => {
    test('generates correct JSON for nested and relationship groups', async () => {
      // Build a complex rule structure directly on localRules for testing currentJsonOutput
      const rootGroupId = wrapper.vm.localRules.id;
      const conditionId1 = uuidv4();
      const relationshipGroupId = uuidv4();
      const relatedGroupId = uuidv4(); // ID for the group inside relatedEntityRules
      const relatedConditionId = uuidv4();

      wrapper.vm.localRules = {
        id: rootGroupId,
        type: 'group',
        entity_id: mockEntityId,
        logical_operator: 'AND',
        rules: [
          { id: conditionId1, type: 'condition', attributeId: 'attr1', attributeName: 'Age', entityId: mockEntityId, operator: '>=', value: 30, valueType: 'integer' },
          { 
            id: relationshipGroupId,
            type: 'relationship_group',
            relationshipId: 'rel1',
            relatedEntityRules: {
              id: relatedGroupId,
              type: 'group',
              entity_id: mockRelatedEntityId, // This should be set when relationship is selected
              logical_operator: 'OR',
              rules: [
                { id: relatedConditionId, type: 'condition', attributeId: 'attr3', attributeName: 'Amount', entityId: mockRelatedEntityId, operator: '>', value: 100, valueType: 'numeric' }
              ]
            }
          }
        ]
      };
      await wrapper.vm.$nextTick();

      const expectedJson = {
        type: "group",
        entity_id: mockEntityId,
        logical_operator: "AND",
        rules: [
          { type: "condition", attribute_id: "attr1", attribute_name: "Age", entity_id: mockEntityId, operator: ">=", value: 30, value_type: "integer" },
          {
            type: "relationship_group",
            relationship_id: "rel1",
            related_entity_rules: {
              type: "group",
              entity_id: mockRelatedEntityId,
              logical_operator: "OR",
              rules: [
                { type: "condition", attribute_id: "attr3", attribute_name: "Amount", entity_id: mockRelatedEntityId, operator: ">", value: 100, value_type: "numeric" }
              ]
            }
          }
        ]
      };
      // Need to parse and compare due to potential field order differences and to avoid issues with reactive objects
      expect(JSON.parse(wrapper.vm.currentJsonOutput)).toEqual(expectedJson);
    });

    test('parses complex JSON into internal structure', async () => {
        const complexJsonString = JSON.stringify({
            type: "group",
            entity_id: mockEntityId,
            logical_operator: "AND",
            rules: [
              { type: "condition", attribute_id: "attr2", attribute_name: "Country", entity_id: mockEntityId, operator: "=", value: "USA", value_type: "string" },
              {
                type: "relationship_group",
                relationship_id: "rel1",
                related_entity_rules: {
                  type: "group",
                  // entity_id for related_entity_rules is determined by relationship target, not from JSON here
                  logical_operator: "AND",
                  rules: [
                    { type: "condition", attribute_id: "attr3", attribute_name: "Amount", entity_id: mockRelatedEntityId, operator: "<", value: 50, value_type: "numeric" }
                  ]
                }
              }
            ]
        });

        await wrapper.setProps({ modelValue: complexJsonString });
        await wrapper.vm.$nextTick(); // Allow watcher for modelValue to trigger parseAndSetRules

        const rules = wrapper.vm.localRules;
        expect(rules.type).toBe('group');
        expect(rules.entity_id).toBe(mockEntityId);
        expect(rules.rules.length).toBe(2);

        const conditionNode = rules.rules[0];
        expect(conditionNode.type).toBe('condition');
        expect(conditionNode.attributeId).toBe('attr2');
        expect(conditionNode.entityId).toBe(mockEntityId);

        const relGroupNode = rules.rules[1];
        expect(relGroupNode.type).toBe('relationship_group');
        expect(relGroupNode.relationshipId).toBe('rel1');
        expect(relGroupNode.relatedEntityRules.type).toBe('group');
        // After parsing, the entity_id of relatedEntityRules might be null initially,
        // and would be set by RuleNodeRenderer when relationship 'rel1' is processed.
        // Or parseJsonNode needs to be smarter with context. For now, let's check structure.
        expect(relGroupNode.relatedEntityRules.rules.length).toBe(1);
        const relatedCondition = relGroupNode.relatedEntityRules.rules[0];
        expect(relatedCondition.attributeId).toBe('attr3');
         // Its entityId should be mockRelatedEntityId if parseJsonNode correctly passes context or if it's in JSON.
        expect(relatedCondition.entityId).toBe(mockRelatedEntityId); 
    });
  });
  
  // More tests would be needed for RuleNodeRenderer.vue itself:
  // - Correctly displaying different node types (condition, group, relationship_group)
  // - Populating and handling changes in relationship dropdown
  // - Passing correct currentEntityId and attributeStore to nested RuleNodeRenderer for related_entity_rules
  // - Filtering availableAttributes based on currentEntityId
  // - Emitting updates correctly for relationshipId changes
});

[end of frontend/tests/unit/GroupRuleBuilder.spec.js]
