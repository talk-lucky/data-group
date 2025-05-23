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

export default apiClient; // Exporting the configured axios instance if direct use is preferred sometimes.
