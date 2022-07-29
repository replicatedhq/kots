import { useQuery } from "react-query";
import { Utilities } from "../../../utilities/utilities";

async function getApps({
  accessToken = Utilities.getToken(),
  apiEndpoint = process.env.API_ENDPOINT,
  _fetch = fetch,
} = {}) {
  try {
    const res = await _fetch(`${apiEndpoint}/apps`, {
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

function useApps({ _getApps = getApps } = {}) {
  return useQuery("apps", () => _getApps(), {
    staleTime: 5000,
  });
}

function UseApps({ children }) {
  const query = useApps();

  return children(query);
}

export { getApps, useApps, UseApps };
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