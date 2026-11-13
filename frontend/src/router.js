import { createRouter, createWebHistory } from 'vue-router'
import Login from './views/Login.vue'
import Applicant from './views/Applicant.vue'
import Approver from './views/Approver.vue'
import Operator from './views/Operator.vue'
import Verification from './views/Verification.vue'
import VehicleSim from './views/VehicleSim.vue'

const routes = [
  { path: '/', redirect: '/login' },
  { path: '/login', name: 'login', component: Login },
  { path: '/applicant', name: 'applicant', component: Applicant, meta: { role: 'APPLICANT' } },
  { path: '/approver', name: 'approver', component: Approver, meta: { role: 'APPROVER' } },
  { path: '/operator', name: 'operator', component: Operator, meta: { role: 'OPERATOR' } },
  { path: '/verification', name: 'verification', component: Verification },
  { path: '/vehicle', name: 'vehicle', component: VehicleSim }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  if (to.path === '/login') return true
  const role = localStorage.getItem('role')
  const token = localStorage.getItem('token')
  if (!token) return '/login'
  if (to.meta && to.meta.role && to.meta.role !== role) {
    return '/' + role.toLowerCase()
  }
  return true
})

export default router
