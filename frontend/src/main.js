import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import axios from 'axios'

axios.defaults.baseURL = '/api'
axios.interceptors.request.use((config) => {
  const token = localStorage.getItem('token') || sessionStorage.getItem('vtoken')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})
axios.interceptors.response.use(
  (r) => r,
  (e) => {
    if (e.response && e.response.data && e.response.data.message) {
      e.message = e.response.data.message
    }
    return Promise.reject(e)
  }
)

const app = createApp(App)
app.use(ElementPlus)
app.use(router)
app.config.globalProperties.$axios = axios
app.mount('#app')
