package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"net/url"
	"os"
	"strconv"
	"testing"
)

func init() {
	file, err := os.ReadFile("../../../env/testConf.json")
	config := RelytClientConfig{}
	if err != nil {
		fmt.Println("error read test file")
		return
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("error read test file")
		return
	}
	client, _ = NewRelytClient(config)
	ctx = context.WithValue(context.Background(), "provider", hclog.NewInterceptLogger)
}

var (
	client RelytClient
	ctx    context.Context
)

func TestReadConf(t *testing.T) {
	fmt.Println(client.RegionApi)
}

func TestCreateDwsu(t *testing.T) {

	request := DwsuModel{
		Alias:  "qingdeng-test",
		Domain: "qqqq-tst",
		Variant: &Variant{
			ID: "basic",
		},
		DefaultDps: &DpsMode{
			Name:        "hybrid",
			Description: "qingdeng-test",
			Engine:      "hybrid",
			Spec: &Spec{
				ID: 2,
			},
		},
		Edition: &Edition{
			ID: "standard",
		},
		Region: &Region{
			Cloud: &Cloud{
				ID: "ksc",
			},
			ID: "beijing-cicd",
		},
	}
	dwsu, err := client.CreateDwsu(ctx, request)
	fmt.Println(fmt.Sprintf("create result:%s resp:%s", strconv.FormatBool(err != nil), dwsu.Msg))

}

func TestListSpec(t *testing.T) {

	spec, err := client.ListSpec(ctx, "standard", "hybrid", "ksc", "beijing-cicd")
	if err != nil {
		fmt.Println("get error" + err.Error())
	}
	marshal, err := json.Marshal(spec)
	if err != nil {
		return
	}
	fmt.Println("spec list:" + string(marshal))
}

func TestListDwsu(t *testing.T) {
	response, err := client.ListDwsu(ctx, 100, 1)
	marshal, _ := json.Marshal(response)
	fmt.Println(fmt.Sprintf("list result:%t resp:%s", err == nil, string(marshal)))

}

func TestDeleteDwsu(t *testing.T) {
	err := client.DropDwsu(ctx, "4679498681344")
	if err != nil {
		fmt.Println(fmt.Sprintf("drop dwsu%s", err.Error()))
	}
}

func TestDeleteDps(t *testing.T) {
	err := client.DropDps(ctx, client.RegionApi, "4679367072512", "4679367072512-1472-abc")
	if err != nil {
		fmt.Println(fmt.Sprintf("drop dwsu%s", err.Error()))
	}
}

func TestPath(t *testing.T) {
	//path := client.ApiHost + "/qingdeng@zbyte-inc.com"
	path := client.ApiHost + "/中午@zbyte-inc.com"
	escape := url.PathEscape(path)
	fmt.Println(escape)

	var st []byte
	sprintf := fmt.Sprintf("abc:%s %t ", string(st), st == nil)
	fmt.Println(sprintf)
}

func TestGetBoto3(t *testing.T) {
	meta, err := client.GetOpenApiMeta(ctx, "aws", "ap-east-1")
	account, err := client.GetBoto3AccessInfo(ctx, meta.URI, "4953896353792", "zhanlu3@zbyte-inc.com")
	if err != nil {
		println("get boto3: " + err.Error())
		return
	}
	marshal, err := json.Marshal(account)
	fmt.Println("get result: " + string(marshal))
}

func TestCreateAccount(t *testing.T) {
	meta, err := client.GetOpenApiMeta(ctx, "aws", "ap-east-1")
	account, err := client.CreateAccount(ctx, meta.URI, "4954356420096", Account{
		InitPassword: "zZefE#12344R*",
		Name:         "demo3",
	})
	if err != nil {
		println("create account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(account)
	fmt.Println("create result: " + string(marshal))
}

func TestGetAccount(t *testing.T) {
	meta, err := client.GetOpenApiMeta(ctx, "aws", "us-east-1")
	account, err := client.GetAccount(ctx, meta.URI, "4679805844736", "zhanlu@zbyte-inc.com")
	if err != nil {
		println("create account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(account)
	fmt.Println("get account result: " + string(marshal))
}

func TestDropAccount(t *testing.T) {
	err := client.DropAccount(ctx, client.RegionApi, "4954356420096", "edit123")
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
}

func TestGetLakeInformation(t *testing.T) {
	//meta, err := client.GetOpenApiMeta(ctx, "ksc", "beijing-cicd")
	//fmt.Println(meta.URI)
	lakeinfo, err := client.GetLakeFormationConfig(ctx, client.RegionApi, "4954776226048", "demo6")
	if err != nil {
		println("get account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestDeleteLakeInformation(t *testing.T) {
	lakeinfo, err := client.DeleteLakeFormationConfig(ctx, client.RegionApi, "4679482247936", "demo6")
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestConfigLakeInformation(t *testing.T) {
	lakeinfo, err := client.LakeFormationConfig(ctx, client.RegionApi, "4954776226048", "demo6", LakeFormation{
		IAMRole: "ttttt3",
	})
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestConfigAsync(t *testing.T) {
	lakeinfo, err := client.AsyncAccountConfig(ctx, client.RegionApi, "4679482247936", "demo6", AsyncResult{
		AwsIamArn:        "agcc",
		S3LocationPrefix: "defs",
	})
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestGetAsyncConfig(t *testing.T) {
	lakeinfo, err := client.GetAsyncAccountConfig(ctx, client.RegionApi, "4679483645184", "demo10")
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestDeleteAsyncConfig(t *testing.T) {
	lakeinfo, err := client.DeleteAsyncAccountConfig(ctx, client.RegionApi, "4954356420096", "demo3")
	if err != nil {
		println("delete account: " + err.Error())
		return
	}
	marshal, err := json.Marshal(lakeinfo)
	fmt.Println("info " + string(marshal))
}

func TestGetOpenApiMeta(t *testing.T) {

	//client.AuthKey = "801b901dce1a98f2QCH6yiakoTAVMgF0ssLc2tjvJ5duk0s5sa4j919DIBfkiCxd"
	//client.AuthKey = "9a3727e5b9c0ddabaGbll2HVLVKLLY1AyjOilAqeyPOBAb74A7VlJRAdTi0bJWJd"
	//client.Role = ""
	//meta, err := client.GetOpenApiMeta(ctx, "aws", "ap-east-1")
	meta, err := client.GetOpenApiMeta(ctx, "ksc", "beijing-cicd")
	if err != nil {
		fmt.Println(fmt.Sprintf("get dwsu%s", err.Error()))
	}
	marshal, err := json.Marshal(meta)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get dwsu%s", string(marshal)))

}

func TestGetDwsu(t *testing.T) {

	mode, err := client.GetDwsu(ctx, "4679805844736")
	if err != nil {
		fmt.Println(fmt.Sprintf("get dwsu%s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get dwsu%s", string(marshal)))
}

func TestGetDwsuServiceAccount(t *testing.T) {
	api, err := client.GetDwsuOpenApiMeta(ctx, "4954339339520")
	mode, err := client.GetDwsuServiceAccount(ctx, api.URI, "4954339339520")
	//mode, err := client.GetDwsuServiceAccount(ctx, "4679344580864")
	if len(mode) > 0 {
		fmt.Println("accoiunt not zero")
	}
	if err != nil {
		fmt.Println(fmt.Sprintf("get dwsu%s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get dwsu%s", string(marshal)))
}

func TestGetDwsuApiMeta(t *testing.T) {
	mode, err := client.GetDwsuOpenApiMeta(ctx, "4679805844736")
	if err != nil {
		fmt.Println(fmt.Sprintf("get api meta%s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get api meta %s", string(marshal)))
}

func TestGetDps(t *testing.T) {

	mode, err := client.GetDps(ctx, "http://120.92.110.101:80", "4679350645248", "4679350645248-1458")
	if err != nil {
		fmt.Println(fmt.Sprintf("get dps error %s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get dps %s", string(marshal)))
}

func TestRegion(t *testing.T) {
	var stt any
	//stt = RelytClient{}
	valueString := stt.(RelytClient)
	fmt.Println("succ" + valueString.RegionApi)

	//provider.RouteRegionUri(ctx, client, diagnose)
}

func TestGetPrivateLinkService(t *testing.T) {
	mode, err := client.GetPrivateLinkService(ctx, client.RegionApi, "4679805844736", "data_api")
	//mode, err := client.GetPrivateLinkService(ctx, client.RegionApi, "4679805844736", "database")
	if err != nil {
		fmt.Println(fmt.Sprintf("get privatelink error %s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("get privatelink %s", string(marshal)))
}

func TestCreatePrivateLinkService(t *testing.T) {
	service := PrivateLinkService{
		AllowedPrincipals: nil,
		ServiceName:       "",
		//ServiceType:     "data_api",
		ServiceType: "database",
		Status:      "",
	}
	mode, err := client.CreatePrivateLinkService(ctx, client.RegionApi, "4679805844736", service)
	if err != nil {
		fmt.Println(fmt.Sprintf("create privatelink error %s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("create privatelink %s", string(marshal)))
}

func TestPatchPrivateLinkService(t *testing.T) {
	service := PrivateLinkService{
		AllowedPrincipals: &[]string{"*"},
		ServiceName:       "",
		ServiceType:       "data_api",
		Status:            "",
	}
	mode, err := client.PatchPrivateLinkService(ctx, client.RegionApi, "4679805844736", "data_api", service)
	if err != nil {
		fmt.Println(fmt.Sprintf("patch privatelink error %s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err patch %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("patch privatelink %s", string(marshal)))
}

func TestDeletePrivateLinkService(t *testing.T) {
	//mode, err := client.DeletePrivateLinkService(ctx, client.RegionApi, "4679805844736", "data_api")
	mode, err := client.DeletePrivateLinkService(ctx, client.RegionApi, "4679805844736", "data_api")
	if err != nil {
		fmt.Println(fmt.Sprintf("delete privatelink error %s", err.Error()))
	}
	marshal, err := json.Marshal(mode)
	if err != nil {
		fmt.Println(fmt.Sprintf("err get %s", err.Error()))
		return
	}
	fmt.Println(fmt.Sprintf("delete privatelink %s", string(marshal)))
}

func TestRelytClient_GetRegionEndpoints(t *testing.T) {
	resp, err := client.GetRegionEndpoints(ctx, "aws", "ap-east-1")
	if err != nil {
		fmt.Println("err" + err.Error())
		return
	}
	marshal, err := json.Marshal(resp)
	fmt.Println(string(marshal))
}

func TestMap(t *testing.T) {
	mm := map[string]RegionEndpoint{"abc": {ID: "abc"}}
	abc, _ := json.Marshal(mm)
	um := map[string]string{}
	err := json.Unmarshal(abc, &um)
	if err != nil {
		fmt.Println("err!", err.Error())
		return
	}
	fmt.Println(um["abc"])
}

func TestRelytClient_GetUserSecurityPolicy(t *testing.T) {
	dwsuId := "4682456933120"
	//resp, err := client.GetDwsuOpenApiMeta(ctx, dwsuId)
	//if err != nil {
	//	fmt.Println("err" + err.Error())
	//	return
	//}
	policy, err := client.GetUserSecurityPolicy(ctx, client.RegionApi, dwsuId)
	if err != nil {
		return
	}
	marshal, err := json.Marshal(policy)
	fmt.Println(string(marshal))

	suc, err := client.PatchUserSecurityPolicy(ctx, client.RegionApi, dwsuId, UserSecurityPolicy{
		ExtraMfaProtectionScopes:     []string{"DPS_OPERATIONS"},
		MFAStrategy:                  "OPTIONAL",
		RequiredChangingInitPassword: true,
	})
	marshal, err = json.Marshal(suc)
	fmt.Println(string(marshal))
}

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
		// (not 200 + business code). See Task 16.
		t.Fatalf("expected nil err on missing config (means CODE_ENTRAID_CONFIG_NOT_FOUND is wrong or backend returns non-200 HTTP); got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil cfg, got %+v", cfg)
	}
}
