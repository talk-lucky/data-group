<template>
  <v-container>
    <v-card class="mx-auto" max-width="800" outlined>
      <v-card-title class="text-h6 pa-4">
        {{ isEditing ? 'Edit Entity Relationship' : 'Create Entity Relationship' }}
      </v-card-title>
      <v-divider></v-divider>

      <v-card-text>
        <v-form ref="form" v-model="validForm">
          <v-text-field
            v-model="formData.name"
            :rules="[rules.required, rules.nameCounter]"
            label="Relationship Name"
            placeholder="e.g., UserProfile"
            required
            class="mb-3"
            variant="outlined"
            density="compact"
          ></v-text-field>

          <v-textarea
            v-model="formData.description"
            label="Description"
            placeholder="A brief description of the relationship"
            rows="3"
            class="mb-3"
            variant="outlined"
            density="compact"
          ></v-textarea>

          <v-row>
            <v-col cols="12" md="6">
              <v-select
                v-model="formData.source_entity_id"
                :items="entityItems"
                item-title="name"
                item-value="id"
                label="Source Entity"
                :rules="[rules.required]"
                required
                @update:modelValue="onSourceEntityChange"
                class="mb-3"
                variant="outlined"
                density="compact"
              ></v-select>
            </v-col>
            <v-col cols="12" md="6">
              <v-select
                v-model="formData.source_attribute_id"
                :items="sourceAttributeItems"
                item-title="name"
                item-value="id"
                label="Source Attribute (e.g., Foreign Key)"
                :rules="[rules.required]"
                :disabled="!formData.source_entity_id"
                required
                class="mb-3"
                variant="outlined"
                density="compact"
              ></v-select>
            </v-col>
          </v-row>

          <v-row>
            <v-col cols="12" md="6">
              <v-select
                v-model="formData.target_entity_id"
                :items="entityItems"
                item-title="name"
                item-value="id"
                label="Target Entity"
                :rules="[rules.required]"
                required
                @update:modelValue="onTargetEntityChange"
                class="mb-3"
                variant="outlined"
                density="compact"
              ></v-select>
            </v-col>
            <v-col cols="12" md="6">
              <v-select
                v-model="formData.target_attribute_id"
                :items="targetAttributeItems"
                item-title="name"
                item-value="id"
                label="Target Attribute (e.g., Primary Key)"
                :rules="[rules.required]"
                :disabled="!formData.target_entity_id"
                required
                class="mb-3"
                variant="outlined"
                density="compact"
              ></v-select>
            </v-col>
          </v-row>

          <v-select
            v-model="formData.relationship_type"
            :items="relationshipTypeItems"
            label="Relationship Type"
            :rules="[rules.required]"
            required
            class="mb-3"
            variant="outlined"
            density="compact"
          ></v-select>

          <v-alert v-if="errorStore.error" type="error" dense class="mb-4">
            {{ errorStore.error }}
          </v-alert>

        </v-form>
      </v-card-text>

      <v-divider></v-divider>
      <v-card-actions class="pa-4">
        <v-spacer></v-spacer>
        <v-btn color="grey darken-1" text @click="cancel">Cancel</v-btn>
        <v-btn
          color="primary"
          @click="submitForm"
          :disabled="!validForm || relationshipStore.loading"
          :loading="relationshipStore.loading"
        >
          {{ isEditing ? 'Save Changes' : 'Create Relationship' }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-container>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useEntityRelationshipStore } from '@/stores/entityRelationshipStore';
import { useEntityStore } from '@/stores/entityStore';
import { useAttributeStore } from '@/stores/attributeStore';
import { useErrorStore } from '@/stores/errorStore'; // Assuming a global error store

const router = useRouter();
const route = useRoute();
const relationshipStore = useEntityRelationshipStore();
const entityStore = useEntityStore();
const attributeStore = useAttributeStore();
const errorStore = useErrorStore(); // For displaying API errors

const form = ref(null); // Template ref for VForm
const validForm = ref(false);
const isEditing = ref(false);
const relationshipId = ref(null);

const formData = ref({
  name: '',
  description: '',
  source_entity_id: null,
  source_attribute_id: null,
  target_entity_id: null,
  target_attribute_id: null,
  relationship_type: null,
});

const relationshipTypeItems = [
  { title: 'One to One', value: 'ONE_TO_ONE' },
  { title: 'One to Many', value: 'ONE_TO_MANY' },
  { title: 'Many to One', value: 'MANY_TO_ONE' },
];

const rules = {
  required: value => !!value || 'This field is required.',
  nameCounter: value => (value && value.length <= 255) || 'Name must be less than 255 characters.',
};

// --- Entity and Attribute Population ---
const entityItems = computed(() => entityStore.entities.map(e => ({ id: e.id, name: e.name })));
const sourceAttributeItems = ref([]);
const targetAttributeItems = ref([]);

async function onSourceEntityChange(entityId) {
  formData.value.source_attribute_id = null; // Reset attribute on entity change
  if (entityId) {
    await attributeStore.fetchAttributesForEntity(entityId);
    sourceAttributeItems.value = attributeStore.attributes.map(a => ({ id: a.id, name: a.name, entity_id: a.entity_id }));
  } else {
    sourceAttributeItems.value = [];
  }
}

async function onTargetEntityChange(entityId) {
  formData.value.target_attribute_id = null; // Reset attribute on entity change
  if (entityId) {
    await attributeStore.fetchAttributesForEntity(entityId); // This might overwrite if attributes are stored globally in attributeStore without context
    // A better approach for attributeStore would be to store attributes per entity or provide a method that doesn't overwrite.
    // For now, assuming fetchAttributesForEntity correctly provides attributes for the given entityId and they can be mapped.
    targetAttributeItems.value = attributeStore.getAttributesByEntityId(entityId).map(a => ({ id: a.id, name: a.name, entity_id: a.entity_id }));
  } else {
    targetAttributeItems.value = [];
  }
}


// --- Lifecycle and Edit Mode ---
onMounted(async () => {
  await entityStore.fetchEntities(); // Fetch all entities for dropdowns

  relationshipId.value = route.params.id;
  if (relationshipId.value) {
    isEditing.value = true;
    try {
      // Use the store's method to fetch and set currentRelationship
      await relationshipStore.fetchEntityRelationship(relationshipId.value);
      if (relationshipStore.currentRelationship) {
        // Populate formData from the store's currentRelationship
        const current = relationshipStore.currentRelationship;
        formData.value = { ...current }; 
        // Pre-load attributes for selected entities
        if (current.source_entity_id) {
          await onSourceEntityChange(current.source_entity_id);
          formData.value.source_attribute_id = current.source_attribute_id; // Ensure it's re-set after items load
        }
        if (current.target_entity_id) {
          await onTargetEntityChange(current.target_entity_id);
           formData.value.target_attribute_id = current.target_attribute_id; // Ensure it's re-set after items load
        }
      } else {
         errorStore.setError(`Relationship with ID ${relationshipId.value} not found.`);
         router.push({ name: 'EntityRelationshipList' }); // Redirect if not found
      }
    } catch (error) {
        console.error("Error fetching relationship for editing:", error);
        // Error already set by store, or use errorStore.setError
        router.push({ name: 'EntityRelationshipList' });
    }
  } else {
    // New form, reset currentRelationship in store if any
    relationshipStore.setCurrentRelationship(null);
  }
});

// Watch for changes in the store's currentRelationship if it's being externally set
// (e.g., after a fetch) and update formData. This is mostly for edit mode.
watch(() => relationshipStore.currentRelationship, (newVal) => {
  if (newVal && isEditing.value && newVal.id === relationshipId.value) {
    formData.value = { ...newVal };
    // Consider re-fetching attributes if necessary, though onMounted should handle initial load.
  } else if (!newVal && !isEditing.value) {
    // If currentRelationship is cleared and we are in "create" mode, reset form
    Object.keys(formData.value).forEach(key => formData.value[key] = null);
    formData.value.name = '';
    formData.value.description = '';
  }
});


// --- Form Actions ---
async function submitForm() {
  errorStore.clearError(); // Clear previous errors
  const { valid } = await form.value.validate();
  if (!valid) {
    validForm.value = false; // Ensure validForm state is updated
    return;
  }
  validForm.value = true;

  const payload = { ...formData.value };

  try {
    if (isEditing.value) {
      await relationshipStore.updateEntityRelationship(relationshipId.value, payload);
    } else {
      await relationshipStore.createEntityRelationship(payload);
    }
    router.push({ name: 'EntityRelationshipList' }); // Navigate back to list on success
  } catch (error) {
    // Error should be set in store, and alert will display it.
    console.error("Form submission error:", error);
    // errorStore.setError(error.message || "An unexpected error occurred."); // Already handled by store
  }
}

function cancel() {
  errorStore.clearError();
  router.push({ name: 'EntityRelationshipList' });
}
</script>

<style scoped>
.v-card {
  border: 1px solid #e0e0e0;
}
</style>
