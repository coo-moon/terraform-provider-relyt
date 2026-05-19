package client

const (
	DPS_STATUS_READY     = "READY"
	DPS_STATUS_DROPPED   = "DROPPED"
	PRIVATE_LINK_READY   = "READY"
	PRIVATE_LINK_UNKNOWN = "UNKNOWN"
	CODE_SUCCESS         = 200
	CODE_USER_NOT_FOUND  = 134084
	//CODE_ROLE_NOT_EXIST = 134085
	CODE_DPS_NOT_FOUND  = 137073
	CODE_DWSU_NOT_FOUND = 65544
)

type CommonRelytResponse[T any] struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data *T     `json:"data,omitempty"`
}

type CommonPage[T any] struct {
	PageNumber int  `json:"pageNumber,omitempty"`
	PageSize   int  `json:"pageSize,omitempty"`
	Records    []*T `json:"records,omitempty"`
	Total      int  `json:"total,omitempty"`
}

type Creator struct {
	CreateTimestamp int64  `json:"createTimestamp,omitempty"`
	Domain          string `json:"domain,omitempty"`
	Email           string `json:"email,omitempty"`
	ID              string `json:"id,omitempty"`
	IsRoot          bool   `json:"isRoot,omitempty"`
	RoleID          string `json:"Role,omitempty"`
	RootAccountID   string `json:"rootAccountId,omitempty"`
	Status          string `json:"status,omitempty"`
}
type UsageRates struct {
	Amount float64 `json:"amount,omitempty"`
	Type   string  `json:"type,omitempty"`
}
type AqsSpec struct {
	ID         int          `json:"id,omitempty"`
	Name       string       `json:"name,omitempty"`
	UsageRates []UsageRates `json:"usageRates,omitempty"`
}
type Owner struct {
	CreateTimestamp int64  `json:"createTimestamp,omitempty"`
	Domain          string `json:"domain,omitempty"`
	Email           string `json:"email,omitempty"`
	ID              string `json:"id,omitempty"`
	IsRoot          bool   `json:"isRoot,omitempty"`
	RoleID          string `json:"Role,omitempty"`
	RootAccountID   string `json:"rootAccountId,omitempty"`
	Status          string `json:"status,omitempty"`
}
type Spec struct {
	ID         int64        `json:"id,omitempty"`
	Name       string       `json:"name,omitempty"`
	UsageRates []UsageRates `json:"usageRates,omitempty"`
}
type DpsMode struct {
	AqsSpec                    *AqsSpec `json:"aqsSpec,omitempty"`
	CreateTime                 int64    `json:"createTime,omitempty"`
	Creator                    *Creator `json:"creator,omitempty"`
	Description                string   `json:"description,omitempty"`
	EnableAdaptiveQueryScaling bool     `json:"enableAdaptiveQueryScaling,omitempty"`
	EnableAutoResume           bool     `json:"enableAutoResume,omitempty"`
	EnableAutoSuspend          bool     `json:"enableAutoSuspend,omitempty"`
	Engine                     string   `json:"engine,omitempty"`
	ID                         string   `json:"id,omitempty"`
	KeepAliveTime              int      `json:"keepAliveTime,omitempty"`
	Name                       string   `json:"name,omitempty"`
	Owner                      *Owner   `json:"owner,omitempty"`
	Spec                       *Spec    `json:"spec,omitempty"`
	Status                     string   `json:"status,omitempty"`
	UpdateTime                 int64    `json:"updateTime,omitempty"`
}

type Features struct {
	Description string `json:"description,omitempty"`
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
}
type Edition struct {
	Description string     `json:"description,omitempty"`
	Features    []Features `json:"features,omitempty"`
	ID          string     `json:"id,omitempty"`
	IsAvailable bool       `json:"isAvailable,omitempty"`
	Name        string     `json:"name,omitempty"`
}

type Endpoints struct {
	Extensions *map[string]string `json:"extensions,omitempty"`
	Host       string             `json:"host,omitempty"`
	ID         string             `json:"id,omitempty"`
	Open       bool               `json:"open,omitempty"`
	Port       int32              `json:"port,omitempty"`
	Protocol   string             `json:"protocol,omitempty"`
	Type       string             `json:"type,omitempty"`
	URI        string             `json:"uri,omitempty"`
}
type Cloud struct {
	ID          string `json:"id,omitempty"`
	IsAvailable bool   `json:"isAvailable,omitempty"`
	IsPublic    bool   `json:"isPublic,omitempty"`
	Link        string `json:"link,omitempty"`
	Name        string `json:"name,omitempty"`
}
type Info struct {
}
type RegionInfo struct {
	Info *Info `json:"info,omitempty"`
}
type Region struct {
	Area       string      `json:"area,omitempty"`
	Cloud      *Cloud      `json:"cloud,omitempty"`
	ID         string      `json:"id,omitempty"`
	Name       string      `json:"name,omitempty"`
	Public     bool        `json:"public,omitempty"`
	RegionInfo *RegionInfo `json:"regionInfo,omitempty"`
}
type Variant struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
type DwsuModel struct {
	Alias           string      `json:"alias,omitempty"`
	CreateTimestamp int64       `json:"createTimestamp,omitempty"`
	Creator         *Creator    `json:"creator,omitempty"`
	DefaultDps      *DpsMode    `json:"defaultDps,omitempty"`
	Domain          string      `json:"domain,omitempty"`
	Edition         *Edition    `json:"edition,omitempty"`
	Endpoints       []Endpoints `json:"endpoints,omitempty"`
	ID              string      `json:"id,omitempty"`
	Owner           *Owner      `json:"owner,omitempty"`
	Region          *Region     `json:"region,omitempty"`
	Status          string      `json:"status,omitempty"`
	Tags            []string    `json:"tags,omitempty"`
	UpdateTimestamp int64       `json:"updateTimestamp,omitempty"`
	Variant         *Variant    `json:"variant,omitempty"`
}

type LakeFormation struct {
	IAMRole string `json:"iamRole,omitempty"`
}

type AsyncResult struct {
	AwsIamArn        string `json:"awsIamArn,omitempty"`
	S3LocationPrefix string `json:"s3LocationPrefix,omitempty"`
}

type Account struct {
	InitPassword string `json:"initPassword,omitempty"`
	Name         string `json:"name,omitempty"`
}

type OpenApiMetaInfo struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	URI      string `json:"uri,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Open     bool   `json:"open,omitempty"`
}

type ServiceAccount struct {
	Type        string             `json:"type,omitempty"`
	AccountInfo *map[string]string `json:"accountInfo,omitempty"`
}

type Boto3AccessInfo struct {
	AccessKeyId string `json:"accessKeyId,omitempty"`
	AccessKey   string `json:"accessKey,omitempty"`
	SecretKey   string `json:"secretKey,omitempty"`
}

type PrivateLinkService struct {
	AllowedPrincipals *[]string `json:"allowedPrincipals,omitempty"`
	ServiceName       string    `json:"serviceName,omitempty"`
	ServiceType       string    `json:"serviceType,omitempty"`
	Status            string    `json:"status,omitempty"`
}

type IntegrationInfo struct {
	ExternalId     string `json:"externalId,omitempty"`
	RelytPrincipal string `json:"relytPrincipal,omitempty"`
	RelytVpc       string `json:"relytVpc,omitempty"`
}

// database
type Database struct {
	Name         *string `json:"name,omitempty"`
	Owner        *string `json:"owner,omitempty"`
	Comments     *string `json:"comments,omitempty"`
	Type         *string `json:"type,omitempty"`
	Hint         *string `json:"hint,omitempty"`
	Oid          *int    `json:"oid,omitempty"`
	Collate      *string `json:"collate,omitempty"`
	Size         *int    `json:"size,omitempty"`
	PrettySize   *string `json:"prettySize,omitempty"`
	CreateSchema *bool   `json:"createSchema,omitempty"`
	UID          *string `json:"uid,omitempty"`
	Ctype        *string `json:"ctype,omitempty"`
}

type PageQuery struct {
	PageSize   int `json:"pageSize"`
	PageNumber int `json:"pageNumber"`
}

type Schema struct {
	Database   *string            `json:"database,omitempty"`
	Catalog    *string            `json:"catalog,omitempty"`
	Name       *string            `json:"name,omitempty"`
	Properties map[string]*string `json:"properties,omitempty"`

	TableFormat *string `json:"tableFormat,omitempty"`
}

//type SchemaProperties struct {
//	Metastore             string `json:"metastore,omitempty"`
//	GlueAccessControlMode string `json:"glue.access-control.mode"`
//	GlueRegion            string `json:"glue.region"`
//	S3Region              string `json:"s3.region"`
//}

type SchemaMeta struct {
	Name         *string             `json:"name,omitempty"`
	Owner        *string             `json:"owner,omitempty"`
	Comments     *string             `json:"comments,omitempty"`
	Type         *string             `json:"type,omitempty"`
	Oid          *int                `json:"oid,omitempty"`
	Database     *string             `json:"database,omitempty"`
	Catalog      *string             `json:"catalog,omitempty"`
	HasPrivilege *bool               `json:"hasPrivilege,omitempty"`
	External     *bool               `json:"external,omitempty"`
	UID          *string             `json:"uid,omitempty"`
	TableFormat  *string             `json:"tableFormat,omitempty"`
	Properties   *map[string]*string `json:"properties,omitempty"`
}

//type SchemaChildren struct {
//	Tables int `json:"tables"`
//	Views  int `json:"views"`
//}

type SchemaPageQuery struct {
	PageQuery
	Database *string `json:"database,omitempty"`
}

type RegionEndpoint struct {
	Host     string `json:"host,omitempty"`
	ID       string `json:"id,omitempty"`
	Open     bool   `json:"open,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Type     string `json:"type,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type UserSecurityPolicy struct {
	ExtraMfaProtectionScopes     []string `json:"extraMfaProtectionScopes,omitempty"`
	MFAStrategy                  string   `json:"mfaStrategy,omitempty"`
	RequiredChangingInitPassword bool     `json:"requiredChangingInitPassword,omitempty"`
}

// EntraIdConfig is the request/response body for /api/entraid-config.
// Note: Enabled has no `omitempty` because `false` must be sent over the wire
// to soft-disable SSO while retaining tenant/client IDs.
type EntraIdConfig struct {
	TenantId   string `json:"tenantId,omitempty"`
	ClientId   string `json:"clientId,omitempty"`
	TenantType string `json:"tenantType,omitempty"`
	Enabled    bool   `json:"enabled"`
}
