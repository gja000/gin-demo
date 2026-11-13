<template>
  <Login v-if="!loggedIn" />
  <el-container v-else style="height:100vh">
    <el-aside width="210px" style="background:#304156">
      <div class="logo">车队日志策略</div>
      <el-menu :default-active="active" background-color="#304156" text-color="#bfcbd9"
        active-text-color="#409eff" router>
        <el-menu-item index="/verification"><el-icon><DataLine/></el-icon>核查看板</el-menu-item>
        <el-menu-item v-if="role==='APPLICANT'" index="/applicant"><el-icon><Edit/></el-icon>我的申请</el-menu-item>
        <el-menu-item v-if="role==='APPROVER'" index="/approver"><el-icon><Stamp/></el-icon>审批与完成</el-menu-item>
        <el-menu-item v-if="role==='OPERATOR'" index="/operator"><el-icon><Promotion/></el-icon>运营发布</el-menu-item>
        <el-menu-item index="/vehicle"><el-icon><Cpu/></el-icon>车端模拟器</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header style="display:flex;align-items:center;justify-content:space-between;background:#fff;border-bottom:1px solid #eee">
        <span>测试车队日志策略 · 申请/审批/分批发布/结果核查</span>
        <div>
          <el-tag>{{ roleName }}</el-tag>
          <el-button size="small" @click="logout" style="margin-left:10px">退出</el-button>
        </div>
      </el-header>
      <el-main><router-view/></el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DataLine, Edit, Stamp, Promotion, Cpu } from '@element-plus/icons-vue'
import Login from './views/Login.vue'

const route = useRoute()
const router = useRouter()
const loggedIn = computed(() => !!localStorage.getItem('token'))
const role = computed(() => localStorage.getItem('role') || '')
const roleName = computed(() => ({ APPLICANT:'研发申请人', APPROVER:'审批负责人', OPERATOR:'发布运营' }[role.value] || role.value))
const active = computed(() => route.path)
function logout() {
  localStorage.clear()
  router.push('/login')
}
</script>

<style>
.logo{color:#fff;font-weight:700;padding:18px 16px;font-size:16px;border-bottom:1px solid #1f2d3d}
body{margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'PingFang SC','Microsoft YaHei',sans-serif}
</style>
