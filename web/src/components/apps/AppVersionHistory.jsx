import React, { Component } from "react";
import { withRouter, Link } from "react-router-dom";
import Helmet from "react-helmet";
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
import AirgapUploadProgress from "../AirgapUploadProgress";
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

import "@src/scss/components/apps/AppVersionHistory.scss";
import DashboardGitOpsCard from "./DashboardGitOpsCard";
dayjs.extend(relativeTime);

class AppVersionHistory extends Component {
  state = {
    logsLoading: false,
    logs: null,
    selectedTab: null,
    showDeployWarningModal: false,
    showSkipModal: false,
    versionToDeploy: null,
    releaseNotes: null,
    selectedDiffReleases: false,
    checkedReleasesToDiff: [],
    diffHovered: false,
    uploadingAirgapFile: false,
    checkingForUpdates: false,
    checkingUpdateMessage: "Checking for updates",
    checkingForUpdateError: false,
    airgapUploadError: null,
    versionDownloadStatuses: {},
    showDiffOverlay: false,
    firstSequence: 0,
    secondSequence: 0,
    appUpdateChecker: new Repeater(),
    uploadProgress: 0,
    uploadSize: 0,
    uploadResuming: false,
    displayShowDetailsModal: false,
    yamlErrorDetails: [],
    deployView: false,
    selectedSequence: -1,
    releaseWithErr: {},
    versionHistoryJob: new Repeater(),
    loadingVersionHistory: true,
    versionHistory: [],
    errorTitle: "",
    errorMsg: "",
    displayErrorModal: false,
    displayConfirmDeploymentModal: false,
    confirmType: "",
    isSkipPreflights: false,
    displayKotsUpdateModal: false,
    kotsUpdateChecker: new Repeater(),
    kotsUpdateRunning: false,
    kotsUpdateStatus: undefined,
    kotsUpdateMessage: undefined,
    kotsUpdateError: undefined,
    numOfSkippedVersions: 0,
    numOfRemainingVersions: 0,
    totalCount: 0,
    currentPage: 0,
    pageSize: 20,
    loadingPage: false,
    hasPreflightChecks: true,
  };

  // moving this out of the state because new repeater instances were getting created
  // and it doesn't really affect the UI
  versionDownloadStatusJobs = {};

  componentDidMount() {
    this.getPreflightState(this.props.app.downstream.currentVersion);
    const urlParams = new URLSearchParams(window.location.search);
    const pageNumber = urlParams.get("page");
    if (pageNumber) {
      this.setState({ currentPage: parseInt(pageNumber) });
    } else {
      this.props.history.push(`${this.props.location.pathname}?page=0`);
    }

    this.fetchKotsDownstreamHistory();
    this.props.refreshAppData();
    if (this.props.app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig();
    }

    // check if there are any updates in progress
    this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);

    const url = window.location.pathname;
    const { params } = this.props.match;
    if (url.includes("/diff")) {
      const firstSequence = params.firstSequence;
      const secondSequence = params.secondSequence;
      if (firstSequence !== undefined && secondSequence !== undefined) {
        // undefined because a sequence can be zero!
        this.setState({ showDiffOverlay: true, firstSequence, secondSequence });
      }
    }

    this._mounted = true;
  }

  componentDidUpdate = async (lastProps) => {
    if (
      lastProps.match.params.slug !== this.props.match.params.slug ||
      lastProps.app.id !== this.props.app.id
    ) {
      this.fetchKotsDownstreamHistory();
    }
    if (
      this.props.app?.downstream &&
      this.props.app?.downstream !== lastProps.app?.downstream
    ) {
      if (this.props.app.downstream.pendingVersions.length > 0) {
        this.setState({ updatesAvailable: true });
      } else {
        this.setState({ updatesAvailable: false });
      }
    }
  };

  componentWillUnmount() {
    this.state.appUpdateChecker.stop();
    this.state.versionHistoryJob.stop();
    for (const j in this.versionDownloadStatusJobs) {
      this.versionDownloadStatusJobs[j].stop();
    }
    this._mounted = false;
  }

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
    }
  };

  setPageSize = (e) => {
    this.setState(
      { pageSize: parseInt(e.target.value), currentPage: 0 },
      () => {
        this.fetchKotsDownstreamHistory();
        this.props.history.push(`${this.props.location.pathname}?page=0`);
      }
    );
  };

  getAirgapConfig = async () => {
    const { app } = this.props;
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

    this.setState({
      airgapUploader: new AirgapUploader(
        true,
        app.slug,
        this.onDropBundle,
        simultaneousUploads
      ),
    });
  };

  onDropBundle = async () => {
    this.setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: null,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });

    this.props.toggleIsBundleUploading(true);

    const params = {
      appId: this.props.app?.id,
    };
    this.state.airgapUploader.upload(
      params,
      this.onUploadProgress,
      this.onUploadError,
      this.onUploadComplete
    );
  };

  onUploadProgress = (progress, size, resuming = false) => {
    this.setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  };

  onUploadError = (message) => {
    this.setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError: message || "Error uploading bundle, please try again",
    });
    this.props.toggleIsBundleUploading(false);
  };

  onUploadComplete = () => {
    this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);
    this.setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.toggleIsBundleUploading(false);
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  showReleaseNotes = (notes) => {
    this.setState({
      releaseNotes: notes,
    });
  };

  hideReleaseNotes = () => {
    this.setState({
      releaseNotes: null,
    });
  };

  toggleDiffErrModal = (release) => {
    this.setState({
      showDiffErrModal: !this.state.showDiffErrModal,
      releaseWithErr: !this.state.showDiffErrModal ? release : {},
    });
  };

  toggleAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: !this.state.showAutomaticUpdatesModal,
    });
  };

  toggleNoChangesModal = (version) => {
    this.setState({
      showNoChangesModal: !this.state.showNoChangesModal,
      releaseWithNoChanges: !this.state.showNoChangesModal ? version : {},
    });
  };

  getVersionDiffSummary = (version) => {
    if (!version.diffSummary || version.diffSummary === "") {
      return null;
    }
    try {
      return JSON.parse(version.diffSummary);
    } catch (err) {
      throw err;
    }
  };

  renderDiff = (version) => {
    const { app } = this.props;
    const downstream = app?.downstream;
    const diffSummary = this.getVersionDiffSummary(version);
    const hasDiffSummaryError =
      version.diffSummaryError && version.diffSummaryError.length > 0;

    let previousSequence = 0;
    for (const v of this.state.versionHistory) {
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
              onClick={() => this.toggleDiffErrModal(version)}
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
              {!this.props.isHelmManaged && !downstream.gitops?.isConnected && (
                <span
                  className="u-fontSize--small replicated-link u-marginLeft--5"
                  onClick={() =>
                    this.setState({
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
                  onClick={() => this.toggleNoChangesModal(version)}
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

  renderLogsTabs = () => {
    const { logs, selectedTab } = this.state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs
          .filter((tab) => tab !== "renderError")
          .map((tab) => (
            <div
              className={`tab-item blue ${tab === selectedTab && "is-active"}`}
              key={tab}
              onClick={() => this.setState({ selectedTab: tab })}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  };

  downloadVersion = (version) => {
    const { app } = this.props;

    if (!this.versionDownloadStatusJobs.hasOwnProperty(version.sequence)) {
      this.versionDownloadStatusJobs[version.sequence] = new Repeater();
    }

    this.setState({
      versionDownloadStatuses: {
        ...this.state.versionDownloadStatuses,
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
          this.setState({
            versionDownloadStatuses: {
              ...this.state.versionDownloadStatuses,
              [version.sequence]: {
                downloadingVersion: false,
                downloadingVersionMessage: response.error,
                downloadingVersionError: true,
              },
            },
          });
          return;
        }
        this.versionDownloadStatusJobs[version.sequence].start(
          () => this.updateVersionDownloadStatus(version),
          1000
        );
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          versionDownloadStatuses: {
            ...this.state.versionDownloadStatuses,
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

  upgradeAdminConsole = (version) => {
    const { app } = this.props;

    this.setState({
      displayKotsUpdateModal: true,
      kotsUpdateRunning: true,
      kotsUpdateStatus: undefined,
      kotsUpdateMessage: undefined,
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
          this.setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "failed",
            kotsUpdateError: response.error,
          });
          return;
        }
        this.state.kotsUpdateChecker.start(this.getKotsUpdateStatus, 1000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          kotsUpdateRunning: false,
          kotsUpdateStatus: "failed",
          kotsUpdateError:
            err?.message || "Something went wrong, please try again.",
        });
      });
  };

  getKotsUpdateStatus = () => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
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
            this.state.kotsUpdateChecker.stop();
            window.location.reload();
          }

          const response = await res.json();
          if (response.status === "successful") {
            window.location.reload();
          } else {
            this.setState({
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
          this.setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "waiting",
            kotsUpdateMessage: "Waiting for pods to restart...",
            kotsUpdateError: "",
          });
          resolve();
        });
    });
  };

  updateVersionDownloadStatus = (version) => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
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
            this.versionDownloadStatusJobs[version.sequence].stop();

            this.setState({
              versionDownloadStatuses: {
                ...this.state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: false,
                  downloadingVersionMessage: response.currentMessage,
                  downloadingVersionError: response.status === "failed",
                },
              },
            });

            if (this.props.updateCallback) {
              this.props.updateCallback();
            }
            this.fetchKotsDownstreamHistory();
          } else {
            this.setState({
              versionDownloadStatuses: {
                ...this.state.versionDownloadStatuses,
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

  renderVersionDownloadStatus = (version) => {
    const { versionDownloadStatuses } = this.state;

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

    const status = versionDownloadStatuses[version.sequence];

    return (
      <div className="flex alignItems--center justifyContent--flexEnd">
        {status.downloadingVersion && (
          <Loader className="u-marginRight--5" size="15" />
        )}
        <span
          className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
            status.downloadingVersionError ? "u-textColor--error" : ""
          }`}
        >
          {status.downloadingVersionMessage
            ? status.downloadingVersionMessage
            : status.downloadingVersion
            ? "Downloading"
            : ""}
        </span>
      </div>
    );
  };

  deployVersion = (
    version,
    force = false,
    continueWithFailedPreflights = false
  ) => {
    const { app } = this.props;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }

    if (!force) {
      if (version.yamlErrors) {
        this.setState({
          displayShowDetailsModal: !this.state.displayShowDetailsModal,
          deployView: true,
          versionToDeploy: version,
          yamlErrorDetails: version.yamlErrors,
        });
        return;
      }
      if (version.status === "pending_preflight") {
        this.setState({
          showSkipModal: true,
          versionToDeploy: version,
          isSkipPreflights: true,
        });
        return;
      }
      if (version?.preflightResult && version.status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState === "fail") {
          this.setState({
            showDeployWarningModal: true,
            versionToDeploy: version,
          });
          return;
        }
      }
      // prompt to make sure user wants to deploy
      this.setState({
        displayConfirmDeploymentModal: true,
        versionToDeploy: version,
        confirmType: "deploy",
      });
      return;
    } else {
      // force deploy is set to true so finalize the deployment
      this.finalizeDeployment(continueWithFailedPreflights);
    }
  };

  finalizeDeployment = async (continueWithFailedPreflights) => {
    const { match, updateCallback } = this.props;
    const { versionToDeploy, isSkipPreflights } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await this.props.makeCurrentVersion(
      match.params.slug,
      versionToDeploy,
      isSkipPreflights,
      continueWithFailedPreflights
    );
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  redeployVersion = (version, isRollback = false) => {
    const { app } = this.props;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }

    // prompt to make sure user wants to redeploy
    if (isRollback) {
      this.setState({
        displayConfirmDeploymentModal: true,
        confirmType: "rollback",
        versionToDeploy: version,
      });
    } else {
      this.setState({
        displayConfirmDeploymentModal: true,
        confirmType: "redeploy",
        versionToDeploy: version,
      });
    }
  };

  finalizeRedeployment = async () => {
    const { match, updateCallback } = this.props;
    const { versionToDeploy } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await this.props.redeployVersion(match.params.slug, versionToDeploy);
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  onForceDeployClick = (continueWithFailedPreflights = false) => {
    this.setState({
      showSkipModal: false,
      showDeployWarningModal: false,
      displayShowDetailsModal: false,
    });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true, continueWithFailedPreflights);
  };

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false,
    });
  };

  hideDeployWarningModal = () => {
    this.setState({
      showDeployWarningModal: false,
    });
  };

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false,
    });
  };

  hideDiffOverlay = (closeReleaseSelect) => {
    this.setState({
      showDiffOverlay: false,
    });
    if (closeReleaseSelect) {
      this.onCloseReleasesToDiff();
    }
  };

  onSelectReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: true,
      diffHovered: false,
    });
  };

  onCloseReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: false,
      checkedReleasesToDiff: [],
      diffHovered: false,
      showDiffOverlay: false,
    });
  };

  onCheckForUpdates = async () => {
    const { app } = this.props;

    this.setState({
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
          this.setState({
            checkingForUpdateError: true,
            checkingForUpdates: false,
            checkingUpdateMessage: text,
          });
          return;
        }
        this.props.refreshAppData();
        const response = await res.json();

        if (response.availableUpdates === 0) {
          if (
            !find(this.state.versionHistory, {
              parentSequence: response.currentAppSequence,
            })
          ) {
            // version history list is out of sync - most probably because of automatic updates happening in the background - refetch list
            this.fetchKotsDownstreamHistory();
            this.setState({ checkingForUpdates: false });
          } else {
            this.setState({
              checkingForUpdates: false,
              noUpdateAvailiableText: "There are no updates available",
            });
            setTimeout(() => {
              this.setState({
                noUpdateAvailiableText: null,
              });
            }, 3000);
          }
        } else {
          this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);
        }
      })
      .catch((err) => {
        this.setState({
          checkingForUpdateError: true,
          checkingForUpdates: false,
          checkingUpdateMessage: String(err),
        });
      });
  };

  getAppUpdateStatus = () => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
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

          if (response.status !== "running" && !this.props.isBundleUploading) {
            this.state.appUpdateChecker.stop();

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed",
            });

            if (this.props.updateCallback) {
              this.props.updateCallback();
            }
            this.fetchKotsDownstreamHistory();
          } else {
            this.setState({
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

  handleViewLogs = async (version, isFailing) => {
    try {
      const { app } = this.props;
      let clusterId = app.downstream.cluster?.id;
      if (this.props.isHelmManaged) {
        clusterId = 0;
      }
      this.setState({
        logsLoading: true,
        showLogsModal: true,
        viewLogsErrMsg: "",
      });

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`,
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
        this.setState({
          logs: response.logs,
          selectedTab,
          logsLoading: false,
          viewLogsErrMsg: "",
        });
      } else {
        this.setState({
          logsLoading: false,
          viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      this.setState({
        logsLoading: false,
        viewLogsErrMsg: err
          ? `Failed to view logs: ${err.message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  renderDiffBtn = () => {
    const { app } = this.props;
    const { showDiffOverlay, selectedDiffReleases, checkedReleasesToDiff } =
      this.state;
    const downstream = app?.downstream;
    const gitopsIsConnected = downstream.gitops?.isConnected;
    const versionHistory = this.state.versionHistory?.length
      ? this.state.versionHistory
      : [];
    return versionHistory.length && selectedDiffReleases ? (
      <div className="flex u-marginLeft--20">
        <button
          className="btn secondary small u-marginRight--10"
          onClick={this.onCloseReleasesToDiff}
        >
          Cancel
        </button>
        <button
          className="btn primary small blue"
          disabled={checkedReleasesToDiff.length !== 2 || showDiffOverlay}
          onClick={() => {
            if (gitopsIsConnected) {
              const { firstHash, secondHash } = this.getDiffCommitHashes();
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
              const { firstSequence, secondSequence } = this.getDiffSequences();
              this.setState({
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
        onClick={this.onSelectReleasesToDiff}
      >
        <span className="icon clickable diffReleasesIcon"></span>
        <span className="u-fontSize--small u-fontWeight--medium u-linkColor u-cursor--pointer u-marginLeft--5">
          Diff versions
        </span>
      </div>
    );
  };

  handleSelectReleasesToDiff = (selectedRelease, isChecked) => {
    if (isChecked) {
      this.setState({
        checkedReleasesToDiff: [{ ...selectedRelease, isChecked }]
          .concat(this.state.checkedReleasesToDiff)
          .slice(0, 2),
      });
    } else {
      this.setState({
        checkedReleasesToDiff: this.state.checkedReleasesToDiff.filter(
          (release) => release.parentSequence !== selectedRelease.parentSequence
        ),
      });
    }
  };

  displayTooltip = (key, value) => {
    return () => {
      this.setState({
        [`${key}Hovered`]: value,
      });
    };
  };

  getDiffSequences = () => {
    let firstSequence = 0,
      secondSequence = 0;

    const { checkedReleasesToDiff } = this.state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort(
        (r1, r2) => r1.parentSequence - r2.parentSequence
      );
      firstSequence = checkedReleasesToDiff[0].parentSequence;
      secondSequence = checkedReleasesToDiff[1].parentSequence;
    }

    return {
      firstSequence,
      secondSequence,
    };
  };

  getDiffCommitHashes = () => {
    let firstCommitUrl = "",
      secondCommitUrl = "";

    const { checkedReleasesToDiff } = this.state;
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

  toggleShowDetailsModal = (yamlErrorDetails, selectedSequence) => {
    this.setState({
      displayShowDetailsModal: !this.state.displayShowDetailsModal,
      deployView: false,
      yamlErrorDetails,
      selectedSequence,
    });
  };

  shouldRenderUpdateProgress = () => {
    if (this.state.uploadingAirgapFile) {
      return true;
    }
    if (this.props.isBundleUploading) {
      return true;
    }
    if (this.state.checkingForUpdateError) {
      return true;
    }
    if (this.state.airgapUploadError) {
      return true;
    }
    if (this.props.app?.isAirgap && this.state.checkingForUpdates) {
      return true;
    }
    return false;
  };

  renderUpdateProgress = () => {
    const { app } = this.props;

    if (!this.shouldRenderUpdateProgress()) {
      return null;
    }

    let updateText;
    if (this.state.airgapUploadError) {
      updateText = (
        <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">
          {this.state.airgapUploadError}
        </p>
      );
    } else if (this.state.checkingForUpdateError) {
      updateText = (
        <div className="flex-column flex-auto u-marginTop--10">
          <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">
            Error updating version:
          </p>
          <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">
            {this.state.checkingUpdateMessage}
          </p>
        </div>
      );
    } else if (this.state.uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          total={this.state.uploadSize}
          progress={this.state.uploadProgress}
          resuming={this.state.uploadResuming}
          onProgressError={undefined}
          smallSize={true}
        />
      );
    } else if (this.props.isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          unkownProgress={true}
          onProgressError={undefined}
          smallSize={true}
        />
      );
    } else if (app?.isAirgap && this.state.checkingForUpdates) {
      let checkingUpdateText = this.state.checkingUpdateMessage;
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

  renderAllVersions = () => {
    // This is kinda hacky. This finds the equivalent downstream version because the midstream
    // version type does not contain metadata like version label or release notes.
    let allVersions = this.state.versionHistory;
    if (this.state.updatesAvailable) {
      allVersions = this.state.versionHistory?.slice(1); // exclude pinned version
    }
    if (!allVersions?.length) {
      return null;
    }

    const { currentPage, pageSize, totalCount, loadingPage } = this.state;

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
            <select className="Select" onChange={(e) => this.setPageSize(e)}>
              <option value="20">20</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </div>
        </div>
        {allVersions?.map((version, index) =>
          this.renderAppVersionHistoryRow(version, index)
        )}
        <Pager
          pagerType="releases"
          currentPage={currentPage}
          pageSize={pageSize}
          totalCount={totalCount}
          loading={loadingPage}
          currentPageLength={allVersions.length}
          goToPage={this.onGotoPage}
        />
      </div>
    );
  };

  onGotoPage = (page, ev) => {
    ev.preventDefault();
    this.setState({ currentPage: page, loadingPage: true }, async () => {
      this.props.history.push(`${this.props.location.pathname}?page=${page}`);
      await this.fetchKotsDownstreamHistory();
      this.setState({ loadingPage: false });
    });
  };

  handleActionButtonClicked = ({ versionLabel, sequence }) => {
    if (this.props.isHelmManaged) {
      this.setState({
        showHelmDeployModalForVersionLabel: versionLabel,
        showHelmDeployModalForSequence: sequence,
      });
    }
  };

  deployButtonStatus = (version) => {
    if (this.props.isHelmManaged) {
      const deployedSequence =
        this.props.app?.downstream?.currentVersion?.sequence;

      if (version.sequence > deployedSequence) {
        return "Deploy";
      }

      if (version.sequence < deployedSequence) {
        return "Rollback";
      }

      return "Redeploy";
    }

    const app = this.props.app;
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
      !this.props.adminConsoleMetadata?.isAirgap &&
      !this.props.adminConsoleMetadata?.isKurl;

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

  renderAppVersionHistoryRow = (version, index) => {
    if (
      !version ||
      isEmpty(version) ||
      (this.state.selectedDiffReleases && version.status === "pending_download")
    ) {
      // non-downloaded versions can't be diffed
      return null;
    }

    const downstream = this.props.app.downstream;
    const gitopsIsConnected = downstream?.gitops?.isConnected;
    const nothingToCommit = gitopsIsConnected && !version.commitUrl;
    const isChecked = !!this.state.checkedReleasesToDiff.find(
      (diffRelease) => diffRelease.parentSequence === version.parentSequence
    );
    const isNew = secondsAgo(version.createdOn) < 10;
    let newPreflightResults = false;
    if (version.preflightResultCreatedAt) {
      newPreflightResults = secondsAgo(version.preflightResultCreatedAt) < 12;
    }
    return (
      <React.Fragment key={index}>
        <AppVersionHistoryRow
          handleActionButtonClicked={() =>
            this.handleActionButtonClicked({
              sequence: version.sequence,
              versionLabel: version.versionLabel,
            })
          }
          isHelmManaged={this.props.isHelmManaged}
          key={version.sequence}
          app={this.props.app}
          match={this.props.match}
          history={this.props.history}
          version={version}
          selectedDiffReleases={this.state.selectedDiffReleases}
          nothingToCommit={nothingToCommit}
          isChecked={isChecked}
          isNew={isNew}
          newPreflightResults={newPreflightResults}
          showReleaseNotes={this.showReleaseNotes}
          renderDiff={this.renderDiff}
          toggleShowDetailsModal={this.toggleShowDetailsModal}
          gitopsEnabled={gitopsIsConnected}
          deployVersion={this.deployVersion}
          redeployVersion={this.redeployVersion}
          downloadVersion={this.downloadVersion}
          upgradeAdminConsole={this.upgradeAdminConsole}
          handleViewLogs={this.handleViewLogs}
          handleSelectReleasesToDiff={this.handleSelectReleasesToDiff}
          renderVersionDownloadStatus={this.renderVersionDownloadStatus}
          isDownloading={
            this.state.versionDownloadStatuses?.[version.sequence]
              ?.downloadingVersion
          }
          adminConsoleMetadata={this.props.adminConsoleMetadata}
        />
        {this.state.showHelmDeployModalForVersionLabel ===
          version.versionLabel &&
          this.state.showHelmDeployModalForSequence === version.sequence && (
            <UseDownloadValues
              appSlug={this.props?.app?.slug}
              fileName="values.yaml"
            >
              {({
                download,
                clearError: clearDownloadError,
                error: downloadError,
                isDownloading,
                name,
                ref,
                url,
              }) => {
                return (
                  <>
                    <HelmDeployModal
                      appSlug={this.props?.app?.slug}
                      chartPath={this.props?.app?.chartPath || ""}
                      downloadClicked={download}
                      error={downloadError}
                      isDownloading={isDownloading}
                      hideHelmDeployModal={() => {
                        this.setState({
                          showHelmDeployModalForVersionLabel: "",
                        });
                        clearDownloadError();
                      }}
                      registryUsername={this.props?.app?.credentials?.username}
                      registryPassword={this.props?.app?.credentials?.password}
                      revision={
                        this.deployButtonStatus(version) === "Rollback"
                          ? version.sequence
                          : null
                      }
                      showHelmDeployModal={true}
                      showDownloadValues={
                        this.deployButtonStatus(version) === "Deploy"
                      }
                      subtitle={
                        this.deployButtonStatus(version) === "Rollback"
                          ? `Follow the steps below to rollback to revision ${version.sequence}.`
                          : this.deployButtonStatus(version) === "Redeploy"
                          ? "Follow the steps below to redeploy the release using the currently deployed chart version and values."
                          : "Follow the steps below to upgrade the release."
                      }
                      title={` ${this.deployButtonStatus(version)} ${
                        this.props?.app.slug
                      } ${
                        this.deployButtonStatus(version) === "Deploy"
                          ? version.versionLabel
                          : ""
                      }`}
                      upgradeTitle={
                        this.deployButtonStatus(version) === "Rollback"
                          ? "Rollback release"
                          : this.deployButtonStatus(version) === "Redeploy"
                          ? "Redeploy release"
                          : "Upgrade release"
                      }
                      version={version.versionLabel}
                      namespace={this.props?.app?.namespace}
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

  getPreflightState = (version) => {
    let preflightState = "";
    if (version?.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      preflightState = getPreflightResultState(preflightResult);
    }
    if (preflightState === "") {
      this.setState({ hasPreflightChecks: false });
    }
  };

  render() {
    const { app, match, makingCurrentVersionErrMsg, redeployVersionErrMsg } =
      this.props;

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
    } = this.state;

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
      sequence: this.state.versionToDeploy?.sequence,
    });

    let checkingUpdateTextShort = checkingUpdateMessage;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    const renderKotsUpgradeStatus =
      this.state.kotsUpdateStatus && !this.state.kotsUpdateMessage;
    let shortKotsUpdateMessage = this.state.kotsUpdateMessage;
    if (shortKotsUpdateMessage && shortKotsUpdateMessage.length > 60) {
      shortKotsUpdateMessage = shortKotsUpdateMessage.substring(0, 60) + "...";
    }

    let sequenceLabel = "Sequence";
    if (this.props.isHelmManaged) {
      sequenceLabel = "Revision";
    }

    return (
      <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{`${app.name} Version History`}</title>
        </Helmet>
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
                                <div>
                                  <span
                                    className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer"
                                    onClick={() =>
                                      this.showReleaseNotes(
                                        currentDownstreamVersion?.releaseNotes
                                      )
                                    }
                                    data-tip="View release notes"
                                  />
                                  <ReactTooltip
                                    effect="solid"
                                    className="replicated-tooltip"
                                  />
                                </div>
                              )}
                              {this.state.hasPreflightChecks ? (
                                <div>
                                  <Link
                                    to={`/app/${app?.slug}/downstreams/${app.downstream.cluster?.slug}/version-history/preflight/${currentDownstreamVersion?.sequence}`}
                                    className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer"
                                    data-tip="View preflight checks"
                                  />
                                  <ReactTooltip
                                    effect="solid"
                                    className="replicated-tooltip"
                                  />
                                </div>
                              ) : null}
                              <div>
                                <span
                                  className="icon deployLogs--icon u-cursor--pointer"
                                  onClick={() =>
                                    this.handleViewLogs(
                                      currentDownstreamVersion,
                                      currentDownstreamVersion?.status ===
                                        "failed"
                                    )
                                  }
                                  data-tip="View deploy logs"
                                />
                                <ReactTooltip
                                  effect="solid"
                                  className="replicated-tooltip"
                                />
                                {currentDownstreamVersion?.status ===
                                "failed" ? (
                                  <span className="icon version-row-preflight-status-icon preflight-checks-failed-icon logs" />
                                ) : null}
                              </div>
                              {app.isConfigurable && (
                                <div>
                                  <Link
                                    to={`/app/${app?.slug}/config/${app?.downstream?.currentVersion?.parentSequence}`}
                                    className="icon configEdit--icon u-cursor--pointer"
                                    data-tip="Edit config"
                                  />
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
                            isBundleUploading={this.props.isBundleUploading}
                            checkingUpdateText={checkingUpdateMessage}
                            checkingUpdateTextShort={checkingUpdateTextShort}
                            noUpdatesAvalable={this.props.noUpdatesAvalable}
                            onCheckForUpdates={this.onCheckForUpdates}
                            showAutomaticUpdatesModal={
                              this.toggleAutomaticUpdatesModal
                            }
                          />
                        </div>
                      ) : (
                        <div className="TableDiff--Wrapper u-marginBottom--30">
                          <div className="flex justifyContent--spaceBetween">
                            <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-marginBottom--15">
                              {this.state.updatesAvailable
                                ? "New version available"
                                : ""}
                            </p>
                            <div className="flex alignItems--center">
                              <div className="flex alignItems--center">
                                {app?.isAirgap && airgapUploader ? (
                                  <MountAware
                                    onMount={(el) =>
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
                                    !this.props.isBundleUploading ? (
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
                                        <span className="icon clickable dashboard-card-check-update-icon u-marginRight--5" />
                                        <span
                                          className="replicated-link u-fontSize--small"
                                          onClick={this.onCheckForUpdates}
                                        >
                                          Check for update
                                        </span>
                                      </div>
                                    )}
                                    <span className="icon clickable dashboard-card-configure-update-icon u-marginRight--5" />
                                    <span
                                      className="replicated-link u-fontSize--small"
                                      onClick={this.toggleAutomaticUpdatesModal}
                                    >
                                      Configure automatic updates
                                    </span>
                                  </div>
                                )}
                              </div>
                              {versionHistory.length > 1 &&
                              !gitopsIsConnected &&
                              !this.props.isHelmManaged
                                ? this.renderDiffBtn()
                                : null}
                            </div>
                          </div>
                          {this.state.updatesAvailable ? (
                            this.renderAppVersionHistoryRow(versionHistory[0])
                          ) : (
                            <div className="flex-column flex1 u-marginTop--20 u-marginBottom--10 alignItems--center justifyContent--center u-backgroundColor--white u-borderRadius--rounded">
                              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--10">
                                Application up to date.
                              </p>
                            </div>
                          )}
                          {(this.state.numOfSkippedVersions > 0 ||
                            this.state.numOfRemainingVersions > 0) && (
                            <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--more u-textColor--header u-marginTop--10">
                              {this.state.numOfSkippedVersions > 0
                                ? `${this.state.numOfSkippedVersions} version${
                                    this.state.numOfSkippedVersions > 1
                                      ? "s"
                                      : ""
                                  } will be skipped in upgrading to ${
                                    versionHistory[0].versionLabel
                                  }. `
                                : ""}
                              {this.state.numOfRemainingVersions > 0
                                ? "Additional versions are available after you deploy this required version."
                                : ""}
                            </p>
                          )}
                        </div>
                      )}
                      {versionHistory?.length > 0 ? (
                        <>
                          {this.renderUpdateProgress()}
                          {this.renderAllVersions()}
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
                      onBackClick={this.hideDiffOverlay}
                      app={this.props.app}
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
            hideLogsModal={this.hideLogsModal}
            viewLogsErrMsg={this.state.viewLogsErrMsg}
            logs={logs}
            selectedTab={selectedTab}
            logsLoading={logsLoading}
            renderLogsTabs={this.renderLogsTabs()}
          />
        )}

        {showDeployWarningModal && (
          <DeployWarningModal
            showDeployWarningModal={showDeployWarningModal}
            hideDeployWarningModal={this.hideDeployWarningModal}
            onForceDeployClick={this.onForceDeployClick}
            showAutoDeployWarning={
              isPastVersion && this.props.app?.autoDeploy !== "disabled"
            }
            confirmType={this.state.confirmType}
          />
        )}

        {showSkipModal && (
          <SkipPreflightsModal
            showSkipModal={showSkipModal}
            hideSkipModal={this.hideSkipModal}
            onForceDeployClick={this.onForceDeployClick}
          />
        )}

        <Modal
          isOpen={!!releaseNotes}
          onRequestClose={this.hideReleaseNotes}
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
            <button className="btn primary" onClick={this.hideReleaseNotes}>
              Close
            </button>
          </div>
        </Modal>

        <Modal
          isOpen={this.state.showDiffErrModal}
          onRequestClose={this.toggleDiffErrModal}
          contentLabel="Unable to Get Diff"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              Unable to generate a file diff for release
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              The release with the{" "}
              <span className="u-fontWeight--bold">
                Upstream {this.state.releaseWithErr.title}, Sequence{" "}
                {this.state.releaseWithErr.sequence}
              </span>{" "}
              was unable to generate a files diff because the following error:
            </p>
            <div className="error-block-wrapper u-marginBottom--30 flex flex1">
              <span className="u-textColor--error">
                {this.state.releaseWithErr.diffSummaryError}
              </span>
            </div>
            <div className="flex u-marginBottom--10">
              <button className="btn primary" onClick={this.toggleDiffErrModal}>
                Ok, got it!
              </button>
            </div>
          </div>
        </Modal>

        {this.state.displayConfirmDeploymentModal && (
          <Modal
            isOpen={true}
            onRequestClose={() =>
              this.setState({
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
                {this.state.confirmType === "rollback"
                  ? "Rollback to"
                  : this.state.confirmType === "redeploy"
                  ? "Redeploy"
                  : "Deploy"}{" "}
                {this.state.versionToDeploy?.versionLabel} (Sequence{" "}
                {this.state.versionToDeploy?.sequence})?
              </p>
              {isPastVersion && this.props.app?.autoDeploy !== "disabled" ? (
                <div className="info-box">
                  <span className="u-fontSize--small u-textColor--header u-lineHeight--normal u-fontWeight--medium">
                    You have automatic deploys enabled.{" "}
                    {this.state.confirmType === "rollback"
                      ? "Rolling back to"
                      : this.state.confirmType === "redeploy"
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
                    this.setState({
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
                    this.state.confirmType === "redeploy"
                      ? this.finalizeRedeployment
                      : () => this.finalizeDeployment(false)
                  }
                >
                  Yes,{" "}
                  {this.state.confirmType === "rollback"
                    ? "rollback"
                    : this.state.confirmType === "redeploy"
                    ? "redeploy"
                    : "deploy"}
                </button>
              </div>
            </div>
          </Modal>
        )}

        {this.state.displayKotsUpdateModal && (
          <Modal
            isOpen={true}
            onRequestClose={() =>
              this.setState({ displayKotsUpdateModal: false })
            }
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
                    {this.state.kotsUpdateStatus}
                  </p>
                ) : null}
                {this.state.kotsUpdateMessage ? (
                  <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                    {shortKotsUpdateMessage}
                  </p>
                ) : null}
              </div>
            </div>
          </Modal>
        )}

        {this.state.displayShowDetailsModal && (
          <ShowDetailsModal
            displayShowDetailsModal={this.state.displayShowDetailsModal}
            toggleShowDetailsModal={this.toggleShowDetailsModal}
            yamlErrorDetails={this.state.yamlErrorDetails}
            deployView={this.state.deployView}
            forceDeploy={this.onForceDeployClick}
            showDeployWarningModal={this.state.showDeployWarningModal}
            showSkipModal={this.state.showSkipModal}
            slug={this.props.match.params.slug}
            sequence={this.state.selectedSequence}
          />
        )}
        {errorMsg && (
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
            appSlug={this.props.match.params.slug}
          />
        )}
        {this.state.showNoChangesModal && (
          <Modal
            isOpen={true}
            onRequestClose={this.toggleNoChangesModal}
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
                <span className="u-fontWeight--bold">
                  Upstream {this.state.releaseWithNoChanges.versionLabel},
                  Sequence {this.state.releaseWithNoChanges.sequence}
                </span>{" "}
                release was unable to generate a diff because the changes made
                do not affect any manifests that will be deployed. Only changes
                affecting the application manifest will be included in a diff.
              </p>
              <div className="flex u-paddingTop--10">
                <button
                  className="btn primary"
                  onClick={this.toggleNoChangesModal}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
        {this.state.showAutomaticUpdatesModal && (
          <AutomaticUpdatesModal
            isOpen={this.state.showAutomaticUpdatesModal}
            onRequestClose={this.toggleAutomaticUpdatesModal}
            updateCheckerSpec={app?.updateCheckerSpec}
            autoDeploy={app?.autoDeploy}
            appSlug={app?.slug}
            isSemverRequired={app?.isSemverRequired}
            gitopsIsConnected={downstream?.gitops?.isConnected}
            onAutomaticUpdatesConfigured={() => {
              this.toggleAutomaticUpdatesModal();
              this.props.updateCallback();
            }}
            isHelmManaged={this.props.isHelmManaged}
            />
        )}
      </div>
    );
  }
}

export default withRouter(AppVersionHistory);
