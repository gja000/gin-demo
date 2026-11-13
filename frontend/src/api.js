import axios from 'axios'

const A = axios.create({ baseURL: '/api' })

// 用户接口（会话令牌）
export function userReq() {
  const token = localStorage.getItem('token')
  return A
}
// 车端接口（车端令牌，独立 header）
export function vehicleReq() {
  const vtoken = sessionStorage.getItem('vtoken')
  const inst = axios.create({ baseURL: '/api' })
  if (vtoken) inst.defaults.headers.Authorization = 'Bearer ' + vtoken
  return inst
}

export default A
