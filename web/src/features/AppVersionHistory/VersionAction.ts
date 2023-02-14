
import { App, Version } from '@types";
import { UseMutationResult } from 'react-query';

type VersionActionType =
  "deployVersion" |
  "downloadVersion" |
  "redeployVersion"  |
  "upgradeAdminConsole";

function getVersionActionType() {
  return "deployVersion";
}

function createVersionAction({ app, version }: { app: App; version: Version; }) {
]
  const deployQuery = useDeployVersionMutation(app, version);
  const downloadQuery = useDownloadVersionMutation(app, version);
  const redeployQuery = useRedeployVersionMutation(app, version);
  const upgradeAdminConsoleQuery = useUpgradeAdminConsoleMutation(app, version);

  switch getVersionActionType() {
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
  }

  return {
    type: "deployVersion",
    callbackAction: () => console.log('deployVersion');
  }
}