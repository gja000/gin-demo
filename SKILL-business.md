---
AIGC:
    Label: "1"
    ContentProducer: 001191440300708461136T1XGW3
    ProduceID: 4434737a4106790ec491403cba0fbf3f_47db616adf0f11f1bc42525400638852
    ReservedCode1: qZmS5OF3PFrAGV0e0kTACL1JCvBm7NTSS8J97BLN9694TPDCIaOMohqdIvMFfZgv7yA9xNqTzdg12sI+ttenHTzvJCAiJWJo4uGaP3vZuqDGKAg3fYLsyT5LE37Eulb3kRwMX9wpaLdzW5n09/HZDuy620WZyInC0v2Hh1Q2KxQRlJYzhKwFsofcu3E=
    ContentPropagator: 001191440300708461136T1XGW3
    PropagateID: 4434737a4106790ec491403cba0fbf3f_47db616adf0f11f1bc42525400638852
    ReservedCode2: qZmS5OF3PFrAGV0e0kTACL1JCvBm7NTSS8J97BLN9694TPDCIaOMohqdIvMFfZgv7yA9xNqTzdg12sI+ttenHTzvJCAiJWJo4uGaP3vZuqDGKAg3fYLsyT5LE37Eulb3kRwMX9wpaLdzW5n09/HZDuy620WZyInC0v2Hh1Q2KxQRlJYzhKwFsofcu3E=
---



# SKILL-business.md — 题目09 测试车队日志策略 · 业务专项 Skill

> 关联产物：`AGENTS.md`（业务规则约束，BR-01~31 / INV-01~05 / 状态机 / AC-01~19）、`AGENTS-git.md`（Git与部署规范）
> 业务基线：`09-测试车队日志策略-需求分析.md` v1.0（2026-11-13）
> 适用范围：研发申请、审批、试点、推广、回执、停止、恢复、核查、完成全链路

---

## 1. 用途

1. 为 AI 开发 Agent 提供「题目 09」业务侧的**逐条可执行断言**：每条业务规则（BR）均映射为「实现核对点 + 测试断言核对点 + 异常路径」，供实现时对照、评审时逐条勾稽。
2. 在接口/页面/车端模拟器开发与自测阶段，快速定位业务红线违规（如自审、越权推广、生效值被旧回执覆盖、出现"已撤销"文案等），并给出拒绝依据与修复指引。
3. 固化状态机与四个必演示场景（正常推广 / 审批试点 / 恢复 / 网络分支），确保业务落地可逐项 PASS/FAIL，杜绝"已下发=已生效"的错误建模。

## 2. 触发场景

| 场景 | 触发条件 |
|---|---|
| 接口实现 | 开发/修改申请、审批、试点、推广、回执、停止、恢复、完成等接口时 |
| 业务评审 | 评审申请审批页、试点推广页、逐车进度页、停止恢复页的行为是否符合 BR |
| 自测与预演 | 执行正常推广、审批试点、恢复、网络分支四大演示场景前/中/后 |
| 异常路径演练 | 覆盖离线、超时、迟到回执、重复回执、跨车队回执、旧版本回执等分支 |
| 规则冲突 | 发现题目未明确事项时，触发假设登记（不猜测、不自行取舍） |

## 3. 输入

| 输入 | 说明 |
|---|---|
| 目标对象 | 接口代码/接口响应/页面行为/车端模拟器参数/数据库记录（至少一类） |
| 角色上下文 | 当前操作者角色（APPLICANT / APPROVER / OPERATOR / VEHICLE） |
| 活动与车辆 | activityId、fleetId、vehicleId、version_no、expected_interval、effective_interval |
| 既有约束 | 本 Skill 内置 BR-01~31 / INV-01~05 / 状态机断言表（无需额外输入） |
| 可选基线 | AGENTS.md 红线清单、需求分析文档 §13 验收标准 |

## 4. 执行步骤

### Step 0 前置判定（必做）
1. 确定本次核对对象属于哪类：接口实现 / 页面行为 / 模拟器行为 / 数据一致性。
2. 确定涉及角色与操作，按 §2.2 权限矩阵锁定可执行集合。
3. 若存在题目未明确事项（如停止审批链、试点数量可配、单车重发、迟到回执对停止活动的生效影响），**先按需求分析文档 §15 默认口径登记假设**，不得直接按猜测实现。

### Step 1 逐条断言核对（核心）
按下表从 BR-01 起逐条执行；每条给出 `实现核对点`（代码/数据层面要满足什么）、`测试断言核对点`（怎么验证）、`异常路径`（失败/边界时系统必须如何表现），并记录 PASS / FAIL / NA 与证据。

| 编号 | 规则 | 实现核对点（Implement） | 测试断言核对点（Assert） | 异常路径（Edge） |
|---|---|---|---|---|
| BR-01 | 申请完整性 | 服务端校验 reason 非空且 ≤500 字、fleetId 必填、目标车辆=车队全部5台（禁止部分选择）、newInterval 整数且 10–300 | 缺项/越界时返回字段级错误（VALIDATION_FAILED）且不创建申请 | 前端正常但服务端仍必须校验；篡改请求体不得绕过 |
| BR-02 | 禁止自审 | approverId != applicantId 由服务端判定（会话令牌取用户ID，忽略请求体字段） | 同账号审批返回 SELF_APPROVAL_FORBIDDEN；双角色用户切换账号可审批 | 双角色同一会话自审必须被拦截 |
| BR-03 | 退回与再提交 | 退回必填理由；退回后可修改 reason/newInterval 并重提交；保留历次快照与 submit_count | 两次提交快照并存、submit_count=2、diff 展示间隔与原因变化 | 未填理由退回被拒绝（REASON_REQUIRED） |
| BR-04 | 批准后冻结 | 批准后 newInterval/目标车辆不可改；修改接口返回 PARAM_FROZEN | 批准后调用修改接口失败且数据未变 | 只能新建申请，原活动保留构成历史成本 |
| BR-05 | 批准前置 | 车队存在未结束活动时禁止批准新申请 | 返回 FLEET_HAS_ACTIVE_ACTIVITY | 并发提交多个申请时唯一索引+事务兜底 |
| BR-06 | 车队单一活动 | 数据库唯一索引（fleet_id + 未结束状态）+ 事务保证，不依赖前端 | 并发创建/批准不产生两个未结束活动 | 前端未拦截时后端仍拒绝（AC-18） |
| BR-07 | 试点范围 | 试点恰好2台、运营显式选择；未试点3台在推广前无任何版本 | 非2台返回 PILOT_SIZE_INVALID；V3 拉取返回 NO_PUBLISHED_VERSION | 重复启动试点返回 ALREADY_STARTED（幂等） |
| BR-08 | 试点门禁 | 2台试点均 SUCCESS → PILOT_PASSED 推广可用；任一 FAILED/TIMEOUT → PAUSED，推广返回 PILOT_NOT_PASSED；不提供强制跳过参数 | 试点未过调用推广接口返回 409；伪造强制参数同样拒绝 | 试点车离线/超时不进入 PILOT_PASSED |
| BR-09 | 批次不可重复 | 试点、推广各执行一次；重复调用幂等返回当前状态、不生成新版本 | 重复推广返回当前状态且版本数不变 | 并发点击推广不产生双份版本 |
| BR-10 | 每车独立版本 | version_no 按车维度从1递增；同活动同车不出现重复版本号；版本行关键字段创建后不可改 | 查询确认 V1 版本 1,2,3 连续；INV-04 满足 | 版本行被修改应触发数据校验告警 |
| BR-11 | 最新可见 | 拉取只返回面向该车的最新已发布版本，且属于当前活动与合法批次；未进入批次车返回空+NO_PUBLISHED_VERSION | V3 未推广前拉取为空；推广后拉到 ROLLOUT 版本 | 离线期间被替代的旧版本（SUPERSEDED）不下发（AC-10） |
| BR-12 | 下发不等于生效 | Published/Fetched 状态不写 effective_interval；仅 SUCCESS 回执更新 | 拉取后生效值不变；回执成功后生效值=expected | 禁止任何 UI/接口将"已下发"当"已生效" |
| BR-13 | 离线车 | 离线不主动拉取保持 PUBLISHED；上线拉取到最新版本，不回放旧版本 | 离线车版本状态 PUBLISHED（未确认） | 恢复/推广场景离线车超时计时正常启动 |
| BR-14 | 回执鉴权归属 | 回执必须带车端凭证绑定 vehicleId；versionId 必须属于该车；未知/他车/跨车队版本拒绝 | 返回 RECEIPT_REJECTED 并写审计（含原始载荷摘要） | 无凭证 401；伪造 vehicleId 不生效 |
| BR-15 | 幂等 | 同 versionId 成功回执重复上报不重复记账、生效值不变 | duplicate_count 增加、生效值不变、无重复事件（AC-12） | 并发同版本回执只产生一次生效更新 |
| BR-16 | 单调生效 | version_no 小于当前生效版本的回执不覆盖生效值 | 记录 STALE_RECEIPT_IGNORED、生效值不变（AC-11） | 旧版本内容不同也不得覆盖 |
| BR-17 | 失败语义 | FAILED 只更新该版本执行结果与失败原因，不改生效值 | 失败后 effective_interval 仍为前值；fail_reason 枚举 APPLY_ERROR/UNSUPPORTED_VALUE/STORAGE_ERROR/TIMEOUT_APPLY | 失败不阻塞后续其他车回执 |
| BR-18 | 重启一致性 | 状态/版本/回执持久化；重启不丢状态、不重放已处理回执；超时扫描续跑 | 重启后核查视图一致；重复回执仍幂等 | 重启期间到达的回执按正常流程处理 |
| BR-19 | 超时判定 | 发布后60秒（可配置）无有效回执 → TIMEOUT；定时扫描（默认5秒一轮），不依赖刷新 | 60秒到无回执标记 TIMEOUT、生效值不变（AC-03） | 超时阈值可缩短（演示环境）、系统时间可注入 |
| BR-20 | 迟到回执 | 超时后针对当前期望版本的合法回执可更新结果（成功则更新生效值）；保留超时历史 | 时间线含 TIMEOUT + SUCCESS 两条（AC-14） | 迟到回执必须仍校验归属与版本合法性 |
| BR-21 | 停止后迟到回执 | 已停止活动的迟到回执仅记录时间线，不自动重启/推广/完成 | 活动状态不变（AC-15） | 回执可更新该车生效值与时间线，但不改活动状态 |
| BR-22 | 停止语义 | 停止只阻止未发布部分；已发布版本/生效值/回执历史保留；文案禁止"已撤销/已回滚"，统一"已停止，车端仍保留已下发配置" | 未发车 NOT_SENT、已发记录保留、文案检索无"已撤销"（AC-07） | 任何位置（接口/页面/日志）不得出现"已撤销车端配置" |
| BR-23 | 停止范围记录 | 停止时固化已发布车辆集合与未发布车辆集合（NOT_SENT） | 恢复范围与核查基于固化范围计算 | 停止后新车不得再进入已发集合 |
| BR-24 | 恢复批准 | 恢复=运营发起 → 负责人批准 → 运营执行；停止中不可跳过批准 | 未批准执行返回 RESTORE_NOT_APPROVED；批准接口拒绝自审 | 负责人亦可直接决定停止，但仍需走恢复批准 |
| BR-25 | 恢复版本 | 为曾发布车辆创建 version_no=历史最大+1、expected_interval=各车前值（可不同） | 恢复版本号+1、目标=前值（AC-08） | 无生效车辆时返回 NO_AFFECTED_VEHICLE（BR-27） |
| BR-26 | 恢复完成判定 | 全部需恢复车辆 SUCCESS 才可 RESTORED；存在未确认保持 RESTORING，fullyRestored=false，禁止"全部恢复"文案 | 1台未确认时界面显示"N台未确认"、无"全部恢复"（AC-09） | 完成接口返回 UNCONFIRMED_REMAINS |
| BR-27 | 无需恢复的停止 | 停止时无车辆实际生效 → 直接 STOPPED 终态，不生成恢复版本 | 恢复接口返回 NO_AFFECTED_VEHICLE | 与 BR-23 固化范围结合判定 |
| BR-28 | 完成条件 | 仅5台全部 SUCCESS 可 COMPLETED；存在失败/超时/未确认完成不可用 | 完成接口返回 NOT_ALL_CONFIRMED（AC-17）；恢复完成同理 | 恢复场景完成后状态为 RESTORED 而非 COMPLETED |
| BR-29 | 逐车核查字段 | 每车展示前值/期望值/生效值/批次/版本号/结果/失败原因/发布/拉取/回执时间 | 核查表同时含成功、失败、未确认行、字段齐全（AC-16） | 未下发车显示 NOT_SENT 而非缺行 |
| BR-30 | 禁止聚合掩盖 | 概览数字可下钻到逐车记录；不只用活动总状态表达结果 | 概览 success/failed/timeout/unconfirmed/notSent 与明细一致 | 总数与逐车行数不一致即 FAIL |
| BR-31 | 审计 | 申请/提交/退回/批准/选试点/启动/推广/超时/回执成功失败/拒绝/停止/恢复批准/恢复下发/恢复完成/完成 全部写审计（操作者/角色/时间/目标车辆/前后值/请求ID） | 按活动、车辆、操作者过滤可查（AC-19）；审计不可篡改 | 非法回执拒绝同样入审计（BR-14） |

### Step 2 状态机校验
按 §7 活动状态机与单车版本状态机核对当前对象状态流转：
- 活动状态：`DRAFT → SUBMITTED → APPROVED → PILOTING → PILOT_PASSED → ROLLING_OUT → PENDING_VERIFY → COMPLETED`，分支 `PAUSED → STOPPED → RESTORING → RESTORED`。
- **门禁校验**：PAUSED → ROLLING_OUT 不存在直达路径；STOPPED → RESTORED 必须经负责人批准 + 恢复版本下发 + 全部需恢复车辆确认；COMPLETED 仅当5台全 SUCCESS。
- 单车版本：`PUBLISHED → FETCHED → SUCCESS/FAILED`；`PUBLISHED → TIMEOUT →（迟到成功）SUCCESS（保留 TIMEOUT 历史）`；旧版本被更高版本替代 → SUPERSEDED。
- 出现未列出的流转（如 PAUSED 直接推广、STOPPED 直接 RESTORED）一律判 FAIL。

### Step 3 四大演示场景预演（AC 映射）
按需求分析 §10 流程逐场景走查，每个场景结束时核对映射 AC：
1. **正常推广**（AC-01）：申请→批准→试点V1/V2→双成功→推广V3~V5→全成功→COMPLETED。
2. **审批与试点分支**（AC-02/03/04/05/06/07）：退回再提交、试点失败/超时 PAUSED、推广被拒、V3拉取为空、负责人停止、文案无"已撤销"。
3. **恢复分支**（AC-08/09）：恢复申请→批准→恢复版本（前值、版本+1）→部分未确认保持 RESTORING → 全确认 → RESTORED。
4. **网络分支**（AC-03/10/11/12/13/14/15）：V5离线→TIMEOUT→上线拉最新→迟到成功→重复回执幂等→未知/他车/跨车队拒绝→停止后迟到回执仅记录。

### Step 4 汇总与结论
1. 汇总每条 BR 与场景的 PASS/FAIL/NA，列出证据（接口响应、数据库查询、页面截图、日志片段）。
2. 任一 FAIL 必须给出：违反规则编号、违反红线描述、建议修复位置。
3. 输出结构化结论报告（见第 5 节）。

## 5. 输出格式

```
[业务专项核验报告]
项目：题目09 测试车队日志策略
核对对象：<接口/页面/模拟器/数据>  <对象标识>
核对角色：<APPLICANT/APPROVER/OPERATOR/VEHICLE>

【BR 断言结果】共 31 条
PASS:  N 条   FAIL: M 条   NA: K 条

| 编号 | 规则 | 结果 | 证据 | 异常路径表现 |
|---|---|---|---|---|
| BR-xx | 规则名 | PASS/FAIL/NA | <接口码/查询/截图> | <符合/不符合> |

【状态机校验】
活动状态流转：<如 APPROVED→PILOTING→PILOT_PASSED>  PASS/FAIL
单车版本流转：<如 PUBLISHED→FETCHED→SUCCESS>      PASS/FAIL
门禁规则：<列出门禁点逐项结果>

【演示场景预演】正常推广 / 审批试点 / 恢复 / 网络分支
场景1（正常推广）：PASS/FAIL  — 关键证据：...
场景2（审批试点）：PASS/FAIL  — 关键证据：...
场景3（恢复分支）：PASS/FAIL  — 关键证据：...
场景4（网络分支）：PASS/FAIL  — 关键证据：...

【红线确认】
1. 是否出现"已下发=已生效"汇报：否/是
2. 是否出现自审/越权推广/旧回执覆盖生效值：否/是
3. 是否出现"已撤销/已回滚"文案：否/是
4. 是否存在未确认时"全部恢复"/自动完成：否/是
5. 是否存在未登记假设：否/是（若有，见假设清单）

【结论】业务侧整体判定：通过 / 不通过（附修复清单）
```

## 6. 失败处理

| 失败类型 | 处理建议 |
|---|---|
| 断言 FAIL（违反 BR/INV/状态机） | 定位到规则编号与实现位置，输出修复指引；修复后重新执行 Step 1 对应条目，禁止带 FAIL 进入验收 |
| 证据不足（无接口响应/查询/截图） | 标记该条为「待补证据」，不得擅自判定 PASS；补齐证据后复验 |
| 题目未明确事项 | 登记假设（docs/assumptions.md），不得猜测实现；未确认前不判定为需求 |
| 场景预演中断（如模拟器不支持某分支） | 明确缺失能力并登记为环境缺口，不虚构结果 |
| 工具/接口调用失败 | 检查凭证、参数、环境后重试 1 次；仍失败则如实记录，禁止伪造返回 |

## 7. 完成标准

1. BR-01~31 全部逐条核对，结论为 PASS（或 NA 且理由成立），无遗留 FAIL。
2. 状态机（活动 + 单车版本）校验通过，无未列出流转。
3. 四个必演示场景全部走通，映射 AC 逐项 PASS。
4. 每条结论均有真实证据（接口响应 / 数据库记录 / 页面截图 / 日志），可追溯。
5. 红线清单 5 项全部确认「否」，未出现越权、覆盖、伪造、误导性文案。
6. 未确认假设均已登记且状态透明。

---

## 8. AI 可使用的 Skill 文件说明与调用记录模板

### 8.1 Skill 文件配置方式（AI 工具侧）

**适用工具**：Claude Code / Cursor / Windsurf / Codex CLI 等支持 Skills/Agent Rules 的 AI 编程工具。

**方式 A（Claude Code / Cursor 等，推荐）**：将本文件保存为项目内 `.claude/skills/business/fleet-log-business-skill.md` 或 `skills/fleet-log-business/SKILL.md`（文件名与 front matter `name` 对应），AI 在项目中自动可检索调用。

**方式 B（作为项目规则注入）**：将本文件内容追加进项目 `AGENTS.md` 引用清单，或直接在工具设置中把本文件路径加入系统提示引用，AI 在业务相关任务开始时按 `触发场景` 主动加载执行。

**方式 C（手动触发）**：向 AI 提问时附带指令「请加载 fleet-log-business-skill 并按 Step 1 逐条核验：<对象>」，AI 按本文件执行步骤输出报告。

### 8.2 调用记录模板（提交验收用）

每次实际调用需按以下模板留存证据（作为 Skill 交付物的一部分）：

```
【Skill 调用记录】
Skill 名称：fleet-log-business-skill
调用时间：YYYY-MM-DD HH:MM:SS
调用工具：<Claude Code / Cursor / 其他>
调用方式：<自动加载 / 手动指令 / 项目规则注入>
任务上下文：<核验对象：如 POST /api/activities/ACT-001/rollout 试点门禁核验>
输入摘要：<activityId、角色、期望状态等>
关键输出：<PASS/FAIL 结论、规则编号、证据引用>
结论：通过 / 需修复（附修复项）
证据留存：<接口响应片段 / 截图路径 / 日志位置>
```

调用记录建议统一维护在 `docs/skill-call-logs/business-skill-log.md`，一次核验一行（或一次核验一段），与验收证据一并提交。

---
*本 Skill 与 AGENTS.md（业务规则约束）配套使用；涉及 Git 提交、PR、部署环节时另见 AGENTS-git.md。*
*（内容由AI生成，仅供参考）*
