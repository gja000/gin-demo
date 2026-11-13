# 测试车队日志策略平台（题目 09）

车队日志策略的「申请 → 审批 → 分批发布（试点/推广）→ 结果核查 → 停止/恢复」全流程平台。

## 技术栈
- 后端：Gin + GORM + MySQL（本地，库名 `car`）+ Redis（本地，仅缓存）
- 前端：Vue 3 + Vite + Element Plus
- 部署：GitHub Actions（CI 构建/测试 + 合并 `main` 后自动部署到远程服务器）

## 角色与账号（密码均为 `123456`）
| 账号 | 角色 | 职责 |
|------|------|------|
| applicant | 研发申请人 | 创建/修改/提交申请 |
| approver  | 审批负责人 | 批准/退回、判定完成、停止 |
| operator  | 发布运营 | 试点/推广/恢复下发 |

车端模拟器：车辆 `V1~V5`，凭证 `secret-V1`…`secret-V5`（在「车端模拟器」页登录）。

## 本地运行
```bash
# 1. 后端（默认读取 .env，MySQL root/root@3306/car，Redis 127.0.0.1:6379）
cp .env.example .env   # 按需修改
go run .              # 监听 :8080

# 2. 前端（另开终端，开发模式带代理到 8080）
cd frontend && npm install && npm run dev   # http://localhost:5173

# 3. 生产：后端同时托管前端构建产物
cd frontend && npm run build   # 产物在 frontend/dist，后端静态托管
```

演示参数（`.env`）：`TIMEOUT_SECONDS=15`（超时阈值）、`SCAN_INTERVAL_SECONDS=3`（扫描周期）、`ENABLE_RESET=1`（开发数据重置，生产务必关闭）。

## 业务规则覆盖（详见 AGENTS.md）
- BR-01~05 申请与审批（完整性、禁自审、退回再提交、冻结、车队唯一活动）
- BR-06~09 分批发布（试点恰好 2 台、试点通过后推广、幂等）
- BR-10~21 车端（拉取/回执/跨车拒绝/重复幂等/单调生效/旧回执忽略/超时扫描/迟到回执/离线）
- BR-22~28 停止与恢复（冻结已发布集合、恢复需批准、恢复版本=前值、全部确认才可恢复完成）
- BR-29~31 核查（逐车字段、概览可加总、审计不可篡改）
- INV-01~04 不变量（车队唯一活动、参数冻结、目标车辆全部、版本记录不可变）
- AC-01~19 验收（正常/异常/停止/恢复/超时/安全场景）

## 验证脚本（开发用）
- `test_smoke.ps1`：AC-01 正常推广闭环
- `test_rules.ps1`：AC-02/04/06/07/12/13
- `test_advanced.ps1`：AC-03 超时、AC-08/09 恢复、AC-11 旧回执
（均需服务运行且 `ENABLE_RESET=1`）

## 假设登记
见 AGENTS.md 第 10 节与接口 `GET /api/assumptions`；含 meegle 占位编号偏离说明（`ASM-MEEgle`，待评审确认）。
