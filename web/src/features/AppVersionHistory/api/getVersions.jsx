import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useParams } from "react-router-dom";
import { useCurrentApp } from "../hooks/useCurrentApp";
import { useMetadata } from "@src/stores";
import { useIsHelmManaged } from "@src/components/hooks";



async function getVersions({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
  currentPage = 0,
  pageSize = 20,
  slug
} = {}) {
  try {
    const res = await _fetch(
      `${apiEndpoint}/app/${slug}/versions?currentPage=${currentPage}&pageSize=${pageSize}&pinLatestDeployable=true`,
      {
        headers: {
          Authorization: accessToken,
          "Content-Type": "application/json",
        },
        method: "GET",
      });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return null;
      }
      throw Error(`Failed to fetch apps with status ${res.status}`);
    }
    return await res.json();
  } catch (err) {
    throw Error(err);
  }
}

// TODO: refactor this function so that the airgapped / nonairgapped are separate
function getVersionsSelectorForKotsManaged({ versions, currentApp, metadata }) {
  console.log("kots managed versions");

  const downstream = currentApp?.downstream;

  const versionHistory = versions?.versionHistory.map(version => {

    const isCurrentVersion =
      version.sequence === downstream?.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";
    const isPastVersion = find(downstream?.pastVersions, {
      sequence: version.sequence,
    });
    const needsConfiguration = version.status === "pending_config";
    const isRollback = isPastVersion && version.deployedAt && currentApp?.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !metadata?.isAirgap &&
      !metadata?.isKurl;

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
      version,
      statusLabel,
    }
  });

  return {
    ...versions,
    versionHistory,
  };
}

// TODO: refactor this function so that the airgapped / nonairgapped are separate
function getVersionsSelectorForAirgapped({ versions, currentApp, metadata }) {
  return getVersionsSelectorForKotsManaged({ versions, currentApp, metadata });
}

function getVersionsSelectorForHelmManaged({ versions }) {
  const deployedSequence = versions?.versionHistory?.find((v) => v.status === "deployed")?.sequence;

  const versionHistory = versions?.versionHistory.map(version => {
    let statusLabel = "Redeploy";

    if (version.sequence > deployedSequence) {
      statusLabel = "Deploy";
    }

    if (version.sequence < deployedSequence) {
      statusLabel = "Rollback";
    }
    return {
      ...version,
      statusLabel,
    }
  });

  return {
    ...versions,
    versionHistory,
  };
}

function chooseVersionsSelector({
  isAirgap,
  isKurl,
  isHelmManaged,
  _getVersionsSelectorForKotsManaged = getVersionsSelectorForKotsManaged,
  _getVersionsSelectorForAirgapped = getVersionsSelectorForAirgapped,
  _getVersionsSelectorForHelmManaged = getVersionsSelectorForHelmManaged
}) {
  // if airgapped
  if (isAirgap && isKurl) {
    return _getVersionsSelectorForAirgapped;
  }

  // if helm managed
  if (isHelmManaged) {
    return _getVersionsSelectorForHelmManaged;
  }

  // if kots managed
  return _getVersionsSelectorForKotsManaged

}

function useVersions({ _getVersions = getVersions } = {}) {
  let { slug } = useParams();
  let { currentApp } = useCurrentApp();
  let { data: metadata } = useMetadata();
  let { data: isHelmManaged } = useIsHelmManaged();

  const versionSelector = chooseVersionsSelector({
    // labels differ by installation manager and if airgapped
    isAirgap: metadata?.isAirgap,
    isHelmManaged,
    isKurl: metadata?.isKurl,
  });
  return useQuery("versions", () => _getVersions({ slug }), {
    // don't call versions until current app is ascertained
    enabled: !!currentApp,
    select: versions => versionSelector({ versions, currentApp, metadata }),
    staleTime: 2000,
  });
}

function UseVersions({ children }) {
  const query = useVersions();

  return children(query);
}

export { useVersions, UseVersions };