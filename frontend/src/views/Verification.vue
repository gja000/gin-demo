<template>
  <div>
    <el-card>
      <h3>核查看板</h3>
      <el-form inline>
        <el-form-item label="活动">
          <el-select v-model="selAct" placeholder="选择活动" @change="loadAll" style="width:320px">
            <el-option v-for="a in acts" :key="a.id" :label="a.id + ' · ' + a.status" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="loadAll">刷新</el-button>
          <el-button type="primary" @click="poll = !poll">{{ poll ? '停止轮询' : '开启轮询(2s)' }}</el-button>
        </el-form-item>
      </el-form>

      <el-row :gutter="12" v-if="overview">
        <el-col :span="4"><el-statistic title="成功" :value="overview.success" /></el-col>
        <el-col :span="4"><el-statistic title="失败" :value="overview.failed" /></el-col>
        <el-col :span="4"><el-statistic title="超时" :value="overview.timeout" /></el-col>
        <el-col :span="4"><el-statistic title="未确认" :value="overview.unconfirmed" /></el-col>
        <el-col :span="4"><el-statistic title="未下发" :value="overview.notSent" /></el-col>
        <el-col :span="4"><el-statistic title="车队总数" :value="overview.total" /></el-col>
      </el-row>
      <el-alert v-if="fullyRestored===false" type="warning" :closable="false" title="存在未确认车辆，禁止“全部恢复”" style="margin-top:10px" />
    </el-card>

    <el-card style="margin-top:16px">
      <h3>逐车核查（前值 / 期望 / 已生效 / 批次 / 版本 / 结果 / 失败原因 / 时间）</h3>
      <el-table :data="states" v-loading="loading" border>
        <el-table-column prop="vehicleId" label="车辆" width="90" />
        <el-table-column prop="prevInterval" label="前值" width="80" />
        <el-table-column prop="expectedInterval" label="期望值" width="90" />
        <el-table-column prop="effectiveInterval" label="已生效值" width="100" />
        <el-table-column prop="batch" label="批次" width="100" />
        <el-table-column prop="versionNo" label="版本号" width="80" />
        <el-table-column prop="status" label="结果" width="120">
          <template #default="{row}">
            <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="failReason" label="失败原因" width="140" />
        <el-table-column prop="publishedAt" label="发布" width="180" />
        <el-table-column prop="fetchedAt" label="拉取" width="180" />
        <el-table-column prop="receiptAt" label="回执" width="180" />
      </el-table>
    </el-card>

    <el-card style="margin-top:16px">
      <el-tabs>
        <el-tab-pane label="时间线（按车辆/版本）">
          <el-table :data="timeline" border>
            <el-table-column prop="vehicleId" label="车辆" width="90" />
            <el-table-column prop="versionNo" label="版本" width="80" />
            <el-table-column prop="batch" label="批次" width="100" />
            <el-table-column prop="expectedInterval" label="期望" width="80" />
            <el-table-column prop="status" label="状态" width="110" />
            <el-table-column prop="timedOut" label="曾超时" width="90" />
            <el-table-column prop="failReason" label="失败原因" width="140" />
            <el-table-column prop="publishedAt" label="发布" width="170" />
            <el-table-column prop="receiptAt" label="回执" width="170" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="审计日志（不可篡改）">
          <el-table :data="audit" border height="360">
            <el-table-column prop="createdAt" label="时间" width="200" />
            <el-table-column prop="action" label="动作" width="200" />
            <el-table-column prop="operatorRole" label="角色" width="120" />
            <el-table-column prop="targetVehicle" label="车辆" width="90" />
            <el-table-column prop="beforeValue" label="前值" width="100" />
            <el-table-column prop="afterValue" label="后值" width="100" />
            <el-table-column prop="requestId" label="请求ID" width="200" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'

const route = useRoute()
const acts = ref([])
const selAct = ref('')
const states = ref([])
const overview = ref(null)
const timeline = ref([])
const audit = ref([])
const loading = ref(false)
const poll = ref(false)
const fullyRestored = ref(null)
let timer = null

function statusType(s) {
  return { SUCCESS:'success', FAILED:'danger', TIMEOUT:'warning', PUBLISHED:'info', FETCHED:'', NOT_SENT:'info' }[s] || ''
}
async function loadActs() {
  const { data } = await axios.get('/api/activities')
  acts.value = data
  if (!selAct.value) {
    const q = route.query.id
    selAct.value = (q && data.find(a => a.id === q)) ? q : (data[0]?.id || '')
  }
  if (selAct.value) loadAll()
}
async function loadAll() {
  if (!selAct.value) return
  loading.value = true
  const [{ data: act }, { data: st }, { data: ov }, { data: tl }, { data: au }] = await Promise.all([
    axios.get(`/api/activities/${selAct.value}`),
    axios.get(`/api/activities/${selAct.value}/vehicles`),
    axios.get(`/api/activities/${selAct.value}/overview`),
    axios.get(`/api/activities/${selAct.value}/timeline`),
    axios.get(`/api/activities/${selAct.value}/audit`)
  ])
  states.value = st
  overview.value = ov
  timeline.value = tl
  audit.value = au
  fullyRestored.value = act.activity.fullyRestored
  loading.value = false
}
watch(poll, (v) => {
  if (v) timer = setInterval(loadAll, 2000)
  else clearInterval(timer)
})
onMounted(loadActs)
onUnmounted(() => clearInterval(timer))
</script>
