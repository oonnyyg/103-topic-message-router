# Topic Message Router

一个本地消息路由工具（仅 Go 标准库，无网络服务）。程序从标准输入逐行读取
JSONL 命令，模拟客户端的订阅、取消订阅与发布，并向标准输出逐行返回 JSON
结果。订阅关系与保留消息持久化在本地状态文件中，重启后继续使用。

## 构建与运行

```sh
go build -o router .
./router -state router-state.json   # -state 指定状态文件，默认 router-state.json
```

运行测试：`go test ./...`

## 输入格式（JSONL，每行一条命令）

```json
{"cmd":"subscribe","client":"c1","filter":"sensors/+"}
{"cmd":"unsubscribe","client":"c1","filter":"sensors/+"}
{"cmd":"publish","topic":"sensors/temp","payload":"22.5","retain":true}
{"cmd":"query","what":"subscriptions"}
{"cmd":"query","what":"retained"}
```

- `subscribe`：为客户端添加过滤器，返回该过滤器当前匹配的保留消息回放
  （`replay`，按主题排序）。
- `unsubscribe`：移除过滤器；未订阅过时返回错误。
- `publish`：投递消息，返回接收客户端列表（`delivered`，按客户端标识排序）。
  `retain:true` 时更新保留消息。
- `query`：查询当前订阅快照或保留消息快照。

每条命令输出一行结果，成功为 `{"ok":true,...}`，失败为
`{"ok":false,"error":"..."}`。非法命令不会改变已有状态。

## 路由规则

- 主题按 `/` 分层，空层级被保留且参与匹配，比较区分大小写。
- 发布主题不能为空，也不能包含 `+` 或 `#`。
- 过滤器中 `+` 必须占据完整层级，且只匹配一层；`#` 必须是完整层级且只能
  出现在最后一层，匹配零层或多层。
- 首层为通配符（`+` 或 `#`）的过滤器不匹配以 `$` 开头的主题。
- 同一客户端的多个过滤器同时命中时，每次发布只投递一次。
- 重复订阅同一过滤器不会增加订阅数量，但仍会触发该过滤器的保留消息回放。
- 保留发布会替换该主题的旧保留消息；空载荷的保留发布会清除旧消息，但仍
  投递给当前订阅者。普通发布不改变保留内容。
- 新订阅只回放当前匹配的保留消息，回放按主题排序，消息内容原样投递不做
  改写。
- 发布结果按客户端标识排序。
