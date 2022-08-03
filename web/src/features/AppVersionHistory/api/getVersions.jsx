import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useParams } from "react-router-dom";
import { useCurrentApp } from "../hooks/useCurrentApp";

const statusToStatusLabel = {
  deployed: "Deployed",
  deploying: "Deploying",
  pending_config: "Configure",

}

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

function getVersionsSelector(versions) {
  console.log("versions selector", versions);

  const versionHistory = versions?.versionHistory.map(version => {
    return {
      ...version,
      statusLabel:
    })
  return { test: "test" };


  /*
  function deployButtonStatus(version) {
    const downstream = currentApp?.downstream;

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
      !adminConsoleMetadata?.isAirgap &&
      !adminConsoleMetadata?.isKurl;

    if (needsConfiguration) {
      return "Configure";
      // not installed
      // but also check if has access to internet
    } else if (downstream?.currentVersion?.sequence == undefined) {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    } else if (isRedeploy) {
      return "Redeploy";
    } else if (isRollback) {
      return "Rollback";
    } else if (isDeploying) {
      return "Deploying";
    } else if (isCurrentVersion) {
      return "Deployed";
    } else {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    }
  }
  */

}

function useVersions({ _getVersions = getVersions } = {}) {
  let { slug } = useParams();
  let { currentApp } = useCurrentApp();
  return useQuery("versions", () => _getVersions({ slug }), {
    // don't call versions until current app is ascertained
    enabled: !!currentApp,
    staleTime: 2000,
    select: getVersionsSelector,
  });
}

export { useVersions };
// looks like this gets refetched if it returns a certain status
/*
  fetchKotsDownstreamHistory = async () => {
    const { match } = this.props;
    const appSlug = match.params.slug;

    this.setState({
      loadingVersionHistory: true,
      errorTitle: "",
      errorMsg: "",
      displayErrorModal: false,
    });

    try {
      const { currentPage, pageSize } = this.state;
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${appSlug}/versions?currentPage=${currentPage}&pageSize=${pageSize}&pinLatestDeployable=true`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          loadingVersionHistory: false,
          errorTitle: "Failed to get version history",
          errorMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
        return;
      }
      const response = await res.json();
      const versionHistory = response.versionHistory;

      if (isAwaitingResults(versionHistory) && this._mounted) {
        this.state.versionHistoryJob.start(
          this.fetchKotsDownstreamHistory,
          2000
        );
      } else {
        this.state.versionHistoryJob.stop();
      }

      this.setState({
        loadingVersionHistory: false,
        versionHistory: versionHistory,
        numOfSkippedVersions: response.numOfSkippedVersions,
        numOfRemainingVersions: response.numOfRemainingVersions,
        totalCount: response.totalCount,
      });
    } catch (err) {
      this.setState({
        loadingVersionHistory: false,
        errorTitle: "Failed to get version history",
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
      */