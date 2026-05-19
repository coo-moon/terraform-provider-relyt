---
page_title: "使用 Terraform 配置 Microsoft Entra ID SSO"
subcategory: "Guides"
description: |-
  从零开始用 Terraform 管理 DWSU 的 Microsoft Entra ID（原 Azure AD）单点登录配置。
---

# 使用 Terraform 配置 Microsoft Entra ID SSO

本指南面向 **DWSU 管理员 / SRE / 平台工程师**，目标是让你在 15 分钟内用 Terraform 把 DMS Console 的 Microsoft 单点登录跑通。

不需要 Terraform 经验 —— 涉及的 TF 概念会在用到的时候解释。

---

## 1. 这是什么 & 我为什么要用

**Microsoft Entra ID**（前身 Azure AD）是微软的企业身份服务。配置 Entra ID SSO 之后：

- 公司员工不再需要 DMS 单独的用户名密码
- 在 DMS Console 点 "Microsoft 登录" → 跳到 Microsoft 登录页 → 走完 SAML/OAuth 流程 → 自动登录到 DMS
- 离职员工在 Azure 一停用，DMS 访问立即失效（不用单独管 DMS 这边）

`relyt_dwsu_entraid_config` 这个 Terraform 资源把"DMS 这一侧的 SSO 配置"（即调 `PUT /api/entraid-config` 的那一步）变成代码声明，让你：

- **可复现**：配置进 git 仓库，跨环境（prod / staging / dev）一份模板即可
- **可审计**：所有变更都有 git diff、PR 记录
- **可回滚**：`terraform destroy` 或回退到前一个 commit 即可关掉 SSO
- **可漂移检测**：有人手工去 curl 改了后端，下次 `terraform plan` 会立刻发现差异

> ⚠️ 本资源**只管 DMS 侧的配置注册**。Azure 侧的 App Registration / App Role / 用户分配仍需要在 Azure portal 手工做（或自己用 `azuread` provider 单独管）。本指南会带你走完这两侧。

---

## 2. 整体流程

```
┌──────────────────────────┐           ┌──────────────────────────┐
│      Azure (手工)        │           │   Terraform (本指南)     │
│                          │           │                          │
│ 1. 创建 App Registration │           │ 4. provider 配置         │
│ 2. 定义 App Roles        │  ─────▶   │    (auth_key + role)     │
│    (value = DMS 用户名)  │           │ 5. relyt_dwsu_entraid_   │
│ 3. 分配 Azure 用户到 Role│           │    config resource       │
│                          │           │ 6. terraform apply       │
└──────────────────────────┘           └──────────────────────────┘
              │                                      │
              │       拿到 tenant_id / client_id     │
              └─────────────────┬────────────────────┘
                                ▼
                       ┌────────────────────┐
                       │ DMS Console        │
                       │ "Microsoft 登录"   │ ◀── 终端用户
                       └────────────────────┘
```

涉及 3 个角色：

| 角色 | 做什么 |
|---|---|
| **Azure 管理员** | Step 1-3：在 Azure portal 注册应用、定义角色、分配用户 |
| **DWSU 管理员（你）** | Step 4-6：本指南主要内容 |
| **终端用户** | 用 Microsoft 账号点登录 |

---

## 3. 前置准备清单

跑下面任何命令之前，确保你有：

| # | 条目 | 怎么验证 |
|---|---|---|
| 1 | Azure AD 租户管理员权限 | 能登 [Azure portal](https://portal.azure.com) 并创建 App registration |
| 2 | 一个已存在的 DWSU | 在 DMS Console 看得到，或 `data "relyt_dwsus" "all" {}` 能查到 |
| 3 | DWSU 上有一个 **ACCOUNTADMIN** 角色的用户 | 该用户的 SecretKey 用于本指南所有 PUT/DELETE 操作 |
| 4 | DWSU 上有至少一个已存在的普通用户（你想给 TA 开 SSO） | DMS Console → Users。SSO 不会自动创建 DMS 用户 |
| 5 | 本地装了 Terraform CLI | `terraform version` 输出 ≥ 1.0 |
| 6 | 本地装了 Relyt provider | 见 [provider 安装](https://registry.terraform.io/providers/relytcloud/relyt) |

> 第 3 条最容易踩坑：**ACCOUNTADMIN ≠ SYSTEMADMIN**。这俩是不同维度的角色。SYSTEMADMIN 是云级别（用来建 DWSU），ACCOUNTADMIN 是 DWSU 内部账号管理员（用来管该 DWSU 的 SSO、用户安全策略等）。如果你只有 SYSTEMADMIN key，本资源的 apply 会被后端拒。

---

## 4. Azure 侧准备（一次性）

### 4.1 创建 App Registration

1. 登 [Azure portal](https://portal.azure.com)
2. **Microsoft Entra ID** → **App registrations** → **+ New registration**
3. 填写：
   - **Name**: `relyt-dms-<dwsu-name>` (示例命名，自由发挥)
   - **Supported account types**: `Accounts in this organizational directory only`（对应 TF 里的 `tenant_type = "single"`）
   - **Redirect URI**:
     - 类型选 **Single-page application (SPA)**
     - 值填 DMS 前端回调地址，如 `https://<dwsu-domain>/console/auth/entra-id/callback`
4. 点 **Register**

### 4.2 拿到 Tenant ID + Client ID

进入刚创建的应用 → **Overview**：

- **Directory (tenant) ID** → 复制下来，TF 里叫 `tenant_id`
- **Application (client) ID** → 复制下来，TF 里叫 `client_id`

### 4.3 定义 App Roles（一个 App Role 对应一个 DMS 用户）

App registration → **App roles** → **+ Create app role**：

| 字段 | 值 |
|---|---|
| Display name | `DMS - <username>` 例如 `DMS - wenbo` |
| Allowed member types | `Users/Groups` |
| **Value** | `wenbo` ← **必须严格等于 DMS 里某个已存在用户的 username** |
| Description | `Login as DMS user wenbo` |
| Enabled | ☑ |

> ⚠️ App Role 的 `Value` 一旦下发就很难改。一个 App Role 对应一个 DMS 身份。需要给多个 DMS 用户开 SSO，就建多个 App Role，每个 Value 对应一个 DMS username。

### 4.4 分配 Azure 用户到 App Role

**Enterprise applications** → 找到刚创建的应用 → **Users and groups** → **+ Add user/group**：

1. 选 Azure 用户（如 `wenbo@yourcompany.com`）
2. 选刚定义的 role（如 `DMS - wenbo`）
3. **Assign**

> ⚠️ 同一 Azure 用户在同一应用里只能分配 **1 个** DMS 相关 role，多于一个会被后端拒登。

---

## 5. Relyt 侧：用 Terraform 配 SSO

### 5.1 第一次：最小可用配置

新建文件 `main.tf`：

```hcl
terraform {
  required_providers {
    relyt = {
      source  = "relytcloud/relyt"
      version = ">= 0.0.4"   # 含 entraid_config 资源的最早版本
    }
  }
}

provider "relyt" {
  # auth_key 是 ACCOUNTADMIN 用户的 SecretKey。
  # 也可以省略此行，改用环境变量 RELYT_AUTH_KEY。
  auth_key = "<accountadmin-secret-key>"
  role     = "ACCOUNTADMIN"
}

# 引用已存在的 DWSU。如果你只有一个 DWSU，索引 [0] 就够；多 DWSU 时
# 用 .records[*].id 然后过滤更稳妥。
data "relyt_dwsus" "all" {}

resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = data.relyt_dwsus.all.records[0].id
  tenant_id = "<paste-your-tenant-id-from-step-4.2>"
  client_id = "<paste-your-client-id-from-step-4.2>"

  # 下面两个可省略（用默认值）：
  # tenant_type = "single"
  # enabled     = true
}
```

字段说明：

| 字段 | 必填 | 说明 |
|---|---|---|
| `dwsu_id` | ✓ | 目标 DWSU 的 ID。**改了会触发销毁重建**（一份 entraid-config 绑死一个 DWSU） |
| `tenant_id` | ✓ | Azure tenant GUID（4.2 步拿的） |
| `client_id` | ✓ | Azure application GUID（4.2 步拿的） |
| `tenant_type` | ✗ | `"single"` 只允许配置租户的用户登录（推荐）；`"multi"` 允许任何 Azure 组织用户。默认 `"single"` |
| `enabled` | ✗ | `true` 启用 SSO；`false` 保留配置但暂时不让登录。默认 `true` |

### 5.2 跑 apply

```bash
# 初始化（下载 provider）
terraform init

# 看会做什么，不实际执行
terraform plan
```

预期输出：

```
Terraform will perform the following actions:

  # relyt_dwsu_entraid_config.sso will be created
  + resource "relyt_dwsu_entraid_config" "sso" {
      + client_id   = "00000000-..."
      + dwsu_id     = "dwsu-abc123"
      + enabled     = true
      + tenant_id   = "11111111-..."
      + tenant_type = "single"
    }

Plan: 1 to add, 0 to change, 0 to destroy.
```

看着对就执行：

```bash
terraform apply
# 输入 yes 确认
```

### 5.3 验证配置生效

```bash
# 直接 curl 后端，应看到刚 PUT 的配置
DMS=$(terraform output -raw dmsHost 2>/dev/null || echo "https://api-<your-dwsu-domain>")
curl -s $DMS/api/entraid-config | python3 -m json.tool
```

期望响应包含你刚 PUT 进去的 `tenantId` / `clientId` / `tenantType` / `enabled: true`。

接着拿 Microsoft 登录授权 URI 试一下：

```bash
curl -s $DMS/api/auth/entra-id/authorize | python3 -m json.tool
```

期望 `data.authorizationUri` 不为 null。

最后**真实登录测试**：在浏览器打开 DMS Console，应该出现 "Microsoft 登录" 按钮，点击 → 跳 Microsoft 登录页 → 用 4.4 步分配的 Azure 用户登录 → 自动跳回 DMS 主界面。

---

## 6. 常见操作

### 6.1 改 tenant_id / client_id（重新绑到新的 Azure 应用）

直接改 .tf 里的字段值，`terraform apply`：

```hcl
resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = ...
  tenant_id = "新的 tenant id"   # ← 改这里
  client_id = "新的 client id"   # ← 改这里
}
```

Plan 会显示：

```
  ~ resource "relyt_dwsu_entraid_config" "sso" {
      ~ client_id = "旧值" -> "新值"
      ~ tenant_id = "旧值" -> "新值"
    }

Plan: 0 to add, 1 to change, 0 to destroy.
```

`~ change` 是"原地修改"（PUT），不是销毁重建。

### 6.2 临时关 SSO（保留配置）

```hcl
resource "relyt_dwsu_entraid_config" "sso" {
  # ... 其他字段不变
  enabled = false   # ← 加这行
}
```

`apply` 后，DMS Console 上 "Microsoft 登录" 按钮自动隐藏，但所有配置仍在后端。下次想开回来把 `enabled = false` 删掉（恢复默认 true）再 apply 即可。

### 6.3 完全移除 SSO

**方式 A**（推荐）：删掉 `.tf` 里整个 `resource "relyt_dwsu_entraid_config" "sso" { ... }` 块，`terraform apply` 自动调用 DELETE。

**方式 B**：destroy 整份 TF：

```bash
terraform destroy
```

执行后后端 entraid-config 被清空，DMS Console 不再出现 Microsoft 登录入口。

### 6.4 把已存在的 SSO 配置纳入 Terraform 管理（import）

场景：你之前已经手工 curl 配过 SSO（按内部 SSO setup 指南做的），现在想转用 Terraform 管。

**Step 1**：在 .tf 里写好 resource 块（字段值先填占位，import 后会被实际值覆盖）：

```hcl
resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = "dwsu-abc123"
  tenant_id = "placeholder"
  client_id = "placeholder"
}
```

**Step 2**：

```bash
terraform import relyt_dwsu_entraid_config.sso dwsu-abc123
#                ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ ^^^^^^^^^^^
#                TF 资源地址                                 import ID = dwsu_id
```

**Step 3**：`terraform plan`。预期 plan 提示你 `.tf` 文件里的 `placeholder` 与后端真实值不一致，你按真实值把 `.tf` 改对，再 plan 应该是 `no changes`。

---

## 7. 进阶：多角色 provider alias

**场景**：你想用一份 `.tf` 同时管 DWSU lifecycle（需 SYSTEMADMIN）和 SSO config（需 ACCOUNTADMIN）。

Terraform 支持"同一个 provider 多个配置实例"，叫 **provider alias**：

```hcl
provider "relyt" {
  alias    = "system"
  auth_key = "<systemadmin-secret-key>"
  role     = "SYSTEMADMIN"
}

provider "relyt" {
  alias    = "account"
  auth_key = "<accountadmin-secret-key>"
  role     = "ACCOUNTADMIN"
}

resource "relyt_dwsu" "dw" {
  provider = relyt.system        # ← 明确用 system alias
  cloud  = "aws"
  region = "ap-southeast-1"
  domain = "my-dw"
  default_dps = {
    name   = "hdps"
    engine = "hybrid"
    size   = "S"
  }
}

resource "relyt_dwsu_entraid_config" "sso" {
  provider = relyt.account       # ← 明确用 account alias
  dwsu_id  = relyt_dwsu.dw.id    # TF 自动识别依赖：先建 DWSU，再配 SSO
  tenant_id = "..."
  client_id = "..."
}
```

> 资源块里只要写一行 `provider = relyt.<alias>`，TF 就会用对应的那份 provider 配置执行该资源的所有 API 调用。资源之间靠引用关系（`relyt_dwsu.dw.id`）自动建立依赖图。

---

## 8. 故障排查速查

按看到的错误信息查：

| 错误信息 | 原因 | 修复 |
|---|---|---|
| `HTTP 403 Permission denied: Account admin role required to manage SSO config` | `auth_key` 用的不是 ACCOUNTADMIN | 用 ACCOUNTADMIN 用户的 SecretKey；或起 provider alias |
| `error fetching DWSU ... DWSU not found` | `dwsu_id` 错了 / DWSU 已被删 | 检查 `dwsu_id`；用 `data "relyt_dwsus"` 列一下 |
| `openapi endpoint not found on DWSU` | DwsuModel.Endpoints[] 里没有 `Type==openapi` 的项 | 联系 Relyt 支持 — 这通常是部署配置问题 |
| `No 'roles' claim in token`（终端用户登录时） | Azure 用户没分配 App Role | Azure portal → Enterprise applications → Users and groups → 分配 role |
| `Multiple app roles assigned: [a, b]` | 同一 Azure 用户分配了多个 DMS role | 删多余的 role assignment，只保留 1 个 |
| `DMS user not found for role: xxx` | App Role 的 Value 在 DMS 里没有对应 username | 在 DMS 创建该用户，或改 App Role 的 Value |
| 浏览器：`AADSTS50011: ...redirect URI...` | Azure 应用没注册当前的回调 URI | App registration → Authentication → 补回调 URI |
| Plan 显示资源要被销毁重建但你没改 `dwsu_id` | 罕见 —— 通常是 state 与 `.tf` 严重不一致 | `terraform refresh` → `terraform plan` 看 diff；必要时 `terraform import` |

如果上面都不匹配，可以：

1. 跑 `TF_LOG=DEBUG terraform apply` 看详细 HTTP 请求 / 响应
2. 看 DMS Pod 日志：`kubectl -n <ns> logs deploy/hdpscluster-<id>-dms --since=10m | grep -iE "entra|verify|token|roles"`
3. 联系 Relyt 支持，附上 plan / apply 的完整输出

---

## 9. FAQ

**Q1：我能不能用同一个 SSO 配置覆盖多个 DWSU？**
不能。`entraid-config` 是 per-DWSU 的，每个 DWSU 都要单独配。常见做法是用 Terraform `for_each`：

```hcl
locals {
  dwsus_with_sso = {
    "prod"    = { dwsu_id = relyt_dwsu.prod.id,    tenant_id = "...", client_id = "..." }
    "staging" = { dwsu_id = relyt_dwsu.staging.id, tenant_id = "...", client_id = "..." }
  }
}

resource "relyt_dwsu_entraid_config" "sso" {
  for_each  = local.dwsus_with_sso
  dwsu_id   = each.value.dwsu_id
  tenant_id = each.value.tenant_id
  client_id = each.value.client_id
}
```

**Q2：tenant_type 设成 multi 安全吗？**
`multi` 允许任何 Azure 组织的用户走你的应用登录流程。能不能登 DMS 仍然由 App Role 分配决定 —— 没分到 role 的人即使过了 Microsoft 认证，到 DMS 这一关也会被拒。**但是**：跨租户场景下 Azure 侧需要明确允许 multi-tenant，否则 Microsoft 会先拒。除非你确实有跨组织协作需求，否则 `single` 更安全。

**Q3：改了 Azure 侧的 App Role / 用户分配，TF 这边要 apply 吗？**
不需要。本资源只管 DMS 端记录 tenant_id/client_id/enabled，Azure 侧的 App Role 不在 TF 管辖范围内。**但是**：如果你想用 Terraform 也管 Azure 侧，可以引入 `hashicorp/azuread` provider，那是另一个独立故事。

**Q4：destroy 了之后，已经登录的用户会被踢出吗？**
**不会**。已经签发的 access token 在过期前仍然有效。destroy 只阻止**新的** Microsoft 登录尝试。要立刻踢人，需要单独的"撤销 token"操作 —— 不在本资源范围内。

**Q5：我能把 auth_key 直接写在 .tf 里吗？**
**强烈不建议**。`.tf` 通常会进 git，泄露的 ACCOUNTADMIN key 风险等同于密码。推荐做法：

- 用环境变量：`export RELYT_AUTH_KEY="..."`，provider 块里省略 `auth_key`
- 用 Terraform Cloud / Vault / Doppler 等 secret 管理工具注入

**Q6：apply 卡住或超时怎么办？**
本资源底层用 `CommonRetry`（5 次 × 1 秒退避），单次 HTTP 超时由 provider 配置 `client_timeout` 控制（默认 10 秒）。如果反复卡：

- 检查 DWSU 是不是 in `READY` 状态
- 检查网络能不能直达 DWSU 的 openapi endpoint（`curl <openapi-uri>/api/entraid-config` 试试）
- `TF_LOG=DEBUG terraform apply` 看到底卡在哪一步

---

## 10. 完整的 main.tf 参考模板

```hcl
terraform {
  required_providers {
    relyt = {
      source  = "relytcloud/relyt"
      version = ">= 0.0.4"
    }
  }
}

provider "relyt" {
  # 推荐：用环境变量 RELYT_AUTH_KEY 注入，不要硬编码
  role = "ACCOUNTADMIN"
}

# 引用已存在的 DWSU
data "relyt_dwsus" "all" {}

locals {
  # 把所有 DWSU 都开 SSO；如果只想给特定 DWSU 配，把这一段改成显式列表
  target_dwsus = {
    for d in data.relyt_dwsus.all.records :
    d.alias => d.id
  }
}

resource "relyt_dwsu_entraid_config" "sso" {
  for_each = local.target_dwsus

  dwsu_id   = each.value
  tenant_id = "<your-azure-tenant-id>"
  client_id = "<your-azure-client-id>"

  tenant_type = "single"
  enabled     = true
}

output "configured_dwsus" {
  value = [for k, v in relyt_dwsu_entraid_config.sso : k]
}
```

---

## 11. 下一步

- 完整资源参考：见 [`relyt_dwsu_entraid_config`](../resources/dwsu_entraid_config.md)
- Azure 侧详细设置：参考 Relyt 内部 SSO setup 指南
- 如果想把 Azure 侧也用 Terraform 管：考虑 [`hashicorp/azuread`](https://registry.terraform.io/providers/hashicorp/azuread/latest/docs) provider
