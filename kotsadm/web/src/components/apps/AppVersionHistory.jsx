import React, { Component } from "react";
import classNames from "classnames";
import { withRouter, Link } from "react-router-dom";
import Helmet from "react-helmet";
import dayjs from "dayjs";
import ReactTooltip from "react-tooltip"
import relativeTime from "dayjs/plugin/relativeTime";
import Modal from "react-modal";
import find from "lodash/find";

import Loader from "../shared/Loader";
import MountAware from "../shared/MountAware";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import UpdateCheckerModal from "@src/components/modals/UpdateCheckerModal";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import ErrorModal from "../modals/ErrorModal";
import { Utilities, isAwaitingResults, secondsAgo, getPreflightResultState, getGitProviderDiffUrl, getCommitHashFromUrl } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import { AirgapUploader } from "../../utilities/airgapUploader";
import get from "lodash/get";

import "@src/scss/components/apps/AppVersionHistory.scss";
dayjs.extend(relativeTime);

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host"
};

class AppVersionHistory extends Component {
  state = {
    viewReleaseNotes: false,
    logsLoading: false,
    logs: null,
    selectedTab: null,
    showDeployWarningModal: false,
    showSkipModal: false,
    versionToDeploy: null,
    downstreamReleaseNotes: null,
    selectedDiffReleases: false,
    checkedReleasesToDiff: [],
    diffHovered: false,
    uploadingAirgapFile: false,
    checkingForUpdates: false,
    checkingUpdateMessage: "Checking for updates",
    errorCheckingUpdate: false,
    airgapUploadError: null,
    showDiffOverlay: false,
    firstSequence: 0,
    secondSequence: 0,
    updateChecker: new Repeater(),
    uploadProgress: 0,
    uploadSize: 0,
    uploadResuming: false,
    showUpdateCheckerModal: false,
    displayShowDetailsModal: false,
    yamlErrorDetails: [],
    deployView: false,
    selectedSequence: "",
    releaseWithErr: {},
    versionHistoryJob: new Repeater(),
    loadingVersionHistory: true,
    versionHistory: [],
    errorTitle: "",
    errorMsg: "",
    displayErrorModal: false,
  }

  componentWillMount() {
    if (this.props.app.isAirgap) {
      this.airgapUploader = new AirgapUploader(true, this.onDropBundle);
    }
  }

  componentDidMount() {
    this.fetchKotsDownstreamHistory();
    this.state.updateChecker.start(this.updateStatus, 1000);

    const url = window.location.pathname;
    if (url.includes("/diff")) {
      const { params } = this.props.match;
      const firstSequence = params.firstSequence;
      const secondSequence = params.secondSequence;
      if (firstSequence !== undefined && secondSequence !== undefined) { // undefined because a sequence can be zero!
        this.setState({ showDiffOverlay: true, firstSequence, secondSequence });
      }
    }
  }

  componentDidUpdate = async (lastProps) => {
    if (lastProps.match.params.slug !== this.props.match.params.slug || lastProps.app.id !== this.props.app.id) {
      this.fetchKotsDownstreamHistory();
    }
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
    this.state.versionHistoryJob.stop();
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
      const res = await fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/versions`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
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

      if (isAwaitingResults(versionHistory)) {
        this.state.versionHistoryJob.start(this.fetchKotsDownstreamHistory, 2000);
      } else {
        this.state.versionHistoryJob.stop();
      }

      this.setState({
        loadingVersionHistory: false,
        versionHistory: versionHistory,
      });
    } catch (err) {
      this.setState({
        loadingVersionHistory: false,
        errorTitle: "Failed to get version history",
        errorMsg: err ? err.message : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  showReleaseNotes = () => {
    this.setState({
      viewReleaseNotes: true
    });
  }

  hideReleaseNotes = () => {
    this.setState({
      viewReleaseNotes: false
    });
  }

  showDownstreamReleaseNotes = notes => {
    this.setState({
      downstreamReleaseNotes: notes
    });
  }

  hideDownstreamReleaseNotes = () => {
    this.setState({
      downstreamReleaseNotes: null
    });
  }

  hideUpdateCheckerModal = () => {
    this.setState({
      showUpdateCheckerModal: false
    });
  }

  showUpdateCheckerModal = () => {
    this.setState({
      showUpdateCheckerModal: true
    });
  }

  toggleDiffErrModal = (release) => {
    this.setState({
      showDiffErrModal: !this.state.showDiffErrModal,
      releaseWithErr: !this.state.showDiffErrModal ? release : {}
    })
  }

  getVersionDiffSummary = version => {
    if (!version.diffSummary || version.diffSummary === "") {
      return null;
    }
    try {
      return JSON.parse(version.diffSummary);
    } catch (err) {
      throw err;
    }
  }

  renderYamlErrors = (yamlErrorsDetails, version) => {
    return (
      <div className="flex alignItems--center u-marginLeft--5">
        <span className="icon error-small" />
        <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-color--red">{yamlErrorsDetails?.length} Invalid file{yamlErrorsDetails?.length !== 1 ? "s" : ""} </span>
        <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => this.toggleShowDetailsModal(yamlErrorsDetails, version.sequence)}> See details </span>
      </div>
    )
  }

  renderSourceAndDiff = version => {
    const { app } = this.props;
    const downstream = app.downstreams?.length && app.downstreams[0];
    const diffSummary = this.getVersionDiffSummary(version);
    const hasDiffSummaryError = version.diffSummaryError && version.diffSummaryError.length > 0;

    if (hasDiffSummaryError) {
      return (
        <div className="flex flex1 alignItems--center">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--dustyGray">Cannot generate diff <span className="replicated-link" onClick={() => this.toggleDiffErrModal(version)}>Why?</span></span>
        </div>
      );
    } else {
      return (
        <div>
          {diffSummary ?
            (diffSummary.filesChanged > 0 ?
              <div
                className="DiffSummary u-cursor--pointer u-marginRight--10"
                onClick={() => {
                  if (!downstream.gitops?.enabled) {
                    this.setState({
                      showDiffOverlay: true,
                      firstSequence: version.parentSequence - 1,
                      secondSequence: version.parentSequence
                    });
                  }
                }}
              >
                <span className="files">{diffSummary.filesChanged} files changed </span>
                <span className="lines-added">+{diffSummary.linesAdded} </span>
                <span className="lines-removed">-{diffSummary.linesRemoved}</span>
              </div>
              :
              <div className="DiffSummary">
                <span className="files">No changes</span>
              </div>
            )
            : <span>&nbsp;</span>}
        </div>
      );
    }

  }

  renderVersionAction = (version, nothingToCommitDiff) => {
    const { app } = this.props;
    const downstream = app.downstreams[0];

    if (downstream.gitops?.enabled) {
      if (version.gitDeployable === false) {
        return (<div className={nothingToCommitDiff && "u-opacity--half"}>Nothing to commit</div>);
      }
      if (!version.commitUrl) {
        return null;
      }
      return (
        <button
          className="btn primary blue"
          onClick={() => window.open(version.commitUrl, '_blank')}
        >
          View
        </button>
      );
    }

    const isCurrentVersion = version.sequence === downstream.currentVersion?.sequence;
    const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
    const needsConfiguration = version.status === "pending_config";
    const showActions = !isPastVersion || app.allowRollback;
    const isSecondaryBtn = isPastVersion || needsConfiguration;
    const isRollback = isPastVersion && version.deployedAt && app.allowRollback;

    return (
      <div className="flex flex1 justifyContent--flexEnd">
        {showActions &&
          <button
            className={classNames("btn", { "secondary blue": isSecondaryBtn, "primary blue": !isSecondaryBtn })}
            disabled={isCurrentVersion}
            onClick={() => needsConfiguration ? this.props.history.push(`/app/${app.slug}/config/${version.sequence}`) : this.deployVersion(version)}
          >
            {needsConfiguration ?
              "Configure" :
              downstream.currentVersion?.sequence == undefined ?
                "Deploy" :
                isRollback ?
                  "Rollback" :
                  isCurrentVersion ?
                    "Deployed" :
                    "Deploy"
            }
          </button>
        }
      </div>
    );
  }

  renderViewPreflights = version => {
    const { match, app } = this.props;
    const downstream = app.downstreams[0];
    const clusterSlug = downstream.cluster?.slug;
    return (
      <Link className="u-marginTop--10" to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version?.sequence}`}>
        <span className="replicated-link" style={{ fontSize: 12 }}>View preflight results</span>
      </Link>
    );
  }


  renderVersionStatus = version => {
    const { app, match } = this.props;
    const downstream = app.downstreams?.length && app.downstreams[0];
    if (!downstream) {
      return null;
    }

    let preflightsFailed = false;
    if (version.status === "pending" && version.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      const preflightState = getPreflightResultState(preflightResult);
      preflightsFailed = preflightState === "fail";
    }

    const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
    const isPendingDeployedVersion = find(downstream.pendingVersions, { sequence: version.sequence, status: "deployed" });
    const clusterSlug = downstream.cluster?.slug;
    let preflightBlock = null;

    if (isPastVersion && app.hasPreflight) {
      if (preflightsFailed) {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">See details</Link>);
      } else if (version.status !== "pending_config") {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
      }
    }
    if (version.status === "pending_preflight") {
      preflightBlock = (
        <span className="flex u-marginLeft--5 alignItems--center">
          <Loader size="20" />
        </span>);
    } else if (app.hasPreflight) {
      if (preflightsFailed) {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">See details</Link>);
      } else if (version.status !== "pending_config") {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
      }
    }

    if (!isPastVersion && !isPendingDeployedVersion) {
      return (
        <div className="flex alignItems--center">
          <div className="flex alignItems--center">
            <div
              data-tip={`${version.versionLabel || version.title}-${version.sequence}`}
              data-for={`${version.versionLabel || version.title}-${version.sequence}`}
              className={classNames("icon", {
                "checkmark-icon": version.status === "deployed" || version.status === "merged" || version.status === "pending",
                "exclamationMark--icon": version.status === "opened",
                "grayCircleMinus--icon": version.status === "closed",
                "error-small": version.status === "failed" || preflightsFailed
              })}
            />
            {version.status === "deploying" &&  <Loader className="flex alignItems--center" size="20" />}
            <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
              "u-color--nevada": version.status === "deployed" || version.status === "merged",
              "u-color--orange": version.status === "opened",
              "u-color--dustyGray": version.status === "closed" || version.status === "pending" || version.status === "pending_preflight",
              "u-color--red": version.status === "failed" || preflightsFailed
            })}>
              {Utilities.toTitleCase(
                version.status === "pending_preflight"
                  ? "Running checks"
                  : preflightsFailed
                    ? "Checks failed"
                    : version.status === "pending"
                      ? "Ready to deploy"
                      : version.status
              ).replace("_", " ")}
            </span>
          </div>
          {preflightBlock}
          {version.status === "failed" &&
            <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => this.handleViewLogs(version, true)}>View logs</span>
          }
        </div>
      );
    } else {
      return (
        <div className="flex alignItems--center">
          <div className="flex alignItems--center">
            <div
              data-tip={`${version.versionLabel || version.title}-${version.sequence}`}
              data-for={`${version.versionLabel || version.title}-${version.sequence}`}
              className={classNames("icon", {
                "analysis-gray_checkmark": version.status === "deployed" || version.status === "merged",
                "exclamationMark--icon": version.status === "opened",
                "grayCircleMinus--icon": version.status === "closed" || version.status === "pending",
                "error-small": version.status === "failed"
              })}
            />
            <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
              "u-color--nevada": version.status === "deployed" || version.status === "merged",
              "u-color--orange": version.status === "opened",
              "u-color--dustyGray": version.status === "closed" || version.status === "pending" || version.status === "pending_preflight",
              "u-color--red": version.status === "failed"
            })}>
              {version.status === "deployed" ?
                "Previously Deployed" :
                version.status === "pending" ?
                  "Skipped" :
                  version.status === "failed" ?
                    "Failed" : ""}
            </span>
          </div>
          {preflightBlock}
          {version.status === "failed" &&
            <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => this.handleViewLogs(version, true)}>View logs</span>
          }
        </div>
      );
    }
  }

  renderLogsTabs = () => {
    const { logs, selectedTab } = this.state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.filter(tab => tab !== "renderError").map(tab => (
          <div className={`tab-item blue ${tab === selectedTab && "is-active"}`} key={tab} onClick={() => this.setState({ selectedTab: tab })}>
            {tab}
          </div>
        ))}
      </div>
    );
  }

  deployVersion = async (version, force = false) => {
    const { match, app } = this.props;
    const clusterSlug = app.downstreams?.length && app.downstreams[0].cluster?.slug;
    if (!clusterSlug) {
      return;
    }
    const downstream = app.downstreams?.length && app.downstreams[0];
    const yamlErrorDetails = this.yamlErrorsDetails(downstream, version);


    if (!force) {
      if (yamlErrorDetails) {
        this.setState({
          displayShowDetailsModal: !this.state.displayShowDetailsModal,
          deployView: true,
          yamlErrorDetails
        });
      }
      if (version.status === "pending_preflight") {
        this.setState({
          showSkipModal: true,
          versionToDeploy: version
        });
        return;
      }
      if (version?.preflightResult && version.status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState === "fail") {
          this.setState({
            showDeployWarningModal: true,
            versionToDeploy: version
          });
          return;
        }
      }
    }
    await this.props.makeCurrentVersion(match.params.slug, version);
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (this.props.updateCallback) {
      this.props.updateCallback();
    }
  }

  onForceDeployClick = () => {
    this.setState({ showSkipModal: false, showDeployWarningModal: false, displayShowDetailsModal: false });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true);
  }

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false
    });
  }

  hideDeployWarningModal = () => {
    this.setState({
      showDeployWarningModal: false
    });
  }

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false
    });
  }

  hideDiffOverlay = (closeReleaseSelect) => {
    this.setState({
      showDiffOverlay: false
    });
    if (closeReleaseSelect) {
      this.onCloseReleasesToDiff();
    }
  }

  onSelectReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: true,
      diffHovered: false
    });
  }

  onCloseReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: false,
      checkedReleasesToDiff: [],
      diffHovered: false,
      showDiffOverlay: false
    });
  }

  handleViewLogs = async (version, isFailing) => {
    try {
      const { app } = this.props;
      const clusterId = app.downstreams?.length && app.downstreams[0].cluster?.id;

      this.setState({ logsLoading: true, showLogsModal: true, viewLogsErrMsg: "" });

      const res = await fetch(`${window.env.API_ENDPOINT}/app/${app?.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status === 200) {
        const response = await res.json();
        let selectedTab;
        if (isFailing) {
          selectedTab = "applyStderr";
        } else {
          selectedTab = Object.keys(response.logs)[0];
        }
        this.setState({ logs: response.logs, selectedTab, logsLoading: false, viewLogsErrMsg: "" });
      } else {
        this.setState({ logsLoading: false, viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}` });
      }
    } catch (err) {
      console.log(err)
      this.setState({ logsLoading: false, viewLogsErrMsg: err ? `Failed to view logs: ${err.message}` : "Something went wrong, please try again." });
    }
  }

  updateStatus = () => {
    return new Promise((resolve, reject) => {
      fetch(`${window.env.API_ENDPOINT}/task/updatedownload`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
        .then(async (res) => {
          const response = await res.json();

          if (response.status !== "running" && !this.props.isBundleUploading) {
            this.state.updateChecker.stop();

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response === "failed"
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
        }).catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
    });
  }

  onCheckForUpdates = async () => {
    const { app } = this.props;

    this.setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
      errorCheckingUpdate: false,
      checkingUpdateMessage: "",
    });

    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          const text = await res.text();
          this.setState({
            errorCheckingUpdate: true,
            checkingForUpdates: false,
            checkingUpdateMessage: text,
          });
          return;
        }
        this.props.refreshAppData();
        const response = await res.json();
        if (response.availableUpdates === 0) {
          if (!find(this.state.versionHistory, { parentSequence: response.currentAppSequence })) {
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
          this.state.updateChecker.start(this.updateStatus, 1000);
        }
      })
      .catch((err) => {
        this.setState({
          errorCheckingUpdate: true,
          checkingForUpdates: false,
          checkingUpdateMessage: String(err),
        });
      });
  }

  onDropBundle = async () => {
    this.setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: null,
      checkingForUpdateError: false,
      checkingUpdateMessage: ""
    });

    this.props.toggleIsBundleUploading(true);

    const params = {
      appId: this.props.app.id,
    };
    this.airgapUploader.upload(params, this.onUploadProgress, this.onUploadError, this.onUploadComplete);
  }

  onUploadProgress = (progress, size, resuming = false) => {
    this.setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  }

  onUploadError = message => {
    this.setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError: message || "Error uploading bundle, please try again"
    });
    this.props.toggleIsBundleUploading(false);
  }

  onUploadComplete = () => {
    this.state.updateChecker.start(this.updateStatus, 1000);
    this.setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.toggleIsBundleUploading(false);
  }

  onProgressError = async (airgapUploadError) => {
    Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
      if (airgapUploadError.includes(errorString)) {
        airgapUploadError = message;
      }
    });
    this.setState({
      uploadingAirgapFile: false,
      airgapUploadError,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.toggleIsBundleUploading(false);
  }

  renderDiffBtn = () => {
    const { app } = this.props;
    const {
      showDiffOverlay,
      selectedDiffReleases,
      checkedReleasesToDiff,
    } = this.state;
    const downstream = app.downstreams.length && app.downstreams[0];
    const gitopsEnabled = downstream.gitops?.enabled;
    const versionHistory = this.state.versionHistory?.length ? this.state.versionHistory : [];
    return (
      versionHistory.length && selectedDiffReleases ?
        <div className="flex">
          <button className="btn secondary small u-marginRight--10" onClick={this.onCloseReleasesToDiff}>Cancel</button>
          <button
            className="btn primary small blue"
            disabled={checkedReleasesToDiff.length !== 2 || showDiffOverlay}
            onClick={() => {
              if (gitopsEnabled) {
                const { firstHash, secondHash } = this.getDiffCommitHashes();
                if (firstHash && secondHash) {
                  const diffUrl = getGitProviderDiffUrl(downstream.gitops?.uri, downstream.gitops?.provider, firstHash, secondHash);
                  window.open(diffUrl, '_blank');
                }
              } else {
                const { firstSequence, secondSequence } = this.getDiffSequences();
                this.setState({ showDiffOverlay: true, firstSequence, secondSequence });
              }
            }}
          >
            Diff releases
          </button>
        </div>
        :
        <div className="flex-auto flex alignItems--center" onClick={this.onSelectReleasesToDiff}>
          <span className="icon diffReleasesIcon"></span>
          <span className="u-fontSize--small u-fontWeight--medium u-color--royalBlue u-cursor--pointer u-marginLeft--5">Diff versions</span>
        </div>
    );
  }

  handleSelectReleasesToDiff = (selectedRelease, isChecked) => {
    if (isChecked) {
      this.setState({
        checkedReleasesToDiff: [{ ...selectedRelease, isChecked }].concat(this.state.checkedReleasesToDiff).slice(0, 2)
      })
    } else {
      this.setState({
        checkedReleasesToDiff: this.state.checkedReleasesToDiff.filter(release => release.parentSequence !== selectedRelease.parentSequence)
      })
    }
  }

  displayTooltip = (key, value) => {
    return () => {
      this.setState({
        [`${key}Hovered`]: value,
      });
    };
  }

  getDiffSequences = () => {
    let firstSequence = 0, secondSequence = 0;

    const { checkedReleasesToDiff } = this.state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort((r1, r2) => r1.parentSequence - r2.parentSequence);
      firstSequence = checkedReleasesToDiff[0].parentSequence;
      secondSequence = checkedReleasesToDiff[1].parentSequence;
    }

    return {
      firstSequence,
      secondSequence
    }
  }

  getDiffCommitHashes = () => {
    let firstCommitUrl = "", secondCommitUrl = "";

    const { checkedReleasesToDiff } = this.state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort((r1, r2) => r1.parentSequence - r2.parentSequence);
      firstCommitUrl = checkedReleasesToDiff[0].commitUrl;
      secondCommitUrl = checkedReleasesToDiff[1].commitUrl;
    }

    return {
      firstHash: getCommitHashFromUrl(firstCommitUrl),
      secondHash: getCommitHashFromUrl(secondCommitUrl)
    }
  }

  yamlErrorsDetails = (downstream, version) => {
    const pendingVersion = downstream?.pendingVersions?.find(v => v.sequence === version?.sequence);
    const pastVersion = downstream?.pastVersions?.find(v => v.sequence === version?.sequence);

    if (downstream?.currentVersion?.sequence === version?.sequence) {
      return downstream?.currentVersion?.yamlErrors ? downstream?.currentVersion?.yamlErrors : false;
    } else if (pendingVersion?.yamlErrors) {
      return pendingVersion?.yamlErrors;
    } else if (pastVersion?.yamlErrors) {
      return pastVersion?.yamlErrors;
    } else {
      return false;
    }
  }

  toggleShowDetailsModal = (yamlErrorDetails, selectedSequence) => {
    this.setState({ displayShowDetailsModal: !this.state.displayShowDetailsModal, deployView: false, yamlErrorDetails, selectedSequence });
  }

  getCurrentVersionStatus = (version) => {
    if (version?.status === "deployed" || version?.status === "merged" || version?.status === "pending") {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center u-marginTop--8"> <span className="icon checkmark-icon u-marginRight--5" /> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
    } else if (version?.status === "failed") {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--red u-fontWeight--medium flex alignItems--center u-marginTop--8"> <span className="icon error-small u-marginRight--5" /> Failed <span className="u-marginLeft--5 replicated-link u-fontSize--small" onClick={() => this.handleViewLogs(version, true)}> See details </span></span>
    } else if (version?.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium u-marginTop--8">
          <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
        </span>);
    } else {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center u-marginTop--8"> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
    }
  }

  render() {
    const {
      app,
      match,
      isBundleUploading,
      makingCurrentVersionErrMsg
    } = this.props;

    const {
      viewReleaseNotes,
      showLogsModal,
      selectedTab,
      logs,
      logsLoading,
      showDeployWarningModal,
      showSkipModal,
      downstreamReleaseNotes,
      selectedDiffReleases,
      checkedReleasesToDiff,
      checkingForUpdates,
      checkingUpdateMessage,
      checkingForUpdateError,
      errorCheckingUpdate,
      airgapUploadError,
      showDiffOverlay,
      firstSequence,
      secondSequence,
      uploadingAirgapFile,
      uploadProgress,
      uploadSize,
      uploadResuming,
      noUpdateAvailiableText,
      showUpdateCheckerModal,
      loadingVersionHistory,
      versionHistory,
      errorTitle,
      errorMsg,
      displayErrorModal,
    } = this.state;

    if (!app) {
      return null;
    }

    let checkingUpdateText = checkingUpdateMessage;
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

    let checkingUpdateTextShort = checkingUpdateText;
    if (!checkingForUpdateError && checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    // only render loader if there is no app yet to avoid flickering
    if (loadingVersionHistory && !versionHistory?.length) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const errorText = checkingUpdateMessage ? checkingUpdateMessage : "Error checking for updates, please try again";
    let updateText;
    if (airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">{airgapUploadError}</p>;
    } else if (uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          total={uploadSize}
          progress={uploadProgress}
          resuming={uploadResuming}
          onProgressError={this.onProgressError}
          smallSize={true}
        />
      );
    } else if (isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          unkownProgress={true}
          onProgressError={this.onProgressError}
          smallSize={true}
        />);
    } else if (errorCheckingUpdate) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">{errorText}</p>
    } else if (checkingForUpdates) {
      updateText = <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium">{checkingUpdateTextShort}</p>
    } else if (app.lastUpdateCheckAt && !noUpdateAvailiableText) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--silverSand u-fontWeight--medium">Last checked {dayjs(app.lastUpdateCheckAt).fromNow()}</p>;
    } else if (!app.lastUpdateCheckat) {
      updateText = null;
    }

    let noUpdateAvailiableMsg;
    if (noUpdateAvailiableText) {
      noUpdateAvailiableMsg = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">{noUpdateAvailiableText}</p>
    } else {
      noUpdateAvailiableMsg = null;
    }

    const showAirgapUI = app.isAirgap && !isBundleUploading;
    const showOnlineUI = !app.isAirgap && !checkingForUpdates;
    const downstream = app.downstreams.length && app.downstreams[0];
    const gitopsEnabled = downstream.gitops?.enabled;
    const currentDownstreamVersion = downstream?.currentVersion;

    // This is kinda hacky. This finds the equivalent downstream version because the midstream
    // version type does not contain metadata like version label or release notes.
    const currentMidstreamVersion = versionHistory.find(version => version.parentSequence === app.currentVersion.sequence) || app.currentVersion;
    const pendingVersions = downstream?.pendingVersions;


    return (
      <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{`${app.name} Version History`}</title>
        </Helmet>
        <div className="flex flex-auto alignItems--center justifyContent--center u-marginTop--10 u-marginBottom--30">
          <div className="upstream-version-box-wrapper flex flex1">
            <div className="flex flex1">
              {app.iconUri &&
                <div className="flex-auto u-marginRight--10">
                  <div className="watch-icon" style={{ backgroundImage: `url(${app.iconUri})` }}></div>
                </div>
              }
              <div className="flex1 flex-column">
                <p className="u-fontSize--small u-fontWeight--bold u-lineHeight--normal u-color--tuna"> Current version </p>
                <div className="flex alignItems--center u-marginTop--5">
                  <p className="u-fontSize--header2 u-fontWeight--bold u-color--tuna"> {currentDownstreamVersion ? currentDownstreamVersion.versionLabel : "---"}</p>
                  <p className="u-fontSize--small u-lineHeight--normal u-color--tundora u-fontWeight--medium u-marginLeft--10"> {currentDownstreamVersion ? `Sequence ${currentDownstreamVersion?.sequence}` : null}</p>
                </div>
                {currentDownstreamVersion?.deployedAt ? <p className="u-fontSize--small u-lineHeight--normal u-color--silverSand u-fontWeight--medium u-marginTop--5">{`${dayjs(currentDownstreamVersion.deployedAt).format("MMMM D, YYYY  @ hh:mm a")}`}</p> : null}
                {currentDownstreamVersion && this.getCurrentVersionStatus(currentDownstreamVersion)}
                {currentDownstreamVersion ?
                  <div className="flex alignItems--center u-marginTop--8 u-marginTop--8">
                    {currentDownstreamVersion?.releaseNotes &&
                      <div>
                        <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => this.showDownstreamReleaseNotes(currentDownstreamVersion?.releaseNotes)} data-tip="View release notes" />
                        <ReactTooltip effect="solid" className="replicated-tooltip" />
                      </div>}
                    <div>
                      <Link to={`/app/${match.params.slug}/downstreams/${app.downstreams[0].cluster?.slug}/version-history/preflight/${currentDownstreamVersion?.sequence}`}
                        className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer"
                        data-tip="View preflight checks" />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    <div>
                      <span className="icon deployLogs--icon u-marginRight--10 u-cursor--pointer" onClick={() => this.handleViewLogs(currentDownstreamVersion)} data-tip="View deploy logs" />
                      <ReactTooltip effect="solid" className="replicated-tooltip" />
                    </div>
                    {app.isConfigurable &&
                      <div>
                        <Link to={`/app/${match.params.slug}/config`} className="icon config--icon u-cursor--pointer" data-tip="Edit config" />
                        <ReactTooltip effect="solid" className="replicated-tooltip" />
                      </div>}
                  </div> : null}
              </div>
            </div>
            {!app.cluster &&
              <div className={`flex flex1 justifyContent--center ${checkingForUpdates && !isBundleUploading && "alignItems--center"}`}>
                {checkingForUpdates && !isBundleUploading
                  ? <Loader size="32" />
                  : showAirgapUI ?
                    <MountAware className="flex alignItems--center" id="bundle-dropzone" onMount={el => this.airgapUploader.assignElement(el)}>
                      <span className="btn secondary blue">Upload new version</span>
                    </MountAware>
                    : showOnlineUI ?
                      <div className="flex1 flex-column">
                        {pendingVersions?.length > 0 ?
                          <div className="flex flex-column">
                            <p className="u-fontSize--small u-lineHeight--normal u-color--selectiveYellow u-fontWeight--bold">New version available</p>
                            <div className="flex flex-column u-marginTop--5">
                              <div className="flex flex1 alignItems--center">
                                <span className="u-fontSize--larger u-lineHeight--medium u-fontWeight--bold u-color--tundora">{pendingVersions[0]?.versionLabel}</span>
                                <span className="u-fontSize--small u-lineHeight--normal u-fontWeight--medium u-color--tundora u-marginLeft--5"> Sequence {pendingVersions[0]?.sequence}</span>
                              </div>
                              <div className="flex flex1 alignItems--center">
                                {pendingVersions[0]?.createdOn || pendingVersions[0].upstreamReleasedAt ? 
                                <p className="u-fontSize--small u-lineHeight--normal u-fontWeight--medium u-color--dustyGray">Released <span className="u-fontWeight--bold">{pendingVersions[0].upstreamReleasedAt ? dayjs(pendingVersions[0]?.upstreamReleasedAt).format("MMMM D, YYYY") : dayjs(pendingVersions[0]?.createdOn).format("MMMM D, YYYY")}</span></p> 
                                : null}
                                {pendingVersions[0]?.releaseNotes ? <span className="release-notes-link u-fontSize--small u-fontWeight--medium u-marginLeft--5 flex alignItems--center" onClick={() => this.showDownstreamReleaseNotes(pendingVersions[0]?.releaseNotes)}><span className="icon releaseNotes-small--icon u-marginRight--5" />Release notes</span> : null}
                              </div>
                            </div>
                          </div>
                          : <p className="u-fontSize--small u-fontWeight--bold u-lineHeight--normal u-color--dustyGray"> No new version available </p>}
                        <div className="flex alignItems--center u-marginTop--10">
                          <button className="btn primary blue" onClick={this.onCheckForUpdates}>Check for update</button>
                          <span className="icon settings-small-icon u-marginLeft--5 u-cursor--pointer" onClick={this.showUpdateCheckerModal} data-tip="Configure automatic update checks"></span>
                          <ReactTooltip effect="solid" className="replicated-tooltip" />
                        </div>
                        {updateText}
                        {noUpdateAvailiableMsg}
                      </div>
                      : null
                }
                {!showOnlineUI && updateText}
                {!showOnlineUI && noUpdateAvailiableMsg}
              </div>
            }
          </div>
        </div>
        {checkingForUpdateError &&
          <div className="flex-column flex-auto u-marginBottom--30">
            <div className="checking-update-error-wrapper">
              <p className="u-color--chestnut u-fontSize--normal u-lineHeight--normal">{checkingUpdateTextShort}</p>
            </div>
          </div>
        }
        <div className="flex-column flex1">
          <div className="flex flex1">
            <div className="flex1 flex-column alignItems--center">
              {makingCurrentVersionErrMsg &&
                <div className="ErrorWrapper flex justifyContent--center">
                  <div className="icon redWarningIcon u-marginRight--10" />
                  <div>
                    <p className="title">Failed to deploy version</p>
                    <p className="err">{makingCurrentVersionErrMsg}</p>
                  </div>
                </div>}

              <div className="TableDiff--Wrapper flex-column flex1">
                <div className={`flex-column flex1 ${showDiffOverlay ? "u-visibility--hidden" : ""}`}>
                  <div className="flex justifyContent--spaceBetween u-borderBottom--gray darker u-paddingBottom--10">
                    <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-lineHeight--normal">All versions</p>
                    {versionHistory.length > 1 && this.renderDiffBtn()}
                  </div>
                  {/* Downstream version history */}
                  {versionHistory.length >= 1 ? versionHistory.map((version) => {
                    const isChecked = !!checkedReleasesToDiff.find(diffRelease => diffRelease.parentSequence === version.parentSequence);
                    const isNew = secondsAgo(version.createdOn) < 10;
                    const nothingToCommit = gitopsEnabled && !version.commitUrl;
                    const yamlErrorsDetails = this.yamlErrorsDetails(downstream, version);
                    return (
                      <div
                        key={version.sequence}
                        className={classNames(`VersionHistoryDeploymentRow ${version.status} flex flex-auto`, { "overlay": selectedDiffReleases, "disabled": nothingToCommit, "selected": (isChecked && !nothingToCommit), "is-new": isNew })}
                        onClick={() => selectedDiffReleases && !nothingToCommit && this.handleSelectReleasesToDiff(version, !isChecked)}
                      >
                        {selectedDiffReleases && <div className={classNames("checkbox u-marginRight--20", { "checked": (isChecked && !nothingToCommit) }, { "disabled": nothingToCommit })} />}
                        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 u-paddingRight--20`}>
                          <div className="flex alignItems--center">
                            <p className="u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-color--tuna">{version.versionLabel || version.title}</p>
                            <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--tundora u-marginLeft--5" style={{ marginTop: "2px" }}>Sequence {version.sequence}</p>
                          </div>
                          <div className="flex alignItems--center u-marginTop--10"></div>
                          <div className="flex flex1 u-marginTop--15 alignItems--center">
                            <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium">Released <span className="u-fontWeight--bold">{version.upstreamReleasedAt ? dayjs(version.upstreamReleasedAt).format("MMMM D, YYYY") : dayjs(version.createdOn).format("MMMM D, YYYY")}</span></p>
                            {version.releaseNotes ?
                              <p className="release-notes-link u-fontSize--small u-lineHeight--normal u-marginLeft--5 flex alignItems--center" onClick={() => this.showDownstreamReleaseNotes(version.releaseNotes)}> <span className="icon releaseNotes-small--icon u-marginRight--5" />Release notes</p> : null}
                          </div>
                        </div>
                        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1`}>
                          <div className="flex flex-column">
                            <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">{version.source}</p>
                            <div className="flex alignItems--center u-fontSize--small u-marginTop--10 u-color--dustyGray">
                              {this.renderSourceAndDiff(version)}
                              {yamlErrorsDetails && this.renderYamlErrors(yamlErrorsDetails, version)}
                            </div>
                          </div>
                          <div className="flex flex1 alignItems--flexEnd"> {gitopsEnabled ? this.renderViewPreflights(version) : this.renderVersionStatus(version)}</div>
                        </div>
                        <div className={`${nothingToCommit && selectedDiffReleases && "u-opacity--half"} flex-column flex1 alignItems--flexEnd`}>
                          <div>
                            {this.renderVersionAction(version, nothingToCommit && selectedDiffReleases)}
                          </div>
                          <p className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium u-marginTop--15">Deployed: <span className="u-fontWeight--bold">{version.deployedAt ? dayjs(version.deployedAt).format("MMMM D, YYYY @ hh:mm a") : "N/A"}</span></p>
                        </div>
                      </div>
                    );
                  }) :
                    <div className="flex-column flex1 alignItems--center justifyContent--center">
                      <p className="u-fontSize--large u-fontWeight--bold u-color--tuna">No versions have been deployed.</p>
                    </div>
                  }
                </div>

                {/* Diff overlay */}
                {showDiffOverlay &&
                  <div className="DiffOverlay">
                    <DownstreamWatchVersionDiff
                      slug={match.params.slug}
                      firstSequence={firstSequence}
                      secondSequence={secondSequence}
                      onBackClick={this.hideDiffOverlay}
                      app={this.props.app}
                    />
                  </div>
                }
              </div>

            </div>
          </div>
        </div>

        <Modal
          isOpen={viewReleaseNotes}
          onRequestClose={this.hideReleaseNotes}
          contentLabel="Release Notes"
          ariaHideApp={false}
          className="Modal LargeSize"
        >
          <div className="flex-column">
            <MarkdownRenderer>
              {currentMidstreamVersion?.releaseNotes || "No release notes for this version"}
            </MarkdownRenderer>
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={this.hideReleaseNotes}>Close</button>
          </div>
        </Modal>

        {showLogsModal &&
          <ShowLogsModal
            showLogsModal={showLogsModal}
            hideLogsModal={this.hideLogsModal}
            viewLogsErrMsg={this.state.viewLogsErrMsg}
            logs={logs}
            selectedTab={selectedTab}
            logsLoading={logsLoading}
            renderLogsTabs={this.renderLogsTabs()}
          />}

        <Modal
          isOpen={showDeployWarningModal}
          onRequestClose={this.hideDeployWarningModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Skip preflight checks"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
              Preflight checks for this version are currently failing. Are you sure you want to make this the current version?
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={this.onForceDeployClick}
                type="button"
                className="btn blue primary"
              >
                Deploy this version
              </button>
              <button
                onClick={this.hideDeployWarningModal}
                type="button"
                className="btn secondary u-marginLeft--20"
              >
                Cancel
              </button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={showSkipModal}
          onRequestClose={this.hideSkipModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Skip preflight checks"
          ariaHideApp={false}
          className="Modal SkipModal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
              Preflight checks have not finished yet. Are you sure you want to deploy this version?
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={this.onForceDeployClick}
                type="button"
                className="btn blue primary">
                Deploy this version
              </button>
              <button type="button" onClick={this.hideSkipModal} className="btn secondary u-marginLeft--20">Cancel</button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={!!downstreamReleaseNotes}
          onRequestClose={this.hideDownstreamReleaseNotes}
          contentLabel="Release Notes"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="flex-column">
            <MarkdownRenderer>
              {downstreamReleaseNotes || ""}
            </MarkdownRenderer>
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={this.hideDownstreamReleaseNotes}>Close</button>
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
            <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--10">Unable to generate a file diff for release</p>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">The release with the <span className="u-fontWeight--bold">Upstream {this.state.releaseWithErr.title}, Sequence {this.state.releaseWithErr.sequence}</span> was unable to generate a files diff because the following error:</p>
            <div className="error-block-wrapper u-marginBottom--30 flex flex1">
              <span className="u-color--chestnut">{this.state.releaseWithErr.diffSummaryError}</span>
            </div>
            <div className="flex u-marginBottom--10">
              <button className="btn primary" onClick={this.toggleDiffErrModal}>Ok, got it!</button>
            </div>
          </div>
        </Modal>

        {showUpdateCheckerModal &&
          <UpdateCheckerModal
            isOpen={showUpdateCheckerModal}
            onRequestClose={this.hideUpdateCheckerModal}
            updateCheckerSpec={app.updateCheckerSpec}
            appSlug={app.slug}
            gitopsEnabled={gitopsEnabled}
            onUpdateCheckerSpecSubmitted={() => {
              this.hideUpdateCheckerModal();
              this.props.refreshAppData();
            }}
          />
        }
        {this.state.displayShowDetailsModal &&
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
          />}
        {errorMsg &&
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
          />}
      </div>
    );
  }
}

export default withRouter(AppVersionHistory);
