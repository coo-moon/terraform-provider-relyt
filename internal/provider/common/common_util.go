package common

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"terraform-provider-relyt/internal/provider/client"
	"time"
)

// 创建一个通道来通知程序退出
var Interrupted = make(chan bool, 1)

func RegSignalHandler() {
	// 创建一个通道来接收信号
	sigs := make(chan os.Signal, 1)

	// 注册要接收的信号
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// 启动一个 goroutine 来处理信号
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("received signal:", sig)
		Interrupted <- true
	}()
}

func RouteRegionUri(ctx context.Context, dwsuId string, relytClient *client.RelytClient,
	diag *diag.Diagnostics) *client.OpenApiMetaInfo {
	meta, err := CommonRetry[client.OpenApiMetaInfo](ctx,
		func() (*client.OpenApiMetaInfo, error) {
			return relytClient.GetDwsuOpenApiMeta(ctx, dwsuId)
		})
	if err != nil || meta == nil {
		errMsg := "get RegionApi is nil"
		if err != nil {
			errMsg = err.Error()
		}
		diag.AddError("error get region api", "fail to get Region uri address dwsuID:"+
			""+dwsuId+" error: "+errMsg)
		return meta
	}
	return meta
}

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

func RetryFunction[T any](ctx context.Context, retryNum, intervalSecond int,
	backoffCoefficient float64,
	retryableFunc func() (*T, error)) (*T, error) {
	var err error
	var result *T
	for i := 0; i < retryNum; i++ {
		result, err = retryableFunc()
		if err == nil {
			return result, nil
		}
		time.Sleep(time.Duration(intervalSecond))
		intervalSecond = int(math.Ceil(backoffCoefficient * float64(intervalSecond)))
		tflog.Warn(ctx, "retry failed func! backoff second: "+strconv.Itoa(intervalSecond)+" retry num:"+strconv.Itoa(i)+" error msg!"+err.Error())
	}

	return result, err
}

func CommonRetry[T any](ctx context.Context, retryableFunc func() (*T, error)) (*T, error) {
	return RetryFunction(ctx, 5, 1, 1.0, retryableFunc)
}

func TimeOutTask(timeoutSec int64, checkIntervalSec int32, task func() (any, error)) (any, error) {
	// 设置超时时间
	timeout := time.Duration(timeoutSec) * time.Second
	interval := time.Duration(checkIntervalSec) * time.Second

	// 创建带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//// 启动任务
	//done := make(chan bool)
	//f := func() (any, error) {
	//
	//}
	//go f
	for {
		if len(Interrupted) > 0 {
			return nil, fmt.Errorf("interrupted by user")
		}
		select {
		case <-ctx.Done():
			fmt.Println("Task timed out")
			//done <- false
			return nil, fmt.Errorf("timeout")
		default:
			a, err := task()
			if err == nil {
				//done <- true
				return a, err
			}
			time.Sleep(interval)
		}
	}
}

func ParseAccessConfig(ctx context.Context, relytClient *client.RelytClient, meta tfsdk.Config, diag *diag.Diagnostics) *client.RelytDatabaseClient {
	//config := model.OptionalProviderConfig{}
	//diags := meta.Get(ctx, &config)
	//tflog.Info(ctx, "msg"+config.Auth.AccessKey.ValueString())
	//diag.Append(diags...)
	if relytClient == nil || relytClient.RelytDatabaseClientConfig == nil {
		diag.AddError("Missing provider data_access_config", "please supply and  check your data access config!")
		return nil
	}
	if relytClient.RelytDatabaseClientConfig.AccessKey == "" {
		diag.AddError("data_access_config error", "access_key can't be empty string")
	}
	if relytClient.RelytDatabaseClientConfig.SecretKey == "" {
		diag.AddError("data_access_config error", "secret_key can't be empty string")
	}
	if relytClient.RelytDatabaseClientConfig.DmsHost == "" {
		diag.AddError("data_access_config error", "endpoint can't be empty string")
	}
	if diag.HasError() {
		return nil
	}

	databaseClient, err := client.NewRelytDatabaseClient(*relytClient.RelytDatabaseClientConfig)
	if err != nil {
		diag.AddError("ProviderMeta parse error", "error parse data access config! "+err.Error())
	}
	return &databaseClient
	//return &config
}

func ScrollPageRecords[T any](diag *diag.Diagnostics, list func(pageSize, pageNum int) ([]T, error)) ([]T, error) {
	var records []T
	pageSize, pageNum := 100, 1
	for {
		databases, err := list(pageSize, pageNum)
		if err != nil {
			msg := "databases read failed"
			if err != nil {
				msg = err.Error()
			}
			diag.AddError("Failed list databases", "error list database "+msg)
			return records, err
		}
		now := 0
		if databases != nil && len(databases) > 0 {
			records = append(records, databases...)
			now = len(databases)
		}
		pageNum++
		if now < pageSize {
			break
		}
	}
	return records, nil
}
