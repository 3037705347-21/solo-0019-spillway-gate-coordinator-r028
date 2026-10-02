# 泄洪闸门操作协调服务

## 项目目标

该服务用于水工控制室协调泄洪闸门的命令申请、安全联锁复核和单次开度调整。系统在同一闸门上只允许一条已通过复核且尚未结束的执行命令，任何冲突都必须被显式拒绝，不能覆盖已经占用的闸门。闸门移动还受值班分配约束：只有当前班次内实际覆盖该闸门所在控制区域、且资格有效的操作员才能启动移动；无人值守、交班竞争、重复分配和已经结束的班次都返回明确结果，执行命令永久保留执行当时的当班事实。

## 使用角色

- 运行协调员：登记闸门并提交开度调整命令。
- 安全复核员：提交联锁结论，决定命令是否通过。
- 值班负责人：维护操作员资格、控制区域、班次与岗位分配。
- 闸门操作员：在本人当班且覆盖目标闸门时，凭执行令牌启动命令并登记执行结果。

## 核心实体

- `gate`：闸门及其当前开度、运行状态、最大开度、所属控制区域和活动命令。
- `gate_command`：一次开度调整申请，包含目标开度、申请原因、状态、执行令牌以及执行时冻结的当班事实。
- `interlock_decision`：安全复核结论，状态为 clear 或 blocked。
- `operator`：控制室操作员资格记录，登记资格不代表当前当班，资格可注销。
- `control_area`：控制区域，即岗位负责范围；区域创建时原子绑定一组此前未归属的闸门。
- `shift`：值班班次窗口，包含计划起止时间，可提前结束；在任意时刻处于 scheduled、active 或 ended 阶段。
- `shift_assignment`：把一名合格操作员在一个班次内分配到一个控制区域，同一班次同一区域只能有一名负责人。

## 业务流程

### register-gate

运行协调员通过 `POST /gates` 登记闸门。服务校验标识、名称、最大开度和初始开度，写入空闲状态；重复标识返回稳定冲突错误，不覆盖原记录。

### request-gate-command

运行协调员通过 `POST /commands` 提交开度调整命令。服务读取闸门当前开度和限制，校验目标开度与申请信息，创建待复核命令并冻结起始开度。

### review-safety-interlock

安全复核员通过 `POST /commands/{id}/interlock` 提交复核结论。clear 结论会原子占用闸门并签发一次执行令牌；blocked 结论会把命令置为 rejected。其他命令先占用同一闸门时，当前 clear 结论返回冲突且不改变闸门。

### execute-gate-command

闸门操作员通过 `POST /commands/{id}/execute` 提交执行令牌。服务在同一临界区内解析值班覆盖关系、校验命令、令牌、闸门占用关系和目标开度，并完成 approved、executing、completed 或 aborted 的转换，同步更新闸门状态。覆盖解析按固定优先级给出明确结果：操作员未登记（`operator_not_found`）、资格失效（`operator_not_qualified`）、闸门无区域或区域不含闸门（`duty_unattended`）、覆盖班次尚未开始（`duty_shift_not_started`）、覆盖班次已结束（`duty_shift_ended`）、多个活动班次同时覆盖（`duty_coverage_conflict`）、操作员不是当前班次该区域负责人（`operator_not_covering_gate`）。移动启动成功时，班次、区域、分配和操作员事实被冻结到命令上；此后交班、班次结束或资格注销都不改变历史命令。令牌在命令终态后失效，不能复用。

### register-operator

值班负责人通过 `POST /operators` 登记操作员资格。重复标识返回稳定冲突。资格可通过 `POST /operators/{id}/revoke` 注销；注销后既不能新建岗位分配，也不能再启动移动。

### create-control-area

值班负责人通过 `POST /areas` 创建控制区域并给出负责的闸门清单。清单内闸门必须全部存在且尚未归属其他区域；任一门禁不满足时整体拒绝，不留下部分绑定。闸门一经绑定不能改挂其他区域。

### create-shift

值班负责人通过 `POST /shifts` 维护班次窗口（`starts_at` 省略时立即开始）。可通过 `POST /shifts/{id}/end` 提前结束班次，重复结束返回 `shift_already_ended`。重叠班次允许登记以支持交班窗口，但同一区域在移动启动时刻被多个活动班次覆盖时明确拒绝。

### assign-duty

值班负责人通过 `POST /assignments` 把合格操作员分配到某班次的某控制区域。班次、区域、操作员必须存在且操作员资格有效；同一班次同一区域重复分配同一操作员返回 `duplicate_assignment`，由其他操作员抢占同一岗位返回 `duplicate_area_assignment`。

## 状态与规则

- `gate` 状态：idle、reserved、moving、faulted。
- `gate_command` 状态：pending、approved、rejected、executing、completed、aborted。
- `shift` 阶段：scheduled（未到开始时间）、active（开始时间至实际结束时间之前）、ended（到达结束时间或已提前结束）。
- 只有 pending 命令可以复核。
- 只有 approved 命令可以执行，且执行令牌必须匹配。
- 只有当前班次内实际覆盖目标闸门、且资格有效的负责操作员可以启动移动；覆盖解析与命令、闸门状态转换在同一临界区内完成，交班无法插入其间。
- 移动启动时冻结当班事实（班次、区域、分配、操作员及名称），completed 与 aborted 命令都保留该事实。
- clear 复核与闸门占用必须同时成功；任一步失败时两者都不写入。
- blocked 复核使命令 rejected，不会占用闸门。
- 模拟故障会令命令 aborted、闸门 faulted，并保留最终开度。
- 已完成或已中止的命令不能再次执行。
- 控制区域原子绑定闸门：未知或已归属闸门使整个区域创建失败。
- 同一班次同一区域至多一条岗位分配；操作员资格注销后不能被分配也不能执行。

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
- `POST /commands`
- `GET /commands/{id}`
- `POST /commands/{id}/interlock`
- `POST /commands/{id}/execute`
- `POST /operators`
- `GET /operators/{id}`
- `POST /operators/{id}/revoke`
- `POST /areas`
- `GET /areas/{id}`
- `POST /shifts`
- `GET /shifts/{id}`
- `POST /shifts/{id}/end`
- `POST /assignments`
- `GET /assignments/{id}`

错误响应统一为：

```json
{"code":"stable_error_code","message":"可读说明"}
```

## 验证计划

本初始化基线刻意不生成单元测试、夹具或浏览器测试文件，也不声明 `test_command`。后续任务阶段负责补充测试，并在该阶段建立红绿验证。当前每条工作流通过 `cmd/workflowcheck` 启动真实 HTTP 服务并调用公开接口，校验成功状态、持久化结果和关键失败分支。
