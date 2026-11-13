---
AIGC:
    Label: "1"
    ContentProducer: 001191440300708461136T1XGW3
    ProduceID: 4434737a4106790ec491403cba0fbf3f_490fa3a6df0f11f1997d52540024e231
    ReservedCode1: 04Mg7JESC716UxSQfro+Gqn+p/8EpyPtSzo3jXsYr9kUp1UfkVVFzwWxCTiwXxDtXULJ6dyCAp3CgKU1Fham7Z9NtXM+vr280WEjZ7ytWB41UUU4XcoHhaPlOWR3jDUlEObnro8yzVR5LT4V0RNShOeCj/O6teVuRlc1yaagf3DRfMXAGCEVDEXzBWc=
    ContentPropagator: 001191440300708461136T1XGW3
    PropagateID: 4434737a4106790ec491403cba0fbf3f_490fa3a6df0f11f1997d52540024e231
    ReservedCode2: 04Mg7JESC716UxSQfro+Gqn+p/8EpyPtSzo3jXsYr9kUp1UfkVVFzwWxCTiwXxDtXULJ6dyCAp3CgKU1Fham7Z9NtXM+vr280WEjZ7ytWB41UUU4XcoHhaPlOWR3jDUlEObnro8yzVR5LT4V0RNShOeCj/O6teVuRlc1yaagf3DRfMXAGCEVDEXzBWc=
---



# SKILL-acceptance.md — 题目09 测试车队日志策略 · 验收检查 Skill

> 关联产物：`AGENTS.md`（AC-01~19 速查）、`SKILL-business.md`（业务规则断言）、`AGENTS-git.md`（Git/部署规范）
> 验收基线：`09-测试车队日志策略-需求分析.md` v1.0（2026-11-13）§13 验收标准 + 开发规范截图（图三/图四/图五/图六）
> 适用范围：对题目 09 系统实现进行验收级检查，含需求实现、异常路径、测试证据与部署结果

---

## 1. 用途

1. 依据 AC-01~AC-19 对系统实现做**逐项验收**：每项给出 PASS/FAIL 判定与判定依据，产出可交付的验收报告。
2. 围绕验收点设计**可执行检查序列**（调用真实接口、读取真实数据、核对真实页面），覆盖正常闭环与全部异常分支。
3. 验收结论**必须基于真实工具/接口返回、数据库记录、页面截图、测试脚本输出等证据**；任何无法取得证据的项一律标「待补证据」，禁止虚构 PASS。
4. 检查部署结果与 Git 交付合规性（与 AGENTS-git.md 对齐：分支、PR、提交格式、meegle 校验、自动部署冒烟、审计索引）。

## 2. 触发场景

| 场景 | 触发条件 |
|---|---|
| 验收测试 | 系统实现完成后，需要按 AC-01~AC-19 逐项验收时 |
| 回归检查 | 修复缺陷后确认未破坏既有验收点 |
| 演示前预检 | 向评审演示前，验证四个演示场景可跑通 |
| 交付审计 | 提交交付物前核对验收证据、审计索引、Git/部署记录完整性 |

## 3. 输入

| 输入 | 说明 |
|---|---|
| 环境信息 | 服务地址、数据库连接、模拟器控制方式、演示用账号（申请人/负责人/运营/车端令牌） |
| 活动对象 | activityId、fleetId、车辆集合（V1~V5）、version_no、expected/effective interval |
| 接口清单 | 见需求分析 §9：applications/submit/approve/reject、activities/{id}/pilot/rollout/stop/restore/*/complete、vehicle-agent/config、vehicle-agent/receipts、vehicles、timeline |
| 验收基线 | AC-01~AC-19 判定表（本文件第 4 节） |
| 交付材料 | 分支/PR/提交记录、meegle 校验输出、部署流水线记录、冒烟结果、审计索引（AGENTS-git.md §6） |

## 4. 执行步骤

### Step 0 环境准备与前置校验
1. 确认服务已启动、数据库可访问、车端模拟器参数可控（在线/离线、回执延迟、失败注入）。
2. 准备三用户（申请人/负责人/运营）与车端令牌，确认会话令牌与角色隔离生效。
3. 若验收涉及超时分支，将超时阈值调短（如 5~10 秒）以便快速验证；记录原配置以便还原。
4. **前置合规**：核对该轮验收对应代码的 Git 记录（分支 dev/09、提交格式 `[Type] [Version] [ID] Description`、feat/fix 已过 meegle 校验、PR 合并后触发部署与冒烟）；不合规即记 FAIL-交付项，不进入业务验收。

### Step 1 逐项验收（AC-01 ~ AC-19）
按下表逐项执行，每项记录判定、证据、证据链接（接口响应文件 / 查询结果 / 截图路径）。

| 编号 | 验收点 | 检查序列 | 判定规则（PASS 条件） | 关键证据 |
|---|---|---|---|---|
| AC-01 | 正常推广闭环 | 创建申请→提交→负责人批准→运营试点V1/V2→双SUCCESS→推广V3~V5→全SUCCESS→负责人 complete | 5台全部 SUCCESS；活动 COMPLETED；逐车 effective_interval=30 | complete 接口响应、vehicles 核查数据 |
| AC-02 | 试点失败阻断推广 | 试点 V1 回执 FAILED（APPLY_ERROR）→ 调推广接口 | 活动 PAUSED；推广接口返回 PILOT_NOT_PASSED；未试点车（V3）拉取为空 + NO_PUBLISHED_VERSION | 推广接口响应、config 拉取响应 |
| AC-03 | 试点超时阻断推广 | 试点车不回执→等待超时阈值→检查状态 | 版本 TIMEOUT → 活动 PAUSED；生效值不变 | 核查视图结果=超时、effective 仍前值 |
| AC-04 | 退回后再提交 | 负责人退回（填理由）→申请人修改 reason/newInterval→再提交 | 两次提交快照并存、submit_count=2、新间隔生效 | 申请详情/快照查询 |
| AC-05 | 自审拦截 | 申请人同一账号调 approve 接口 | 返回 SELF_APPROVAL_FORBIDDEN，状态未变 | approve 接口响应 |
| AC-06 | 参数冻结 | 批准后调修改 newInterval 接口 | 返回 PARAM_FROZEN，数据未变 | 修改接口响应 + 数据查询 |
| AC-07 | 停止语义 | 试点 V2 成功、V1 失败→负责人停止→检查 | 未发车辆 NOT_SENT、已发记录保留；接口与页面文案检索不含"已撤销/已回滚"，统一为"已停止，车端仍保留已下发配置" | 停止响应、核查数据、页面文案截图 |
| AC-08 | 恢复全流程 | 运营发起恢复→负责人批准→运营执行→车辆确认→负责人判定恢复完成 | 恢复版本 version_no=历史最大+1、expected=各车前值；全确认后 RESTORED；逐车前值=生效值 | restore/* 接口响应、vehicles 数据 |
| AC-09 | 未确认禁止"全部恢复" | 恢复执行后 1 台车离线不回执→检查 | 活动保持 RESTORING；fullyRestored=false；界面显示"N 台未确认"；无"全部恢复"文案/标志 | 核查响应 fullyRestored 字段、页面截图 |
| AC-10 | 离线晚拉取 | 推广时 V5 离线→V5 上线拉取 | 上线后获得最新版本；不获得离线期间被替代的旧版本（SUPERSEDED 不下发） | config 拉取响应版本号 |
| AC-11 | 旧回执不覆盖 | 先成功 version2 后，提交 version1 旧回执 | 返回/记录 STALE_RECEIPT_IGNORED；生效值不变 | receipts 响应、核查视图生效值 |
| AC-12 | 重复回执幂等 | 同一 versionId 成功回执重复上报多次 | 生效值不变；duplicate_count 增加；无重复记账事件 | receipts 响应 + delivery_event 查询 |
| AC-13 | 跨车队/未知版本拒绝 | 上报未知 versionId、他车版本、跨车队版本 | 返回 RECEIPT_REJECTED；写审计（含原始载荷摘要）；任何车生效值不变 | receipts 响应、audit_log 查询 |
| AC-14 | 超时后成功 | 版本 TIMEOUT 后该车迟到成功回执 | 结果改判 SUCCESS；生效值更新；时间线保留 TIMEOUT + SUCCESS 两条 | timeline 查询 |
| AC-15 | 停止后迟到回执 | 停止后 V5 迟到回执 | 仅记录时间线；活动状态不变，无自动完成/重启 | timeline 查询 + 活动状态复核 |
| AC-16 | 逐车核查完整性 | 构造成功/失败/未确认混合场景→查核查视图 | 核查表同时含成功、失败、未确认行；每车字段齐全（前值/期望/生效/批次/版本/结果/失败原因/发布/拉取/回执时间）；未下发车显示 NOT_SENT | vehicles 响应字段逐项核对 |
| AC-17 | 完成门禁 | 存在未确认/失败时负责人调 complete | 返回 NOT_ALL_CONFIRMED；活动未进入 COMPLETED | complete 接口响应 |
| AC-18 | 车队单一活动 | 并发创建/批准第二个申请（活动未结束时） | 不产生两个未结束活动；批准/提交返回 FLEET_HAS_ACTIVE_ACTIVITY | 并发脚本输出 + activity 表查询 |
| AC-19 | 审计完整 | 走完一个完整生命周期→按活动/车辆/操作者过滤审计 | 申请/提交/退回/批准/选试点/启动/推广/超时/回执成功失败/拒绝/停止/恢复批准/恢复下发/恢复完成/完成 全部可查，含操作者/角色/时间/目标车辆/前后值/请求ID | audit_log 查询结果 |

### Step 2 异常路径与边界复核
在 Step 1 基础上补充（可与 Step 1 合并执行）：
1. **校验边界**：reason 501 字、newInterval 9/301/非整数 → 字段级错误；目标车辆非全 5 台 → 拒绝。
2. **鉴权越权**：无令牌 401；车端无凭证 401；请求体伪造 operatorId/applicantId/vehicleId → 服务端以会话令牌为准（忽略请求体）。
3. **幂等复核**：重复试点/推广/停止/恢复下发均返回当前状态、不生成新版本。
4. **重启一致性**：服务重启后状态/版本/回执不丢、不重放；超时扫描续跑（可作为附加验收项记录）。
5. **性能抽查**：单活动车辆核查接口 P95 ≤ 300ms；试点/推广/回执 P95 ≤ 200ms（5 台规模）。

### Step 3 演示场景串联验收
按需求分析 §10 完整串联执行四场景（正常推广 / 审批试点 / 恢复 / 网络分支），作为整体闭环证据；每场景结束时引用 Step 1 对应 AC 结论。

### Step 4 交付合规检查（与 AGENTS-git.md 对齐）
1. 分支：main + dev/09，无直推 main 记录。
2. 提交：格式合规、feat/fix 均过 meegle 校验且输出留存、无伪造 ID、无敏感数据。
3. PR：均有检查记录与合并记录，历史可追溯。
4. 部署：合并 main 后自动部署、冒烟通过、版本标识与提交 [Version] 对齐、含回滚记录（如需）。
5. 审计交付索引：docs/delivery-index.md 或 README 覆盖工作项/分支/提交/PR/验收/部署，不含敏感数据。

### Step 5 汇总报告
1. 逐项 AC-01~19 汇总 PASS/FAIL/待补证据，附证据链接。
2. 异常路径与交付合规结果独立成段。
3. 对 FAIL 项给出处置建议（见第 6 节）。

## 5. 输出格式

```
【验收检查报告】
项目：题目09 测试车队日志策略
验收时间：YYYY-MM-DD HH:MM:SS
验收环境：<服务地址 / 数据库 / 模拟器参数摘要>
验收基线：需求分析 v1.0（2026-11-13）§13 + AGENTS.md / AGENTS-git.md

【AC 验收结果】共 19 项
PASS: N 项   FAIL: M 项   待补证据: K 项

| 编号 | 验收点 | 结果 | 判定依据/证据 | 证据链接 |
|---|---|---|---|---|
| AC-01 | 正常推广闭环 | PASS | 5 台 SUCCESS、effective=30 | <文件/截图/URL> |
| ... | ... | ... | ... | ... |

【异常路径与边界】
| 检查项 | 结果 | 证据 |
|---|---|---|
| 字段边界校验 | PASS/FAIL | ... |
| 鉴权越权 | PASS/FAIL | ... |
| 幂等复核 | PASS/FAIL | ... |
| 重启一致性 | PASS/FAIL | ... |
| 性能抽查 | PASS/FAIL | P95 实测值 |

【演示场景串联】正常推广 / 审批试点 / 恢复 / 网络分支：PASS/FAIL（引用对应 AC）

【交付合规】（分支 / 提交 / meegle / PR / 部署冒烟 / 审计索引）
| 检查项 | 结果 | 证据 |
|---|---|---|
| 分支策略 | PASS/FAIL | ... |
| 提交格式与 meegle 校验 | PASS/FAIL | ... |
| PR 与合并记录 | PASS/FAIL | ... |
| 自动部署与冒烟 | PASS/FAIL | ... |
| 审计交付索引 | PASS/FAIL | ... |

【未通过项处置建议】
| 编号 | 问题描述 | 建议处置 | 复验方式 |
|---|---|---|---|
| ... | ... | ... | ... |

【总体结论】验收通过（19/19 PASS 且交付合规） / 有条件通过（修复指定 FAIL 后复验） / 验收不通过
```

## 6. 失败处理

| 失败类型 | 处置建议 |
|---|---|
| AC 项 FAIL（功能/规则不满足） | 记录具体失败现象与复现步骤；建议按根因修复后重新执行该项验收，禁止以"口头修复"代替复验 |
| 验收点无法取得真实证据 | 标记「待补证据」并说明缺失原因（环境不可用/接口未实现等）；未补齐前该点不得记为 PASS |
| 环境问题（服务未启动/数据库不可达/模拟器不可控） | 修复环境后重试 1 次；仍失败如实记录为环境阻断，不得伪造数据 |
| 交付不合规（提交/PR/部署/meegle 缺失） | 按 AGENTS-git.md 补齐流程记录后再验收；禁止用旧记录冒充本次证据 |
| 部分 AC 阻塞（如时间限制） | 明确列出未覆盖项并说明原因，输出「有条件通过」结论，绝不整体宣称 PASS |

## 7. 完成标准

1. AC-01~AC-19 逐项执行完毕，每项有明确结果（PASS / FAIL / 待补证据）与真实证据链接。
2. 异常路径与边界复核完成，无未说明的越权、覆盖、伪造、误导性文案问题。
3. 演示场景串联通过，映射 AC 结论一致。
4. 交付合规检查完成（分支/提交/meegle/PR/部署冒烟/审计索引）。
5. 所有 PASS 结论均有证据支撑；待补证据项 ≤ 可接受范围且已列明原因与补齐计划。
6. 报告含未通过项处置建议与总体结论，可交付评审。

---

## 8. AI 可使用的 Skill 文件说明与调用记录模板

### 8.1 Skill 文件配置方式（AI 工具侧）

**适用工具**：Claude Code / Cursor / Windsurf / Codex CLI 等支持 Skills/Agent Rules 的 AI 编程工具，以及支持 MCP Skill 注册的平台。

**方式 A（项目内 Skill 目录，推荐）**：保存为 `.claude/skills/acceptance/fleet-log-acceptance-skill.md` 或 `skills/fleet-log-acceptance/SKILL.md`（文件名与 front matter `name` 对应），AI 在验收类任务中自动可检索加载。

**方式 B（作为验收流程注入）**：将本文件内容加入交付仓库 `docs/acceptance/` 并写入 AGENTS.md 引用清单；验收时向 AI 声明「按 fleet-log-acceptance-skill 执行 AC-01~19 验收」。

**方式 C（手动触发）**：向 AI 提问附带指令「请加载 fleet-log-acceptance-skill，对 <环境信息> 执行逐项验收并输出报告」，AI 按本文件 Step 1~5 执行并产出第 5 节格式报告。

### 8.2 调用记录模板（提交验收用）

```
【Skill 调用记录】
Skill 名称：fleet-log-acceptance-skill
调用时间：YYYY-MM-DD HH:MM:SS
调用工具：<Claude Code / Cursor / 其他>
调用方式：<自动加载 / 手动指令 / 项目规则注入>
验收范围：AC-01 ~ AC-19（或部分：如 AC-07~AC-09）
环境摘要：<服务地址 / 超时阈值 / 模拟器参数>
执行结果：PASS N 项 / FAIL M 项 / 待补证据 K 项
关键证据留存：<接口响应文件路径 / 数据库查询输出 / 截图路径 / 部署记录链接>
总体结论：通过 / 有条件通过 / 不通过
```

调用记录建议统一维护在 `docs/skill-call-logs/acceptance-skill-log.md`，一次验收一段，与验收报告、审计索引一并提交。

---
*本 Skill 与 AGENTS.md（业务规则约束）、SKILL-business.md（业务专项断言）、AGENTS-git.md（Git/部署规范）配套使用；验收证据必须真实可追溯，禁止虚构 PASS。*
*（内容由AI生成，仅供参考）*
