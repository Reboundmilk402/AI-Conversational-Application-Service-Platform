import axios from 'axios'

const backendOrigin = typeof window !== 'undefined'
  ? `http://${window.location.hostname}:9091`
  : 'http://localhost:9091'

const api = axios.create({
  baseURL: `${backendOrigin}/api/v1`,
  timeout: 0
})

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers = config.headers || {}
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api
