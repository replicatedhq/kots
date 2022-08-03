import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";
import { useParams } from "react-router-dom";
import { useCurrentApp } from "../hooks/useCurrentApp";

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
  return { test: "test" };
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