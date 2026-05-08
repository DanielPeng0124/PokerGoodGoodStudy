# poker-frontend

React + TypeScript + Vite 前端，配套 `poker-backend` 使用。

## 启动

```bash
npm install
npm run dev
```

默认后端地址：`http://localhost:8080`。

如需修改：

```bash
VITE_API_BASE=http://localhost:8080 npm run dev
```

## 功能

- 创建房间
- 加入房间
- 选择座位和 buy-in
- WebSocket 实时同步桌面状态
- 展示公共牌、底池、玩家状态、当前行动位
- fold / check / call / bet / raise / all-in
- 聊天

## 注意

这是娱乐筹码 MVP 前端，不包含真钱支付、KYC、反洗钱、监管合规等能力。
