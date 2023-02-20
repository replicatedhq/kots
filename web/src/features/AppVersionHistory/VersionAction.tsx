// WIP
import { useDeployAppVersion, useRedeployAppVersion } from "@features/App/api";
import { useDownloadAppVersion } from "./api";
import { useUpdateAdminConsole } from "@features/AdminConsole/api/postUpdateAdminConsole";

type VersionActionType =
  | "deployVersion"
  | "downloadVersion"
  | "redeployVersion"
  | "upgradeAdminConsole";

type VersionAction = {
  type: VersionActionType;
} & (
  | ReturnType<typeof useDeployAppVersion>
  | ReturnType<typeof useDownloadAppVersion>
  | ReturnType<typeof useRedeployAppVersion>
  | ReturnType<typeof useUpdateAdminConsole>
);

function getVersionActionType() {
  return "deployVersion";
}

function createVersionAction({
  slug,
  sequence,
}: {
  slug: string;
  sequence: string;
}): VersionAction {
  const deployQuery = useDeployAppVersion({ slug, sequence });
  const downloadQuery = useDownloadAppVersion({ slug, sequence });
  const redeployQuery = useRedeployAppVersion({ slug, sequence });
  const updateAdminConsole = useUpdateAdminConsole({ slug, sequence });

  switch (getVersionActionType()) {
    case "deployVersion":
      return {
        type: "deployVersion",
        ...deployQuery,
      };
    case "downloadVersion":
      return {
        type: "downloadVersion",
        ...downloadQuery,
      };
    case "redeployVersion":
      return {
        type: "redeployVersion",
        ...redeployQuery,
      };
    case "upgradeAdminConsole":
      return {
        type: "upgradeAdminConsole",
        ...updateAdminConsole,
      };
    default:
      throw new Error("Invalid version action type");
  }
}

export { createVersionAction };
