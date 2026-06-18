import axios from 'axios'

// 动态获取API基础URL
const getBaseURL = () => {
  const hostname = window.location.hostname
  const port = window.location.port
  const protocol = window.location.protocol // 获取当前页面协议（http: 或 https:）
  
  // 如果是localhost，使用localhost
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return 'http://localhost:8080/api'
  }
  
  // 否则使用当前访问的域名和协议，后端端口改为8080
  // 根据当前页面协议动态调整API协议，避免Mixed Content错误
  return `${protocol}//${hostname}:8080/api`
}

const api = axios.create({
  baseURL: getBaseURL(),
  timeout: 10000
})

// VPN相关API
export const vpnAPI = {
  // 下载VPN配置（无需认证）
  downloadConfig: () => api.get('/vpn/download', {
    responseType: 'blob'
  }),
  
  // 检查VPN连接
  checkConnection: () => api.get('/check')
}

// 位置相关API
export const locationAPI = {
  // 锁定位置
  lockLocation: (data) => api.post('/map/lock', data),
  
  // 设置虚拟定位
  setSpoofLocation: (data) => api.post('/map/spoof', data)
}

export default api
