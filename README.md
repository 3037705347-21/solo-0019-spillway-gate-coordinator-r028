# 泄洪闸门操作协调服务

这是一个无外部依赖的 Go 后端，用本地内存仓库协调闸门登记、开度调整命令、安全联锁复核、值班岗位分配和执行结果。服务强调单闸门活动命令互斥、原子状态转换、执行令牌生命周期、值班覆盖校验和稳定错误传播。

## 运行

```sh
go build ./...
go run ./cmd/server --addr 127.0.0.1:18119
```

健康入口为 `GET http://127.0.0.1:18119/health`。

## API

```text
POST /gates
GET  /gates/{id}
POST /commands
GET  /commands/{id}
POST /commands/{id}/interlock
POST /commands/{id}/execute
POST /operators
GET  /operators/{id}
POST /operators/{id}/revoke
POST /areas
GET  /areas/{id}
POST /shifts
GET  /shifts/{id}
POST /shifts/{id}/end
POST /assignments
GET  /assignments/{id}
```

登记闸门示例：

```json
{
  "id": "gate-east",
  "name": "东侧泄洪闸",
  "max_aperture": 4.5,
  "initial_aperture": 0.5
}
```

提交命令示例：

```json
{
  "gate_id": "gate-east",
  "target_aperture": 2.25,
  "reason": "上游水位调整",
  "requested_by": "dispatcher-lin"
}
```

联锁复核示例：

```json
{
  "verdict": "clear",
  "verifier": "safety-wu",
  "note": "上下游条件满足"
}
```

## 值班分配与岗位覆盖

登记操作员资格不等于当班。启动闸门移动前，值班负责人需要先建立完整的值班链路：

1. `POST /operators` 登记合格操作员（可 `POST /operators/{id}/revoke` 注销资格）。
2. `POST /areas` 创建控制区域并绑定其负责的闸门，闸门必须已登记且未归属其他区域。
3. `POST /shifts` 创建班次窗口；`starts_at` 省略时立即开始，可通过 `POST /shifts/{id}/end` 提前结束。
4. `POST /assignments` 把一名合格操作员分配到某班次的某控制区域；同一班次同一区域只能有一名负责人。

登记操作员：

```json
{"id": "operator-chen", "name": "陈操作"}
```

创建控制区域：

```json
{"id": "area-east", "name": "东侧控制区", "gate_ids": ["gate-east"]}
```

创建班次：

```json
{
  "id": "shift-day-1",
  "name": "白班",
  "starts_at": "2026-10-02T00:00:00Z",
  "ends_at": "2026-10-02T12:00:00Z"
}
```

岗位分配：

```json
{"shift_id": "shift-day-1", "area_id": "area-east", "operator_id": "operator-chen"}
```

只有在当前时刻班次处于 active、该班次的区域绑定包含目标闸门、且请求执行的操作员正是该岗位负责人时，才能执行命令。执行请求使用 `operator_id`：

```json
{
  "operator_id": "operator-chen",
  "token": "复核响应中的 execution_token",
  "simulate_fault": false
}
```

明确的失败结果（均为稳定错误码）：

- `operator_not_found` / `operator_not_qualified`：操作员未登记或资格已注销。
- `duty_unattended`：闸门没有控制区域，或当前没有任何当班岗位覆盖它。
- `duty_shift_not_started` / `duty_shift_ended`：只有未开始或已结束的班次覆盖该闸门。
- `duty_coverage_conflict`：交班窗口内多个活动班次同时覆盖，无人能单独负责。
- `operator_not_covering_gate`：操作员有资格但不是当前班次该闸门的负责人。
- `duplicate_assignment` / `duplicate_area_assignment`：同一班次同一区域重复分配。
- `shift_already_ended`：重复结束已经结束的班次。

移动启动时，班次、区域、分配和操作员事实会冻结为命令上的 `execution_duty`；交班、班次结束或资格注销后，历史命令仍显示执行当时的当班事实（completed 与 aborted 均保留）。

## 目录

- `cmd/server`：服务进程。
- `cmd/workflowcheck`：工作流的真实 HTTP 检查入口。
- `internal/bootstrap`：依赖装配。
- `internal/contracts`：公开数据契约。
- `internal/coordination`：业务工作流。
- `internal/domain`：实体、不变量和状态机。
- `internal/httpapi`：路由、解码和错误映射。
- `internal/storage`：互斥保护的内存仓库。

## 工作流检查

```sh
go run ./cmd/workflowcheck --workflow register-gate
go run ./cmd/workflowcheck --workflow request-gate-command
go run ./cmd/workflowcheck --workflow review-safety-interlock
go run ./cmd/workflowcheck --workflow execute-gate-command
go run ./cmd/workflowcheck --workflow duty-coverage
```

这些检查会创建临时 HTTP 服务并调用真实路由。它们不调用外部网络，也不依赖数据库。`duty-coverage` 覆盖无人值守、未开始/已结束班次、交班竞争、重复分配、资格注销、故障路径当班快照和历史命令当班事实。

## 测试状态

测试有意延后。当前交付仅包含生产级 smoke 检查；后续任务阶段会添加自动化测试与红绿验证，不应把当前没有测试文件视为项目未完成。
