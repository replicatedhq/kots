
import { App, Version } from '@types";
import { useDeployAppVersion } from '@features/App/api';

type VersionActionType =
  "deployVersion" |
  "downloadVersion" |
  "redeployVersion"  |
  "upgradeAdminConsole";

type VersionAction = {
  type: VersionActionType;
} & ReturnType<typeof useDeployAppVersion>

function getVersionActionType() {
  return "deployVersion";
}

function createVersionAction({ slug, sequence }: { slug: string; sequence: string; }) : VersionAction {
]
  const deployQuery = useDeployAppVersion({ slug, sequence } );
  const downloadQuery = useDownloadVersionMutation(app, version);
  const redeployQuery = useRedeployVersionMutation(app, version);
  const upgradeAdminConsoleQuery = useUpgradeAdminConsoleMutation(app, version);

  switch (getVersionActionType()) {
    case "deployVersion":
      return {
        type: "deployVersion",
        ...deployQuery
      };
    case "downloadVersion":
      return {
        type: "downloadVersion",
        ...downloadQuery
      };
    case "redeployVersion":
      return {
        type: "redeployVersion",
        ...redeployQuery
      };
    case "upgradeAdminConsole":
      return {
        type: "upgradeAdminConsole",
        ...upgradeAdminConsoleQuery
      };
    default:
      throw new Error("Invalid version action type");
  }
}