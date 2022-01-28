import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import Modal from "react-modal";
import find from "lodash/find";

import Loader from "../shared/Loader";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import ErrorModal from "../modals/ErrorModal";
import AppVersionHistoryRowClassic from "@src/components/apps/AppVersionHistoryRowClassic";
import AppVersionHistoryHeader from "./AppVersionHistoryHeader";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
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

class AppVersionHistoryClassic extends Component {
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
    showAutomaticUpdatesModal: false,
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
    displayConfirmDeploymentModal: false,
    confirmType: "",
    isSkipPreflights: false
  }

  componentDidMount() {
    if (this.props.app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig()
    }

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

  getAirgapConfig = async () => {
    const { app } = this.props;
    const configUrl = `${process.env.API_ENDPOINT}/app/${app.slug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "Authorization": Utilities.getToken(),
        }
      });
      if (res.ok) {
        const response = await res.json();
        simultaneousUploads = response.simultaneousUploads;
      }
    } catch {
      // no-op
    }

    this.setState({
      airgapUploader: new AirgapUploader(true, app.slug, this.onDropBundle, simultaneousUploads),
    });
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
      const res = await fetch(`${process.env.API_ENDPOINT}/app/${appSlug}/versions`, {
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

  hideAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: false
    });
  }

  showAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: true
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

  renderSourceAndDiff = version => {
    const { app } = this.props;
    const downstream = app.downstreams?.length && app.downstreams[0];
    const diffSummary = this.getVersionDiffSummary(version);
    const hasDiffSummaryError = version.diffSummaryError && version.diffSummaryError.length > 0;

    if (hasDiffSummaryError) {
      return (
        <div className="flex flex1 alignItems--center">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">Cannot generate diff <span className="replicated-link" onClick={() => this.toggleDiffErrModal(version)}>Why?</span></span>
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

  deployVersion = (version, force = false, continueWithFailedPreflights = false) => {
    const { app } = this.props;
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
          versionToDeploy: version,
          yamlErrorDetails
        });
        return;
      }
      if (version.status === "pending_preflight") {
        this.setState({
          showSkipModal: true,
          versionToDeploy: version,
          isSkipPreflights: true
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
      // prompt to make sure user wants to deploy
      this.setState({
        displayConfirmDeploymentModal: true,
        versionToDeploy: version,
        confirmType: "deploy"
      });
      return;
    } else { // force deploy is set to true so finalize the deployment
      this.finalizeDeployment(continueWithFailedPreflights);
    }
  }

  finalizeDeployment = async (continueWithFailedPreflights) => {
    const { match, updateCallback } = this.props;
    const { versionToDeploy, isSkipPreflights } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await this.props.makeCurrentVersion(match.params.slug, versionToDeploy, isSkipPreflights, continueWithFailedPreflights);
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  }

  redeployVersion = (version, isRollback = false) => {
    const { app } = this.props;
    const clusterSlug = app.downstreams?.length && app.downstreams[0].cluster?.slug;
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
  }

  finalizeRedeployment = async () => {
    const { match, updateCallback } = this.props;
    const { versionToDeploy } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "", });
    await this.props.redeployVersion(match.params.slug, versionToDeploy);
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  }

  onForceDeployClick = (continueWithFailedPreflights = false) => {
    this.setState({ showSkipModal: false, showDeployWarningModal: false, displayShowDetailsModal: false });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true, continueWithFailedPreflights);
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

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`, {
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
          selectedTab = Utilities.getDeployErrorTab(response.logs);
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
    const { app } = this.props;

    return new Promise((resolve, reject) => {
      fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/task/updatedownload`, {
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

            if (response.status === "failed") {
              if (response.currentMessage.includes("Upgrade KOTS to version")) {
                this.setState({
                  checkingForUpdates: false,
                  checkingForUpdateError: true,
                  checkingUpdateMessage: response.currentMessage,
                  incompatibleKotsVersionError: true
                });
              }
            } else {
              this.setState({
                checkingForUpdates: false,
                checkingUpdateMessage: response.currentMessage,
                checkingForUpdateError: response.status === "failed"
              });
            }

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

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
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
            checkingUpdateMessage: text
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
    this.state.airgapUploader.upload(params, this.onUploadProgress, this.onUploadError, this.onUploadComplete);
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
                  window.open(diffUrl, "_blank");
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
          <span className="icon clickable diffReleasesIcon"></span>
          <span className="u-fontSize--small u-fontWeight--medium u-linkColor u-cursor--pointer u-marginLeft--5">Diff versions</span>
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

  render() {
    const {
      app,
      match,
      isBundleUploading,
      makingCurrentVersionErrMsg,
      redeployVersionErrMsg
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
      showAutomaticUpdatesModal,
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
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">{airgapUploadError}</p>;
    } else if (uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
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
          appSlug={app.slug}
          unkownProgress={true}
          onProgressError={this.onProgressError}
          smallSize={true}
        />);
    } else if (errorCheckingUpdate || checkingForUpdateError) {
      if (checkingForUpdateError && this.state.incompatibleKotsVersionError) {
        updateText = <span className="u-marginTop--10 u-fontSize--small u-lineHeight--normal u-textColor--error u-fontWeight--medium flex alignItems--center"> <span className="icon error-small u-marginRight--5" /> Incompatible KOTS Version <span className="u-marginLeft--5 replicated-link u-fontSize--small" onClick={() => this.props.toggleDisplayRequiredKotsUpdateModal(checkingUpdateMessage)}> See details </span></span>
      } else {
        updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">{errorText}</p>
      }
    } else if (checkingForUpdates) {
      updateText = <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium">{checkingUpdateTextShort}</p>
    } else if (app.lastUpdateCheckAt && !noUpdateAvailiableText) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--info u-fontWeight--medium">Last checked {dayjs(app.lastUpdateCheckAt).fromNow()}</p>;
    } else if (!app.lastUpdateCheckat) {
      updateText = null;
    }

    let noUpdateAvailiableMsg;
    if (noUpdateAvailiableText) {
      noUpdateAvailiableMsg = <p className="u-marginTop--10 u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium">{noUpdateAvailiableText}</p>
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
    const isPastVersion = find(downstream?.pastVersions, { sequence: this.state.versionToDeploy?.sequence });

    return (
      <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{`${app.name} Version History`}</title>
        </Helmet>
        {gitopsEnabled &&
          <div className="edit-files-banner gitops-enabled-banner u-fontSize--small u-fontWeight--normal u-textColor--secondary flex alignItems--center justifyContent--center">
            <span className={`icon gitopsService--${downstream.gitops?.provider} u-marginRight--10`}/>Gitops is enabled for this application. Versions are tracked {app.isAirgap ? "at" : "on"}&nbsp;<a target="_blank" rel="noopener noreferrer" href={downstream.gitops?.uri} className="replicated-link">{app.isAirgap ? downstream.gitops?.uri : Utilities.toTitleCase(downstream.gitops?.provider)}</a>
          </div>
        }
        <AppVersionHistoryHeader
          app={app}
          slug={this.props.match.params.slug}
          currentDownstreamVersion={currentDownstreamVersion}
          showDownstreamReleaseNotes={this.showDownstreamReleaseNotes}
          handleViewLogs={this.handleViewLogs}
          checkingForUpdates={checkingForUpdates}
          isBundleUploading={isBundleUploading}
          airgapUploader={this.state.airgapUploader}
          pendingVersions={pendingVersions}
          showOnlineUI={showOnlineUI}
          showAirgapUI={showAirgapUI}
          noUpdateAvailiableMsg={noUpdateAvailiableMsg}
          updateText={updateText}
          onCheckForUpdates={this.onCheckForUpdates}
          showAutomaticUpdatesModal={this.showAutomaticUpdatesModal}
        />
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
              {redeployVersionErrMsg &&
                <div className="ErrorWrapper flex justifyContent--center">
                  <div className="icon redWarningIcon u-marginRight--10" />
                  <div>
                    <p className="title">Failed to redeploy version</p>
                    <p className="err">{redeployVersionErrMsg}</p>
                  </div>
                </div>
              }

              <div className="TableDiff--Wrapper classic flex-column flex1">
                <div className={`flex-column flex1 ${showDiffOverlay ? "u-visibility--hidden" : ""}`}>
                  <div className="flex justifyContent--spaceBetween u-borderBottom--gray darker u-paddingBottom--10">
                    <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-lineHeight--normal">All versions</p>
                    {versionHistory.length > 1 && this.renderDiffBtn()}
                  </div>
                  {/* Downstream version history */}
                  {versionHistory.length >= 1 ? versionHistory.map((version) => {
                    const isChecked = !!checkedReleasesToDiff.find(diffRelease => diffRelease.parentSequence === version.parentSequence);
                    const isNew = secondsAgo(version.createdOn) < 10;
                    const nothingToCommit = gitopsEnabled && !version.commitUrl;
                    const yamlErrorsDetails = this.yamlErrorsDetails(downstream, version);
                    return (
                      <AppVersionHistoryRowClassic
                        key={version.sequence}
                        app={this.props.app}
                        match={this.props.match}
                        history={this.props.history}
                        version={version}
                        latestVersion={versionHistory[0]}
                        selectedDiffReleases={selectedDiffReleases}
                        nothingToCommit={nothingToCommit}
                        isChecked={isChecked}
                        isNew={isNew}
                        showDownstreamReleaseNotes={this.showDownstreamReleaseNotes}
                        renderSourceAndDiff={this.renderSourceAndDiff}
                        yamlErrorsDetails={yamlErrorsDetails}
                        toggleShowDetailsModal={this.toggleShowDetailsModal}
                        gitopsEnabled={gitopsEnabled}
                        deployVersion={this.deployVersion}
                        handleViewLogs={this.handleViewLogs}
                        handleSelectReleasesToDiff={this.handleSelectReleasesToDiff}
                        redeployVersion={this.redeployVersion}
                      />
                    );
                  }) :
                    <div className="flex-column flex1 alignItems--center justifyContent--center">
                      <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary">No versions have been deployed.</p>
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
            <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
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

        {showDeployWarningModal &&
          <DeployWarningModal
            showDeployWarningModal={showDeployWarningModal}
            hideDeployWarningModal={this.hideDeployWarningModal}
            onForceDeployClick={this.onForceDeployClick}
            showAutoDeployWarning={isPastVersion && this.props.app?.semverAutoDeploy !== "disabled"}
            confirmType={this.state.confirmType}
          />}

        {showSkipModal &&
          <SkipPreflightsModal
            showSkipModal={showSkipModal}
            hideSkipModal={this.hideSkipModal}
            onForceDeployClick={this.onForceDeployClick} 
            />
            }

        <Modal
          isOpen={!!downstreamReleaseNotes}
          onRequestClose={this.hideDownstreamReleaseNotes}
          contentLabel="Release Notes"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="flex-column">
            <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
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
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">Unable to generate a file diff for release</p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">The release with the <span className="u-fontWeight--bold">Upstream {this.state.releaseWithErr.title}, Sequence {this.state.releaseWithErr.sequence}</span> was unable to generate a files diff because the following error:</p>
            <div className="error-block-wrapper u-marginBottom--30 flex flex1">
              <span className="u-textColor--error">{this.state.releaseWithErr.diffSummaryError}</span>
            </div>
            <div className="flex u-marginBottom--10">
              <button className="btn primary" onClick={this.toggleDiffErrModal}>Ok, got it!</button>
            </div>
          </div>
        </Modal>

        {this.state.displayConfirmDeploymentModal &&
          <Modal
            isOpen={true}
            onRequestClose={() => this.setState({ displayConfirmDeploymentModal: false, confirmType: "", versionToDeploy: null })}
            contentLabel="Confirm deployment"
            ariaHideApp={false}
            className="Modal DefaultSize"
          >
            <div className="Modal-body">
              <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">{this.state.confirmType === "rollback" ? "Rollback to" : this.state.confirmType === "redeploy" ? "Redeploy" : "Deploy"} {this.state.versionToDeploy?.versionLabel} (Sequence {this.state.versionToDeploy?.sequence})?</p>
              {isPastVersion && this.props.app?.semverAutoDeploy !== "disabled" ? 
                <div className="info-box">
                  <span className="u-fontSize--small u-textColor--header u-lineHeight--normal u-fontWeight--medium">You have automatic deploys enabled. {this.state.confirmType === "rollback" ? "Rolling back to" : this.state.confirmType === "redeploy" ? "Redeploying" : "Deploying"} this version will disable automatic deploys. You can turn it back on after this version finishes deployment.</span>
                </div>
              : null}
              <div className="flex u-paddingTop--10">
                <button className="btn secondary blue" onClick={() => this.setState({ displayConfirmDeploymentModal: false, confirmType: "", versionToDeploy: null })}>Cancel</button>
                <button className="u-marginLeft--10 btn primary" onClick={this.state.confirmType === "redeploy" ? this.finalizeRedeployment : () => this.finalizeDeployment(false)}>Yes, {this.state.confirmType === "rollback" ? "rollback" : this.state.confirmType === "redeploy" ? "redeploy" : "deploy"}</button>
              </div>
            </div>
          </Modal>
        }

        {showAutomaticUpdatesModal &&
          <AutomaticUpdatesModal
            isOpen={showAutomaticUpdatesModal}
            onRequestClose={this.hideAutomaticUpdatesModal}
            updateCheckerSpec={app.updateCheckerSpec}
            semverAutoDeploy={app.semverAutoDeploy}
            appSlug={app.slug}
            gitopsEnabled={gitopsEnabled}
            onAutomaticUpdatesConfigured={() => {
              this.hideAutomaticUpdatesModal();
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
            appSlug={this.props.match.params.slug}
          />}
      </div>
    );
  }
}

export default withRouter(AppVersionHistoryClassic);
