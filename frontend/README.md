# Chatroom 前端

基于 React + TypeScript + Vite 构建的聊天室 Web 客户端，覆盖当前后端已实现的全部功能，包括用户认证、WebSocket 调试、管理员用户管理等。

---

## 技术栈

| 分类 | 选型 |
|------|------|
| 框架 | React 19 + TypeScript |
| 构建工具 | Vite 8 |
| 包管理器 | pnpm |
| UI 组件库 | Ant Design 5 |
| 全局状态管理 | Zustand（持久化至 localStorage） |
| HTTP 请求 | Axios（统一封装，自动携带 Cookie） |
| 日期处理 | dayjs |

---

## 目录结构

```
frontend/
├── public/                  # 静态资源
├── src/
│   ├── api/                 # 接口层
│   │   ├── http.ts          # axios 实例，统一前缀 /api/v1
│   │   ├── auth.ts          # 认证相关接口（注册/登录/登出/改密）
│   │   └── admin.ts         # 管理员接口（用户列表/封禁/解封等）
│   ├── components/
│   │   └── layout/
│   │       └── AppLayout.tsx  # 全局布局（侧边栏 + 顶栏 + 内容区）
│   ├── hooks/
│   │   └── useWebSocket.ts  # WebSocket 封装 Hook（含客户端心跳）
│   ├── pages/
│   │   ├── auth/
│   │   │   └── AuthPage.tsx   # 登录 / 注册页
│   │   ├── chat/
│   │   │   └── ChatTestPage.tsx  # WebSocket 调试页（Echo + 流式）
│   │   └── admin/
│   │       └── AdminPage.tsx  # 用户管理页（仅管理员可见）
│   ├── store/
│   │   └── authStore.ts     # 认证状态（Zustand + persist）
│   ├── types/
│   │   └── index.ts         # 全局 TypeScript 类型定义
│   ├── App.tsx              # 根组件（按认证状态路由至登录页或主布局）
│   ├── main.tsx             # 应用入口
│   └── index.css            # 全局样式重置
├── index.html
├── vite.config.ts           # Vite 配置（含开发代理）
├── tsconfig.app.json
├── tsconfig.node.json
└── package.json
```

---

## 快速开始

### 前置条件

- Node.js >= 18
- pnpm >= 8（`npm install -g pnpm`）
- 后端服务已启动并监听 `localhost:8080`

### 安装依赖

```bash
cd frontend
pnpm install
```

### 启动开发服务器

```bash
pnpm dev
```

启动后访问 [http://localhost:5173](http://localhost:5173)。

开发服务器通过 Vite 代理将以下路径自动转发至后端，无需手动处理跨域：

| 前端路径 | 转发目标 | 说明 |
|----------|----------|------|
| `/api/*` | `http://localhost:8080` | REST 接口 |
| `/websocket_test` | `ws://localhost:8080` | Echo WebSocket |
| `/websocket_stream` | `ws://localhost:8080` | 流式 WebSocket |
| `/ws` | `ws://localhost:8080` | 主 WebSocket（预留） |

### 构建生产包

```bash
pnpm build
```

产物输出至 `dist/` 目录。

### 代码检查

```bash
pnpm lint
```

---

## 功能说明

### 登录 / 注册（AuthPage）

- 未登录时自动展示认证页，登录成功后进入主界面
- 注册时对用户名和密码进行前端格式校验
  - 用户名：3~30 位字母、数字或下划线
  - 密码：同时包含字母与数字，6 位以上
- 认证状态通过 Zustand `persist` 持久化至 `localStorage`，刷新页面后免重新登录

### 主布局（AppLayout）

- 左侧侧边栏菜单，当前支持两个页面：
  - **WS 调试**：所有用户均可访问
  - **用户管理**：仅 `role === 'admin'` 的用户可见
- 顶栏展示当前登录用户昵称和角色标识，提供退出登录按钮

### WebSocket 调试页（ChatTestPage）

包含两个标签页：

**Echo 调试（/websocket_test）**
- 可手动连接 / 断开 WebSocket
- 发送任意文字，服务端返回逐字符反转内容（支持中文）
- 消息以气泡形式展示（蓝色：自发，灰色：服务端）
- 连接状态实时显示（已连接 / 已断开）

**流式输出（/websocket_stream）**
- 发送内容后，服务端以 20 字/秒的速度将反转结果逐字符流式推送
- 使用 JSON 帧协议：`{"type":"chunk","content":"X"}` / `{"type":"done"}`
- 输出区展示打字机效果，末尾显示闪烁光标 `▌`

### 用户管理页（AdminPage）

仅管理员可访问，功能包括：

| 操作 | 说明 |
|------|------|
| 查询用户列表 | 支持按状态（全部 / 正常 / 封禁 / 已注销）和关键词筛选，分页展示 |
| 封禁用户 | 填写封禁原因，可选截止时间（不填则永久封禁） |
| 解封用户 | 将封禁用户恢复为正常状态 |
| 注销用户 | 软删除 |
| 恢复用户 | 将已注销用户恢复为正常状态 |
| 重置密码 | 为指定用户设置新密码 |
| 踢下线 | 强制使指定用户的当前 WebSocket 连接断开 |

---

## WebSocket 心跳机制

客户端通过 `useWebSocket` Hook 管理连接：

- 服务端每 **30 秒**发送一次 Ping 帧
- 若 **35 秒**内未收到 Pong 响应，服务端主动断开连接
- 客户端每 **25 秒**额外发送一次文本心跳包 `__ping__`，服务端回复 `__gnip__`（前端自动过滤，不展示给用户）

---

## 状态管理说明

| Store | 文件 | 持久化 | 说明 |
|-------|------|--------|------|
| authStore | `src/store/authStore.ts` | 是（localStorage） | 存储当前用户信息和 sessionId |

---

## 环境要求与注意事项

- 前端开发阶段通过 Vite 代理访问后端，**生产部署时需自行配置 Nginx 反向代理或同源部署**
- `sessionId` 通过 Cookie（`HttpOnly`）传递，Axios 已配置 `withCredentials: true`
- Ant Design 组件库体积较大（gzip 后约 373 KB），后续可按需引入优化
