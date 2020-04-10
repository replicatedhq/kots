package types

import "time"

type StoreAWS struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"` // added for unmarshaling, redacted on marshaling
	UseInstanceRole bool   `json:"useInstanceRole"`
}

type StoreGoogle struct {
	ServiceAccount  string `json:"serviceAccount"`
	UseInstanceRole bool   `json:"useInstanceRole"`
}

type StoreAzure struct {
	ResourceGroup  string `json:"resourceGroup"`
	StorageAccount string `json:"storageAccount"`
	SubscriptionID string `json:"subscriptionId"`
	TenantID       string `json:"tenantId"`
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
	CloudName      string `json:"cloudName"`
}

type StoreOther struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"` // added for unmarshaling, redacted on marshaling
	Endpoint        string `json:"endpoint"`
}

type Store struct {
	Provider string       `json:"provider"`
	Bucket   string       `json:"bucket"`
	Path     string       `json:"path"`
	AWS      *StoreAWS    `json:"aws,omitempty"`
	Azure    *StoreAzure  `json:"azure,omitempty"`
	Google   *StoreGoogle `json:"google,omitempty"`
	Other    *StoreOther  `json:"other,omitempty"`
}

type Backup struct {
	Name               string     `json:"name"`
	Status             string     `json:"status"`
	Trigger            string     `json:"trigger"`
	AppID              string     `json:"appID"`
	Sequence           int64      `json:"sequence"`
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	VolumeCount        int        `json:"volumeCount"`
	VolumeSuccessCount int        `json:"volumeSuccessCount"`
	VolumeBytes        int64      `json:"volumeBytes"`
	VolumeSizeHuman    string     `json:"volumeSizeHuman"`
}
