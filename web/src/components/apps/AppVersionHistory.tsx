import React, { useEffect, useReducer } from "react";
import { withRouter, Link } from "react-router-dom";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import Modal from "react-modal";
import find from "lodash/find";
import isEmpty from "lodash/isEmpty";
import get from "lodash/get";
import MountAware from "../shared/MountAware";
import Loader from "../shared/Loader";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import AirgapUploadProgress from "@src/features/Dashboard/components/AirgapUploadProgress";
import ErrorModal from "../modals/ErrorModal";
import { AppVersionHistoryRow } from "@features/AppVersionHistory/AppVersionHistoryRow";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
import {
  Utilities,
  isAwaitingResults,
  secondsAgo,
  getPreflightResultState,
  getGitProviderDiffUrl,
  getCommitHashFromUrl,
} from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import { AirgapUploader } from "../../utilities/airgapUploader";
import ReactTooltip from "react-tooltip";
import Pager from "../shared/Pager";
import { HelmDeployModal } from "../shared/modals/HelmDeployModal";
import { UseDownloadValues } from "../hooks";
import { KotsPageTitle } from "@components/Head";

import "@src/scss/components/apps/AppVersionHistory.scss";
import { DashboardGitOpsCard } from "@features/Dashboard";
import Icon from "../Icon";
import { App, KotsParams, Version, VersionDownloadStatus } from "@types";
import { RouteComponentProps } from "react-router-dom";
dayjs.extend(relativeTime);

type Release = {
  versionLabel?: string;
  sequence?: number;
};

type ReleaseWithError = {
  title?: string;
  sequence: number;
  diffSummaryError?: string;
};

type Props = {
  adminConsoleMetadata: { isAirgap: boolean; isKurl: boolean };
  app: App;
  displayErrorModal: boolean;
  isBundleUploading: boolean;
  isHelmManaged: boolean;
  makeCurrentVersion: (
    slug: string,
    version: Version | null,
    isSkipPreflights: boolean,
    continueWithFailedPreflights: boolean
  ) => void;
  makingCurrentRelease: boolean;
  makingCurrentVersionErrMsg: string;
  redeployVersion: (slug: string, version: Version | null) => void;
  redeployVersionErrMsg: string;
  refreshAppData: () => void;
  toggleErrorModal: () => void;
  toggleIsBundleUploading: (isUploading: boolean) => void;
  updateCallback: () => void;
} & RouteComponentProps<KotsParams>;

type State = {
  logsLoading: boolean;
  logs: Object | null;
  selectedTab: Object | null;
  showDeployWarningModal: boolean;
  showSkipModal: boolean;
  versionToDeploy: Version | null;
  releaseNotes: Object | null;
  selectedDiffReleases: boolean;
  checkedReleasesToDiff: Version[];
  diffHovered: boolean;
  uploadingAirgapFile: boolean;
  checkingForUpdates: boolean;
  checkingUpdateMessage: string;
  checkingForUpdateError: boolean;
  airgapUploadError: string;
  versionDownloadStatuses: {
    [x: number]: VersionDownloadStatus;
  };
  showDiffOverlay: boolean;
  firstSequence: Number | string;
  secondSequence: Number | string;
  appUpdateChecker: Repeater;
  uploadProgress: Number;
  uploadSize: Number;
  uploadResuming: boolean;
  displayShowDetailsModal: boolean;
  yamlErrorDetails: string[];
  deployView: boolean;
  selectedSequence: Number;
  releaseWithErr: ReleaseWithError | null | undefined;
  versionHistoryJob: Repeater;
  loadingVersionHistory: boolean;
  versionHistory: Version[];
  errorTitle: string;
  errorMsg: string;
  displayErrorModal: boolean;
  displayConfirmDeploymentModal: boolean;
  confirmType: string;
  isSkipPreflights: boolean;
  displayKotsUpdateModal: boolean;
  kotsUpdateChecker: Repeater;
  kotsUpdateRunning: boolean;
  kotsUpdateStatus: string;
  kotsUpdateMessage: string;
  kotsUpdateError: Object | undefined;
  numOfSkippedVersions: Number;
  numOfRemainingVersions: Number;
  totalCount: Number;
  currentPage: Number;
  pageSize: Number;
  loadingPage: boolean;
  hasPreflightChecks: boolean;
  airgapUploader: AirgapUploader | null;
  updatesAvailable: boolean;
  showNoChangesModal: boolean;
  showAutomaticUpdatesModal: boolean;
  releaseWithNoChanges: Release | null | undefined;
  showDiffErrModal: boolean;
  showLogsModal: boolean;
  noUpdateAvailiableText: string;
  viewLogsErrMsg: string;
  showHelmDeployModalForVersionLabel: string;
  showHelmDeployModalForSequence: number | null;
};

const AppVersionHistory = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      airgapUploader: null,
      airgapUploadError: "",
      appUpdateChecker: new Repeater(),
      checkedReleasesToDiff: [],
      checkingForUpdateError: false,
      checkingForUpdates: false,
      checkingUpdateMessage: "Checking for updates",
      confirmType: "",
      currentPage: 0,
      deployView: false,
      diffHovered: false,
      displayConfirmDeploymentModal: false,
      displayErrorModal: false,
      displayKotsUpdateModal: false,
      displayShowDetailsModal: false,
      errorMsg: "",
      errorTitle: "",
      firstSequence: 0,
      hasPreflightChecks: true,
      isSkipPreflights: false,
      kotsUpdateChecker: new Repeater(),
      kotsUpdateError: undefined,
      kotsUpdateMessage: "",
      kotsUpdateRunning: false,
      kotsUpdateStatus: "",
      loadingPage: false,
      loadingVersionHistory: true,
      logs: null,
      logsLoading: false,
      noUpdateAvailiableText: "",
      numOfRemainingVersions: 0,
      numOfSkippedVersions: 0,
      pageSize: 20,
      releaseNotes: null,
      releaseWithErr: { title: "", sequence: 0, diffSummaryError: "" },
      releaseWithNoChanges: { versionLabel: "", sequence: 0 },
      secondSequence: 0,
      selectedDiffReleases: false,
      selectedSequence: -1,
      selectedTab: null,
      showAutomaticUpdatesModal: false,
      showDeployWarningModal: false,
      showDiffErrModal: false,
      showDiffOverlay: false,
      showHelmDeployModalForSequence: null,
      showHelmDeployModalForVersionLabel: "",
      showLogsModal: false,
      showNoChangesModal: false,
      showSkipModal: false,
      totalCount: 0,
      updatesAvailable: false,
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadResuming: false,
      uploadSize: 0,
      versionDownloadStatuses: {},
      versionHistory: [],
      versionHistoryJob: new Repeater(),
      versionToDeploy: null,
      viewLogsErrMsg: "",
      yamlErrorDetails: [],
    }
  );

  // moving this out of the state because new repeater instances were getting created
  // and it doesn't really affect the UI
  const versionDownloadStatusJobs: { [key: number]: Repeater } = {};
  const getPreflightState = (version: Version) => {
    let preflightState = "";
    if (version?.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      preflightState = getPreflightResultState(preflightResult);
    }
    if (preflightState === "") {
      setState({ hasPreflightChecks: false });
    }
  };

  const fetchKotsDownstreamHistory = async ({
    currentPage,
    pageSize,
  }: { currentPage?: Number; pageSize?: Number } = {}) => {
    const { match } = props;
    const appSlug = match.params.slug;

    setState({
      loadingVersionHistory: true,
      errorTitle: "",
      errorMsg: "",
      displayErrorModal: false,
    });

    try {
      const currentPageToQuery = currentPage ? currentPage : state.currentPage;
      const pageSizeToQuery = pageSize ? pageSize : state.pageSize;
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${appSlug}/versions?currentPage=${currentPageToQuery}&pageSize=${pageSizeToQuery}&pinLatestDeployable=true`,
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
        setState({
          loadingVersionHistory: false,
          errorTitle: "Failed to get version history",
          errorMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
        return;
      }
      const response = await res.json();
      const versionHistory = response.versionHistory;

      if (
        isAwaitingResults(versionHistory) &&
        !state.versionHistoryJob.isRunning
      ) {
        state.versionHistoryJob.start(fetchKotsDownstreamHistory, 2000);
      } else {
        state.versionHistoryJob.stop();
      }

      setState({
        loadingPage: false,
        loadingVersionHistory: false,
        versionHistory: versionHistory,
        numOfSkippedVersions: response.numOfSkippedVersions,
        numOfRemainingVersions: response.numOfRemainingVersions,
        totalCount: response.totalCount,
      });
    } catch (err) {
      setState({
        loadingVersionHistory: false,
        errorTitle: "Failed to get version history",
        errorMsg: err
          ? (err as Error).message
          : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  const getAppUpdateStatus = () => {
    const { app } = props;

    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/task/updatedownload`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      )
        .then(async (res) => {
          const response = await res.json();

          if (response.status !== "running" && !props.isBundleUploading) {
            state.appUpdateChecker.stop();

            setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed",
            });

            if (props.updateCallback) {
              props.updateCallback();
            }
            fetchKotsDownstreamHistory();
          } else {
            setState({
              checkingForUpdates: true,
              checkingUpdateMessage: response.currentMessage,
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get update status", err);
          reject();
        });
    });
  };

  const onUploadProgress = (
    progress: number,
    size: number,
    resuming = false
  ) => {
    setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  };

  const onUploadError = (message: String) => {
    setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError:
        (message as string) || "Error uploading bundle, please try again",
    });
    props.toggleIsBundleUploading(false);
  };

  const onUploadComplete = () => {
    state.appUpdateChecker.start(getAppUpdateStatus, 1000);
    setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    props.toggleIsBundleUploading(false);
  };

  const onDropBundle = async () => {
    setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: "",
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });

    props.toggleIsBundleUploading(true);

    const params = {
      appId: props.app?.id,
    };
    state.airgapUploader?.upload(
      params,
      onUploadProgress,
      onUploadError,
      onUploadComplete
    );
  };

  const getAirgapConfig = async () => {
    const { app } = props;
    const configUrl = `${process.env.API_ENDPOINT}/app/${app.slug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: Utilities.getToken(),
        },
      });
      if (res.ok) {
        const response = await res.json();
        simultaneousUploads = response.simultaneousUploads;
      }
    } catch {
      // no-op
    }

    setState({
      airgapUploader: new AirgapUploader(
        true,
        app.slug,
        onDropBundle,
        simultaneousUploads
      ),
    });
  };

  useEffect(() => {
    getPreflightState(props.app.downstream.currentVersion);
    const urlParams = new URLSearchParams(window.location.search);
    const pageNumber = urlParams.get("page");
    if (pageNumber) {
      setState({ currentPage: parseInt(pageNumber) });
    } else {
      props.history.push(`${props.location.pathname}?page=0`);
    }

    fetchKotsDownstreamHistory();
    props.refreshAppData();
    if (props.app?.isAirgap && !state.airgapUploader) {
      getAirgapConfig();
    }

    // check if there are any updates in progress
    state.appUpdateChecker.start(getAppUpdateStatus, 1000);

    const url = window.location.pathname;
    const { params } = props.match;
    if (url.includes("/diff")) {
      const firstSequence = params.firstSequence;
      const secondSequence = params.secondSequence;
      if (firstSequence !== undefined && secondSequence !== undefined) {
        // undefined because a sequence can be zero!
        setState({ showDiffOverlay: true, firstSequence, secondSequence });
      }
    }

    return () => {
      state.appUpdateChecker.stop();
      state.versionHistoryJob.stop();
      for (const j in versionDownloadStatusJobs) {
        versionDownloadStatusJobs[j].stop();
      }
    };
  }, []);

  useEffect(() => {
    fetchKotsDownstreamHistory();
    if (
      props.app.downstream.pendingVersions.length > 0 &&
      state.updatesAvailable === false
    ) {
      setState({ updatesAvailable: true });
    }
    if (
      props.app.downstream.pendingVersions.length === 0 &&
      state.updatesAvailable === true
    ) {
      setState({ updatesAvailable: false });
    }
  }, [
    props.match.params.slug,
    props.app.id,
    props.app.downstream.pendingVersions.length,
    state.updatesAvailable,
  ]);

  const setPageSize = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newPageSize = parseInt(e.target.value);
    setState({ pageSize: newPageSize, currentPage: 0 });
    // TODO: refactor this to use the query params to trigger refetch. track the page size in the query params
    fetchKotsDownstreamHistory({ currentPage: 0, pageSize: newPageSize });
    props.history.push(`${props.location.pathname}?page=0`);
  };

  const toggleErrorModal = () => {
    setState({ displayErrorModal: !state.displayErrorModal });
  };

  const showReleaseNotes = (notes: string) => {
    setState({
      releaseNotes: notes,
    });
  };

  const hideReleaseNotes = () => {
    setState({
      releaseNotes: null,
    });
  };

  const toggleDiffErrModal = (release?: ReleaseWithError) => {
    setState({
      showDiffErrModal: !state.showDiffErrModal,
      releaseWithErr: !state.showDiffErrModal ? release : null,
    });
  };

  const toggleAutomaticUpdatesModal = () => {
    setState({
      showAutomaticUpdatesModal: !state.showAutomaticUpdatesModal,
    });
  };

  const toggleNoChangesModal = (version?: Version) => {
    setState({
      showNoChangesModal: !state.showNoChangesModal,
      releaseWithNoChanges: !state.showNoChangesModal ? version : {},
    });
  };

  const getVersionDiffSummary = (version: Version) => {
    if (!version.diffSummary || version.diffSummary === "") {
      return null;
    }
    try {
      return JSON.parse(version.diffSummary);
    } catch (err) {
      throw err;
    }
  };

  const renderDiff = (version: Version) => {
    const { app } = props;
    const downstream = app?.downstream;
    const diffSummary = getVersionDiffSummary(version);
    const hasDiffSummaryError =
      version.diffSummaryError && version.diffSummaryError.length > 0;
    let previousSequence = 0;
    for (const v of state.versionHistory as Version[]) {
      if (v.status === "pending_download") {
        continue;
      }
      if (v.parentSequence < version.parentSequence) {
        previousSequence = v.parentSequence;
        break;
      }
    }

    if (hasDiffSummaryError) {
      return (
        <div className="flex flex1 alignItems--center">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
            Unable to generate diff{" "}
            <span
              className="replicated-link"
              onClick={() => toggleDiffErrModal(version)}
            >
              Why?
            </span>
          </span>
        </div>
      );
    } else if (diffSummary) {
      return (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
          {diffSummary.filesChanged > 0 ? (
            <div className="DiffSummary u-marginRight--10">
              <span className="files">
                {diffSummary.filesChanged} files changed{" "}
              </span>
              {!props.isHelmManaged && !downstream.gitops?.isConnected && (
                <span
                  className="u-fontSize--small replicated-link u-marginLeft--5"
                  onClick={() =>
                    setState({
                      showDiffOverlay: true,
                      firstSequence: previousSequence,
                      secondSequence: version.parentSequence,
                    })
                  }
                >
                  View diff
                </span>
              )}
            </div>
          ) : (
            <div className="DiffSummary">
              <span className="files">
                No changes to show.{" "}
                <span
                  className="replicated-link"
                  onClick={() => toggleNoChangesModal(version)}
                >
                  Why?
                </span>
              </span>
            </div>
          )}
        </div>
      );
    }
  };

  const renderLogsTabs = () => {
    const { logs, selectedTab } = state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs
          .filter((tab) => tab !== "renderError")
          .filter((tab) => {
            if (props.isHelmManaged) {
              return tab.startsWith("helm");
            }
            return true;
          })
          .map((tab) => (
            <div
              className={`tab-item blue ${tab === selectedTab && "is-active"}`}
              key={tab}
              onClick={() => setState({ selectedTab: tab })}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  };

  const updateVersionDownloadStatus = (version: Version) => {
    const { app } = props;

    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/sequence/${version?.parentSequence}/task/updatedownload`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      )
        .then(async (res) => {
          const response = await res.json();

          if (response.status !== "running") {
            versionDownloadStatusJobs[version.sequence].stop();

            setState({
              versionDownloadStatuses: {
                ...state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: false,
                  downloadingVersionMessage: response.currentMessage,
                  downloadingVersionError: response.status === "failed",
                },
              },
            });

            if (props.updateCallback) {
              props.updateCallback();
            }
            fetchKotsDownstreamHistory();
          } else {
            setState({
              versionDownloadStatuses: {
                ...state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: true,
                  downloadingVersionMessage: response.currentMessage,
                },
              },
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get version download status", err);
          reject();
        });
    });
  };

  const downloadVersion = (version: Version) => {
    const { app } = props;

    if (!versionDownloadStatusJobs.hasOwnProperty(version.sequence)) {
      versionDownloadStatusJobs[version.sequence] = new Repeater();
    }

    setState({
      versionDownloadStatuses: {
        ...state.versionDownloadStatuses,
        [version.sequence]: {
          downloadingVersion: true,
          downloadingVersionMessage: "",
          downloadingVersionError: false,
        },
      },
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/download`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          setState({
            versionDownloadStatuses: {
              ...state.versionDownloadStatuses,
              [version.sequence]: {
                downloadingVersion: false,
                downloadingVersionMessage: response.error,
                downloadingVersionError: true,
              },
            },
          });
          return;
        }
        versionDownloadStatusJobs[version.sequence].start(
          () => updateVersionDownloadStatus(version),
          1000
        );
      })
      .catch((err) => {
        console.log(err);
        setState({
          versionDownloadStatuses: {
            ...state.versionDownloadStatuses,
            [version.sequence]: {
              downloadingVersion: false,
              downloadingVersionMessage:
                err?.message || "Something went wrong, please try again.",
              downloadingVersionError: true,
            },
          },
        });
      });
  };

  const getKotsUpdateStatus = () => {
    const { app } = props;

    return new Promise<void>((resolve) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app.slug}/task/update-admin-console`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      )
        .then(async (res) => {
          if (res.status === 404) {
            // TODO: remove... this is for testing with older kots releases
            state.kotsUpdateChecker.stop();
            window.location.reload();
          }

          const response = await res.json();
          if (response.status === "successful") {
            window.location.reload();
          } else {
            setState({
              kotsUpdateRunning: true,
              kotsUpdateStatus: response.status,
              kotsUpdateMessage: response.message,
              kotsUpdateError: response.error,
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get upgrade status", err);
          setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "waiting",
            kotsUpdateMessage: "Waiting for pods to restart...",
            kotsUpdateError: "",
          });
          resolve();
        });
    });
  };

  const upgradeAdminConsole = (version: Version) => {
    const { app } = props;

    setState({
      displayKotsUpdateModal: true,
      kotsUpdateRunning: true,
      kotsUpdateStatus: "",
      kotsUpdateMessage: "",
      kotsUpdateError: undefined,
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/update-console`,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "failed",
            kotsUpdateError: response.error,
          });
          return;
        }
        state.kotsUpdateChecker.start(getKotsUpdateStatus, 1000);
      })
      .catch((err) => {
        console.log(err);
        setState({
          kotsUpdateRunning: false,
          kotsUpdateStatus: "failed",
          kotsUpdateError:
            err?.message || "Something went wrong, please try again.",
        });
      });
  };

  const renderVersionDownloadStatus = (version: Version) => {
    const { versionDownloadStatuses } = state;
    if (!versionDownloadStatuses.hasOwnProperty(version.sequence)) {
      // user hasn't tried to re-download the version yet, show last known download status if exists
      if (version.downloadStatus) {
        return (
          <div className="flex alignItems--center justifyContent--flexEnd">
            <span
              className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
                version.downloadStatus.status === "failed"
                  ? "u-textColor--error"
                  : ""
              }`}
            >
              {version.downloadStatus.message}
            </span>
          </div>
        );
      }
      return null;
    }

    if (versionDownloadStatuses !== null) {
      const status = versionDownloadStatuses[version.sequence];
      return (
        <div className="flex alignItems--center justifyContent--flexEnd">
          {status?.downloadingVersion && (
            <Loader className="u-marginRight--5" size="15" />
          )}
          <span
            className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
              status.downloadingVersionError ? "u-textColor--error" : ""
            }`}
          >
            {status?.downloadingVersionMessage
              ? status?.downloadingVersionMessage
              : status?.downloadingVersion
              ? "Downloading"
              : ""}
          </span>
        </div>
      );
    }
  };

  const finalizeDeployment = async (continueWithFailedPreflights: boolean) => {
    const { match, updateCallback } = props;
    const { versionToDeploy, isSkipPreflights } = state;
    setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await props.makeCurrentVersion(
      match.params.slug,
      versionToDeploy,
      isSkipPreflights,
      continueWithFailedPreflights
    );
    await fetchKotsDownstreamHistory();
    setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  const deployVersion = (
    version: Version | null,
    force = false,
    continueWithFailedPreflights = false
  ) => {
    const { app } = props;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }
    if (!force && version !== null) {
      if (version.yamlErrors) {
        setState({
          displayShowDetailsModal: !state.displayShowDetailsModal,
          deployView: true,
          versionToDeploy: version,
          yamlErrorDetails: version.yamlErrors,
        });
        return;
      }
      if (version.status === "pending_preflight") {
        setState({
          showSkipModal: true,
          versionToDeploy: version,
          isSkipPreflights: true,
        });
        return;
      }
      if (version?.preflightResult && version.status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult as string);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState === "fail") {
          setState({
            showDeployWarningModal: true,
            versionToDeploy: version,
          });
          return;
        }
      }
      // prompt to make sure user wants to deploy
      setState({
        displayConfirmDeploymentModal: true,
        versionToDeploy: version,
        confirmType: "deploy",
      });
      return;
    } else {
      // force deploy is set to true so finalize the deployment
      finalizeDeployment(continueWithFailedPreflights);
    }
  };

  const redeployVersion = (version: Version, isRollback = false) => {
    const { app } = props;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }

    // prompt to make sure user wants to redeploy
    if (isRollback) {
      setState({
        displayConfirmDeploymentModal: true,
        confirmType: "rollback",
        versionToDeploy: version,
      });
    } else {
      setState({
        displayConfirmDeploymentModal: true,
        confirmType: "redeploy",
        versionToDeploy: version,
      });
    }
  };

  const finalizeRedeployment = async () => {
    const { match, updateCallback } = props;
    const { versionToDeploy } = state;
    setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await props.redeployVersion(match.params.slug, versionToDeploy);
    await fetchKotsDownstreamHistory();
    setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  const onForceDeployClick = (continueWithFailedPreflights = false) => {
    setState({
      showSkipModal: false,
      showDeployWarningModal: false,
      displayShowDetailsModal: false,
    });
    const versionToDeploy = state.versionToDeploy;
    deployVersion(versionToDeploy, true, continueWithFailedPreflights);
  };

  const hideLogsModal = () => {
    setState({
      showLogsModal: false,
    });
  };

  const hideDeployWarningModal = () => {
    setState({
      showDeployWarningModal: false,
    });
  };

  const hideSkipModal = () => {
    setState({
      showSkipModal: false,
    });
  };

  const onCloseReleasesToDiff = () => {
    setState({
      selectedDiffReleases: false,
      checkedReleasesToDiff: [],
      diffHovered: false,
      showDiffOverlay: false,
    });
  };

  const hideDiffOverlay = (closeReleaseSelect: boolean) => {
    setState({
      showDiffOverlay: false,
    });
    if (closeReleaseSelect) {
      onCloseReleasesToDiff();
    }
  };

  const onSelectReleasesToDiff = () => {
    setState({
      selectedDiffReleases: true,
      diffHovered: false,
    });
  };

  const onCheckForUpdates = async () => {
    const { app } = props;

    setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
      checkingUpdateMessage: "",
    });

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          const text = await res.text();
          setState({
            checkingForUpdateError: true,
            checkingForUpdates: false,
            checkingUpdateMessage: text,
          });
          return;
        }
        props.refreshAppData();
        const response = await res.json();

        if (response.availableUpdates === 0) {
          if (
            !find(state.versionHistory, {
              parentSequence: response.currentAppSequence,
            })
          ) {
            // version history list is out of sync - most probably because of automatic updates happening in the background - refetch list
            fetchKotsDownstreamHistory();
            setState({ checkingForUpdates: false });
          } else {
            setState({
              checkingForUpdates: false,
              noUpdateAvailiableText: "There are no updates available",
            });
            setTimeout(() => {
              setState({
                noUpdateAvailiableText: "",
              });
            }, 3000);
          }
        } else {
          state.appUpdateChecker.start(getAppUpdateStatus, 1000);
        }
      })
      .catch((err) => {
        setState({
          checkingForUpdateError: true,
          checkingForUpdates: false,
          checkingUpdateMessage: String(err),
        });
      });
  };

  const handleViewLogs = async (
    version: Version | null,
    isFailing: boolean
  ) => {
    try {
      const { app } = props;
      let clusterId = app.downstream.cluster?.id;
      if (props.isHelmManaged) {
        clusterId = 0;
      }
      setState({
        logsLoading: true,
        showLogsModal: true,
        viewLogsErrMsg: "",
      });

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${app.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      );
      if (res.ok && res.status === 200) {
        const response = await res.json();
        let selectedTab;
        if (isFailing) {
          selectedTab = Utilities.getDeployErrorTab(response.logs);
        } else {
          selectedTab = Object.keys(response.logs)[0];
        }
        setState({
          logs: response.logs,
          selectedTab,
          logsLoading: false,
          viewLogsErrMsg: "",
        });
      } else {
        setState({
          logsLoading: false,
          viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      setState({
        logsLoading: false,
        viewLogsErrMsg: err
          ? `Failed to view logs: ${(err as Error).message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  const getDiffCommitHashes = () => {
    let firstCommitUrl = "",
      secondCommitUrl = "";

    const { checkedReleasesToDiff } = state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort(
        (r1, r2) => r1.parentSequence - r2.parentSequence
      );
      firstCommitUrl = checkedReleasesToDiff[0].commitUrl;
      secondCommitUrl = checkedReleasesToDiff[1].commitUrl;
    }

    return {
      firstHash: getCommitHashFromUrl(firstCommitUrl),
      secondHash: getCommitHashFromUrl(secondCommitUrl),
    };
  };

  const getDiffSequences = () => {
    let firstSequence = 0,
      secondSequence = 0;

    const { checkedReleasesToDiff } = state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort(
        (r1: Version, r2: Version) => r1.parentSequence - r2.parentSequence
      );
      firstSequence = checkedReleasesToDiff[0].parentSequence;
      secondSequence = checkedReleasesToDiff[1].parentSequence;
    }

    return {
      firstSequence,
      secondSequence,
    };
  };

  const renderDiffBtn = () => {
    const { app } = props;
    const { showDiffOverlay, selectedDiffReleases, checkedReleasesToDiff } =
      state;
    const downstream = app?.downstream;
    const gitopsIsConnected = downstream.gitops?.isConnected;
    const versionHistory = state.versionHistory?.length
      ? state.versionHistory
      : [];
    return versionHistory.length && selectedDiffReleases ? (
      <div className="flex u-marginLeft--20">
        <button
          className="btn secondary small u-marginRight--10"
          onClick={onCloseReleasesToDiff}
        >
          Cancel
        </button>
        <button
          className="btn primary small blue"
          disabled={checkedReleasesToDiff.length !== 2 || showDiffOverlay}
          onClick={() => {
            if (gitopsIsConnected) {
              const { firstHash, secondHash } = getDiffCommitHashes();
              if (firstHash && secondHash) {
                const diffUrl = getGitProviderDiffUrl(
                  downstream.gitops?.uri,
                  downstream.gitops?.provider,
                  firstHash,
                  secondHash
                );
                window.open(diffUrl, "_blank");
              }
            } else {
              const { firstSequence, secondSequence } = getDiffSequences();
              setState({
                showDiffOverlay: true,
                firstSequence,
                secondSequence,
              });
            }
          }}
        >
          Diff versions
        </button>
      </div>
    ) : (
      <div
        className="flex-auto flex alignItems--center u-marginLeft--20"
        onClick={onSelectReleasesToDiff}
      >
        <Icon
          icon="diff-icon"
          size={21}
          className="clickable"
          color={""}
          style={{}}
          disableFill={false}
          removeInlineStyle={false}
        />
        <span className="u-fontSize--small u-fontWeight--medium u-linkColor u-cursor--pointer u-marginLeft--5">
          Diff versions
        </span>
      </div>
    );
  };

  const handleSelectReleasesToDiff = (
    selectedRelease: Version,
    isChecked: boolean
  ) => {
    if (isChecked) {
      setState({
        checkedReleasesToDiff: (
          [{ ...selectedRelease, isChecked }] as Version[]
        )
          .concat(state.checkedReleasesToDiff)
          .slice(0, 2),
      });
    } else {
      setState({
        checkedReleasesToDiff: state.checkedReleasesToDiff.filter(
          (release: Version) =>
            release.parentSequence !== selectedRelease.parentSequence
        ),
      });
    }
  };

  const toggleShowDetailsModal = (
    yamlErrorDetails: string[],
    selectedSequence: number
  ) => {
    setState({
      displayShowDetailsModal: !state.displayShowDetailsModal,
      deployView: false,
      yamlErrorDetails,
      selectedSequence,
    });
  };

  const shouldRenderUpdateProgress = () => {
    if (state.uploadingAirgapFile) {
      return true;
    }
    if (props.isBundleUploading) {
      return true;
    }
    if (state.checkingForUpdateError) {
      return true;
    }
    if (state.airgapUploadError) {
      return true;
    }
    if (props.app?.isAirgap && state.checkingForUpdates) {
      return true;
    }
    return false;
  };

  const renderUpdateProgress = () => {
    const { app } = props;

    if (!shouldRenderUpdateProgress()) {
      return null;
    }

    let updateText;
    if (state.airgapUploadError) {
      updateText = (
        <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">
          {state.airgapUploadError}
        </p>
      );
    } else if (state.checkingForUpdateError) {
      updateText = (
        <div className="flex-column flex-auto u-marginTop--10">
          <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">
            Error updating version:
          </p>
          <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">
            {state.checkingUpdateMessage}
          </p>
        </div>
      );
    } else if (state.uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          total={state.uploadSize}
          progress={state.uploadProgress}
          resuming={state.uploadResuming}
          onProgressError={undefined}
          smallSize={true}
        />
      );
    } else if (props.isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          unkownProgress={true}
          onProgressError={undefined}
          smallSize={true}
        />
      );
    } else if (app?.isAirgap && state.checkingForUpdates) {
      let checkingUpdateText = state.checkingUpdateMessage;
      try {
        const jsonMessage = JSON.parse(checkingUpdateText);
        const type = get(jsonMessage, "type");
        if (type === "progressReport") {
          checkingUpdateText = jsonMessage.compatibilityMessage;
          // TODO: handle image upload progress here
        }
      } catch {
        // empty
      }
      if (checkingUpdateText && checkingUpdateText.length > 65) {
        checkingUpdateText = checkingUpdateText.slice(0, 65) + "...";
      }
      updateText = (
        <div className="flex-column justifyContent--center alignItems--center">
          <Loader className="u-marginBottom--10" size="30" />
          <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">
            {checkingUpdateText}
          </span>
        </div>
      );
    }

    return (
      <div className="u-marginTop--20 u-marginBottom--20">{updateText}</div>
    );
  };

  const deployButtonStatus = (version: Version) => {
    if (props.isHelmManaged) {
      const deployedSequence = props.app?.downstream?.currentVersion?.sequence;

      if (version.sequence > deployedSequence) {
        return "Deploy";
      }

      if (version.sequence < deployedSequence) {
        return "Rollback";
      }

      return "Redeploy";
    }

    const app = props.app;
    const downstream = app?.downstream;

    const isCurrentVersion =
      version.sequence === downstream.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";

    const isPastVersion = find(downstream.pastVersions, {
      sequence: version.sequence,
    });
    const needsConfiguration = version.status === "pending_config";
    const isRollback = isPastVersion && version.deployedAt && app.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !props.adminConsoleMetadata?.isAirgap &&
      !props.adminConsoleMetadata?.isKurl;

    if (needsConfiguration) {
      return "Configure";
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
  };

  const handleActionButtonClicked = (
    versionLabel: string | null | undefined,
    sequence: number
  ) => {
    if (props.isHelmManaged && versionLabel) {
      setState({
        showHelmDeployModalForVersionLabel: versionLabel,
        showHelmDeployModalForSequence: sequence,
      });
    }
  };

  const renderAppVersionHistoryRow = (version: Version, index?: number) => {
    if (
      !version ||
      isEmpty(version) ||
      (state.selectedDiffReleases && version.status === "pending_download")
    ) {
      // non-downloaded versions can't be diffed
      return null;
    }

    const downstream = props.app.downstream;
    const gitopsIsConnected = downstream?.gitops?.isConnected;
    const nothingToCommit = gitopsIsConnected && !version.commitUrl;
    const isChecked = !!state.checkedReleasesToDiff.find(
      (diffRelease) => diffRelease.parentSequence === version.parentSequence
    );
    const isNew = secondsAgo(version.createdOn) < 10;
    let newPreflightResults = false;
    if (version.preflightResultCreatedAt) {
      newPreflightResults = secondsAgo(version.preflightResultCreatedAt) < 12;
    }
    let isPending = false;
    if (props.isHelmManaged && version.status.startsWith("pending")) {
      isPending = true;
    }

    return (
      <React.Fragment key={index}>
        <AppVersionHistoryRow
          handleActionButtonClicked={() =>
            handleActionButtonClicked(version.versionLabel, version.sequence)
          }
          isHelmManaged={props.isHelmManaged}
          key={version.sequence}
          app={props.app}
          match={props.match}
          history={props.history}
          version={version}
          selectedDiffReleases={state.selectedDiffReleases}
          nothingToCommit={nothingToCommit}
          isChecked={isChecked}
          isNew={isNew}
          newPreflightResults={newPreflightResults}
          showReleaseNotes={showReleaseNotes}
          renderDiff={renderDiff}
          toggleShowDetailsModal={toggleShowDetailsModal}
          gitopsEnabled={gitopsIsConnected}
          deployVersion={deployVersion}
          redeployVersion={redeployVersion}
          downloadVersion={downloadVersion}
          upgradeAdminConsole={upgradeAdminConsole}
          handleViewLogs={handleViewLogs}
          handleSelectReleasesToDiff={handleSelectReleasesToDiff}
          renderVersionDownloadStatus={renderVersionDownloadStatus}
          isDownloading={
            state.versionDownloadStatuses?.[version.sequence]
              ?.downloadingVersion
          }
          adminConsoleMetadata={props.adminConsoleMetadata}
        />
        {state.showHelmDeployModalForVersionLabel === version.versionLabel &&
          state.showHelmDeployModalForSequence === version.sequence && (
            <UseDownloadValues
              appSlug={props?.app?.slug}
              fileName="values.yaml"
              sequence={version.parentSequence}
              versionLabel={version.versionLabel}
              isPending={isPending}
            >
              {({
                download,
                clearError: clearDownloadError,
                downloadError: downloadError,
                // isDownloading,
                name,
                ref,
                url,
              }: {
                download: () => void;
                clearError: () => void;
                downloadError: boolean;
                //  isDownloading: boolean;
                name: string;
                ref: string;
                url: string;
              }) => {
                return (
                  <>
                    <HelmDeployModal
                      appSlug={props?.app?.slug}
                      chartPath={props?.app?.chartPath || ""}
                      downloadClicked={download}
                      downloadError={downloadError}
                      //isDownloading={isDownloading}
                      hideHelmDeployModal={() => {
                        setState({
                          showHelmDeployModalForVersionLabel: "",
                        });
                        clearDownloadError();
                      }}
                      registryUsername={props?.app?.credentials?.username}
                      registryPassword={props?.app?.credentials?.password}
                      revision={
                        deployButtonStatus(version) === "Rollback"
                          ? version.sequence
                          : null
                      }
                      showHelmDeployModal={true}
                      showDownloadValues={
                        deployButtonStatus(version) === "Deploy"
                      }
                      subtitle={
                        deployButtonStatus(version) === "Rollback"
                          ? `Follow the steps below to rollback to revision ${version.sequence}.`
                          : deployButtonStatus(version) === "Redeploy"
                          ? "Follow the steps below to redeploy the release using the currently deployed chart version and values."
                          : "Follow the steps below to upgrade the release."
                      }
                      title={` ${deployButtonStatus(version)} ${
                        props?.app.slug
                      } ${
                        deployButtonStatus(version) === "Deploy"
                          ? version.versionLabel
                          : ""
                      }`}
                      upgradeTitle={
                        deployButtonStatus(version) === "Rollback"
                          ? "Rollback release"
                          : deployButtonStatus(version) === "Redeploy"
                          ? "Redeploy release"
                          : "Upgrade release"
                      }
                      version={version.versionLabel}
                      namespace={props?.app?.namespace}
                    />
                    <a
                      href={url}
                      download={name}
                      className="hidden"
                      ref={ref}
                    />
                  </>
                );
              }}
            </UseDownloadValues>
          )}
      </React.Fragment>
    );
  };

  const onGotoPage = (page: Number, ev: { preventDefault: () => void }) => {
    ev.preventDefault();
    setState({ currentPage: page, loadingPage: true });
    props.history.push(`${props.location.pathname}?page=${page}`);
    fetchKotsDownstreamHistory({ currentPage: page });
  };

  const renderAllVersions = () => {
    // This is kinda hacky. This finds the equivalent downstream version because the midstream
    // version type does not contain metadata like version label or release notes.
    let allVersions = state.versionHistory;

    // exclude pinned version
    if (props.isHelmManaged) {
      // Only show pending versions in the "New version available" card. Helm, unlike kots, always adds a new version, even when we rollback.
      if (state.updatesAvailable && allVersions?.length > 0) {
        if (allVersions[0].status.startsWith("pending")) {
          allVersions = allVersions?.slice(1);
        }
      }
    } else {
      if (state.updatesAvailable) {
        allVersions = state.versionHistory?.slice(1);
      }
    }

    if (!allVersions?.length) {
      return null;
    }

    const { currentPage, pageSize, totalCount, loadingPage } = state;

    return (
      <div className="TableDiff--Wrapper">
        <div className="flex u-marginBottom--15 justifyContent--spaceBetween">
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy">
            All versions
          </p>
          <div className="flex flex-auto alignItems--center">
            <span className="flex-auto u-marginRight--5 u-fontSize--small u-textColor--secondary u-lineHeight--normal u-fontWeight--medium">
              Results per page:
            </span>
            <select className="Select" onChange={(e) => setPageSize(e)}>
              <option value="20">20</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </div>
        </div>
        {allVersions?.map((version, index) =>
          renderAppVersionHistoryRow(version, index)
        )}
        <Pager
          pagerType="releases"
          currentPage={currentPage}
          pageSize={pageSize}
          totalCount={totalCount}
          loading={loadingPage}
          currentPageLength={allVersions.length}
          goToPage={onGotoPage}
        />
      </div>
    );
  };

  const { app, match, makingCurrentVersionErrMsg, redeployVersionErrMsg } =
    props;

  const {
    showLogsModal,
    selectedTab,
    logs,
    logsLoading,
    showDeployWarningModal,
    showSkipModal,
    releaseNotes,
    showDiffOverlay,
    firstSequence,
    secondSequence,
    loadingVersionHistory,
    versionHistory,
    errorTitle,
    errorMsg,
    displayErrorModal,
    airgapUploader,
    checkingForUpdates,
    checkingUpdateMessage,
  } = state;

  if (!app) {
    return null;
  }

  // only render loader if there is no app yet to avoid flickering
  if (loadingVersionHistory && !versionHistory?.length) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  const downstream = app?.downstream;
  const gitopsIsConnected = downstream.gitops?.isConnected;
  const currentDownstreamVersion = downstream?.currentVersion;
  const isPastVersion = find(downstream?.pastVersions, {
    sequence: state.versionToDeploy?.sequence,
  });

  let checkingUpdateTextShort = checkingUpdateMessage;
  if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
    checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
  }

  const renderKotsUpgradeStatus =
    state.kotsUpdateStatus && !state.kotsUpdateMessage;
  let shortKotsUpdateMessage = state.kotsUpdateMessage;
  if (shortKotsUpdateMessage && shortKotsUpdateMessage.length > 60) {
    shortKotsUpdateMessage = shortKotsUpdateMessage.substring(0, 60) + "...";
  }

  let sequenceLabel = "Sequence";
  if (props.isHelmManaged) {
    sequenceLabel = "Revision";
  }

  // In Helm, only pending versions are updates.  In kots native, a deployed version can be an update after a rollback.
  let pendingVersion;
  if (props.isHelmManaged) {
    if (
      state.updatesAvailable &&
      versionHistory[0].status.startsWith("pending")
    ) {
      pendingVersion = versionHistory[0];
    }
  } else {
    if (state.updatesAvailable) {
      pendingVersion = versionHistory[0];
    }
  }

  return (
    <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
      <KotsPageTitle pageName="Version History" showAppSlug />
      <div className="flex-column flex1">
        <div className="flex flex1 justifyContent--center">
          <div className="flex1 flex AppVersionHistory">
            {makingCurrentVersionErrMsg && (
              <div className="ErrorWrapper flex justifyContent--center">
                <div className="icon redWarningIcon u-marginRight--10" />
                <div>
                  <p className="title">Failed to deploy version</p>
                  <p className="err">{makingCurrentVersionErrMsg}</p>
                </div>
              </div>
            )}
            {redeployVersionErrMsg && (
              <div className="ErrorWrapper flex justifyContent--center">
                <div className="icon redWarningIcon u-marginRight--10" />
                <div>
                  <p className="title">Failed to redeploy version</p>
                  <p className="err">{redeployVersionErrMsg}</p>
                </div>
              </div>
            )}

            {!gitopsIsConnected && (
              <div
                className="flex-column flex1"
                style={{ maxWidth: "370px", marginRight: "20px" }}
              >
                <div className="TableDiff--Wrapper currentVersionCard--wrapper">
                  <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
                    {currentDownstreamVersion?.versionLabel
                      ? "Currently deployed version"
                      : "No current version deployed"}
                  </p>
                  <div className="currentVersion--wrapper u-marginTop--10">
                    <div className="flex flex1">
                      {app?.iconUri && (
                        <div className="flex-auto u-marginRight--10">
                          <div
                            className="watch-icon"
                            style={{
                              backgroundImage: `url(${app?.iconUri})`,
                            }}
                          ></div>
                        </div>
                      )}
                      <div className="flex1 flex-column">
                        <div className="flex alignItems--center u-marginTop--5">
                          <p className="u-fontSize--header2 u-fontWeight--bold u-textColor--primary">
                            {" "}
                            {currentDownstreamVersion
                              ? currentDownstreamVersion.versionLabel
                              : "---"}
                          </p>
                          <p className="u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">
                            {" "}
                            {currentDownstreamVersion
                              ? `${sequenceLabel} ${currentDownstreamVersion?.sequence}`
                              : null}
                          </p>
                        </div>
                        {currentDownstreamVersion?.deployedAt ? (
                          <p className="u-fontSize--small u-lineHeight--normal u-textColor--info u-fontWeight--medium u-marginTop--10">
                            {currentDownstreamVersion?.status === "deploying"
                              ? "Deploy started at"
                              : "Deployed"}{" "}
                            {Utilities.dateFormat(
                              currentDownstreamVersion.deployedAt,
                              "MM/DD/YY @ hh:mm a z"
                            )}
                          </p>
                        ) : null}
                        {currentDownstreamVersion ? (
                          <div className="flex alignItems--center u-marginTop--10">
                            {currentDownstreamVersion?.releaseNotes && (
                              <div className="u-marginRight--5">
                                <Icon
                                  icon="release-notes"
                                  className="clickable"
                                  size={24}
                                  onClick={() =>
                                    showReleaseNotes(
                                      currentDownstreamVersion?.releaseNotes
                                    )
                                  }
                                  data-tip="View release notes"
                                  color={""}
                                  style={{}}
                                  disableFill={false}
                                  removeInlineStyle={false}
                                />
                                <ReactTooltip
                                  effect="solid"
                                  className="replicated-tooltip"
                                />
                              </div>
                            )}
                            {state.hasPreflightChecks ? (
                              <div className="u-marginRight--5">
                                <Link
                                  to={`/app/${app?.slug}/downstreams/${app.downstream.cluster?.slug}/version-history/preflight/${currentDownstreamVersion?.sequence}`}
                                  data-tip="View preflight checks"
                                >
                                  <Icon
                                    icon="preflight-checks"
                                    size={22}
                                    className="clickable"
                                    color={""}
                                    style={{}}
                                    disableFill={false}
                                    removeInlineStyle={false}
                                  />
                                </Link>
                                <ReactTooltip
                                  effect="solid"
                                  className="replicated-tooltip"
                                />
                              </div>
                            ) : null}
                            {app ? (
                              <div>
                                <span
                                  onClick={() =>
                                    handleViewLogs(
                                      currentDownstreamVersion,
                                      currentDownstreamVersion?.status ===
                                        "failed"
                                    )
                                  }
                                  data-tip="View deploy logs"
                                >
                                  <Icon
                                    icon="view-logs"
                                    size={22}
                                    className="clickable"
                                    color={""}
                                    style={{}}
                                    disableFill={false}
                                    removeInlineStyle={false}
                                  />
                                </span>
                                <ReactTooltip
                                  effect="solid"
                                  className="replicated-tooltip"
                                />
                              </div>
                            ) : null}
                            {currentDownstreamVersion?.status === "failed" ? (
                              <div className="u-position--relative u-marginLeft--10 u-marginRight--10">
                                <Icon
                                  icon="preflight-checks"
                                  size={22}
                                  className="clickable"
                                  color={""}
                                  style={{}}
                                  disableFill={false}
                                  removeInlineStyle={false}
                                />
                                <Icon
                                  icon={"warning-circle-filled"}
                                  size={12}
                                  className="version-row-preflight-status-icon warning-color"
                                  style={{ left: "15px", top: "-6px" }}
                                  color={""}
                                  disableFill={false}
                                  removeInlineStyle={false}
                                />
                              </div>
                            ) : null}
                            {app.isConfigurable && (
                              <div>
                                <Link
                                  to={`/app/${app?.slug}/config/${app?.downstream?.currentVersion?.parentSequence}`}
                                  data-tip="Edit config"
                                >
                                  <Icon
                                    icon="edit-config"
                                    size={22}
                                    color={""}
                                    style={{}}
                                    className={""}
                                    disableFill={false}
                                    removeInlineStyle={false}
                                  />
                                </Link>
                                <ReactTooltip
                                  effect="solid"
                                  className="replicated-tooltip"
                                />
                              </div>
                            )}
                          </div>
                        ) : null}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div
              className={`flex-column flex1 alignSelf--start ${
                gitopsIsConnected ? "gitops-enabled" : ""
              }`}
            >
              <div
                className={`flex-column flex1 version ${
                  showDiffOverlay ? "u-visibility--hidden" : ""
                }`}
              >
                {(versionHistory.length === 0 && gitopsIsConnected) ||
                versionHistory?.length > 0 ? (
                  <>
                    {gitopsIsConnected ? (
                      <div
                        style={{ maxWidth: "1030px" }}
                        className="u-width--full u-marginBottom--30"
                      >
                        <DashboardGitOpsCard
                          gitops={downstream?.gitops}
                          isAirgap={app?.isAirgap}
                          appSlug={app?.slug}
                          checkingForUpdates={checkingForUpdates}
                          latestConfigSequence={
                            versionHistory[0]?.parentSequence
                          }
                          isBundleUploading={props.isBundleUploading}
                          checkingUpdateText={checkingUpdateMessage}
                          checkingUpdateTextShort={checkingUpdateTextShort}
                          onCheckForUpdates={onCheckForUpdates}
                          showAutomaticUpdatesModal={
                            toggleAutomaticUpdatesModal
                          }
                        />
                      </div>
                    ) : (
                      <div className="TableDiff--Wrapper u-marginBottom--30">
                        <div className="flex justifyContent--spaceBetween">
                          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-marginBottom--15">
                            {state.updatesAvailable
                              ? "New version available"
                              : ""}
                          </p>
                          <div className="flex alignItems--center">
                            <div className="flex alignItems--center">
                              {app?.isAirgap && airgapUploader ? (
                                <MountAware
                                  onMount={(el: Element) =>
                                    airgapUploader?.assignElement(el)
                                  }
                                >
                                  <div className="flex alignItems--center">
                                    <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
                                    <span className="replicated-link u-fontSize--small u-lineHeight--default">
                                      Upload new version
                                    </span>
                                  </div>
                                </MountAware>
                              ) : (
                                <div className="flex alignItems--center">
                                  {checkingForUpdates &&
                                  !props.isBundleUploading ? (
                                    <div className="flex alignItems--center u-marginRight--20">
                                      <Loader
                                        className="u-marginRight--5"
                                        size="15"
                                      />
                                      <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                                        {checkingUpdateMessage === ""
                                          ? "Checking for updates"
                                          : checkingUpdateTextShort}
                                      </span>
                                    </div>
                                  ) : (
                                    <div className="flex alignItems--center u-marginRight--20">
                                      <span
                                        className="replicated-link u-fontSize--small"
                                        onClick={onCheckForUpdates}
                                      >
                                        <Icon
                                          icon="check-update"
                                          size={18}
                                          className="clickable u-marginRight--5"
                                          color={""}
                                          style={{}}
                                          disableFill={false}
                                          removeInlineStyle={false}
                                        />
                                        Check for update
                                      </span>
                                    </div>
                                  )}
                                  <span
                                    className="flex-auto flex alignItems--center replicated-link u-fontSize--small"
                                    onClick={toggleAutomaticUpdatesModal}
                                  >
                                    <Icon
                                      icon="schedule-sync"
                                      size={16}
                                      className="clickable u-marginRight--5"
                                      color={""}
                                      style={{}}
                                      disableFill={false}
                                      removeInlineStyle={false}
                                    />
                                    Configure automatic updates
                                  </span>
                                </div>
                              )}
                            </div>
                            {versionHistory.length > 1 &&
                            !gitopsIsConnected &&
                            !props.isHelmManaged
                              ? renderDiffBtn()
                              : null}
                          </div>
                        </div>
                        {pendingVersion ? (
                          renderAppVersionHistoryRow(pendingVersion)
                        ) : (
                          <div className="flex-column flex1 u-marginTop--20 u-marginBottom--10 alignItems--center justifyContent--center u-backgroundColor--white u-borderRadius--rounded">
                            <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--10">
                              Application up to date.
                            </p>
                          </div>
                        )}
                        {(state.numOfSkippedVersions > 0 ||
                          state.numOfRemainingVersions > 0) && (
                          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--more u-textColor--header u-marginTop--10">
                            {state.numOfSkippedVersions > 0
                              ? `${state.numOfSkippedVersions} version${
                                  state.numOfSkippedVersions > 1 ? "s" : ""
                                } will be skipped in upgrading to ${
                                  versionHistory[0].versionLabel
                                }. `
                              : ""}
                            {state.numOfRemainingVersions > 0
                              ? "Additional versions are available after you deploy this required version."
                              : ""}
                          </p>
                        )}
                      </div>
                    )}
                    {versionHistory?.length > 0 ? (
                      <>
                        {renderUpdateProgress()}
                        {renderAllVersions()}
                      </>
                    ) : null}
                  </>
                ) : (
                  <div className="flex-column flex1 alignItems--center justifyContent--center">
                    <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary">
                      No versions have been deployed.
                    </p>
                  </div>
                )}
              </div>

              {/* Diff overlay */}
              {showDiffOverlay && (
                <div className="DiffOverlay">
                  <DownstreamWatchVersionDiff
                    slug={match.params.slug}
                    firstSequence={firstSequence}
                    secondSequence={secondSequence}
                    onBackClick={hideDiffOverlay}
                    app={props.app}
                  />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {showLogsModal && (
        <ShowLogsModal
          showLogsModal={showLogsModal}
          hideLogsModal={hideLogsModal}
          viewLogsErrMsg={state.viewLogsErrMsg}
          logs={logs}
          selectedTab={selectedTab}
          logsLoading={logsLoading}
          renderLogsTabs={renderLogsTabs()}
        />
      )}

      {showDeployWarningModal && (
        <DeployWarningModal
          showDeployWarningModal={showDeployWarningModal}
          hideDeployWarningModal={hideDeployWarningModal}
          onForceDeployClick={onForceDeployClick}
          showAutoDeployWarning={
            isPastVersion && props.app?.autoDeploy !== "disabled"
          }
          confirmType={state.confirmType}
        />
      )}

      {showSkipModal && (
        <SkipPreflightsModal
          showSkipModal={showSkipModal}
          hideSkipModal={hideSkipModal}
          onForceDeployClick={onForceDeployClick}
        />
      )}

      <Modal
        isOpen={!!releaseNotes}
        onRequestClose={hideReleaseNotes}
        contentLabel="Release Notes"
        ariaHideApp={false}
        className="Modal MediumSize"
      >
        <div className="flex-column">
          <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
            {releaseNotes || ""}
          </MarkdownRenderer>
        </div>
        <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
          <button className="btn primary" onClick={hideReleaseNotes}>
            Close
          </button>
        </div>
      </Modal>

      <Modal
        isOpen={state.showDiffErrModal}
        onRequestClose={() => toggleDiffErrModal()}
        contentLabel="Unable to Get Diff"
        ariaHideApp={false}
        className="Modal MediumSize"
      >
        <div className="Modal-body">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
            Unable to generate a file diff for release
          </p>
          {state.releaseWithErr && (
            <>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
                The release with the{" "}
                <span className="u-fontWeight--bold">
                  Upstream {state.releaseWithErr.title}, Sequence{" "}
                  {state.releaseWithErr.sequence}
                </span>{" "}
                was unable to generate a files diff because the following error:
              </p>
              <div className="error-block-wrapper u-marginBottom--30 flex flex1">
                <span className="u-textColor--error">
                  {state.releaseWithErr.diffSummaryError}
                </span>
              </div>
            </>
          )}
          <div className="flex u-marginBottom--10">
            <button
              className="btn primary"
              onClick={() => toggleDiffErrModal()}
            >
              Ok, got it!
            </button>
          </div>
        </div>
      </Modal>

      {state.displayConfirmDeploymentModal && (
        <Modal
          isOpen={true}
          onRequestClose={() =>
            setState({
              displayConfirmDeploymentModal: false,
              confirmType: "",
              versionToDeploy: null,
            })
          }
          contentLabel="Confirm deployment"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              {state.confirmType === "rollback"
                ? "Rollback to"
                : state.confirmType === "redeploy"
                ? "Redeploy"
                : "Deploy"}{" "}
              {state.versionToDeploy?.versionLabel} (Sequence{" "}
              {state.versionToDeploy?.sequence})?
            </p>
            {isPastVersion && props.app?.autoDeploy !== "disabled" ? (
              <div className="info-box">
                <span className="u-fontSize--small u-textColor--header u-lineHeight--normal u-fontWeight--medium">
                  You have automatic deploys enabled.{" "}
                  {state.confirmType === "rollback"
                    ? "Rolling back to"
                    : state.confirmType === "redeploy"
                    ? "Redeploying"
                    : "Deploying"}{" "}
                  this version will disable automatic deploys. You can turn it
                  back on after this version finishes deployment.
                </span>
              </div>
            ) : null}
            <div className="flex u-paddingTop--10">
              <button
                className="btn secondary blue"
                onClick={() =>
                  setState({
                    displayConfirmDeploymentModal: false,
                    confirmType: "",
                    versionToDeploy: null,
                  })
                }
              >
                Cancel
              </button>
              <button
                className="u-marginLeft--10 btn primary"
                onClick={
                  state.confirmType === "redeploy"
                    ? finalizeRedeployment
                    : () => finalizeDeployment(false)
                }
              >
                Yes,{" "}
                {state.confirmType === "rollback"
                  ? "rollback"
                  : state.confirmType === "redeploy"
                  ? "redeploy"
                  : "deploy"}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {state.displayKotsUpdateModal && (
        <Modal
          isOpen={true}
          onRequestClose={() => setState({ displayKotsUpdateModal: false })}
          contentLabel="Upgrade is in progress"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body u-textAlign--center">
            <div className="flex-column justifyContent--center alignItems--center">
              <p className="u-fontSize--large u-textColor--primary u-lineHeight--bold u-marginBottom--10">
                Upgrading...
              </p>
              <Loader className="flex alignItems--center" size="32" />
              {renderKotsUpgradeStatus ? (
                <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                  {state.kotsUpdateStatus}
                </p>
              ) : null}
              {state.kotsUpdateMessage ? (
                <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                  {shortKotsUpdateMessage}
                </p>
              ) : null}
            </div>
          </div>
        </Modal>
      )}

      {state.displayShowDetailsModal && (
        <ShowDetailsModal
          displayShowDetailsModal={state.displayShowDetailsModal}
          toggleShowDetailsModal={toggleShowDetailsModal}
          yamlErrorDetails={state.yamlErrorDetails}
          deployView={state.deployView}
          forceDeploy={onForceDeployClick}
          showDeployWarningModal={state.showDeployWarningModal}
          showSkipModal={state.showSkipModal}
          slug={props.match.params.slug}
          sequence={state.selectedSequence}
        />
      )}
      {errorMsg && (
        <ErrorModal
          errorModal={displayErrorModal}
          toggleErrorModal={toggleErrorModal}
          err={errorTitle}
          errMsg={errorMsg}
          appSlug={props.match.params.slug}
        />
      )}
      {state.showNoChangesModal && (
        <Modal
          isOpen={true}
          onRequestClose={() => toggleNoChangesModal()}
          contentLabel="No Changes"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              No changes to show
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              The{" "}
              {state.releaseWithNoChanges && (
                <span className="u-fontWeight--bold">
                  Upstream {state.releaseWithNoChanges.versionLabel}, Sequence{" "}
                  {state.releaseWithNoChanges.sequence}{" "}
                </span>
              )}
              release was unable to generate a diff because the changes made do
              not affect any manifests that will be deployed. Only changes
              affecting the application manifest will be included in a diff.
            </p>
            <div className="flex u-paddingTop--10">
              <button
                className="btn primary"
                onClick={() => toggleNoChangesModal()}
              >
                Ok, got it!
              </button>
            </div>
          </div>
        </Modal>
      )}
      {state.showAutomaticUpdatesModal && (
        <AutomaticUpdatesModal
          isOpen={state.showAutomaticUpdatesModal}
          onRequestClose={toggleAutomaticUpdatesModal}
          updateCheckerSpec={app?.updateCheckerSpec}
          autoDeploy={app?.autoDeploy}
          appSlug={app?.slug}
          isSemverRequired={app?.isSemverRequired}
          gitopsIsConnected={downstream?.gitops?.isConnected}
          onAutomaticUpdatesConfigured={() => {
            toggleAutomaticUpdatesModal();
            props.updateCallback();
          }}
          isHelmManaged={props.isHelmManaged}
        />
      )}
    </div>
  );
};

// @ts-ignore
// eslint-disable-next-line
export default withRouter(AppVersionHistory) as any;
