<template>
  <div class="login-wrap">
    <el-card class="box">
      <h2>测试车队日志策略平台</h2>
      <p class="sub">题目09 · 申请 / 审批 / 分批发布 / 结果核查</p>
      <el-alert type="info" :closable="false" title="演示账号（密码均为 123456）" description="applicant 研发申请人 / approver 审批负责人 / operator 发布运营；车端模拟请在「车端模拟器」页登录" />
      <div style="margin-top:18px">
        <el-button type="primary" @click="quick('applicant')">以研发申请人登录</el-button>
        <el-button type="success" @click="quick('approver')">以审批负责人登录</el-button>
        <el-button type="warning" @click="quick('operator')">以发布运营登录</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const router = useRouter()
async function quick(role) {
  try {
    const { data } = await axios.post('/api/login', { username: role, password: '123456' })
    localStorage.setItem('token', data.token)
    localStorage.setItem('role', data.user.role)
    ElMessage.success('登录成功：' + data.user.name)
    router.push('/' + role.toLowerCase())
  } catch (e) {
    ElMessage.error('登录失败：' + (e.message || e))
  }
}
</script>

<style scoped>
.login-wrap{height:100vh;display:flex;align-items:center;justify-content:center;background:#f0f2f5}
.box{width:520px}
.sub{color:#888;margin:6px 0 18px}
</style>
