import axios from 'axios'

// 动态获取API基础URL
const getBaseURL = () => {
  const hostname = window.location.hostname
  const protocol = window.location.protocol // 获取当前页面协议（http: 或 https:）
  
  // 如果是localhost，使用localhost
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return 'http://localhost:8080/api'
  }
  
  // 生产环境：使用当前访问的域名和协议，通过Nginx反代到后端
  // 不指定端口，使用默认的80/443端口
  return `${protocol}//${hostname}/api`
}

const api = axios.create({
  baseURL: getBaseURL(),
  timeout: 10000,
  withCredentials: true
})

// 认证相关API
export const authAPI = {
  register: (data) => api.post('/register', data),
  login: (data) => api.post('/login', data),
  logout: () => api.post('/logout'),
  getProfile: () => api.get('/profile')
}

// VPN相关API
export const vpnAPI = {
  downloadConfig: () => api.get('/vpn/download', {
    responseType: 'blob'
  }),
  checkConnection: (clientIP) => api.get('/check', { params: { client_ip: clientIP } })
}

// 位置相关API —— 直接用客户端IP作为key
export const locationAPI = {
  lockLocation: (data) => api.post('/map/lock', data),
  setSpoofLocation: (data) => api.post('/map/spoof', data),
  getStatus: () => api.get('/map/status')
}

export default api