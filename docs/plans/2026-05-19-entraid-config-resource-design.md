# Design: `relyt_dwsu_entraid_config` Resource

**Date**: 2026-05-19
**Status**: Approved (awaiting implementation plan)
**Scope**: 仅交付 DMS 侧 Terraform resource + 示例 + 文档；Azure 侧的 App Registration / Role 分配仍由客户手工或自带 azuread provider 完成。

---

## 1. 背景与目标

让客户能在 Terraform 中**声明式管理 DWSU 的 Microsoft Entra ID (Azure AD) SSO 配置**，覆盖完整生命周期（创建 / 更新 / 读取 / 删除 / 导入），目标是"开箱即用"。

后端 REST API 已存在（详见 `~/source/bluewhale/docs/entra-id-sso-guide.md`）：

- `PUT /api/entraid-config` — 写 / 更新（需 ACCOUNTADMIN）
- `GET /api/entraid-config` — 读（无需认证）
- `DELETE /api/entraid-config` — 删（需 ACCOUNTADMIN）

接口寄宿于**每个 DWSU 自己的 openapi endpoint**（`DwsuModel.Endpoints[Type=="openapi"].URI`），路径里**不带 dwsu_id**。

---

## 2. 设计决策摘要

| 决策点 | 结论 | 理由 |
|---|---|---|
| 资源形态 | 独立 singleton resource，与现有 `dwsu_user_policy` 同套路 | 一致性、最小改动 |
| TF 资源类型名 | `relyt_dwsu_entraid_config` | 与 `dwsu_<feature>` 命名传统对齐；为将来 SAML / Okta 留命名空间 |
| 字段暴露 | 完整暴露 4 个后端字段（`tenant_id` / `client_id` / `tenant_type` / `enabled`） | 与后端 schema 1:1，不屏蔽 enabled |
| Singleton 标识 | `dwsu_id` 既作输入又作 TF state 隐式 ID | 一个 DWSU 一份配置，天然约束 |
| Delete 语义 | 调 `DELETE /api/entraid-config`（hard delete） | 与 destroy 词义对齐，TF 状态与云端对齐 |
| Drift 处理 | Read 检测到后端 `data == null` → `RemoveResource` → 下次 plan 自动重建 | TF 社区标准 drift 处理 |
| Host 解析 | `client.GetDwsu(dwsu_id).Endpoints[Type=="openapi"].URI` | 与指南字面一致；**不复用** `RouteRegionUri`（那个是 region 级 gateway，不是 DWSU 私有 host） |
| 鉴权传递 | 复用 provider 级 `auth_key`（已被 HTTP 工具塞进 `x-maxone-api-key` 头） | 现有约定，零代码变更 |
| 多角色场景 | 文档提示用 provider alias，不在 provider 内塞两套 credential | 与 TF 社区惯例对齐 |
| 测试形态 | 沿用项目现有"冒烟脚本"风格 + 1 个纯 Go 单测 + 手工 smoke 流程脚本 | 与项目现状对齐，不引入新依赖 |

---

## 3. 文件清单

| 文件 | 动作 | 内容 |
|---|---|---|
| `client/relyt_data.go` | 追加 | `EntraIdConfig` struct + `CODE_ENTRAID_CONFIG_NOT_FOUND` 常量（值待定，实现期间确认） |
| `client/relyt_client.go` | 追加 | `GetEntraIdConfig` / `PutEntraIdConfig` / `DeleteEntraIdConfig` |
| `client/relyt_client_test.go` | 追加 | 4 个冒烟测试 |
| `common/common_util.go` | 追加 | `RouteDwsuOpenApiHost` + 纯函数 `PickOpenApiURIFromEndpoints` |
| `common/common_util_test.go` | 追加 | 4 个纯 Go 单测覆盖 `PickOpenApiURIFromEndpoints` 边界 |
| `model/entraid_config.go` | **新建** | `EntraIdConfigModel` |
| `resource/dwsu_entraid_config_resource.go` | **新建** | resource 主体 |
| `provider.go` | 改 | `Resources()` 列表追加 `NewDwsuEntraIdConfig` |
| `examples/resources/relyt_dwsu_entraid_config/resource.tf` | **新建** | tfplugindocs 引用的示例 |
| `docs/resources/dwsu_entraid_config.md` | **新建**（由 `go generate` 生成 + 手工补 Prerequisites 段） | 注册中心文档 |
| `docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/{main,variables,outputs}.tf` | **新建** | demo module |

**不动**：顶层 `docs/demo/terraform/modules/relyt/main.tf`（不默认启用 SSO，客户按需拼接）。

---

## 4. Schema

```go
"dwsu_id":     Required, RequiresReplace
"tenant_id":   Required
"client_id":   Required
"tenant_type": Optional, Computed, Default("single"),  Validators: OneOf("single","multi")
"enabled":     Optional, Computed, Default(true)
```

| 字段 | RequiresReplace | 默认 | 备注 |
|---|---|---|---|
| `dwsu_id` | ✓ | — | 换 DWSU 等于换对象，销毁重建 |
| `tenant_id` | ✗ | — | 改了走 PUT |
| `client_id` | ✗ | — | 同上 |
| `tenant_type` | ✗ | `"single"` | `single ↔ multi` 是无损切换（指南确认） |
| `enabled` | ✗ | `true` | 写资源 = 想开 SSO；显式关需手填 `enabled = false` |

无 `id` 显式字段 — `dwsu_id` 既是输入又是 state 标识。

---

## 5. Client 层

### 5.1 类型（追加到 `relyt_data.go`）

```go
type EntraIdConfig struct {
    TenantId   string `json:"tenantId,omitempty"`
    ClientId   string `json:"clientId,omitempty"`
    TenantType string `json:"tenantType,omitempty"`
    Enabled    bool   `json:"enabled"`           // 注意：无 omitempty —— false 是有意义的值，必须发送
}
```

### 5.2 方法签名

```go
func (p *RelytClient) GetEntraIdConfig(ctx context.Context, dmsHost, dwsuId string) (*EntraIdConfig, error)
func (p *RelytClient) PutEntraIdConfig(ctx context.Context, dmsHost, dwsuId string, cfg EntraIdConfig) (*EntraIdConfig, error)
func (p *RelytClient) DeleteEntraIdConfig(ctx context.Context, dmsHost, dwsuId string) error
```

- `dmsHost` = DWSU 的 openapi endpoint URI（如 `https://api-<dwsu-domain>`）
- `dwsuId` 仅用于 tflog 日志关联，**不进路径**（路径恒为 `/api/entraid-config`）
- `Get` 的 codeHandler 必须把 `CODE_ENTRAID_CONFIG_NOT_FOUND` 视为成功，data 返回 nil
- `Delete` 同上（幂等 DELETE）

### 5.3 错误码占位

```go
const CODE_ENTRAID_CONFIG_NOT_FOUND = ???  // <TBD> 实现期间确认（抓包或后端源码）
```

**风险项**：若 GET/DELETE 在 not-found 情形下不是返回 200 + 业务错误码、而是直接 HTTP 4xx，则 `doHttpRequest` 在 `resp.StatusCode != 200` 时已经直接返回 `err != nil`，codeHandler 拿不到。这种情况下要给 `doHttpRequest` 加上"允许非 200 状态码"的扩展。**这条会在 plan 文档中作为待验证项标注。**

---

## 6. Host 解析（`common/common_util.go`）

```go
// 纯函数 — 可单测
func PickOpenApiURIFromEndpoints(endpoints []client.Endpoints) (string, error) {
    for _, ep := range endpoints {
        if ep.Type == "openapi" && ep.URI != "" {
            return ep.URI, nil
        }
    }
    return "", fmt.Errorf("no endpoint of type 'openapi' found")
}

// resource 层调用入口
func RouteDwsuOpenApiHost(ctx context.Context, dwsuId string,
    c *client.RelytClient, diag *diag.Diagnostics) string {
    dwsu, err := CommonRetry(ctx, func() (*client.DwsuModel, error) {
        return c.GetDwsu(ctx, dwsuId)
    })
    if err != nil || dwsu == nil {
        diag.AddError("error fetching DWSU",
            fmt.Sprintf("dwsu_id=%s err=%v", dwsuId, err))
        return ""
    }
    uri, err := PickOpenApiURIFromEndpoints(dwsu.Endpoints)
    if err != nil {
        diag.AddError("openapi endpoint not found on DWSU",
            fmt.Sprintf("dwsu_id=%s err=%v", dwsuId, err))
        return ""
    }
    return uri
}
```

---

## 7. Resource CRUD（`resource/dwsu_entraid_config_resource.go`）

接口断言：

```go
var (
    _ resource.Resource                = &dwsuEntraIdConfig{}
    _ resource.ResourceWithConfigure   = &dwsuEntraIdConfig{}
    _ resource.ResourceWithImportState = &dwsuEntraIdConfig{}
)
```

骨架：

```go
type dwsuEntraIdConfig struct { RelytClientResource }

func NewDwsuEntraIdConfig() resource.Resource { return &dwsuEntraIdConfig{} }

func (r *dwsuEntraIdConfig) Metadata(...)  { resp.TypeName = req.ProviderTypeName + "_dwsu_entraid_config" }
func (r *dwsuEntraIdConfig) Schema(...)    { /* 见 §4 */ }

func (r *dwsuEntraIdConfig) Create(ctx, req, resp) { r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State) }
func (r *dwsuEntraIdConfig) Update(ctx, req, resp) { r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State) }

func (r *dwsuEntraIdConfig) Read(ctx, req, resp) {
    var state EntraIdConfigModel
    diags := req.State.Get(ctx, &state); resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, state.DwsuId.ValueString(), r.client, &resp.Diagnostics)
    if resp.Diagnostics.HasError() { return }

    cfg, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
        return r.client.GetEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString())
    })
    if err != nil {
        resp.Diagnostics.AddError("Error reading entraid-config", err.Error())
        return
    }
    if cfg == nil {
        // 后端已被外部 DELETE / 未配置 → drift 处理
        resp.State.RemoveResource(ctx)
        return
    }
    state.TenantId   = types.StringValue(cfg.TenantId)
    state.ClientId   = types.StringValue(cfg.ClientId)
    state.TenantType = types.StringValue(cfg.TenantType)
    state.Enabled    = types.BoolValue(cfg.Enabled)
    resp.State.Set(ctx, &state)
}

func (r *dwsuEntraIdConfig) Delete(ctx, req, resp) {
    var state EntraIdConfigModel
    diags := req.State.Get(ctx, &state); resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, state.DwsuId.ValueString(), r.client, &resp.Diagnostics)
    if resp.Diagnostics.HasError() { return }

    _, err := common.CommonRetry(ctx, func() (*string, error) {
        if e := r.client.DeleteEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString()); e != nil {
            return nil, e
        }
        s := "ok"; return &s, nil
    })
    if err != nil {
        resp.Diagnostics.AddError("Error deleting entraid-config", err.Error())
    }
}

func (r *dwsuEntraIdConfig) ImportState(ctx, req, resp) {
    resource.ImportStatePassthroughID(ctx, path.Root("dwsu_id"), req, resp)
}

// 私有助手 — Create / Update 公用
func (r *dwsuEntraIdConfig) put(ctx, plan, diag, dst) {
    var p EntraIdConfigModel
    diag.Append(plan.Get(ctx, &p)...)
    if diag.HasError() { return }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, p.DwsuId.ValueString(), r.client, diag)
    if diag.HasError() { return }

    _, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
        return r.client.PutEntraIdConfig(ctx, dmsHost, p.DwsuId.ValueString(), client.EntraIdConfig{
            TenantId:   p.TenantId.ValueString(),
            ClientId:   p.ClientId.ValueString(),
            TenantType: p.TenantType.ValueString(),
            Enabled:    p.Enabled.ValueBool(),
        })
    })
    if err != nil {
        diag.AddError("Error writing entraid-config", err.Error()); return
    }
    dst.Set(ctx, &p)
}
```

**Drift 行为对照表**：

| 后端实际状态 | Read 行为 | 下次 plan | apply 行为 |
|---|---|---|---|
| 字段被外部改 | 把实际值刷回 state | `~ update` | PUT 覆盖回客户期望 |
| 配置被外部 DELETE | `RemoveResource` | `+ create` | PUT 重建 |
| DWSU 整个被删 | `RouteDwsuOpenApiHost` 失败 → 报错 | — | 用户排查 |

---

## 8. 文档 & 示例

### 8.1 `examples/resources/relyt_dwsu_entraid_config/resource.tf`

```hcl
data "relyt_dwsus" "all" {}

resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = data.relyt_dwsus.all.records[0].id
  tenant_id = "00000000-0000-0000-0000-000000000000"  # Azure portal → Directory (tenant) ID
  client_id = "00000000-0000-0000-0000-000000000000"  # Azure portal → Application (client) ID

  tenant_type = "single"   # "single" | "multi"，默认 single
  enabled     = true       # 默认 true
}
```

### 8.2 `docs/resources/dwsu_entraid_config.md`（Prerequisites 段）

```markdown
## Prerequisites

1. Azure AD App registration with App Roles defined and assigned to users
   (see Relyt SSO guide for full walkthrough).
2. The provider's `auth_key` must belong to a user with **ACCOUNTADMIN** role
   on the target DWSU — only ACCOUNTADMIN can manage SSO config. A SYSTEMADMIN
   key will result in `HTTP 403 Permission denied`.
3. The DWSU must already exist; reference it via `relyt_dwsu.<name>.id` or
   `data.relyt_dwsus.<name>.records[*].id`.

## Multi-role example (provider alias)

If you need to manage both DWSU lifecycle (SYSTEMADMIN) and SSO config
(ACCOUNTADMIN) in one Terraform configuration:

\`\`\`hcl
provider "relyt" {
  alias    = "system"
  auth_key = "<systemadmin-key>"
  role     = "SYSTEMADMIN"
}

provider "relyt" {
  alias    = "account"
  auth_key = "<accountadmin-key>"
  role     = "ACCOUNTADMIN"
}

resource "relyt_dwsu" "dw" {
  provider = relyt.system
  # ...
}

resource "relyt_dwsu_entraid_config" "sso" {
  provider = relyt.account
  dwsu_id  = relyt_dwsu.dw.id
  # ...
}
\`\`\`
```

### 8.3 Demo module

`docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/{main,variables,outputs}.tf` — 形状对齐 `dw_user`、`relyt_dwsu_user_policy`。**不**自动接入顶层 `main.tf`。

---

## 9. 测试

### 9.1 Client 冒烟脚本（`client/relyt_client_test.go` 追加）

- `TestPutEntraIdConfig`
- `TestGetEntraIdConfig`
- `TestDeleteEntraIdConfig`
- `TestEntraIdConfig_NotFoundReadIdempotent` ← 借此验证 `CODE_ENTRAID_CONFIG_NOT_FOUND` 实际数值

### 9.2 纯 Go 单测（`common/common_util_test.go` 追加）

- `TestPickOpenApiURI_Found`
- `TestPickOpenApiURI_NotFound`
- `TestPickOpenApiURI_EmptyList`
- `TestPickOpenApiURI_FirstMatchWins`

### 9.3 手工 smoke test 脚本

写进本文档"附录 A"（见下），客户/QA 复制即跑。

### 9.4 显式不做

| 不做 | 理由 |
|---|---|
| `terraform-plugin-testing` acceptance test | 项目现状未用，引入新依赖；§9.3 已覆盖 |
| `httptest.NewServer` mock | 与项目"打真实 API"测试约定不一致 |
| Resource 层 framework mock 单测 | mock 装机大，性价比低 |

---

## 10. 风险 & 待确认项

| # | 项 | 何时确认 | 默认假设 |
|---|---|---|---|
| 1 | `CODE_ENTRAID_CONFIG_NOT_FOUND` 实际数值 | 实现期间抓包 / 看后端源码 | 占位常量，跑 `TestEntraIdConfig_NotFoundReadIdempotent` 时填入 |
| 2 | NOT_FOUND 时后端是返回 HTTP 200 + 业务错误码，还是直接 HTTP 4xx？ | 同上 | 假设 200 + 业务码（与 `CODE_DWSU_NOT_FOUND` / `CODE_USER_NOT_FOUND` 一致）；若实际是 4xx，需扩展 `doHttpRequest` 允许非 200 状态码进入 codeHandler |
| 3 | `DwsuModel.Endpoints[]` 在所有部署下都包含 `Type=="openapi"` 的项？ | smoke test 第 1 步 | 假设是；若否，host 解析报"openapi endpoint not found" |
| 4 | SSO PUT 路径在 `openapi` endpoint 上确实 reachable？（指南字面是 `<dwsu-domain>`，未明确说是哪个 type） | smoke test 第 2 步 curl 验证 | 假设是；用户已在 brainstorming 中确认 |

---

## 附录 A · 手工 smoke test 脚本

```bash
# 0. 准备
export RELYT_AUTH_KEY="<ACCOUNTADMIN secret key>"

# 假设 main.tf 已包含 provider + relyt_dwsu + relyt_dwsu_entraid_config

# 1. apply
terraform apply -auto-approve

# 2. 后端校验：配置确实落库
DMS="https://<openapi-uri-of-the-dwsu>"
curl -s $DMS/api/entraid-config | python3 -m json.tool
# 期望 data 字段含 tenantId/clientId/tenantType/enabled=true

# 3. 改 enabled = false，再 apply
sed -i '' 's/enabled[[:space:]]*=[[:space:]]*true/enabled = false/' main.tf
terraform apply -auto-approve
curl -s $DMS/api/entraid-config | python3 -m json.tool
# 期望 enabled:false

# 4. 模拟外部 drift：手动 PUT 把 enabled 改回 true
curl -X PUT $DMS/api/entraid-config -H "x-maxone-api-key: $RELYT_AUTH_KEY" \
  -H "Content-Type: application/json" \
  -d '{"tenantId":"...","clientId":"...","tenantType":"single","enabled":true}'
terraform plan
# 期望输出：~ enabled = true -> false

# 5. 模拟外部删除：手动 DELETE
curl -X DELETE $DMS/api/entraid-config -H "x-maxone-api-key: $RELYT_AUTH_KEY"
terraform plan
# 期望输出：+ create

# 6. import：把云端已存在配置纳入 TF
curl -X PUT $DMS/api/entraid-config ...   # 手动建一份
terraform state rm relyt_dwsu_entraid_config.sso
terraform import relyt_dwsu_entraid_config.sso <dwsu_id>
terraform plan
# 期望输出：no changes

# 7. destroy
terraform destroy -auto-approve
curl -s $DMS/api/entraid-config | python3 -m json.tool
# 期望 data: null
```
