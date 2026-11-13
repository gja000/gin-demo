<template>
  <div>
    <el-card>
      <h3>运营发布</h3>
      <el-alert type="info" :closable="false" title="流程" description="选择试点车(恰好2台)→启动试点→（试点通过后）推广→（停止后）恢复。恢复需负责人先批准。" />
      <el-form label-width="120px" style="margin-top:14px;max-width:600px">
        <el-form-item label="选择活动">
          <el-select v-model="selAct" placeholder="选择活动" @change="loadAct">
            <el-option v-for="a in acts" :key="a.id" :label="a.id + ' · ' + a.status" :value="a.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template v-if="cur">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="活动ID">{{ cur.id }}</el-descriptions-item>
          <el-descriptions-item label="状态"><el-tag>{{ cur.status }}</el-tag></el-descriptions-item>
          <el-descriptions-item label="冻结间隔">{{ cur.newInterval }}</el-descriptions-item>
          <el-descriptions-item label="恢复已批准">{{ cur.restoreApproved ? '是' : '否' }}</el-descriptions-item>
        </el-descriptions>

        <div style="margin:14px 0">
          <el-button :disabled="cur.status!=='APPROVED'" type="primary" @click="openPilot">选择试点车并启动</el-button>
          <el-button :disabled="cur.status!=='PILOT_PASSED'" type="success" @click="rollout">推广剩余车辆</el-button>
        </div>
        <div style="margin:8px 0">
          <el-button :disabled="cur.status!=='STOPPED'" @click="requestRestore">发起恢复（需负责人批准）</el-button>
          <el-button :disabled="!(cur.status==='STOPPED'&&cur.restoreApproved)" type="warning" @click="executeRestore">执行恢复下发</el-button>
        </div>
        <el-alert v-if="cur.status==='STOPPED'&&!cur.restoreApproved" type="warning" :closable="false" title="已发起恢复，待负责人在「审批与完成」页批准" />
        <el-alert v-if="cur.status==='RESTORING'" type="info" :closable="false" title="恢复下发中，请到车端模拟器让车辆拉取并回执；全部确认后由负责人判定恢复完成" />
      </template>
    </el-card>

    <el-dialog v-model="pilotVis" title="选择试点车辆（恰好 2 台）">
      <el-checkbox-group v-model="pilotSel">
        <el-checkbox v-for="v in vehicles" :key="v" :label="v" :value="v" border />
      </el-checkbox-group>
      <template #footer><el-button @click="pilotVis=false">取消</el-button><el-button type="primary" @click="doPilot">启动试点</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const acts = ref([])
const vehicles = ['V1','V2','V3','V4','V5']
const selAct = ref('')
const cur = ref(null)
const pilotVis = ref(false); const pilotSel = ref([])

async function load() {
  const { data } = await axios.get('/api/activities')
  acts.value = data
  if (!selAct.value && data.length) { selAct.value = data[0].id; loadAct() }
}
async function loadAct() {
  if (!selAct.value) return
  const { data } = await axios.get(`/api/activities/${selAct.value}`)
  cur.value = data.activity
}
function openPilot() { pilotSel.value = []; pilotVis.value = true }
async function doPilot() {
  if (pilotSel.value.length !== 2) return ElMessage.error('试点必须恰好 2 台')
  try { await axios.post(`/api/activities/${selAct.value}/pilot`, { vehicleIds: pilotSel.value }); ElMessage.success('试点已启动'); pilotVis.value = false; loadAct() }
  catch (e) { ElMessage.error(e.message) }
}
async function rollout() {
  try { await axios.post(`/api/activities/${selAct.value}/rollout`); ElMessage.success('推广已下发'); loadAct() }
  catch (e) { ElMessage.error(e.message) }
}
async function requestRestore() {
  try { await axios.post(`/api/activities/${selAct.value}/restore`); ElMessage.success('已发起恢复'); loadAct() }
  catch (e) { ElMessage.error(e.message) }
}
async function executeRestore() {
  try { await axios.post(`/api/activities/${selAct.value}/restore/execute`); ElMessage.success('恢复版本已下发'); loadAct() }
  catch (e) { ElMessage.error(e.message) }
}
onMounted(load)
</script>
