# 泄洪闸门操作协调服务

这是一个无外部依赖的 Go 后端，用本地内存仓库协调闸门登记、开度调整命令、安全联锁复核、值班岗位覆盖和执行结果。服务强调单闸门活动命令互斥、原子状态转换、执行令牌生命周期、当班资格校验和稳定错误传播。

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
GET  /gates/{id}/coverage
POST /commands
GET  /commands/{id}
POST /commands/{id}/interlock
POST /commands/{id}/execute
POST /operators
GET  /operators/{id}
POST /operators/{id}/revoke
POST /shifts
GET  /shifts/{id}
POST /shifts/{id}/assignments
POST /shifts/{id}/end
```

### 值班分配与岗位覆盖

一次性执行令牌之外，移动闸门还必须满足当班要求：

1. 值班负责人登记操作员资格（可选 `valid_until` 有效期）：

```json
{"id": "operator-chen", "name": "陈操作"}
```

2. 维护班次：指定 RFC3339 起止时间和负责的闸门范围，并把当班执行人加入班次：

```json
{
  "id": "shift-day-1",
  "label": "白班一组",
  "starts_at": "2026-10-02T08:00:00Z",
  "ends_at": "2026-10-02T16:00:00Z",
  "gate_ids": ["gate-east"]
}
```

```json
{"operator_id": "operator-chen"}
```

3. `POST /shifts/shift-day-1/end` 可提前交班；`GET /gates/gate-east/coverage`
   返回 `covered`、`unattended` 或 `contended` 的岗位覆盖结论。

只有资格有效、且在覆盖该闸门的 active 班次名单内的操作员才能启动移动。
无人值守（`gate_unattended`）、交班竞争（`shift_handover_contention`）、
旧班次已结束（`shift_already_ended`）、操作员不在当班机组
（`operator_not_on_duty`）、重复分配（`duplicate_shift_assignment`）和资格失效
（`operator_credential_invalid`）都返回明确冲突，状态不发生任何变化。
命令执行时会把班次、操作员、闸门和校验时间冻结为 `duty` 快照，
交班或吊销资格不会改写历史命令上的当班事实。

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

执行示例：

```json
{
  "operator": "operator-chen",
  "token": "复核响应中的 execution_token",
  "simulate_fault": false
}
```

## 目录

- `cmd/server`：服务进程。
- `cmd/workflowcheck`：四条工作流的真实 HTTP 检查入口。
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

这些检查会创建临时 HTTP 服务并调用真实路由。它们不调用外部网络，也不依赖数据库。

## 测试状态

测试有意延后。当前交付仅包含初始化基线和生产级 smoke 检查；后续任务阶段会添加自动化测试与红绿验证，不应把当前没有测试文件视为项目未完成。
