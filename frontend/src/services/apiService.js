import axios from 'axios';

const apiClient = axios.create({
  baseURL: 'http://localhost:8080/api/v1', // Backend API base URL
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptors can be added here if needed (e.g., for error handling, auth tokens)

// --- Entity Endpoints ---

export const getEntities = () => {
  return apiClient.get('/entities/');
};

export const getEntityById = (entityId) => {
  return apiClient.get(`/entities/${entityId}`);
};

export const createEntity = (entityData) => {
  return apiClient.post('/entities/', entityData);
};

export const updateEntity = (entityId, entityData) => {
  return apiClient.put(`/entities/${entityId}`, entityData);
};

export const deleteEntity = (entityId) => {
  return apiClient.delete(`/entities/${entityId}`);
};

// --- Attribute Endpoints ---

export const getAttributesForEntity = (entityId) => {
  return apiClient.get(`/entities/${entityId}/attributes/`);
};

export const getAttributeById = (entityId, attributeId) => {
  return apiClient.get(`/entities/${entityId}/attributes/${attributeId}`);
};

export const createAttribute = (entityId, attributeData) => {
  return apiClient.post(`/entities/${entityId}/attributes/`, attributeData);
};

export const updateAttribute = (entityId, attributeId, attributeData) => {
  return apiClient.put(`/entities/${entityId}/attributes/${attributeId}`, attributeData);
};

export const deleteAttribute = (entityId, attributeId) => {
  return apiClient.delete(`/entities/${entityId}/attributes/${attributeId}`);
};

// --- Data Source Endpoints ---

export const getDataSources = () => {
  return apiClient.get('/datasources/');
};

export const getDataSourceById = (dataSourceId) => { // Renamed for clarity vs getEntityById
  return apiClient.get(`/datasources/${dataSourceId}`);
};

export const createDataSource = (dataSourceData) => {
  return apiClient.post('/datasources/', dataSourceData);
};

export const updateDataSource = (dataSourceId, dataSourceData) => {
  return apiClient.put(`/datasources/${dataSourceId}`, dataSourceData);
};

export const deleteDataSource = (dataSourceId) => {
  return apiClient.delete(`/datasources/${dataSourceId}`);
};

// --- Field Mapping Endpoints ---

export const getFieldMappingsForDataSource = (sourceId) => { // Renamed for clarity
  return apiClient.get(`/datasources/${sourceId}/mappings/`);
};

export const getFieldMappingById = (sourceId, mappingId) => { // Renamed
  return apiClient.get(`/datasources/${sourceId}/mappings/${mappingId}`);
};

export const createFieldMapping = (sourceId, fieldMappingData) => {
  return apiClient.post(`/datasources/${sourceId}/mappings/`, fieldMappingData);
};

export const updateFieldMapping = (sourceId, mappingId, fieldMappingData) => {
  return apiClient.put(`/datasources/${sourceId}/mappings/${mappingId}`, fieldMappingData);
};

export const deleteFieldMapping = (sourceId, mappingId) => {
  return apiClient.delete(`/datasources/${sourceId}/mappings/${mappingId}`);

  // Group Definition Endpoints
  getGroupDefinitions: () => apiClient.get('/groups'),
  getGroupDefinitionById: (groupId) => apiClient.get(`/groups/${groupId}`),
  createGroupDefinition: (groupData) => apiClient.post('/groups', groupData),
  updateGroupDefinition: (groupId, groupData) => apiClient.put(`/groups/${groupId}`, groupData),
  deleteGroupDefinition: (groupId) => apiClient.delete(`/groups/${groupId}`),
  calculateGroup: (groupId) => apiClient.post(`/groups/${groupId}/calculate`),
  getGroupResults: (groupId) => apiClient.get(`/groups/${groupId}/results`),

  // Workflow Definition Endpoints
  getWorkflows: () => apiClient.get('/workflows'),
  getWorkflowById: (workflowId) => apiClient.get(`/workflows/${workflowId}`),
  createWorkflow: (workflowData) => apiClient.post('/workflows', workflowData),
  updateWorkflow: (workflowId, workflowData) => apiClient.put(`/workflows/${workflowId}`, workflowData),
  deleteWorkflow: (workflowId) => apiClient.delete(`/workflows/${workflowId}`),

  // Action Template Endpoints
  getActionTemplates: () => apiClient.get('/actiontemplates'),
  getActionTemplateById: (templateId) => apiClient.get(`/actiontemplates/${templateId}`),
  createActionTemplate: (templateData) => apiClient.post('/actiontemplates', templateData),
  updateActionTemplate: (templateId, templateData) => apiClient.put(`/actiontemplates/${templateId}`, templateData),
  deleteActionTemplate: (templateId) => apiClient.delete(`/actiontemplates/${templateId}`),
};


export default apiClient; // Exporting the configured axios instance if direct use is preferred sometimes.
