# 04. API 使用手册

> 契约先行：本项目的三个 HTTP 服务对外接口如下，前后端/上下游对接前请先过一遍这份契约。所有示例默认本地 `docker compose` 环境端口。

## 服务与 Base URL

| 服务 | Base URL | 说明 |
|---|---|---|
| event-gateway | `http://localhost:8081` | 打点上报、实验分流（面向前端/App 直连） |
| experiment-service | `http://localhost:8082` | 实验管理与统计（面向后台/内部服务） |
| admin-api | `http://localhost:8083` | 后台聚合查询（面向 admin-web，也可直接调） |

所有接口统一返回 JSON；错误响应统一形如 `{"error": "错误信息"}`。

---

## 一、event-gateway

### 1.1 `GET /healthz`

存活探针，返回纯文本 `ok`，用于容器编排的 liveness 检查。

### 1.2 `POST /v1/events` — 上报事件

**用途**：所有用户行为埋点、转化事件的统一上报入口。异步写入 Kafka，接口本身不等待落库完成，所以响应很快（P99 应 < 200ms）。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `event_name` | string | ✅ | 事件名，如 `page_view`、`purchase`、`signup` |
| `user_id` | string | ✅ | 用户唯一标识 |
| `device_id` | string | ❌ | 设备标识 |
| `click_id` | string | ❌ | 广告点击 ID（如 `gclid`/`ttclid`/`fbc`），要触发转化回传必须带 |
| `ad_platform` | string | ❌ | `meta` / `google` / `tiktok`，要触发转化回传必须带 |
| `campaign_id` | int64 | ❌ | 广告计划 ID，要触发转化回传必须带 |
| `properties` | object | ❌ | 自定义属性，任意 JSON |
| `occurred_at` | string(RFC3339) | ❌ | 事件实际发生时间，缺省用服务端接收时间 |

> 转化回传的触发条件是 `click_id`、`ad_platform`、`campaign_id` **三者都非空**——只上报 `event_name`+`user_id` 也完全合法，只是不会触发回传（例如纯站内行为事件）。

**请求示例**：

```bash
curl -X POST http://localhost:8081/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "purchase",
    "user_id": "u-10086",
    "click_id": "gclid-abc123",
    "ad_platform": "google",
    "campaign_id": 1,
    "properties": {"value": 299, "currency": "CNY"}
  }'
```

**成功响应** `202 Accepted`：

```json
{"status": "queued"}
```

**失败响应** `400 Bad Request`（缺少必填字段）：

```json
{"error": "Key: 'incomingEvent.EventName' Error:Field validation for 'EventName' failed on the 'required' tag"}
```

### 1.3 `GET /v1/experiments/:key/assign` — 实验分流

**用途**：给指定用户分配实验变体（幂等——同一用户重复调用结果不变），并自动上报一条 `experiment_exposure` 曝光事件。

**路径参数**：`key` 为实验的 `exp_key`。

**Query 参数**：`user_id`（必填）。

**请求示例**：

```bash
curl "http://localhost:8081/v1/experiments/checkout_button_color/assign?user_id=u-10086"
```

**成功响应（参与实验）** `200 OK`：

```json
{"experiment_key": "checkout_button_color", "variant": "treatment", "in_experiment": true}
```

**成功响应（落入灰度 holdout，不参与实验）** `200 OK`：

```json
{"experiment_key": "checkout_button_color", "variant": null, "in_experiment": false}
```

**失败响应** `400`（缺 `user_id`）/ `404`（实验不存在）：

```json
{"error": "user_id is required"}
```

---

## 二、experiment-service

### 2.1 `POST /v1/experiments` — 创建实验

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `key` | string | ✅ | 实验唯一标识（业务侧代码里引用这个 key） |
| `name` | string | ✅ | 实验展示名 |
| `traffic_allocation` | int | ❌ | 灰度百分比 0-100，缺省 100（全量参与） |
| `variants` | array | ✅（至少2个） | `[{"key","name","weight"}]`，`weight` 决定流量比例 |

**请求示例**：

```bash
curl -X POST http://localhost:8082/v1/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "key": "checkout_button_color",
    "name": "结算按钮颜色",
    "traffic_allocation": 100,
    "variants": [
      {"key": "control", "name": "蓝色（对照组）", "weight": 50},
      {"key": "treatment", "name": "橙色（实验组）", "weight": 50}
    ]
  }'
```

**成功响应** `201 Created`：

```json
{"id": 1, "key": "checkout_button_color"}
```

> ⚠️ **变体顺序即"谁是对照组"**：`getExperimentStats` 会把创建时传入的**第一个 variant** 当作对照组，计算其余变体相对它的 p 值。建好实验后不要再调整变体顺序。

### 2.2 `GET /v1/experiments` — 实验列表

```bash
curl http://localhost:8082/v1/experiments
```

```json
[
  {"id": 1, "key": "checkout_button_color", "name": "结算按钮颜色", "status": "running", "traffic_allocation": 100}
]
```

### 2.3 `GET /v1/experiments/:key` — 实验详情

```bash
curl http://localhost:8082/v1/experiments/checkout_button_color
```

```json
{
  "id": 1, "key": "checkout_button_color", "name": "结算按钮颜色",
  "status": "running", "traffic_allocation": 100,
  "variants": [
    {"key": "control", "name": "蓝色（对照组）", "weight": 50},
    {"key": "treatment", "name": "橙色（实验组）", "weight": 50}
  ]
}
```

### 2.4 `GET /v1/experiments/:key/stats?goal_event=xxx` — 实验统计（含显著性与SRM）

**Query 参数**：`goal_event`，默认 `purchase`，即以哪个事件名作为"转化"的判定口径。

```bash
curl "http://localhost:8082/v1/experiments/checkout_button_color/stats?goal_event=purchase"
```

**响应字段说明**：

```json
{
  "variants": [
    {
      "variant_key": "control",
      "assigned": 91,
      "converted": 55,
      "conversion_rate": 0.6044,
      "confidence_interval": [0.5017, 0.6987],
      "p_value_vs_control": null
    },
    {
      "variant_key": "treatment",
      "assigned": 109,
      "converted": 69,
      "conversion_rate": 0.6330,
      "confidence_interval": [0.5395, 0.7175],
      "p_value_vs_control": 0.6778
    }
  ],
  "srm_check": {
    "chi_square": 1.62,
    "p_value": 0.2031,
    "flagged": false
  }
}
```

| 字段 | 含义 | 怎么解读 |
|---|---|---|
| `conversion_rate` | 该变体转化率 = converted/assigned | |
| `confidence_interval` | 95% Wilson 置信区间 | 两组区间大幅重叠通常意味着差异不显著 |
| `p_value_vs_control` | 与对照组（第一个 variant）比较的双侧 z 检验 p 值 | 通常 p < 0.05 认为差异显著；对照组自身此字段为空 |
| `srm_check.flagged` | 是否触发样本比例失衡告警（p < 0.01） | **为 true 时不要相信上面的转化率数字**，先去排查分桶/曝光上报是否有 bug，参考 [06 调试手册](06-debugging-guide.md) |

---

## 三、admin-api（后台聚合/代理接口）

> 这一组接口主要给 admin-web 用，也可以直接调用做自动化脚本/报表导出。除标注"代理"的以外，均直读 MySQL。

| 方法 | 路径 | 用途 | 备注 |
|---|---|---|---|
| GET | `/v1/events/recent` | 最近 100 条事件 | 按 `received_at` 倒序 |
| GET | `/v1/conversions/summary` | 各转化状态计数 | 返回 `{"success": N, "failed": N, "pending": N}` |
| GET | `/v1/conversions/recent` | 最近 100 条转化记录 | 含 `last_error`、`attempt_count` |
| GET | `/v1/queue/status` | asynq 队列积压情况 | 代理 Redis Inspector，字段见下 |
| GET | `/v1/campaigns` | 广告计划列表 | |
| POST | `/v1/campaigns` | 新建广告计划 | 见下方示例 |
| GET | `/v1/experiments` | 代理到 experiment-service | 原样透传 |
| GET | `/v1/experiments/:key` | 代理到 experiment-service | 原样透传 |
| GET | `/v1/experiments/:key/stats` | 代理到 experiment-service | 原样透传，支持 `goal_event` 参数 |

### 3.1 `POST /v1/campaigns` 示例

```bash
curl -X POST http://localhost:8083/v1/campaigns \
  -H "Content-Type: application/json" \
  -d '{
    "name": "618大促-谷歌搜索",
    "ad_platform": "google",
    "external_campaign_id": "google-camp-001",
    "attribution_window_days": 7,
    "is_active": true
  }'
```

```json
{"id": 1}
```

### 3.2 `GET /v1/queue/status` 响应字段

```json
{
  "pending": 0, "active": 0, "scheduled": 2,
  "retry": 3, "archived": 0, "completed": 1489, "processed": 1512, "failed": 23
}
```

`retry` 不为 0 是正常现象（本项目 stub 了 15% 概率的模拟转化失败用于演示重试链路）；持续增长且不回落才需要关注，见 [06 调试手册](06-debugging-guide.md)。

---

## 四、下游广告平台回传格式（供排查参考，非本项目对外接口）

`conversion-worker` 会把标准化事件转成各广告平台要求的请求体结构（当前为 stub，不发真实网络请求，但结构已对齐官方文档）：

| 平台 | 关键字段 | 参考文档 |
|---|---|---|
| Meta | `data[].event_id`、`data[].user_data.external_id/fbc` | [Meta Conversions API](https://developers.facebook.com/docs/marketing-api/conversions-api) |
| Google | `conversions[].conversion_action`（资源名格式）、`conversion_date_time`（`yyyy-MM-dd HH:mm:ss+HH:MM`） | [Google Ads Enhanced Conversions](https://developers.google.com/google-ads/api/docs/conversions/upload-clicks) |
| TikTok | `data[].context.user.external_id`、`data[].context.ad.ttclid`、`data[].timestamp` | [TikTok Events API](https://business-api.tiktok.com/portal/docs?id=1741601162187777) |

生成的请求体保存在 `conversions.callback_payload` 字段，调试时可直接查库对照文档核对格式，见 [06 调试手册](06-debugging-guide.md) 的"广告回传格式核对"一节。
