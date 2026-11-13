<template>
  <div>
    <el-card>
      <h3>车端模拟器</h3>
      <el-alert type="info" :closable="false" title="模拟车辆行为" description="登录车辆后：拉取面向本车的最新版本（在线才拉取），上报成功/失败回执。可模拟离线（不拉取）、失败注入、延迟/迟到回执。" />
      <el-form inline style="margin-top:12px">
        <el-form-item label="车辆">
          <el-select v-model="vehId" style="width:120px">
            <el-option v-for="v in vehicles" :key="v" :label="v" :value="v" />
          </el-select>
        </el-form-item>
        <el-form-item label="凭证">
          <el-input v-model="secret" placeholder="secret-Vx" style="width:160px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="login">车端登录</el-button>
          <el-button @click="logout">登出</el-button>
        </el-form-item>
      </el-form>
      <el-tag v-if="logged" type="success">已登录：{{ vehId }}</el-tag>
      <el-tag v-if="!online" type="info" style="margin-left:8px">离线（不拉取）</el-tag>
    </el-card>

    <el-card v-if="logged" style="margin-top:16px">
      <h3>当前配置与回执</h3>
      <el-form inline>
        <el-form-item>
          <el-switch v-model="online" active-text="在线" inactive-text="离线" />
        </el-form-item>
        <el-form-item>
          <el-button @click="pull" :disabled="!online">拉取配置（Pull）</el-button>
        </el-form-item>
      </el-form>

      <el-descriptions v-if="config" :column="2" border style="margin-top:8px">
        <el-descriptions-item label="是否有配置">{{ config.hasConfig ? '是' : '否（' + (config.code||'') + '）' }}</el-descriptions-item>
        <el-descriptions-item label="版本ID">{{ config.versionId }}</el-descriptions-item>
        <el-descriptions-item label="版本号">{{ config.versionNo }}</el-descriptions-item>
        <el-descriptions-item label="期望间隔">{{ config.expectedInterval }}</el-descriptions-item>
        <el-descriptions-item label="批次">{{ config.batch }}</el-descriptions-item>
      </el-descriptions>

      <el-divider />
      <el-form inline>
        <el-form-item label="回执结果">
          <el-radio-group v-model="result">
            <el-radio value="SUCCESS">成功</el-radio>
            <el-radio value="FAILED">失败</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="result==='FAILED'" label="失败原因">
          <el-select v-model="failReason" style="width:220px">
            <el-option v-for="f in failReasons" :key="f" :label="f" :value="f" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="success" @click="submitReceipt" :disabled="!config || !config.hasConfig">上报回执</el-button>
        </el-form-item>
      </el-form>
      <el-alert v-if="lastResult" type="success" :closable="false" style="margin-top:8px">
        回执响应：状态 {{ lastResult.status }} ｜ 重复 {{ lastResult.duplicate }} ｜ 重复计数 {{ lastResult.duplicateCount }} ｜ 代码 {{ lastResult.code || '-' }}
      </el-alert>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const vehicles = ['V1','V2','V3','V4','V5']
const vehId = ref('V1')
const secret = ref('secret-V1')
const logged = ref(false)
const online = ref(true)
const config = ref(null)
const result = ref('SUCCESS')
const failReason = ref('APPLY_ERROR')
const failReasons = ['APPLY_ERROR','UNSUPPORTED_VALUE','STORAGE_ERROR','TIMEOUT_APPLY']
const lastResult = ref(null)

async function login() {
  try {
    const { data } = await axios.post('/api/vehicle-agent/login', { vehicleId: vehId.value, secret: secret.value })
    sessionStorage.setItem('vtoken', data.token)
    logged.value = true; config.value = null; lastResult.value = null
    ElMessage.success('车端登录成功：' + data.vehicleId)
  } catch (e) { ElMessage.error('登录失败：' + e.message) }
}
function logout() { sessionStorage.removeItem('vtoken'); logged.value = false; config.value = null }

async function pull() {
  config.value = null
  try {
    const { data } = await axios.get('/api/vehicle-agent/config')
    config.value = data
  } catch (e) { ElMessage.error(e.message) }
}
async function submitReceipt() {
  if (!config.value || !config.value.hasConfig) return ElMessage.error('请先拉取配置')
  try {
    const body = { versionId: config.value.versionId, result: result.value }
    if (result.value === 'FAILED') body.failReason = failReason.value
    const { data } = await axios.post('/api/vehicle-agent/receipts', body)
    lastResult.value = data
    ElMessage.success('回执已上报')
    pull()
  } catch (e) { ElMessage.error(e.message) }
}
</script>
