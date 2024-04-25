// This hook has not been integrated yet. It's considered a work in progress
import { useQuery } from "@tanstack/react-query";
import { Utilities } from "../../../utilities/utilities";
import { useParams } from "react-router-dom";
import { useSelectedApp } from "@features/App";
import { useMetadata } from "@src/stores";
import { App, KotsParams, Metadata, Version } from "@types";

async function getVersions({
  apiEndpoint = process.env.API_ENDPOINT,
  currentPage = 0,
  pageSize = 20,
  slug,
}: {
  accessToken?: string;
  apiEndpoint?: string;
  currentPage?: number;
  pageSize?: number;
  slug?: string;
} = {}) {
  try {
    const res = await fetch(
      `${apiEndpoint}/app/${slug}/versions?currentPage=${currentPage}&pageSize=${pageSize}&pinLatestDeployable=true`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return null;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    return await res.json();
  } catch (err) {
    if (err instanceof Error) {
      throw Error(`Failed to fetch apps: ${err.message}`);
    }
    throw Error(`Failed to fetch apps`);
  }
}

interface SelectorParams {
  metadata: Metadata;
  selectedApp: App;
  versions: {
    versionHistory: Version[];
  };
}

// TODO: refactor this function so that the airgapped / nonairgapped are separate
function getVersionsSelectorForKotsManaged({
  metadata,
  selectedApp,
  versions,
}: SelectorParams) {
  const downstream = selectedApp?.downstream;

  const versionHistory = versions?.versionHistory.map((version) => {
    const isCurrentVersion =
      version.sequence === downstream?.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";
    const isPastVersion = !!downstream?.pastVersions?.find(
      (downstreamVersion: Version) =>
        downstreamVersion.sequence === version.sequence
    );
    const needsConfiguration = version.status === "pending_config";
    const isRollback =
      isPastVersion && version.deployedAt && selectedApp?.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade && !metadata?.isAirgap && !metadata?.isKurl;

    let statusLabel = "";

    if (needsConfiguration) {
      statusLabel = "Configure";
    } else if (downstream?.currentVersion?.sequence == undefined) {
      if (canUpdateKots) {
        statusLabel = "Upgrade";
      } else {
        statusLabel = "Deploy";
      }
    } else if (isRedeploy) {
      statusLabel = "Redeploy";
    } else if (isRollback) {
      statusLabel = "Rollback";
    } else if (isDeploying) {
      statusLabel = "Deploying";
    } else if (isCurrentVersion) {
      statusLabel = "Deployed";
    } else {
      if (canUpdateKots) {
        statusLabel = "Upgrade";
      } else {
        statusLabel = "Deploy";
      }
    }
    return {
      ...version,
      statusLabel,
    };
  });

  return {
    ...versions,
    versionHistory,
  };
}

// TODO: refactor this function so that the airgapped / nonairgapped are separate
function getVersionsSelectorForAirgapped({
  versions,
  selectedApp,
  metadata,
}: SelectorParams) {
  return getVersionsSelectorForKotsManaged({ versions, selectedApp, metadata });
}

function chooseVersionsSelector({
  isAirgap,
  isKurl,
}: {
  isAirgap?: boolean;
  isKurl?: boolean;
}) {
  // if airgapped
  if (isAirgap && isKurl) {
    return getVersionsSelectorForAirgapped;
  }

  // if kots managed
  return getVersionsSelectorForKotsManaged;
}

function useVersions({
  currentPage,
  pageSize,
}: {
  currentPage?: number;
  pageSize?: number;
} = {}) {
  let { slug } = useParams<KotsParams>();
  let selectedApp = useSelectedApp();
  let { data: metadata } = useMetadata();

  const versionSelector = chooseVersionsSelector({
    // labels differ by installation manager and if airgapped
    isAirgap: metadata?.isAirgap,
    isKurl: metadata?.isKurl,
  });

  return useQuery(
    ["versions", currentPage, pageSize],
    () => getVersions({ slug, currentPage, pageSize }),
    {
      // don't call versions until current app is ascertained
      enabled: !!selectedApp,
      select: (versions) =>
        selectedApp !== null
          ? versionSelector({ versions, selectedApp, metadata })
          : versions,
    }
  );
}

export { useVersions };
