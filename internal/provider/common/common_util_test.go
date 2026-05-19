package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"strconv"
	"terraform-provider-relyt/internal/provider/client"
	"testing"
	"time"
)

func TestScroll(t *testing.T) {
	relytDatabaseClientConfig := client.RelytDatabaseClientConfig{
		DmsHost:   "http://pl-4679805844736-api-d3410a34c78f4386.elb.us-east-1.amazonaws.com:8180",
		AccessKey: "AK8DoEFMRPWBGG0eY1JyNBVj7OnrTO3B6t3uJFyibDcGwz56HrAlg8uKtxf9hQeoHphJzOw",
		SecretKey: "HHJU4NBSLKZVGKTGRM41FCLGZVH4VPWS",
	}
	databaseClient, _ := client.NewRelytDatabaseClient(relytDatabaseClientConfig)
	start := time.Now()
	catalog := "catalog"
	records, _ := ScrollPageRecords(&diag.Diagnostics{}, func(pageSize, pageNum int) ([]*client.SchemaMeta, error) {
		listRecords, err := CommonRetry(context.TODO(), func() (*client.CommonPage[client.SchemaMeta], error) {
			start = time.Now()
			schemas, err := databaseClient.ListSchemas(context.TODO(), client.SchemaPageQuery{
				PageQuery: client.PageQuery{
					PageSize:   pageSize,
					PageNumber: pageNum,
				},
				Database: &catalog,
			})
			msg := ""
			if err != nil {
				msg += err.Error()
			}
			println("http call cost:" + (time.Now().Sub(start)).String() + " err" + msg)
			return schemas, err
		})
		if err != nil {
			return nil, err
		}
		if listRecords == nil {
			return nil, fmt.Errorf(" shouldn't get nil CommonPage resp")
		}
		return listRecords.Records, nil
	})
	fmt.Println("get all cost: " + (time.Now().Sub(start).String()))
	marshal, _ := json.Marshal(records)
	fmt.Println("size:" + strconv.Itoa(len(records)) + "body" + string(marshal))
}

func TestScrollDatabase(t *testing.T) {
	relytDatabaseClientConfig := client.RelytDatabaseClientConfig{
		DmsHost:   "http://pl-4679805844736-api-d3410a34c78f4386.elb.us-east-1.amazonaws.com:8180",
		AccessKey: "AK8DoEFMRPWBGG0eY1JyNBVj7OnrTO3B6t3uJFyibDcGwz56HrAlg8uKtxf9hQeoHphJzOw",
		SecretKey: "HHJU4NBSLKZVGKTGRM41FCLGZVH4VPWS",
	}
	dbClient, _ := client.NewRelytDatabaseClient(relytDatabaseClientConfig)
	start := time.Now()
	records, _ := ScrollPageRecords(&diag.Diagnostics{}, func(pageSize, pageNum int) ([]*client.Database, error) {
		listRecords, err := CommonRetry(context.TODO(), func() (*client.CommonPage[client.Database], error) {
			database, err := dbClient.ListDatabase(context.TODO(), pageSize, pageNum)
			return database, err
		})
		if err != nil {
			return nil, err
		}
		if listRecords == nil {
			return nil, fmt.Errorf(" shouldn't get nil CommonPage resp")
		}
		return listRecords.Records, nil
	})
	fmt.Println("get all cost: " + (time.Now().Sub(start).String()))
	marshal, _ := json.Marshal(records)
	fmt.Println("size:" + strconv.Itoa(len(records)) + "body" + string(marshal))
}

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
