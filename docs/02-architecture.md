# 02. 系统架构设计

## 1. 整体架构图

```mermaid
flowchart TB
    subgraph Client["客户端 / 上游"]
        FE[前端 / App]
    end

    subgraph Gateway["接入层"]
        EG["event-gateway<br/>:8081"]
    end

    subgraph MQ["消息队列"]
        K1[["Kafka Topic<br/>events.raw"]]
        K2[["Kafka Topic<br/>events.processed"]]
    end

    subgraph Workers["异步处理层"]
        CW["conversion-worker<br/>转化回传"]
        PW["pipeline-worker<br/>数据管道"]
    end

    subgraph Async["任务队列 (asynq)"]
        RQ[("Redis<br/>:6380")]
    end

    subgraph AdPlatforms["外部广告平台"]
        META[Meta CAPI]
        GOOGLE[Google Ads]
        TIKTOK[TikTok Events API]
    end

    subgraph ExpPlatform["实验平台"]
        ES["experiment-service<br/>:8082"]
    end

    subgraph Admin["管理后台"]
        AA["admin-api<br/>:8083"]
        AW["admin-web<br/>:3000"]
        AM["asynqmon<br/>:8090"]
    end

    subgraph Storage["存储层"]
        DB[("MySQL<br/>:3306")]
    end

    FE -->|"POST /v1/events"| EG
    FE -->|"GET /v1/experiments/:key/assign"| EG
    EG -->|publish| K1
    EG -->|读写分桶| ES
    ES --> DB

    K1 --> CW
    K1 --> PW
    CW -->|写 events/conversions| DB
    CW -->|失败重试任务| RQ
    CW -->|publish processed| K2
    CW -->|S2S 回传| META
    CW -->|S2S 回传| GOOGLE
    CW -->|S2S 回传| TIKTOK

    K2 --> PW
    PW -->|写 daily_funnel_stats| DB
    PW -->|定时任务| RQ

    AW --> AA
    AA --> DB
    AA -->|代理| ES
    AA -->|查询任务状态| RQ
    AM --> RQ
```

## 2. 技术栈与选型理由

| 组件 | 选型 | 理由 |
|---|---|---|
| 语言 | Go 1.25 | 高并发 I/O 密集场景表现好，编译产物单一二进制，适合多服务微服务化部署 |
| HTTP 框架 | Gin | 轻量、生态成熟，团队熟悉度高 |
| 消息队列 | Kafka | 需要多消费者独立消费同一份事件流（回传 + 管道两条线），且要求可重放；对比 Redis Stream/RabbitMQ，Kafka 在"多消费者组独立 offset + 高吞吐持久化"上更合适 |
| 异步任务队列 | asynq (Redis) | 用于"失败重试"和"定时任务"这类偏任务调度语义的场景，和 Kafka 的"事件流"语义分工明确，不混用 |
| 数据库 | MySQL 8.0 | 团队现有运维能力覆盖，事务/外键约束满足强一致性需求（如 conversions 外键关联 events） |
| 迁移工具 | golang-migrate | 版本化 schema 迁移，`migrations/0001_init.up.sql` / `.down.sql` 成对提供回滚能力 |
| 后台前端 | Next.js (App Router) | Server Component 直接查后端聚合数据，无需额外 BFF 层 |

## 3. 服务清单与端口

| 服务 | 类型 | 端口（宿主机） | 职责 |
|---|---|---|---|
| `event-gateway` | HTTP | 8081→8080 | 事件打点接收、实验分流请求入口 |
| `conversion-worker` | Kafka 消费者 + asynq worker | 无对外端口 | 消费 `events.raw`，落库 + 触发广告转化回传，失败重试 |
| `pipeline-worker` | Kafka 消费者 + asynq scheduler/worker | 无对外端口 | 消费 `events.processed`，定时汇总每日漏斗统计 |
| `experiment-service` | HTTP | 8082→8080 | 实验/变体 CRUD、统计显著性与 SRM 检测 |
| `admin-api` | HTTP | 8083→8080 | 后台聚合查询、代理转发实验服务、queue 状态查询 |
| `admin-web` | Next.js | 3000 | 可视化后台 |
| `asynqmon` | 第三方 UI | 8090 | Redis 任务队列可视化（内置组件，非本项目自研） |
| `mysql` / `redis` / `kafka` | 基础设施 | 3306 / 6380 / 9092 | 存储与中间件 |

## 4. 数据模型（ER 图）

```mermaid
erDiagram
    campaigns ||--o{ conversions : "campaign_id"
    events ||--o{ conversions : "event_id"
    experiments ||--o{ experiment_variants : "experiment_id"
    experiments ||--o{ experiment_assignments : "experiment_id"
    experiment_variants ||--o{ experiment_assignments : "variant_id"

    campaigns {
        bigint id PK
        varchar name
        varchar ad_platform
        varchar external_campaign_id UK
        int attribution_window_days
        bool is_active
    }
    events {
        bigint id PK
        varchar event_name
        varchar user_id
        varchar click_id
        varchar ad_platform
        bigint campaign_id
        json properties
        timestamp occurred_at
        timestamp received_at
    }
    conversions {
        bigint id PK
        bigint event_id FK
        varchar ad_platform
        bigint campaign_id
        varchar status "pending/success/failed"
        int attempt_count
        json callback_payload
    }
    experiments {
        bigint id PK
        varchar exp_key UK
        varchar status
        int traffic_allocation "灰度百分比 0-100"
    }
    experiment_variants {
        bigint id PK
        bigint experiment_id FK
        varchar variant_key
        int weight
    }
    experiment_assignments {
        bigint id PK
        bigint experiment_id FK
        bigint variant_id FK
        varchar user_id
        timestamp assigned_at
    }
    daily_funnel_stats {
        bigint id PK
        date stat_date
        bigint campaign_id
        varchar event_name
        bigint event_count
    }
```

## 5. 关键数据流时序图

### 5.1 事件上报 → 转化回传

```mermaid
sequenceDiagram
    participant FE as 前端
    participant EG as event-gateway
    participant K as Kafka(events.raw)
    participant CW as conversion-worker
    participant DB as MySQL
    participant AD as 广告平台

    FE->>EG: POST /v1/events
    EG->>K: Publish(事件JSON)
    EG-->>FE: 202 Accepted
    K-->>CW: FetchMessage(不自动提交)
    CW->>DB: INSERT events
    alt 有click_id/campaign_id/ad_platform
        CW->>DB: INSERT conversions(status=pending)
        CW->>AD: 转化回传请求
        alt 回传成功
            CW->>DB: UPDATE conversions SET status=success
        else 回传失败
            CW->>DB: UPDATE conversions SET status=failed
            CW->>CW: enqueueRetry(asynq)
        end
    end
    CW->>K: CommitMessages(处理成功才提交)
```

### 5.2 实验分流

```mermaid
sequenceDiagram
    participant FE as 前端
    participant EG as event-gateway
    participant DB as MySQL

    FE->>EG: GET /v1/experiments/:key/assign?user_id=xxx
    EG->>DB: 查 traffic_allocation
    EG->>EG: hash(key+":holdout", user_id) % 100 >= 灰度比例?
    alt 命中holdout（不参与实验）
        EG-->>FE: {in_experiment: false, variant: null}
    else 参与实验
        EG->>DB: 查 variants + weight
        EG->>EG: 按累积权重映射 bucket 到具体 variant
        EG->>DB: INSERT/忽略 experiment_assignments（幂等）
        EG->>K: Publish experiment_exposure 事件
        EG-->>FE: {in_experiment: true, variant: "xxx"}
    end
```

## 6. 关键设计要点（面试/评审常被追问的点）

| 设计点 | 做法 | 为什么这么做 |
|---|---|---|
| Kafka 消费语义 | Fetch 不自动提交 → 处理成功才 Commit | 避免"取到消息即视为已消费"导致进程崩溃时丢数据（详见 [01 SOP 中 ADR-001](01-build-sop.md)） |
| 优雅关闭 | `signal.NotifyContext` 只 gate 下一次 Fetch，已取到的消息始终用 `context.Background()` 跑完 | 避免两种极端：硬杀丢数据 / 关闭信号打断正在写库的事务 |
| 确定性分桶 | `murmur3(experimentKey+userID)` → `[0,10000)` 区间，按变体权重累积映射阈值区间 | 同一用户每次请求结果一致，且避免"哈希后再取模"引入的分布偏差 |
| 灰度与分桶解耦 | holdout 判断用独立的 `key+":holdout"` 加盐哈希，不复用分桶哈希 | 保证调整流量灰度比例不会连带改变已进组用户的分组结果 |
| 转化去重 | `EventID` 贯穿 events 表主键 → 广告平台 `event_id` 字段 | 满足 Meta/TikTok 官方要求的像素+服务端双路径去重匹配 |
| 统计显著性 | Wilson score 置信区间（而非正态近似） | 小样本/极端转化率下更稳健，是 GrowthBook 等平台的标准做法 |
| SRM 检测 | 卡方拟合优度检验，p<0.01 判定异常 | 行业标准阈值，帮助在看转化率之前先验证"分流本身有没有 bug" |
