<template>
  <div>
    <el-card>
      <h3>待审批申请</h3>
      <el-table :data="pending" v-loading="loading">
        <el-table-column prop="id" label="申请ID" width="200" />
        <el-table-column prop="reason" label="原因" />
        <el-table-column prop="newInterval" label="新间隔" width="90" />
        <el-table-column prop="submitCount" label="提交次数" width="100" />
        <el-table-column label="操作" width="240">
          <template #default="{row}">
            <el-button size="small" type="success" @click="approve(row)">批准</el-button>
            <el-button size="small" type="danger" @click="openReject(row)">退回</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card style="margin-top:16px">
      <h3>活动决策（完成 / 停止 / 恢复完成）</h3>
      <el-table :data="acts" v-loading="loading">
        <el-table-column prop="id" label="活动ID" width="200" />
        <el-table-column prop="status" label="状态" width="140">
          <template #default="{row}"><el-tag>{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="newInterval" label="冻结间隔" width="100" />
        <el-table-column label="操作" width="320">
          <template #default="{row}">
            <el-button v-if="row.status==='PENDING_VERIFY'" size="small" type="success" @click="complete(row)">判定完成</el-button>
            <el-button v-if="row.status==='RESTORING'" size="small" type="success" @click="completeRestore(row)">判定恢复完成</el-button>
            <el-button v-if="['PILOTING','PILOT_PASSED','ROLLING_OUT','PENDING_VERIFY','PAUSED','RESTORING'].includes(row.status)" size="small" type="warning" @click="openStop(row)">停止</el-button>
            <el-button size="small" @click="$router.push('/verification?id='+row.id)">核查</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="rejVis" title="退回申请（必须填写理由）">
      <el-input v-model="rejReason" type="textarea" :rows="3" placeholder="退回理由" />
      <template #footer><el-button @click="rejVis=false">取消</el-button><el-button type="danger" @click="doReject">确认退回</el-button></template>
    </el-dialog>
    <el-dialog v-model="stopVis" title="停止活动">
      <el-input v-model="stopReason" type="textarea" :rows="2" placeholder="停止原因（可选）" />
      <template #footer><el-button @click="stopVis=false">取消</el-button><el-button type="warning" @click="doStop">确认停止</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const pending = ref([])
const acts = ref([])
const loading = ref(false)
const rejVis = ref(false); const rejReason = ref(''); let rejId = ''
const stopVis = ref(false); const stopReason = ref(''); let stopId = ''

async function load() {
  loading.value = true
  const { data: apps } = await axios.get('/api/applications')
  pending.value = apps.filter(a => a.status === 'SUBMITTED')
  const { data: activities } = await axios.get('/api/activities')
  acts.value = activities
  loading.value = false
}
async function approve(row) {
  try { await axios.post(`/api/applications/${row.id}/approve`); ElMessage.success('已批准，活动已创建'); load() }
  catch (e) { ElMessage.error(e.message) }
}
function openReject(row) { rejId = row.id; rejReason.value = ''; rejVis.value = true }
async function doReject() {
  if (!rejReason.value.trim()) return ElMessage.error('退回必须填写理由')
  try { await axios.post(`/api/applications/${rejId}/reject`, { reason: rejReason.value }); ElMessage.success('已退回'); rejVis.value = false; load() }
  catch (e) { ElMessage.error(e.message) }
}
async function complete(row) {
  try { await axios.post(`/api/activities/${row.id}/complete`); ElMessage.success('活动已完成'); load() }
  catch (e) { ElMessage.error(e.message) }
}
async function completeRestore(row) {
  try { await axios.post(`/api/activities/${row.id}/restore/complete`); ElMessage.success('恢复已完成'); load() }
  catch (e) { ElMessage.error(e.message) }
}
function openStop(row) { stopId = row.id; stopReason.value = ''; stopVis.value = true }
async function doStop() {
  try { await axios.post(`/api/activities/${stopId}/stop`, { reason: stopReason.value }); ElMessage.success('已停止'); stopVis.value = false; load() }
  catch (e) { ElMessage.error(e.message) }
}
onMounted(load)
</script>
