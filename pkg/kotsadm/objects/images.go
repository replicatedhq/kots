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
	rqliteTag, _ := image.GetTag(image.Rqlite)
	dexTag, _ := image.GetTag(image.Dex)

	minioImage := fmt.Sprintf("minio/minio:%s", minioTag)
	rqliteImage := fmt.Sprintf("rqlite/rqlite:%s", rqliteTag)
	dexImage := fmt.Sprintf("kotsadm/dex:%s", dexTag)

	if s := kotsadmversion.KotsadmPullSecret(deployOptions.Namespace, deployOptions.RegistryConfig); s != nil {
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), minioTag)
		rqliteImage = fmt.Sprintf("%s/rqlite:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), rqliteTag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), dexTag)
	} else if deployOptions.RegistryConfig.OverrideRegistry != "" {
		// if there is a registry specified, use images there and not the ones from docker hub - even though there's not a username/password specified
		minioImage = fmt.Sprintf("%s/minio:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), minioTag)
		rqliteImage = fmt.Sprintf("%s/rqlite:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), rqliteTag)
		dexImage = fmt.Sprintf("%s/dex:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), dexTag)
	}

	return map[string]string{
		"kotsadm-migrations": fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"kotsadm":            fmt.Sprintf("%s/kotsadm:%s", kotsadmversion.KotsadmRegistry(deployOptions.RegistryConfig), kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"minio":              minioImage,
		"rqlite":             rqliteImage,
		"dex":                dexImage,
	}
}

func GetOriginalAdminConsoleImages(deployOptions types.DeployOptions) map[string]string {
	dexTag, _ := image.GetTag(image.Dex) // dex image is special; we host a copy
	return map[string]string{
		"kotsadm-migrations": fmt.Sprintf("kotsadm/kotsadm-migrations:%s", kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"kotsadm":            fmt.Sprintf("kotsadm/kotsadm:%s", kotsadmversion.KotsadmTag(deployOptions.RegistryConfig)),
		"minio":              image.Minio,
		"rqlite":             image.Rqlite,
		"dex":                fmt.Sprintf("kotsadm/dex:%s", dexTag),
	}
}
