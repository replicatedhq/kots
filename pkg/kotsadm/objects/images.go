package kotsadm

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
)

func GetAdminConsoleImage(deployOptions types.DeployOptions, imageKey string) string {
	return GetAdminConsoleImages(deployOptions)[imageKey]
}

func GetAdminConsoleImages(deployOptions types.DeployOptions) map[string]string {
	// TODO: Add error handling to this function
	minioTag, _ := image.GetTag(image.Minio)
	postgres10Tag, _ := image.GetTag(image.Postgres10)
	postgres14Tag, _ := image.GetTag(image.Postgres14)
	dexTag, _ := image.GetTag(image.Dex)

	minioImage := fmt.Sprintf("minio/minio:%s", minioTag)
	postgres10Image := fmt.Sprintf("postgres:%s", postgres10Tag)
	postgres14Image := fmt.Sprintf("postgres:%s", postgres14Tag)
	dexImage := fmt.Sprintf("kotsadm/dex:%s", dexTag)

	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), minioTag)
		postgres10Image = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), postgres10Tag)
		postgres14Image = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), postgres14Tag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), dexTag)
	} else if deployOptions.RegistryConfig.OverrideRegistry != "" {
		// if there is a registry specified, use images there and not the ones from docker hub - even though there's not a username/password specified
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), minioTag)
		postgres10Image = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), postgres10Tag)
		postgres14Image = fmt.Sprintf("%s/postgres:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), postgres14Tag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), dexTag)
	}

	return map[string]string{
		"kotsadm-migrations": fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"kotsadm":            fmt.Sprintf("%s/kotsadm:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"minio":              minioImage,
		"postgres-10":        postgres10Image,
		"postgres-14":        postgres14Image,
		"dex":                dexImage,
	}
}

func GetOriginalAdminConsoleImages(deployOptions types.DeployOptions) map[string]string {
	dexTag, _ := image.GetTag(image.Dex) // dex image is special; we host a copy
	return map[string]string{
		"kotsadm-migrations": fmt.Sprintf("kotsadm/kotsadm-migrations:%s", kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"kotsadm":            fmt.Sprintf("kotsadm/kotsadm:%s", kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"minio":              image.Minio,
		"postgres-10":        image.Postgres10,
		"postgres-14":        image.Postgres14,
		"dex":                fmt.Sprintf("kotsadm/dex:%s", dexTag),
	}
}
