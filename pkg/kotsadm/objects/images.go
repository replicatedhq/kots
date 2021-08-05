package kotsadm

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
)

func GetAdminConsoleImage(deployOptions types.DeployOptions, imageKey string) string {
	return GetAdminConsoleImages(deployOptions)[imageKey]
}

func GetAdminConsoleImages(deployOptions types.DeployOptions) map[string]string {
	minioTag := "RELEASE.2021-07-27T02-40-15Z"
	postgresTag := getPostgresTag(deployOptions)
	dexTag := "v2.28.1"

	if deployOptions.KotsadmOptions.OverrideVersion != "" {
		minioTag = deployOptions.KotsadmOptions.OverrideVersion
		postgresTag = deployOptions.KotsadmOptions.OverrideVersion
		dexTag = deployOptions.KotsadmOptions.OverrideVersion
	}

	minioImage := fmt.Sprintf("minio/minio:%s", minioTag)
	postgresImage := fmt.Sprintf("postgres:%s", postgresTag)
	dexImage := fmt.Sprintf("kotsadm/dex:%s", dexTag)

	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.KotsadmOptions); s != nil {
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), minioTag)
		postgresImage = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), postgresTag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), dexTag)
	} else if deployOptions.KotsadmOptions.OverrideRegistry != "" {
		// if there is a registry specified, use images there and not the ones from docker hub - even though there's not a username/password specified
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), minioTag)
		postgresImage = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), postgresTag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), dexTag)
	}

	return map[string]string{
		"kotsadm-operator":   fmt.Sprintf("%s/kotsadm-operator:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), kotsadmversion.KotsadmTag(deployOptions.KotsadmOptions)),
		"kotsadm-migrations": fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), kotsadmversion.KotsadmTag(deployOptions.KotsadmOptions)),
		"kotsadm":            fmt.Sprintf("%s/kotsadm:%s", kotsadmversion.KotsadmRegistry(deployOptions.KotsadmOptions), kotsadmversion.KotsadmTag(deployOptions.KotsadmOptions)),
		"minio":              minioImage,
		"postgres":           postgresImage,
		"dex":                dexImage,
	}
}
