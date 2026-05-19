# `relyt_dwsu_entraid_config` Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship a new Terraform resource `relyt_dwsu_entraid_config` that manages Microsoft Entra ID SSO config on a DWSU's DMS via existing `PUT/GET/DELETE /api/entraid-config` endpoints, plus complete docs/examples/demo for "out-of-the-box" customer experience.

**Architecture:** Independent singleton resource mirroring the existing `dwsu_user_policy` pattern. Host is resolved per-DWSU by `client.GetDwsu(dwsu_id) → Endpoints[Type=="openapi"].URI` (new helper `RouteDwsuOpenApiHost`). Auth reuses the provider-level `auth_key` header (`x-maxone-api-key`) — no provider schema change. Customer must use an ACCOUNTADMIN-role key; multi-role scenarios use provider aliases.

**Tech Stack:** Go 1.21, terraform-plugin-framework v1.11.0, terraform-plugin-docs v0.19.4. Existing helpers: `common.CommonRetry`, `client.doHttpRequest`, `RelytClientResource`.

**Reference:** Design decisions and rationale → `docs/plans/2026-05-19-entraid-config-resource-design.md`.

---

## Task ordering & rationale

1. **Pure-Go logic first (TDD)**: `PickOpenApiURIFromEndpoints` is the only function we can clean-TDD without network — do it first.
2. **Client types + methods**: build bottom-up so resource code can compile when we get to it.
3. **Resource code in 3 slices**: Schema/Metadata → Create+Update+put → Read → Delete+Import. Each slice compiles independently (`go build ./...`).
4. **Wire into provider** only after all resource methods exist (otherwise `go build` breaks).
5. **Customer-facing surface** (examples, docs, demo module) last — code is stable, easy to generate.
6. **Verification (smoke tests against real DWSU)** is the final task. The 4 risk items in design §10 are *discovered & confirmed* in this step.

---

## Task 1: Add `PickOpenApiURIFromEndpoints` pure helper + unit tests (TDD)

**Why first**: only piece pure enough to TDD. Catches misunderstandings about `Endpoints[]` shape before they propagate.

**Files:**
- Modify: `internal/provider/common/common_util_test.go` (append 4 tests)
- Modify: `internal/provider/common/common_util.go` (append function)

**Step 1: Write the failing tests**

Append to `internal/provider/common/common_util_test.go`:

```go
func TestPickOpenApiURI_Found(t *testing.T) {
    endpoints := []client.Endpoints{
        {Type: "web_console", URI: "https://console.example.com"},
        {Type: "openapi", URI: "https://api.example.com"},
        {Type: "database", URI: "db.example.com:5432"},
    }
    uri, err := PickOpenApiURIFromEndpoints(endpoints)
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if uri != "https://api.example.com" {
        t.Fatalf("expected https://api.example.com, got %q", uri)
    }
}

func TestPickOpenApiURI_NotFound(t *testing.T) {
    endpoints := []client.Endpoints{
        {Type: "web_console", URI: "https://console.example.com"},
        {Type: "database", URI: "db.example.com:5432"},
    }
    _, err := PickOpenApiURIFromEndpoints(endpoints)
    if err == nil {
        t.Fatal("expected error when no openapi endpoint present, got nil")
    }
}

func TestPickOpenApiURI_EmptyList(t *testing.T) {
    _, err := PickOpenApiURIFromEndpoints([]client.Endpoints{})
    if err == nil {
        t.Fatal("expected error for empty endpoint list, got nil")
    }
}

func TestPickOpenApiURI_FirstMatchWins(t *testing.T) {
    endpoints := []client.Endpoints{
        {Type: "openapi", URI: "https://first.example.com"},
        {Type: "openapi", URI: "https://second.example.com"},
    }
    uri, err := PickOpenApiURIFromEndpoints(endpoints)
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if uri != "https://first.example.com" {
        t.Fatalf("expected first match, got %q", uri)
    }
}
```

Add `"terraform-provider-relyt/internal/provider/client"` to the test file imports if not already present.

**Step 2: Run tests — expect compile failure**

```bash
cd /Users/coo/source/terraform-provider-relyt
go test ./internal/provider/common/ -run TestPickOpenApiURI -v
```

Expected: build fails — `PickOpenApiURIFromEndpoints` is undefined.

**Step 3: Implement minimal function**

Append to `internal/provider/common/common_util.go`:

```go
// PickOpenApiURIFromEndpoints returns the URI of the first endpoint whose
// Type is "openapi". Pure function (no network), unit-testable.
func PickOpenApiURIFromEndpoints(endpoints []client.Endpoints) (string, error) {
    for _, ep := range endpoints {
        if ep.Type == "openapi" && ep.URI != "" {
            return ep.URI, nil
        }
    }
    return "", fmt.Errorf("no endpoint of type 'openapi' found")
}
```

(`client` and `fmt` packages already imported in this file.)

**Step 4: Run tests — expect PASS**

```bash
go test ./internal/provider/common/ -run TestPickOpenApiURI -v
```

Expected: 4 tests pass.

**Step 5: Commit**

```bash
git add internal/provider/common/common_util.go internal/provider/common/common_util_test.go
git commit -m "$(cat <<'EOF'
add PickOpenApiURIFromEndpoints helper

Pure function (no network) used by upcoming RouteDwsuOpenApiHost to locate
the per-DWSU DMS API host from DwsuModel.Endpoints[].

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add `RouteDwsuOpenApiHost` wrapper (network)

**Files:**
- Modify: `internal/provider/common/common_util.go` (append function)

**Note:** No unit test for the wrapper itself — it just composes `client.GetDwsu` + `PickOpenApiURIFromEndpoints` + retry. The pure logic is already covered by Task 1; network behavior is covered by smoke tests in Task 14.

**Step 1: Implement**

Append to `internal/provider/common/common_util.go`:

```go
// RouteDwsuOpenApiHost fetches the DWSU model and returns its openapi endpoint URI.
// Adds diagnostics on failure (mirrors RouteRegionUri's style).
func RouteDwsuOpenApiHost(ctx context.Context, dwsuId string,
    relytClient *client.RelytClient, diag *diag.Diagnostics) string {
    dwsu, err := CommonRetry(ctx, func() (*client.DwsuModel, error) {
        return relytClient.GetDwsu(ctx, dwsuId)
    })
    if err != nil || dwsu == nil {
        errMsg := "GetDwsu returned nil"
        if err != nil {
            errMsg = err.Error()
        }
        diag.AddError("error fetching DWSU",
            "fail to fetch DWSU for openapi host resolution. dwsuId: "+dwsuId+" error: "+errMsg)
        return ""
    }
    uri, perr := PickOpenApiURIFromEndpoints(dwsu.Endpoints)
    if perr != nil {
        diag.AddError("openapi endpoint not found on DWSU",
            "dwsuId: "+dwsuId+" error: "+perr.Error())
        return ""
    }
    return uri
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/common/common_util.go
git commit -m "$(cat <<'EOF'
add RouteDwsuOpenApiHost helper

Composes client.GetDwsu + PickOpenApiURIFromEndpoints + CommonRetry to
resolve the per-DWSU DMS API host (e.g. https://api-<dwsu>.region.cloud)
that the upcoming entraid-config resource needs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add `EntraIdConfig` type + NOT_FOUND constant

**Files:**
- Modify: `internal/provider/client/relyt_data.go` (append type + const)

**Step 1: Add the error code constant**

Modify the existing `const (...)` block at the top of `internal/provider/client/relyt_data.go` to add one line:

```go
const (
    DPS_STATUS_READY     = "READY"
    DPS_STATUS_DROPPED   = "DROPPED"
    PRIVATE_LINK_READY   = "READY"
    PRIVATE_LINK_UNKNOWN = "UNKNOWN"
    CODE_SUCCESS         = 200
    CODE_USER_NOT_FOUND  = 134084
    CODE_DPS_NOT_FOUND   = 137073
    CODE_DWSU_NOT_FOUND  = 65544

    // CODE_ENTRAID_CONFIG_NOT_FOUND will be confirmed during smoke testing (Task 14).
    // Placeholder used by GetEntraIdConfig/DeleteEntraIdConfig to allow idempotent
    // not-found handling once the real code is known.
    CODE_ENTRAID_CONFIG_NOT_FOUND = 0 // TBD — replace after Task 14 smoke test
)
```

**Step 2: Add the type**

Append to the bottom of `internal/provider/client/relyt_data.go` (after `UserSecurityPolicy`):

```go
// EntraIdConfig is the request/response body for /api/entraid-config.
// Note: Enabled has no `omitempty` because `false` must be sent over the wire
// to soft-disable SSO while retaining tenant/client IDs.
type EntraIdConfig struct {
    TenantId   string `json:"tenantId,omitempty"`
    ClientId   string `json:"clientId,omitempty"`
    TenantType string `json:"tenantType,omitempty"`
    Enabled    bool   `json:"enabled"`
}
```

**Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 4: Commit**

```bash
git add internal/provider/client/relyt_data.go
git commit -m "$(cat <<'EOF'
add EntraIdConfig type and not-found error code placeholder

Adds the JSON model for /api/entraid-config plus a placeholder
CODE_ENTRAID_CONFIG_NOT_FOUND. The real code value is TBD and will be
filled in during the smoke-test phase. Enabled deliberately lacks omitempty
so false can be sent to soft-disable SSO.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add 3 client methods (Get / Put / Delete)

**Files:**
- Modify: `internal/provider/client/relyt_client.go` (append 3 methods at file end)

**Step 1: Append methods**

Append to the end of `internal/provider/client/relyt_client.go`:

```go
func (p *RelytClient) GetEntraIdConfig(ctx context.Context, dmsHost, dwsuId string) (*EntraIdConfig, error) {
    path := "/api/entraid-config"
    resp := CommonRelytResponse[EntraIdConfig]{}
    handler := func(response *CommonRelytResponse[EntraIdConfig], respString []byte) (*CommonRelytResponse[EntraIdConfig], error) {
        if response.Code != CODE_SUCCESS && response.Code != CODE_ENTRAID_CONFIG_NOT_FOUND {
            body := string(respString)
            tflog.Error(ctx, "error call api! entraid-config GET resp code not success! dwsuId: "+dwsuId+" body: "+body)
            return response, fmt.Errorf(body)
        }
        return response, nil
    }
    err := doHttpRequest(p, ctx, dmsHost, path, "GET", &resp, nil, nil, handler)
    if err != nil {
        tflog.Error(ctx, "Error get entraid-config: "+err.Error())
        return nil, err
    }
    return resp.Data, nil
}

func (p *RelytClient) PutEntraIdConfig(ctx context.Context, dmsHost, dwsuId string, cfg EntraIdConfig) (*EntraIdConfig, error) {
    path := "/api/entraid-config"
    resp := CommonRelytResponse[EntraIdConfig]{}
    err := doHttpRequest(p, ctx, dmsHost, path, "PUT", &resp, cfg, nil, nil)
    if err != nil {
        tflog.Error(ctx, "Error put entraid-config: "+err.Error())
        return nil, err
    }
    return resp.Data, nil
}

func (p *RelytClient) DeleteEntraIdConfig(ctx context.Context, dmsHost, dwsuId string) error {
    path := "/api/entraid-config"
    resp := CommonRelytResponse[string]{}
    handler := func(response *CommonRelytResponse[string], respString []byte) (*CommonRelytResponse[string], error) {
        if response.Code != CODE_SUCCESS && response.Code != CODE_ENTRAID_CONFIG_NOT_FOUND {
            body := string(respString)
            tflog.Error(ctx, "error call api! entraid-config DELETE resp code not success! dwsuId: "+dwsuId+" body: "+body)
            return response, fmt.Errorf(body)
        }
        return nil, nil
    }
    err := doHttpRequest(p, ctx, dmsHost, path, "DELETE", &resp, nil, nil, handler)
    if err != nil {
        tflog.Info(ctx, "delete entraid-config err: "+err.Error())
    }
    return err
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/client/relyt_client.go
git commit -m "$(cat <<'EOF'
add entraid-config client methods

Adds GetEntraIdConfig / PutEntraIdConfig / DeleteEntraIdConfig against
/api/entraid-config on the DWSU's openapi endpoint. dwsuId is passed for
log correlation only; the path is fixed (no dwsu in URL because the DMS
host is already per-DWSU).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Create `EntraIdConfigModel` (TF-side struct)

**Files:**
- Create: `internal/provider/model/entraid_config.go`

**Step 1: Write the model**

Create `internal/provider/model/entraid_config.go`:

```go
package model

import "github.com/hashicorp/terraform-plugin-framework/types"

type EntraIdConfigModel struct {
    DwsuId     types.String `tfsdk:"dwsu_id"`
    TenantId   types.String `tfsdk:"tenant_id"`
    ClientId   types.String `tfsdk:"client_id"`
    TenantType types.String `tfsdk:"tenant_type"`
    Enabled    types.Bool   `tfsdk:"enabled"`
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/model/entraid_config.go
git commit -m "$(cat <<'EOF'
add EntraIdConfigModel for terraform state

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Resource skeleton + Schema + Metadata

**Files:**
- Create: `internal/provider/resource/dwsu_entraid_config_resource.go`

**Step 1: Write skeleton (Schema + Metadata only; CRUD methods are stubs)**

Create the file with this content:

```go
package resource

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator/stringvalidator"
)

var (
    _ resource.Resource                = &dwsuEntraIdConfig{}
    _ resource.ResourceWithConfigure   = &dwsuEntraIdConfig{}
    _ resource.ResourceWithImportState = &dwsuEntraIdConfig{}
)

func NewDwsuEntraIdConfig() resource.Resource {
    return &dwsuEntraIdConfig{}
}

type dwsuEntraIdConfig struct {
    RelytClientResource
}

func (r *dwsuEntraIdConfig) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_dwsu_entraid_config"
}

func (r *dwsuEntraIdConfig) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Version: 0,
        Attributes: map[string]schema.Attribute{
            "dwsu_id": schema.StringAttribute{
                Required:    true,
                PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
                Description: "The ID of the DWSU to attach Entra ID SSO config to.",
            },
            "tenant_id": schema.StringAttribute{
                Required:    true,
                Description: "Azure AD tenant (directory) GUID.",
            },
            "client_id": schema.StringAttribute{
                Required:    true,
                Description: "Azure AD application (client) GUID.",
            },
            "tenant_type": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Default:     stringdefault.StaticString("single"),
                Validators:  []validator.String{stringvalidator.OneOf("single", "multi")},
                Description: "'single' = only users in the configured tenant can log in; 'multi' = any Azure org user. Default 'single'.",
            },
            "enabled": schema.BoolAttribute{
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(true),
                Description: "Whether SSO is active. Set false to retain config but disable login. Default true.",
            },
        },
    }
}

// Stubbed; implemented in subsequent tasks.
func (r *dwsuEntraIdConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *dwsuEntraIdConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *dwsuEntraIdConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *dwsuEntraIdConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
func (r *dwsuEntraIdConfig) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("dwsu_id"), req, resp)
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

> **Note for executing engineer**: If any import path doesn't resolve (terraform-plugin-framework reorganized some modules across minor versions), check `internal/provider/resource/dwUser_resource.go` to see how the same packages are imported there.

**Step 3: Commit**

```bash
git add internal/provider/resource/dwsu_entraid_config_resource.go
git commit -m "$(cat <<'EOF'
scaffold relyt_dwsu_entraid_config resource

Adds Metadata + Schema + stub CRUD + ImportState. CRUD bodies are filled
in by subsequent tasks. Schema mirrors design doc 4.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Implement `put` helper + Create + Update

**Files:**
- Modify: `internal/provider/resource/dwsu_entraid_config_resource.go`

**Step 1: Add the private `put` helper + replace Create/Update bodies**

Edit the file:

1. Add imports (top of file):
   ```go
   import (
       // ... existing imports
       "github.com/hashicorp/terraform-plugin-framework/diag"
       "github.com/hashicorp/terraform-plugin-framework/tfsdk"
       "terraform-provider-relyt/internal/provider/client"
       "terraform-provider-relyt/internal/provider/common"
       tfModel "terraform-provider-relyt/internal/provider/model"
   )
   ```

2. Replace the stub `Create` and `Update`:

```go
func (r *dwsuEntraIdConfig) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State)
}

func (r *dwsuEntraIdConfig) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    r.put(ctx, req.Plan, &resp.Diagnostics, &resp.State)
}
```

3. Append the private helper at the bottom of the file:

```go
// put applies the plan to the backend via PUT /api/entraid-config and writes
// the same plan into TF state on success. Shared by Create and Update because
// the backend endpoint is upsert-semantics.
func (r *dwsuEntraIdConfig) put(ctx context.Context, plan tfsdk.Plan, diags *diag.Diagnostics, state *tfsdk.State) {
    var m tfModel.EntraIdConfigModel
    diags.Append(plan.Get(ctx, &m)...)
    if diags.HasError() {
        return
    }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, m.DwsuId.ValueString(), r.client, diags)
    if diags.HasError() {
        return
    }

    _, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
        return r.client.PutEntraIdConfig(ctx, dmsHost, m.DwsuId.ValueString(), client.EntraIdConfig{
            TenantId:   m.TenantId.ValueString(),
            ClientId:   m.ClientId.ValueString(),
            TenantType: m.TenantType.ValueString(),
            Enabled:    m.Enabled.ValueBool(),
        })
    })
    if err != nil {
        diags.AddError("Error writing entraid-config",
            "PUT /api/entraid-config failed for dwsu_id="+m.DwsuId.ValueString()+": "+err.Error())
        return
    }
    diags.Append(state.Set(ctx, &m)...)
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/resource/dwsu_entraid_config_resource.go
git commit -m "$(cat <<'EOF'
implement Create + Update for entraid-config resource

Shares a single 'put' helper because the backend PUT is upsert-semantics.
Errors carry the dwsu_id for easier debugging.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Implement Read (with drift handling)

**Files:**
- Modify: `internal/provider/resource/dwsu_entraid_config_resource.go`

**Step 1: Add `types` import + replace Read stub**

1. Ensure import block contains:
   ```go
   "github.com/hashicorp/terraform-plugin-framework/types"
   ```

2. Replace the stub `Read`:

```go
func (r *dwsuEntraIdConfig) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state tfModel.EntraIdConfigModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, state.DwsuId.ValueString(), r.client, &resp.Diagnostics)
    if resp.Diagnostics.HasError() {
        return
    }

    cfg, err := common.CommonRetry(ctx, func() (*client.EntraIdConfig, error) {
        return r.client.GetEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString())
    })
    if err != nil {
        resp.Diagnostics.AddError("Error reading entraid-config",
            "GET /api/entraid-config failed for dwsu_id="+state.DwsuId.ValueString()+": "+err.Error())
        return
    }

    // Drift: backend has no config (externally deleted or never created).
    // Remove from state → next plan will show '+ create'.
    if cfg == nil {
        resp.State.RemoveResource(ctx)
        return
    }

    state.TenantId = types.StringValue(cfg.TenantId)
    state.ClientId = types.StringValue(cfg.ClientId)
    state.TenantType = types.StringValue(cfg.TenantType)
    state.Enabled = types.BoolValue(cfg.Enabled)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

**Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/resource/dwsu_entraid_config_resource.go
git commit -m "$(cat <<'EOF'
implement Read with drift handling for entraid-config

When the backend returns data=null (config externally deleted), the
resource is removed from TF state so the next plan triggers re-create.
Field-level drift is reflected by overwriting state with backend truth.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Implement Delete

**Files:**
- Modify: `internal/provider/resource/dwsu_entraid_config_resource.go`

**Step 1: Replace Delete stub**

```go
func (r *dwsuEntraIdConfig) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state tfModel.EntraIdConfigModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    dmsHost := common.RouteDwsuOpenApiHost(ctx, state.DwsuId.ValueString(), r.client, &resp.Diagnostics)
    if resp.Diagnostics.HasError() {
        return
    }

    _, err := common.CommonRetry(ctx, func() (*string, error) {
        if e := r.client.DeleteEntraIdConfig(ctx, dmsHost, state.DwsuId.ValueString()); e != nil {
            return nil, e
        }
        s := "ok"
        return &s, nil
    })
    if err != nil {
        resp.Diagnostics.AddError("Error deleting entraid-config",
            "DELETE /api/entraid-config failed for dwsu_id="+state.DwsuId.ValueString()+": "+err.Error())
    }
    // No explicit state clear — framework removes the resource automatically on success.
}
```

**Step 2: Verify build + run any tests we have**

```bash
go build ./...
go test ./internal/provider/...
```

Expected: build passes, all existing tests pass (no behavior changes to anything but the new resource).

**Step 3: Commit**

```bash
git add internal/provider/resource/dwsu_entraid_config_resource.go
git commit -m "$(cat <<'EOF'
implement Delete for entraid-config resource

DeleteEntraIdConfig already swallows CODE_ENTRAID_CONFIG_NOT_FOUND, so
destroy is idempotent even if the config was already removed externally.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Register resource in provider

**Files:**
- Modify: `internal/provider/provider.go:283-295` (the `Resources()` function)

**Step 1: Add the registration**

Edit `internal/provider/provider.go`. In the `Resources()` function, append `relytRS.NewDwsuEntraIdConfig` to the slice:

```go
func (p *RelytProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        relytRS.NewdwUserResource,
        relytRS.NewDpsResource,
        relytRS.NewDwsuResource,
        relytRS.NewPrivateLinkResource,
        relytRS.NewDwsuIntegrationInfoResource,
        relytRS.NewDwsuDatabaseResource,
        relytRS.NewDwsuExternalSchemaResource,
        relytRS.NewdwsuUserPolicy,
        relytRS.NewDwsuEntraIdConfig, // added
        //relytRS.NewTestResource,
    }
}
```

**Step 2: Verify build + go vet**

```bash
go build ./...
go vet ./...
```

Expected: no errors.

**Step 3: Commit**

```bash
git add internal/provider/provider.go
git commit -m "$(cat <<'EOF'
register relyt_dwsu_entraid_config resource

Wires the new resource into the provider so terraform CLI can discover it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Add client smoke tests

**Note:** These are smoke scripts, not unit tests. They hit the real backend and require `env/testConf.json` + a real DWSU with ACCOUNTADMIN auth_key. They are deliberately not run in `go test ./...`-by-default workflows (the existing `relyt_client_test.go` tests have the same property).

**Files:**
- Modify: `internal/provider/client/relyt_client_test.go` (append tests + 2 test constants)

**Step 1: Append constants + tests**

Append to end of `internal/provider/client/relyt_client_test.go`:

```go
// === Entra ID SSO smoke tests ===
// Fill in with values from your local test environment before running.
const (
    testEntraIdDmsHost = "<https://api-<your-dwsu-domain>>"
    testEntraIdDwsuId  = "<your-dwsu-id>"
)

func TestPutEntraIdConfig(t *testing.T) {
    resp, err := client.PutEntraIdConfig(ctx, testEntraIdDmsHost, testEntraIdDwsuId, EntraIdConfig{
        TenantId:   "00000000-0000-0000-0000-000000000000",
        ClientId:   "11111111-1111-1111-1111-111111111111",
        TenantType: "single",
        Enabled:    true,
    })
    if err != nil {
        t.Fatalf("put err: %v", err)
    }
    fmt.Printf("put resp: %+v\n", resp)
}

func TestGetEntraIdConfig(t *testing.T) {
    cfg, err := client.GetEntraIdConfig(ctx, testEntraIdDmsHost, testEntraIdDwsuId)
    if err != nil {
        t.Fatalf("get err: %v", err)
    }
    fmt.Printf("get resp: %+v\n", cfg)
}

func TestDeleteEntraIdConfig(t *testing.T) {
    err := client.DeleteEntraIdConfig(ctx, testEntraIdDmsHost, testEntraIdDwsuId)
    if err != nil {
        t.Fatalf("delete err: %v", err)
    }
}

func TestEntraIdConfig_NotFoundReadIdempotent(t *testing.T) {
    // First DELETE to guarantee not-found state, then GET.
    _ = client.DeleteEntraIdConfig(ctx, testEntraIdDmsHost, testEntraIdDwsuId)
    cfg, err := client.GetEntraIdConfig(ctx, testEntraIdDmsHost, testEntraIdDwsuId)
    if err != nil {
        // If err != nil, the not-found path is NOT being handled — likely because
        // CODE_ENTRAID_CONFIG_NOT_FOUND is wrong, or backend returns HTTP 4xx
        // (not 200 + business code). See Task 14.
        t.Fatalf("expected nil err on missing config (means CODE_ENTRAID_CONFIG_NOT_FOUND is wrong or backend returns non-200 HTTP); got %v", err)
    }
    if cfg != nil {
        t.Fatalf("expected nil cfg, got %+v", cfg)
    }
}
```

**Step 2: Verify build only (do NOT run these tests yet — values are placeholders)**

```bash
go build ./...
go vet ./...
```

Expected: no errors. (Tests will run in Task 14.)

**Step 3: Commit**

```bash
git add internal/provider/client/relyt_client_test.go
git commit -m "$(cat <<'EOF'
add entraid-config client smoke tests

Sibling to the existing relyt_client_test.go smoke scripts — requires
env/testConf.json + a real DWSU. The NotFoundReadIdempotent test is the
mechanism for confirming CODE_ENTRAID_CONFIG_NOT_FOUND during Task 14.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Example .tf file

**Files:**
- Create: `examples/resources/relyt_dwsu_entraid_config/resource.tf`

**Step 1: Write the example**

Create the directory and file with this content:

```hcl
# Reference an existing DWSU
data "relyt_dwsus" "all" {}

resource "relyt_dwsu_entraid_config" "sso" {
  dwsu_id   = data.relyt_dwsus.all.records[0].id
  tenant_id = "00000000-0000-0000-0000-000000000000" # Azure portal → Directory (tenant) ID
  client_id = "00000000-0000-0000-0000-000000000000" # Azure portal → Application (client) ID

  # The two below are optional and default to "single" and true.
  tenant_type = "single" # "single" | "multi"
  enabled     = true
}
```

**Step 2: Verify with `terraform fmt`**

```bash
terraform fmt -check examples/resources/relyt_dwsu_entraid_config/resource.tf
```

Expected: no output (file is already canonically formatted).

If terraform CLI is not installed, skip this check — the `go generate` step in Task 13 also formats examples.

**Step 3: Commit**

```bash
git add examples/resources/relyt_dwsu_entraid_config/
git commit -m "$(cat <<'EOF'
add example for relyt_dwsu_entraid_config

This example is automatically embedded into the auto-generated registry doc
by tfplugindocs (via the //go:generate directive in main.go).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Generate docs

**Files:**
- Generate: `docs/resources/dwsu_entraid_config.md` (via tfplugindocs)

**Step 1: Run go generate**

```bash
cd /Users/coo/source/terraform-provider-relyt
go generate ./...
```

This runs two directives from `main.go:19,23`:
1. `terraform fmt -recursive ./examples/` — formats all examples
2. `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name relyt` — generates `docs/resources/*.md` from `Schema()` descriptions + example files

**Step 2: Inspect the generated file**

```bash
ls -la docs/resources/dwsu_entraid_config.md
cat docs/resources/dwsu_entraid_config.md
```

Expected:
- File exists
- Contains a "Schema" section with all 5 attributes
- Contains an "Example Usage" block embedding the .tf from Task 12

> **If generation fails**: check that `examples/` directory has correct structure (compare with `examples/resources/relyt_dwsu_user_policy/resource.tf`).

**Step 3: Commit**

```bash
git add docs/resources/dwsu_entraid_config.md
git diff --cached --stat   # sanity check what's being committed
git commit -m "$(cat <<'EOF'
generate registry doc for relyt_dwsu_entraid_config

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If `go generate` reformatted other example files, include those too — they should be no-op formatting fixes.

---

## Task 14: Augment docs with Prerequisites + alias example

**Files:**
- Modify: `docs/resources/dwsu_entraid_config.md`

**Step 1: Open the generated file**

It will look roughly like:

```markdown
---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "relyt_dwsu_entraid_config Resource - relyt"
subcategory: ""
description: |-
---

# relyt_dwsu_entraid_config (Resource)

## Example Usage

```terraform
data "relyt_dwsus" "all" {}
...
```

<!-- schema generated by tfplugindocs -->
## Schema
...
```

**Step 2: Insert Prerequisites section between the H1 and `## Example Usage`**

Use Edit tool. After the line `# relyt_dwsu_entraid_config (Resource)` (and any blank line that follows), insert:

```markdown

Registers Microsoft Entra ID (formerly Azure AD) SSO configuration on a DWSU's
DMS instance. After applying this resource, users assigned an App Role in your
Azure tenant can log into the DMS Console using their Microsoft account.

## Prerequisites

1. Azure AD App registration with App Roles defined and assigned to users.
   See the Relyt SSO setup guide for the Azure-side walkthrough.
2. The provider's `auth_key` must belong to a user with **ACCOUNTADMIN** role on
   the target DWSU — only ACCOUNTADMIN can manage SSO config. Using a key from
   a SYSTEMADMIN-only user will result in:
   ```
   HTTP 403 Permission denied: Account admin role required to manage SSO config
   ```
3. The DWSU must already exist; reference it via `relyt_dwsu.<name>.id` or
   `data.relyt_dwsus.<name>.records[*].id`.

## Multi-role example (provider alias)

If you need to manage both DWSU lifecycle (requires SYSTEMADMIN) and SSO config
(requires ACCOUNTADMIN) in a single Terraform configuration, use provider aliases:

```terraform
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
  provider = relyt.system
  # ...
}

resource "relyt_dwsu_entraid_config" "sso" {
  provider = relyt.account
  dwsu_id  = relyt_dwsu.dw.id
  # ...
}
```

## Import

```bash
terraform import relyt_dwsu_entraid_config.sso <dwsu_id>
```

```

> **Note**: The next time someone runs `go generate`, tfplugindocs may overwrite this file. Project currently has no `templates/` directory; if regeneration becomes painful, introduce `templates/resources/dwsu_entraid_config.md.tmpl` in a follow-up. For now we accept the manual maintenance burden — it's a one-off doc.

**Step 3: Commit**

```bash
git add docs/resources/dwsu_entraid_config.md
git commit -m "$(cat <<'EOF'
add Prerequisites and multi-role alias guidance to entraid-config doc

Spells out the ACCOUNTADMIN role requirement and the provider-alias pattern
for customers who need to manage both DWSU lifecycle and SSO config from
one Terraform configuration.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Demo module

**Files:**
- Create: `docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/main.tf`
- Create: `docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/variables.tf`
- Create: `docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/outputs.tf`

**Step 1: Write the three files**

`main.tf`:
```hcl
resource "relyt_dwsu_entraid_config" "this" {
  dwsu_id     = var.dwsu_id
  tenant_id   = var.tenant_id
  client_id   = var.client_id
  tenant_type = var.tenant_type
  enabled     = var.enabled
}
```

`variables.tf`:
```hcl
variable "dwsu_id" {
  type        = string
  description = "The ID of the DWSU to attach SSO config to."
}

variable "tenant_id" {
  type        = string
  description = "Azure AD tenant (directory) GUID."
}

variable "client_id" {
  type        = string
  description = "Azure AD application (client) GUID."
}

variable "tenant_type" {
  type        = string
  default     = "single"
  description = "'single' or 'multi'. Default 'single'."
}

variable "enabled" {
  type        = bool
  default     = true
  description = "Whether SSO is active. Default true."
}
```

`outputs.tf`:
```hcl
output "dwsu_id" {
  value       = relyt_dwsu_entraid_config.this.dwsu_id
  description = "Passthrough of dwsu_id for chained module composition."
}
```

**Step 2: Format**

```bash
terraform fmt docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/
```

(Skip if terraform CLI is unavailable.)

**Step 3: Commit**

```bash
git add docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/
git commit -m "$(cat <<'EOF'
add demo module for relyt_dwsu_entraid_config

Mirrors the shape of dw_user and relyt_dwsu_user_policy demo modules so
customers can compose it into the top-level main.tf as needed. Not wired
into the top-level main.tf by default (customers opt in when their
Azure-side App Registration is ready).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: Smoke test verification (RESOLVES 4 RISK ITEMS from design §10)

**Prerequisites**:
- A real DWSU with ACCOUNTADMIN auth_key available
- `env/testConf.json` populated (see existing `relyt_client_test.go` for the format)
- `kubectl` access to the DMS pod if you need to dump backend logs

**Step 1: Fill in real values in `relyt_client_test.go` (locally only — do NOT commit real values)**

Edit the two constants you added in Task 11:

```go
const (
    testEntraIdDmsHost = "https://api-<your-real-dwsu-domain>"  // from DwsuModel.Endpoints[Type=="openapi"].URI
    testEntraIdDwsuId  = "<your-real-dwsu-id>"
)
```

**Step 2: Run the smoke tests one by one and observe**

```bash
cd /Users/coo/source/terraform-provider-relyt

# Test PUT — confirms risk item #3 (Endpoints[openapi] exists & reachable) + #4 (PUT path lives on openapi endpoint)
go test ./internal/provider/client/ -run TestPutEntraIdConfig -v

# Test GET on existing config
go test ./internal/provider/client/ -run TestGetEntraIdConfig -v

# Test DELETE
go test ./internal/provider/client/ -run TestDeleteEntraIdConfig -v

# Test NOT_FOUND idempotency — this is where risk items #1 and #2 are resolved
go test ./internal/provider/client/ -run TestEntraIdConfig_NotFoundReadIdempotent -v
```

**Step 3: Resolve risk items**

| If `TestEntraIdConfig_NotFoundReadIdempotent` … | Means … | Do |
|---|---|---|
| **PASSES** with CODE_ENTRAID_CONFIG_NOT_FOUND=0 | Backend already returns code=200 data=null on not-found (`response.Code == CODE_SUCCESS` path matches; placeholder 0 is harmless) | Optionally remove the constant, or leave it as documented-but-unused |
| **FAILS** with err containing `code`: e.g. `{"code":134099,...}` | Backend returns HTTP 200 + a specific business error code | Update `CODE_ENTRAID_CONFIG_NOT_FOUND = <real value>` in `client/relyt_data.go`; rerun |
| **FAILS** with err containing `Error http code not 200! respCode: 404` | Backend returns plain HTTP 404 — `doHttpRequest` returns before calling codeHandler | Decision: either (a) extend `doHttpRequest` to allow specific HTTP statuses to enter codeHandler (significant change to `client/relyt_http_util.go`); or (b) accept that Read shows an error instead of triggering re-create. Default to (a) — see "If risk path 3 occurs" below. |

**If risk path 3 occurs** (HTTP 404 not handled by codeHandler), add this branch to `signedHttpRequestWithHeader` in `client/relyt_http_util.go` around line 215 (right before the current `resp.StatusCode != CODE_SUCCESS` check):

```go
// Allow codeHandler to inspect known-acceptable non-200 statuses
if resp.StatusCode != CODE_SUCCESS && codeHandler != nil {
    // Try to parse body into respMode so the handler can decide
    _ = json.Unmarshal(body, respMode)
    handler, herr := codeHandler(respMode, body)
    if handler != nil {
        respMode.Code = handler.Code
        respMode.Data = handler.Data
        respMode.Msg = handler.Msg
    }
    if herr == nil {
        return nil
    }
    return herr
}
```

Then change `GetEntraIdConfig` and `DeleteEntraIdConfig` handlers to also accept HTTP 404 (read `respMode.Code == CODE_ENTRAID_CONFIG_NOT_FOUND` after unmarshal — already in our handler). Rerun the test.

**Step 4: End-to-end smoke test via Terraform**

This validates the resource itself (not just client methods). Follow the script in `docs/plans/2026-05-19-entraid-config-resource-design.md` Appendix A:

```bash
# Build provider locally and override the registry
cd /Users/coo/source/terraform-provider-relyt
go install .
# (this installs the provider binary; see HashiCorp docs for ~/.terraformrc dev_overrides setup)

# Then in a separate test directory with provider, dwsu, entraid_config resources:
export RELYT_AUTH_KEY="<ACCOUNTADMIN secret key>"
terraform init
terraform apply -auto-approve

# Verify backend state
DMS="<openapi URI of the DWSU>"
curl -s $DMS/api/entraid-config | python3 -m json.tool
# Expect: data field populated

# Repeat steps 3–7 of Appendix A
```

**Step 5: If `CODE_ENTRAID_CONFIG_NOT_FOUND` was updated, commit the fix**

```bash
# Only if you had to change the constant value or http_util.go
git add internal/provider/client/relyt_data.go internal/provider/client/relyt_http_util.go
git commit -m "$(cat <<'EOF'
fix CODE_ENTRAID_CONFIG_NOT_FOUND from smoke test discovery

Real value confirmed via TestEntraIdConfig_NotFoundReadIdempotent against
a live DWSU. [Include any http_util.go change rationale here.]

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

**Step 6: Revert local-only test constants**

Before any PR / push, restore `testEntraIdDmsHost` and `testEntraIdDwsuId` to placeholder values (real values are environment-specific and must not be committed):

```go
const (
    testEntraIdDmsHost = "<https://api-<your-dwsu-domain>>"
    testEntraIdDwsuId  = "<your-dwsu-id>"
)
```

Do not commit these reverts as a separate commit — amend Task 11's commit only if they were never committed with real values in the first place. (Better: never edit them in a committed state. Use a `.local.go` shadow or a checked-out copy that lives outside git.)

---

## Done state

After Task 16:

- New resource `relyt_dwsu_entraid_config` is registered and functional
- 4 unit tests for `PickOpenApiURIFromEndpoints` pass in `go test ./...`
- Client smoke tests pass against a real DWSU
- `examples/resources/relyt_dwsu_entraid_config/resource.tf` exists
- `docs/resources/dwsu_entraid_config.md` includes prerequisites + multi-role alias
- Demo module exists at `docs/demo/terraform/modules/relyt/relyt_dwsu_entraid_config/`
- Appendix A smoke test in design doc has been walked end-to-end against a real DWSU at least once

## Skills to reference during execution

- `superpowers:verification-before-completion` — before marking each task done, confirm the verification command's output (don't just trust that "go build" was called)
- `superpowers:systematic-debugging` — if a smoke test fails with an unexpected error, work the failure systematically rather than guessing at the cause
- `superpowers:test-driven-development` — Task 1 is the canonical TDD task; treat its red-green-commit rhythm as the template

---

## Out of scope (explicitly NOT in this plan)

- Azure-side automation (App Registration, App Roles, user assignment) — separate follow-up if/when needed
- Wiring SSO module into top-level demo `main.tf` — left as opt-in
- Switching docs to a `templates/` system — only if regeneration becomes painful
- Acceptance test framework (`terraform-plugin-testing`) — project has no existing acceptance test culture; would be a separate cross-cutting initiative
