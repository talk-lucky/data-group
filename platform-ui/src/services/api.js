// src/services/api.js
import axios from 'axios'
import router from '../router' // To handle navigation on auth errors
// import { useAppStore } from '../store/appStore' // If you need to access store for token or loading state

const apiClient = axios.create({
  baseURL: process.env.VUE_APP_API_BASE_URL || '/api/v1', // Use environment variable or default
  headers: {
    'Content-Type': 'application/json',
    // You can add other default headers here, like Authorization if token is available
  }
})

// Request Interceptor (optional)
// apiClient.interceptors.request.use(
//   config => {
//     const appStore = useAppStore()
//     const token = appStore.user?.token // Example: get token from Pinia store
//     if (token) {
//       config.headers.Authorization = `Bearer ${token}`;
//     }
//     appStore.setLoading(true) // Example: set global loading state
//     return config;
//   },
//   error => {
//     const appStore = useAppStore()
//     appStore.setLoading(false)
//     return Promise.reject(error);
//   }
// );

// Response Interceptor
apiClient.interceptors.response.use(
  response => {
    // const appStore = useAppStore()
    // appStore.setLoading(false) // Example: set global loading state
    return response
  },
  error => {
    // const appStore = useAppStore()
    // appStore.setLoading(false) // Example: set global loading state

    if (error.response) {
      // The request was made and the server responded with a status code
      // that falls out of the range of 2xx
      console.error('API Error Response:', error.response.data);
      console.error('Status:', error.response.status);
      console.error('Headers:', error.response.headers);

      if (error.response.status === 401) {
        // Handle unauthorized errors (e.g., redirect to login)
        // appStore.logout() // Example: clear user session
        if (router.currentRoute.value.name !== 'Login') { // Avoid redirect loop if already on login
          // router.push({ name: 'Login' });
          console.warn('Unauthorized (401). Would redirect to login.')
        }
      } else if (error.response.status === 403) {
        // Handle forbidden errors
        console.warn('Forbidden (403). Access denied.')
        // router.push({ name: 'Forbidden' }); // Or show a generic error
      } else {
        // Handle other errors (e.g., show a generic error message)
        // You might want to use a notification system here
        // alert(`Error: ${error.response.data.message || error.message}`);
      }
    } else if (error.request) {
      // The request was made but no response was received
      console.error('API Error Request (no response):', error.request);
      // alert('Network Error: Please check your connection.');
    } else {
      // Something happened in setting up the request that triggered an Error
      console.error('API Error Message:', error.message);
      // alert(`Error: ${error.message}`);
    }
    return Promise.reject(error)
  }
)

export default apiClient
