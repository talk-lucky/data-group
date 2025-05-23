import { defineStore } from 'pinia';
import {
  getDataSources,
  getDataSourceById,
  createDataSource,
  updateDataSource,
  deleteDataSource,
} from '@/services/apiService';

export const useDataSourceStore = defineStore('dataSource', {
  state: () => ({
    dataSources: [],
    currentDataSource: null,
    loading: false,
    error: null,
  }),
  getters: {
    allDataSources: (state) => state.dataSources,
    isLoading: (state) => state.loading,
    dataSourceOptions: (state) => {
      return state.dataSources.map(ds => ({
        title: `${ds.name} (${ds.type})`,
        value: ds.id,
      }));
    },
  },
  actions: {
    async fetchDataSources() {
      this.loading = true;
      this.error = null;
      try {
        const response = await getDataSources();
        this.dataSources = response.data;
      } catch (error) {
        this.error = 'Failed to fetch data sources: ' + (error.response?.data?.error || error.message);
        console.error(this.error, error);
      } finally {
        this.loading = false;
      }
    },
    async fetchDataSourceById(id) {
      this.loading = true;
      this.error = null;
      try {
        const response = await getDataSourceById(id);
        this.currentDataSource = response.data;
        return response.data;
      } catch (error) {
        this.error = `Failed to fetch data source ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        this.currentDataSource = null;
      } finally {
        this.loading = false;
      }
    },
    async addDataSource(dataSourceData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await createDataSource(dataSourceData);
        this.dataSources.push(response.data);
        return response.data;
      } catch (error) {
        this.error = 'Failed to create data source: ' + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async editDataSource(id, dataSourceData) {
      this.loading = true;
      this.error = null;
      try {
        const response = await updateDataSource(id, dataSourceData);
        const index = this.dataSources.findIndex(ds => ds.id === id);
        if (index !== -1) {
          this.dataSources[index] = response.data;
        }
        if (this.currentDataSource && this.currentDataSource.id === id) {
          this.currentDataSource = response.data;
        }
        return response.data;
      } catch (error) {
        this.error = `Failed to update data source ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    async removeDataSource(id) {
      this.loading = true;
      this.error = null;
      try {
        await deleteDataSource(id);
        this.dataSources = this.dataSources.filter(ds => ds.id !== id);
        if (this.currentDataSource && this.currentDataSource.id === id) {
          this.currentDataSource = null;
        }
      } catch (error) {
        this.error = `Failed to delete data source ${id}: ` + (error.response?.data?.error || error.message);
        console.error(this.error, error);
        throw error;
      } finally {
        this.loading = false;
      }
    },
    clearCurrentDataSource() {
      this.currentDataSource = null;
    },
  },
});
