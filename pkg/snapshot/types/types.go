package types

import (
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

type StoreAWS struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"` // added for unmarshaling, redacted on marshaling
	UseInstanceRole bool   `json:"useInstanceRole"`
}

type StoreGoogle struct {
	JSONFile        string `json:"jsonFile"`
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

type StoreInternal struct {
	Region               string `json:"region"`
	AccessKeyID          string `json:"accessKeyID"`
	SecretAccessKey      string `json:"secretAccessKey"` // added for unmarshaling, redacted on marshaling
	Endpoint             string `json:"endpoint"`
	ObjectStoreClusterIP string `json:"objectStoreClusterIP"`
}

type StoreFileSystem struct {
	Region               string            `json:"region"`               // Legacy field for minio backwards compatibility
	AccessKeyID          string            `json:"accessKeyID"`          // Legacy field for minio backwards compatibility
	SecretAccessKey      string            `json:"secretAccessKey"`      // Legacy field for minio backwards compatibility; added for unmarshaling, redacted on marshaling
	Endpoint             string            `json:"endpoint"`             // Legacy field for minio backwards compatibility
	ObjectStoreClusterIP string            `json:"objectStoreClusterIP"` // Legacy field for minio backwards compatibility
	Config               *FileSystemConfig `json:"config,omitempty"`
}

type Store struct {
	Provider   string           `json:"provider"`
	Bucket     string           `json:"bucket"`
	Path       string           `json:"path"`
	CACertData []byte           `json:"caCert,omitempty"`
	AWS        *StoreAWS        `json:"aws,omitempty"`
	Azure      *StoreAzure      `json:"azure,omitempty"`
	Google     *StoreGoogle     `json:"gcp,omitempty"`
	Other      *StoreOther      `json:"other,omitempty"`
	Internal   *StoreInternal   `json:"internal,omitempty"`
	FileSystem *StoreFileSystem `json:"fileSystem,omitempty"`
}

type FileSystemConfig struct {
	NFS      *NFSConfig `json:"nfs,omitempty"`
	HostPath *string    `json:"hostPath,omitempty"`
}

type NFSConfig struct {
	Path   string `json:"path,omitempty"`
	Server string `json:"server,omitempty"`
}

type App struct {
	Slug       string `json:"slug"`
	Sequence   int64  `json:"sequence"`
	Name       string `json:"name"`
	AppIconURI string `json:"iconUri"`
}

type Backup struct {
	Name               string     `json:"name"`
	Status             string     `json:"status"`
	Trigger            string     `json:"trigger"`
	AppID              string     `json:"appID"`    // TODO: remove with app backups
	Sequence           int64      `json:"sequence"` // TODO: remove with app backups
	StartedAt          *time.Time `json:"startedAt,omitempty"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	VolumeCount        int        `json:"volumeCount"`
	VolumeSuccessCount int        `json:"volumeSuccessCount"`
	VolumeBytes        int64      `json:"volumeBytes"`
	VolumeSizeHuman    string     `json:"volumeSizeHuman"`
	SupportBundleID    string     `json:"supportBundleId,omitempty"`
	IncludedApps       []App      `json:"includedApps,omitempty"`
}

type BackupDetail struct {
	Name            string           `json:"name"`
	Status          string           `json:"status"`
	VolumeSizeHuman string           `json:"volumeSizeHuman"`
	Namespaces      []string         `json:"namespaces"`
	Hooks           []*SnapshotHook  `json:"hooks"`
	Volumes         []SnapshotVolume `json:"volumes"`
	Errors          []SnapshotError  `json:"errors"`
	Warnings        []SnapshotError  `json:"warnings"`
}

type RestoreDetail struct {
	Name     string                `json:"name"`
	Phase    velerov1.RestorePhase `json:"phase"`
	Volumes  []RestoreVolume       `json:"volumes"`
	Errors   []SnapshotError       `json:"errors"`
	Warnings []SnapshotError       `json:"warnings"`
}

type SnapshotHook struct {
	Name          string          `json:"name"`
	Namespace     string          `json:"namespace"`
	Phase         string          `json:"phase"`
	PodName       string          `json:"podName"`
	ContainerName string          `json:"containerName"`
	Command       string          `json:"command"`
	Stdout        string          `json:"stdout"`
	Stderr        string          `json:"stderr"`
	StartedAt     *time.Time      `json:"startedAt,omitempty"`
	FinishedAt    *time.Time      `json:"finishedAt,omitempty"`
	Errors        []SnapshotError `json:"errors"`
	Warnings      []SnapshotError `json:"warnings"`
}

type SnapshotVolume struct {
	Name                 string     `json:"name"`
	SizeBytesHuman       string     `json:"sizeBytesHuman"`
	DoneBytesHuman       string     `json:"doneBytesHuman"`
	CompletionPercent    int        `json:"completionPercent"`
	TimeRemainingSeconds int        `json:"timeRemainingSeconds"`
	StartedAt            *time.Time `json:"startedAt,omitempty"`
	FinishedAt           *time.Time `json:"finishedAt,omitempty"`
	Phase                string     `json:"phase"`
}
type RestoreVolume struct {
	Name                  string     `json:"name"`
	PodName               string     `json:"podName"`
	PodNamespace          string     `json:"podNamespace"`
	PodVolumeName         string     `json:"podVolumeName"`
	SizeBytesHuman        string     `json:"sizeBytesHuman"`
	DoneBytesHuman        string     `json:"doneBytesHuman"`
	CompletionPercent     int        `json:"completionPercent"`
	RemainingSecondsExist bool       `json:"remainingSecondsExist"`
	TimeRemainingSeconds  int        `json:"timeRemainingSeconds"`
	StartedAt             *time.Time `json:"startedAt,omitempty"`
	FinishedAt            *time.Time `json:"finishedAt,omitempty"`
	Phase                 string     `json:"phase"`
}

type SnapshotError struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Namespace string `json:"namespace"`
}

type VolumeSummary struct {
	VolumeCount        int    `json:"volumeCount"`
	VolumeSuccessCount int    `json:"volumeSuccessCount"`
	VolumeBytes        int64  `json:"volumeBytes"`
	VolumeSizeHuman    string `json:"volumeSizeHuman"`
}

type SnapshotSchedule struct {
	Schedule string `json:"schedule"`
}

type SnapshotTTL struct {
	InputValue    string `json:"inputValue"`
	InputTimeUnit string `json:"inputTimeUnit"`
	Converted     string `json:"converted"`
}

type ParsedTTL struct {
	Quantity int64  `json:"quantity"`
	Unit     string `json:"unit"`
}

type ScheduledSnapshot struct {
	ID                 string    `json:"id"`
	AppID              string    `json:"appId"`
	ScheduledTimestamp time.Time `json:"scheduledTimestamp"`
	// name of Backup CR will be set once scheduled
	BackupName string `json:"backupName,omitempty"`
}

type ScheduledInstanceSnapshot struct {
	ID                 string    `json:"id"`
	ClusterID          string    `json:"clusterId"`
	ScheduledTimestamp time.Time `json:"scheduledTimestamp"`
	// name of Backup CR will be set once scheduled
	BackupName string `json:"backupName,omitempty"`
}
