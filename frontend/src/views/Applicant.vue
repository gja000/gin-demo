<template>
  <div>
    <el-card>
      <h3>发起日志策略申请</h3>
      <el-form :model="form" label-width="120px" style="max-width:560px">
        <el-form-item label="车队">
          <el-input :model-value="'FLEET-A（测试车队 A）'" disabled />
        </el-form-item>
        <el-form-item label="申请原因">
          <el-input v-model="form.reason" type="textarea" :rows="3" placeholder="必填，≤500 字" />
        </el-form-item>
        <el-form-item label="目标日志间隔(秒)">
          <el-input-number v-model="form.newInterval" :min="10" :max="300" />
        </el-form-item>
        <el-form-item label="目标车辆">
          <el-tag v-for="v in targetVehicles" :key="v" style="margin-right:6px">{{ v }}</el-tag>
          <span style="color:#999">（车队全部 5 台，禁止部分选择）</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="createApp">创建申请</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <h3>我的申请</h3>
      <el-table :data="apps" v-loading="loading">
        <el-table-column prop="id" label="申请ID" width="200" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{row}"><el-tag>{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="newInterval" label="新间隔" width="90" />
        <el-table-column prop="submitCount" label="提交次数" width="100" />
        <el-table-column prop="reason" label="原因" />
        <el-table-column label="操作" width="260">
          <template #default="{row}">
            <el-button size="small" @click="showDetail(row)">详情</el-button>
            <el-button v-if="row.status==='DRAFT'||row.status==='REJECTED'" size="small" type="warning" @click="openEdit(row)">修改</el-button>
            <el-button v-if="row.status==='DRAFT'||row.status==='REJECTED'" size="small" type="primary" @click="submit(row)">提交</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="editVis" title="修改申请（仅 DRAFT/REJECTED 可改）">
      <el-form :model="editForm">
        <el-form-item label="原因"><el-input v-model="editForm.reason" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="新间隔"><el-input-number v-model="editForm.newInterval" :min="10" :max="300" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVis=false">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="detailVis" title="申请详情 / 提交快照">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="申请ID">{{ detail.application?.id }}</el-descriptions-item>
        <el-descriptions-item label="当前状态">{{ detail.application?.status }}</el-descriptions-item>
        <el-descriptions-item label="当前原因">{{ detail.application?.reason }}</el-descriptions-item>
        <el-descriptions-item label="当前间隔">{{ detail.application?.newInterval }}</el-descriptions-item>
      </el-descriptions>
      <h4 style="margin-top:14px">提交历史（共 {{ detail.snapshots?.length }} 次）</h4>
      <el-timeline>
        <el-timeline-item v-for="(s,i) in detail.snapshots" :key="i" :timestamp="String(s.submittedAt)">
          第 {{ i+1 }} 次提交：间隔 {{ s.newInterval }} 秒，原因「{{ s.reason }}」
        </el-timeline-item>
      </el-timeline>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const targetVehicles = ['V1','V2','V3','V4','V5']
const form = ref({ reason: '', newInterval: 30 })
const apps = ref([])
const loading = ref(false)
const editVis = ref(false)
const detailVis = ref(false)
const editForm = ref({ id:'', reason:'', newInterval:30 })
const detail = ref({ application:null, snapshots:[] })

async function load() {
  loading.value = true
  const { data } = await axios.get('/api/applications')
  apps.value = data
  loading.value = false
}
async function createApp() {
  try {
    await axios.post('/api/applications', {
      fleetId: 'FLEET-A', reason: form.value.reason, newInterval: form.value.newInterval, targetVehicles
    })
    ElMessage.success('已创建申请（DRAFT）')
    form.value.reason = ''
    load()
  } catch (e) { ElMessage.error(e.message) }
}
async function submit(row) {
  try { await axios.post(`/api/applications/${row.id}/submit`); ElMessage.success('已提交'); load() }
  catch (e) { ElMessage.error(e.message) }
}
function openEdit(row) { editForm.value = { id: row.id, reason: row.reason, newInterval: row.newInterval }; editVis.value = true }
async function saveEdit() {
  try {
    await axios.patch(`/api/applications/${editForm.value.id}`, { reason: editForm.value.reason, newInterval: editForm.value.newInterval })
    ElMessage.success('已保存（批准后将被冻结）'); editVis.value = false; load()
  } catch (e) { ElMessage.error(e.message) }
}
async function showDetail(row) {
  const { data } = await axios.get(`/api/applications/${row.id}`)
  detail.value = data; detailVis.value = true
}
onMounted(load)
</script>
