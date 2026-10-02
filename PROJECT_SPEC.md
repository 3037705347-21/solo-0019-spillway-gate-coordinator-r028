# 泄洪闸门操作协调服务

## 项目目标

该服务用于水工控制室协调泄洪闸门的命令申请、安全联锁复核和单次开度调整。系统在同一闸门上只允许一条已通过复核且尚未结束的执行命令，任何冲突都必须被显式拒绝，不能覆盖已经占用的闸门。

## 使用角色

- 运行协调员：登记闸门并提交开度调整命令。
- 安全复核员：提交联锁结论，决定命令是否通过。
- 闸门操作员：登记操作员资格，凭执行令牌启动命令并登记执行结果。
- 值班负责人：维护操作员资格、值班班次、当班执行人和负责闸门范围。

## 核心实体

- `gate`：闸门及其当前开度、运行状态、最大开度和活动命令。
- `gate_command`：一次开度调整申请，包含目标开度、申请原因、状态、执行令牌和启动移动时冻结的当班事实。
- `interlock_decision`：安全复核结论，状态为 clear 或 blocked。
- `operator`：操作员资格登记，状态为 active 或 revoked，可带有效期。
- `duty_shift`：值班班次，包含起止窗口、负责闸门范围、当班执行人名单和可能的提前结束时间。

## 业务流程

### register-gate

运行协调员通过 `POST /gates` 登记闸门。服务校验标识、名称、最大开度和初始开度，写入空闲状态；重复标识返回稳定冲突错误，不覆盖原记录。

### request-gate-command

运行协调员通过 `POST /commands` 提交开度调整命令。服务读取闸门当前开度和限制，校验目标开度与申请信息，创建待复核命令并冻结起始开度。

### review-safety-interlock

安全复核员通过 `POST /commands/{id}/interlock` 提交复核结论。clear 结论会原子占用闸门并签发一次执行令牌；blocked 结论会把命令置为 rejected。其他命令先占用同一闸门时，当前 clear 结论返回冲突且不改变闸门。

### execute-gate-command

闸门操作员通过 `POST /commands/{id}/execute` 提交执行令牌。服务校验命令、令牌、闸门占用关系和目标开度，并在同一临界区内解析当前当班事实：操作员资格必须存在且 active 未过期，必须有一条未结束的当班班次覆盖该闸门，且操作员在该班次的执行人名单中。校验通过后命令进入 executing，并把班次、操作员、负责闸门和校验时间作为当班快照冻结在命令上，随后完成 approved、executing、completed 或 aborted 的转换并同步更新闸门状态。令牌在命令结束后失效，不能复用；当班快照不随后交班、改派或资格吊销而改写。

### register-operator

值班负责人通过 `POST /operators` 登记操作员资格，可提供 RFC3339 的 `valid_until`；重复标识返回稳定冲突。`POST /operators/{id}/revoke` 吊销资格，吊销是单向终态，不影响已完成命令上冻结的当班事实。

### manage-duty-shift

值班负责人通过 `POST /shifts` 创建班次（RFC3339 起止时间、负责闸门列表），通过 `POST /shifts/{id}/assignments` 维护当班执行人，通过 `POST /shifts/{id}/end` 提前交班。同一闸门的两个班次在有效窗口上重叠时（仅端点相接不算重叠），创建新班次返回 `shift_handover_contention`，必须先结束旧班次；把同一操作员重复加入班次返回 `duplicate_shift_assignment`；向已结束班次加人或再次结束返回 `shift_already_ended`。`GET /gates/{id}/coverage` 返回某闸门当前是 covered、unattended 还是 contended。

## 状态与规则

- `gate` 状态：idle、reserved、moving、faulted。
- `gate_command` 状态：pending、approved、rejected、executing、completed、aborted。
- `operator` 状态：active、revoked；存在 `valid_until` 时，到点即视为资格失效。
- `duty_shift` 派生状态：scheduled、active、ended；提前结束后有效窗口截断到结束时间。
- 只有 pending 命令可以复核。
- 只有 approved 命令可以执行，且执行令牌必须匹配。
- 只有“资格有效 + 被一条 active 班次覆盖 + 在该班次名单内”的操作员才能启动移动。
- 无班次覆盖闸门返回 `gate_unattended`；多条 active 班次竞争同一闸门返回 `shift_handover_contention`；操作员所在覆盖班次已结束返回 `shift_already_ended`；操作员不属于当班机组返回 `operator_not_on_duty`；资格缺失、被吊销或过期返回 not found 或 `operator_credential_invalid`。
- clear 复核与闸门占用必须同时成功；任一步失败时两者都不写入。
- blocked 复核使命令 rejected，不会占用闸门。
- 模拟故障会令命令 aborted、闸门 faulted，并保留最终开度。
- 已完成或已中止的命令不能再次执行。
- 命令上的当班快照在执行瞬间写入，历史命令永远保留执行当时的当班事实。

## 模块边界

- `cmd/server`：进程入口、监听地址和退出控制。
- `internal/httpapi`：HTTP 路由、JSON 解码、状态码映射。
- `internal/coordination`：工作流编排、命令标识和执行令牌签发。
- `internal/domain`：实体、不变量和状态转换规则。
- `internal/storage`：带互斥保护的内存仓库和原子写入。
- `internal/contracts`：API 请求、响应和稳定错误结构。
- `internal/bootstrap`：运行时依赖装配。

## 接口

- `GET /health`
- `POST /gates`
- `GET /gates/{id}`
- `GET /gates/{id}/coverage`
- `POST /commands`
- `GET /commands/{id}`
- `POST /commands/{id}/interlock`
- `POST /commands/{id}/execute`
- `POST /operators`
- `GET /operators/{id}`
- `POST /operators/{id}/revoke`
- `POST /shifts`
- `GET /shifts/{id}`
- `POST /shifts/{id}/assignments`
- `POST /shifts/{id}/end`

错误响应统一为：

```json
{"code":"stable_error_code","message":"可读说明"}
```

## 验证计划

本初始化基线刻意不生成单元测试、夹具或浏览器测试文件，也不声明 `test_command`。后续任务阶段负责补充测试，并在该阶段建立红绿验证。当前每条工作流通过 `cmd/workflowcheck` 启动真实 HTTP 服务并调用公开接口，校验成功状态、持久化结果和关键失败分支。
