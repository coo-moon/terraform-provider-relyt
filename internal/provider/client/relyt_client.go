package client

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/url"
	"strconv"
)

func NewRelytClient(config RelytClientConfig) (RelytClient, error) {
	return RelytClient{config}, nil
}

type RelytClientConfig struct {
	ApiHost                   string                     `json:"apiHost"`
	AuthKey                   string                     `json:"authKey"`
	Role                      string                     `json:"role"`
	RegionApi                 string                     `json:"regionApi"`
	CheckTimeOut              int64                      `json:"checkTimeOut"`
	CheckInterval             int32                      `json:"checkInterval"`
	ClientTimeout             int32                      `json:"clientTimeout"`
	RelytDatabaseClientConfig *RelytDatabaseClientConfig `json:"relytDatabaseClientConfig"`
}

type RelytClient struct {
	RelytClientConfig
}

func (p *RelytClient) ListDwsu(ctx context.Context, pageSize, pageNumber int) ([]*DwsuModel, error) {
	resp := CommonRelytResponse[CommonPage[DwsuModel]]{}
	pageQuery := map[string]string{
		"pageSize":   strconv.Itoa(pageSize),
		"pageNumber": strconv.Itoa(pageNumber),
	}
	err := doHttpRequest(p, ctx, "", "/dwsu", "GET", &resp, nil, pageQuery, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data.Records, nil
}

func (p *RelytClient) CreateDwsu(ctx context.Context, request DwsuModel) (*CommonRelytResponse[string], error) {
	url := "/dwsu"
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, "", url, "POST", &resp, request, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetDwsu(ctx context.Context, dwServiceUnitId string) (*DwsuModel, error) {
	path := fmt.Sprintf("/dwsu/%s", dwServiceUnitId)
	resp := CommonRelytResponse[DwsuModel]{}
	handler := func(response *CommonRelytResponse[DwsuModel], respString []byte) (*CommonRelytResponse[DwsuModel], error) {
		if response.Code != CODE_SUCCESS && resp.Code != CODE_DWSU_NOT_FOUND {
			body := string(respString)
			tflog.Error(ctx, "error call api! resp code not success! body: "+body)
			return response, fmt.Errorf(body)
		}
		return response, nil
	}
	err := doHttpRequest(p, ctx, "", path, "GET", &resp, nil, nil, handler)
	if err != nil {
		tflog.Error(ctx, "Error get dwsu:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) DropDwsu(ctx context.Context, dwServiceUnitId string) error {
	path := fmt.Sprintf("/dwsu/%s", dwServiceUnitId)
	resp := CommonRelytResponse[string]{}
	handler := func(response *CommonRelytResponse[string], respString []byte) (*CommonRelytResponse[string], error) {
		if response.Code != CODE_SUCCESS && resp.Code != CODE_DWSU_NOT_FOUND {
			body := string(respString)
			tflog.Error(ctx, "error call api! resp code not success! body: "+body)
			return response, fmt.Errorf(body)
		}
		return response, nil
	}
	err := doHttpRequest(p, ctx, "", path, "DELETE", &resp, nil, nil, handler)
	if err != nil {
		tflog.Info(ctx, "delete dwsu err:"+err.Error())
		return err
	}
	return nil
}

func (p *RelytClient) ListDps(ctx context.Context, pageSize, pageNumber int, dwServiceUnitId string) ([]*DpsMode, error) {
	resp := CommonRelytResponse[CommonPage[DpsMode]]{}
	pageQuery := map[string]string{
		"pageSize":   strconv.Itoa(pageSize),
		"pageNumber": strconv.Itoa(pageNumber),
	}
	path := fmt.Sprintf("/dwsu/%s/dps", dwServiceUnitId)
	err := doHttpRequest(p, ctx, "", path, "GET", &resp, nil, pageQuery, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data.Records, nil
}

func (p *RelytClient) CreateDps(ctx context.Context, regionUri string, dwServiceUnitId string, mode DpsMode) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/dps", dwServiceUnitId)
	resp := CommonRelytResponse[string]{}
	if err := doHttpRequest(p, ctx, regionUri, path, "POST", &resp, mode, nil, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) PatchDps(ctx context.Context, regionUri string, dwServiceUnitId, dpsId string, mode DpsMode) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/dps/%s", dwServiceUnitId, dpsId)
	resp := CommonRelytResponse[string]{}
	if err := doHttpRequest(p, ctx, regionUri, path, "PATCH", &resp, mode, nil, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetDps(ctx context.Context, regionUri, dwServiceUnitId, dpsBizId string) (*DpsMode, error) {
	path := fmt.Sprintf("/dwsu/%s/dps/%s", dwServiceUnitId, dpsBizId)
	resp := CommonRelytResponse[DpsMode]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get dps:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) DropDps(ctx context.Context, regionUri, dwServiceUnitId, dpsBizId string) error {
	path := fmt.Sprintf("/dwsu/%s/dps/%s", dwServiceUnitId, dpsBizId)
	resp := CommonRelytResponse[string]{}
	handler := func(response *CommonRelytResponse[string], respString []byte) (*CommonRelytResponse[string], error) {
		if response.Code != CODE_SUCCESS && resp.Code != CODE_DPS_NOT_FOUND {
			body := string(respString)
			tflog.Error(ctx, "error call api! resp code not success! body: "+body)
			return response, fmt.Errorf(body)
		}
		return nil, nil
	}
	err := doHttpRequest(p, ctx, regionUri, path, "DELETE", &resp, nil, nil, handler)
	if err != nil {
		tflog.Info(ctx, "delete dps err:"+err.Error())
		return err
	}
	return nil
}

func (p *RelytClient) ListSpec(ctx context.Context, edition, dpsType, cloud, region string) ([]Spec, error) {
	path := fmt.Sprintf("/dwsu/edition/%s/dps/%s/specs", edition, dpsType)
	specList := CommonRelytResponse[[]Spec]{}
	parameter := map[string]string{"cloud": cloud, "region": region}
	err := doHttpRequest(p, ctx, "", path, "GET", &specList, nil, parameter, nil)
	if err != nil {
		return nil, err
	}
	return *specList.Data, nil
}

func (p *RelytClient) CreateAccount(ctx context.Context, regionUri string, dwsuId string, account Account) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/account", dwsuId)
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "POST", &resp, account, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) PatchAccount(ctx context.Context, regionUri string, dwsuId string, userId, passwd string) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s", dwsuId, url.PathEscape(userId))
	patchPwd := map[string]any{"initPassword": passwd, "resetMfa": true}
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PATCH", &resp, patchPwd, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetAccount(ctx context.Context, regionUri string, dwsuId string, userId string) (*CommonRelytResponse[Account], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[Account]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) DropAccount(ctx context.Context, regionUri string, dwsuId string, userId string) error {
	path := fmt.Sprintf("/dwsu/%s/user/%s", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[string]{}
	handler := func(response *CommonRelytResponse[string], respString []byte) (*CommonRelytResponse[string], error) {
		if response.Code != CODE_SUCCESS && resp.Code != CODE_USER_NOT_FOUND {
			body := string(respString)
			tflog.Error(ctx, "error call api! resp code not success! body: "+body)
			return response, fmt.Errorf(body)
		}
		return nil, nil
	}
	err := doHttpRequest(p, ctx, regionUri, path, "DELETE", &resp, nil, nil, handler)
	return err
}

func (p *RelytClient) AsyncAccountConfig(ctx context.Context, regionUri, dwsuId, userId string, asyncResult AsyncResult) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/asyncresult", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PUT", &resp, asyncResult, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) DeleteAsyncAccountConfig(ctx context.Context, regionUri, dwsuId, userId string) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/asyncresult", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "DELETE", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetAsyncAccountConfig(ctx context.Context, regionUri, dwsuId, userId string) (*AsyncResult, error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/asyncresult", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[AsyncResult]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) LakeFormationConfig(ctx context.Context, regionUri, dwsuId, userId string, formation LakeFormation) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/lakeformation", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PUT", &resp, formation, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetLakeFormationConfig(ctx context.Context, regionUri, dwsuId, userId string) (*LakeFormation, error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/lakeformation", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[LakeFormation]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) DeleteLakeFormationConfig(ctx context.Context, regionUri, dwsuId, userId string) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/lakeformation", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "DELETE", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetBoto3AccessInfo(ctx context.Context, regionUri, dwsuId, userId string) ([]*Boto3AccessInfo, error) {
	path := fmt.Sprintf("/dwsu/%s/user/%s/ak", dwsuId, url.PathEscape(userId))
	resp := CommonRelytResponse[[]*Boto3AccessInfo]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return *resp.Data, nil
}

func (p *RelytClient) GetOpenApiMeta(ctx context.Context, cloud, region string) (*OpenApiMetaInfo, error) {
	path := fmt.Sprintf("/infra/%s/%s/endpoint", url.PathEscape(cloud), url.PathEscape(region))
	resp := CommonRelytResponse[[]*OpenApiMetaInfo]{}
	err := doHttpRequest(p, ctx, "", path, "GET", &resp, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	lengthOfApi := len(*resp.Data)
	if lengthOfApi != 1 {
		return nil, fmt.Errorf("error read regionApi! length of api " + strconv.Itoa(lengthOfApi))
	}
	return (*resp.Data)[0], nil
}

func (p *RelytClient) GetDwsuOpenApiMeta(ctx context.Context, dwsuId string) (*OpenApiMetaInfo, error) {
	dwsu, err := p.GetDwsu(ctx, dwsuId)
	if err != nil {
		return nil, err
	}
	if dwsu == nil {
		return nil, fmt.Errorf("can't find dwsu meta! %s", dwsuId)
	}
	meta, err := p.GetOpenApiMeta(ctx, dwsu.Region.Cloud.ID, dwsu.Region.ID)
	return meta, err
}

func (p *RelytClient) GetDwsuServiceAccount(ctx context.Context, regionUri, dwServiceUnitId string) ([]*ServiceAccount, error) {
	path := fmt.Sprintf("/dwsu/%s/service-accounts", dwServiceUnitId)
	resp := CommonRelytResponse[[]*ServiceAccount]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get dwsu:"+err.Error())
		return nil, err
	}
	return *resp.Data, nil
}

func (p *RelytClient) CreatePrivateLinkService(ctx context.Context, regionUri, dwServiceUnitId string, pl PrivateLinkService) (*PrivateLinkService, error) {
	path := fmt.Sprintf("/dwsu/%s/private-link-services", dwServiceUnitId)
	resp := CommonRelytResponse[PrivateLinkService]{}
	pl.ServiceName = ""
	pl.Status = ""
	header := map[string]string{"x-maxone-idempotent": "false"}
	err := doHttpRequestWithHeader(p, ctx, regionUri, path, "PUT", &resp, pl, nil, header, nil)
	if err != nil {
		tflog.Error(ctx, "Error create private-link:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) GetPrivateLinkService(ctx context.Context, regionUri, dwServiceUnitId, serviceType string) (*PrivateLinkService, error) {
	path := fmt.Sprintf("/dwsu/%s/private-link-services/%s", dwServiceUnitId, serviceType)
	resp := CommonRelytResponse[PrivateLinkService]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get private-link:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) PatchPrivateLinkService(ctx context.Context, regionUri, dwServiceUnitId, serviceType string, pl PrivateLinkService) (*CommonRelytResponse[PrivateLinkService], error) {
	path := fmt.Sprintf("/dwsu/%s/private-link-services/%s", dwServiceUnitId, serviceType)
	pl.ServiceType = ""
	pl.ServiceName = ""
	pl.Status = ""
	resp := CommonRelytResponse[PrivateLinkService]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PATCH", &resp, pl, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error patch private-link:"+err.Error())
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) DeletePrivateLinkService(ctx context.Context, regionUri, dwServiceUnitId, serviceType string) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/private-link-services/%s", dwServiceUnitId, serviceType)
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "DELETE", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error delete private-link:"+err.Error())
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetIntegration(ctx context.Context, regionUri, dwServiceUnitId string) (*IntegrationInfo, error) {
	path := fmt.Sprintf("/dwsu/%s/integration", dwServiceUnitId)
	resp := CommonRelytResponse[IntegrationInfo]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get dwsu integration:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) PatchIntegration(ctx context.Context, regionUri, dwServiceUnitId string, info IntegrationInfo) (*CommonRelytResponse[string], error) {
	path := fmt.Sprintf("/dwsu/%s/integration", dwServiceUnitId)
	//这两个字段暂不支持更新
	info.RelytVpc = ""
	info.RelytPrincipal = ""
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PATCH", &resp, info, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error patch dwsu integration:"+err.Error())
		return nil, err
	}
	return &resp, nil
}

func (p *RelytClient) GetRegionEndpoints(ctx context.Context, cloud, region string) (*[]RegionEndpoint, error) {
	path := fmt.Sprintf("/infra/%s/%s/endpoint", url.PathEscape(cloud), url.PathEscape(region))
	resp := CommonRelytResponse[[]RegionEndpoint]{}
	err := doHttpRequest(p, ctx, "", path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error list region endpoints:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) GetUserSecurityPolicy(ctx context.Context, regionUri, dwsuId string) (*UserSecurityPolicy, error) {
	path := fmt.Sprintf("/dwsu/%s/user-security-policy", url.PathEscape(dwsuId))
	resp := CommonRelytResponse[UserSecurityPolicy]{}
	err := doHttpRequest(p, ctx, regionUri, path, "GET", &resp, nil, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get user-security-policy:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) PatchUserSecurityPolicy(ctx context.Context, regionUri, dwsuId string, userSecPolicy UserSecurityPolicy) (*string, error) {
	path := fmt.Sprintf("/dwsu/%s/user-security-policy", url.PathEscape(dwsuId))
	resp := CommonRelytResponse[string]{}
	err := doHttpRequest(p, ctx, regionUri, path, "PATCH", &resp, userSecPolicy, nil, nil)
	if err != nil {
		tflog.Error(ctx, "Error get user-security-policy:"+err.Error())
		return nil, err
	}
	return resp.Data, nil
}

func (p *RelytClient) GetEntraIdConfig(ctx context.Context, dmsHost, dwsuId string) (*EntraIdConfig, error) {
	path := "/api/entraid-config"
	resp := CommonRelytResponse[EntraIdConfig]{}
	// Backend returns code:200, data:null when the config is absent — no special
	// not-found code needed. resp.Data == nil signals "not configured" to callers.
	err := doHttpRequest(p, ctx, dmsHost, path, "GET", &resp, nil, nil, nil)
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
	// Backend returns code:200 success even when the config doesn't exist — DELETE
	// is already idempotent on the server side, no special handler needed.
	err := doHttpRequest(p, ctx, dmsHost, path, "DELETE", &resp, nil, nil, nil)
	if err != nil {
		tflog.Info(ctx, "delete entraid-config err: "+err.Error())
	}
	return err
}
