import React, { Component } from "react";
import { withRouter, Link } from "react-router-dom";
import Helmet from "react-helmet";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import Modal from "react-modal";
import find from "lodash/find";
import get from "lodash/get";
import MountAware from "../shared/MountAware";
import Loader from "../shared/Loader";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import AirgapUploadProgress from "../AirgapUploadProgress";
import ErrorModal from "../modals/ErrorModal";
import AppVersionHistoryRow from "@src/components/apps/AppVersionHistoryRow";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
import { Utilities, isAwaitingResults, secondsAgo, getPreflightResultState, getGitProviderDiffUrl, getCommitHashFromUrl } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import { AirgapUploader } from "../../utilities/airgapUploader";
import ReactTooltip from "react-tooltip"

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
    this.fetchKotsDownstreamHistory();

    if (this.props.app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig()
    }

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

  toggleDiffErrModal = (release) => {
    this.setState({
      showDiffErrModal: !this.state.showDiffErrModal,
      releaseWithErr: !this.state.showDiffErrModal ? release : {}
    })
  }

  toggleAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: !this.state.showAutomaticUpdatesModal
    });
  }

  toggleNoChangesModal = (version) => {
    this.setState({
      showNoChangesModal: !this.state.showNoChangesModal,
      releaseWithNoChanges: !this.state.showNoChangesModal ? version: {}
    });
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
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">Unable to generate diff <span className="replicated-link" onClick={() => this.toggleDiffErrModal(version)}>Why?</span></span>
        </div>
      );
    } else {
      return (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
          {diffSummary ?
            (diffSummary.filesChanged > 0 ?
              <div className="DiffSummary u-marginRight--10">
                <span className="files">{diffSummary.filesChanged} files changed </span>
                {!downstream.gitops?.enabled &&
                  <span className="u-fontSize--small replicated-link u-marginLeft--5" onClick={() => this.setState({ showDiffOverlay: true, firstSequence: version.parentSequence - 1, secondSequence: version.parentSequence})}>View diff</span>
                }
              </div>
              :
              <div className="DiffSummary">
                <span className="files">No changes to show. <span className="replicated-link" onClick={() => this.toggleNoChangesModal(version)}>Why?</span></span>
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

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed"
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
        <div className="flex u-marginLeft--20">
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
        <div className="flex-auto flex alignItems--center u-marginLeft--20" onClick={this.onSelectReleasesToDiff}>
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

  renderAirgapVersionUploading = () => {
    const { app, isBundleUploading } = this.props;

    let updateText;
    if (this.state.airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">Error uploading bundle <span className="u-linkColor u-textDecoration--underlineOnHover" onClick={this.props.viewAirgapUploadError}>See details</span></p>
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
    } else if (isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          unkownProgress={true}
          onProgressError={undefined}
          smallSize={true}
        />);
    } else if (this.state.errorCheckingUpdate) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">Error checking for updates, please try again</p>
    } else if (!app.lastUpdateCheckAt) {
      updateText = null;
    }

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

    return (
      <div>
        {updateText}
        {app?.isAirgap && this.state.checkingForUpdates && !isBundleUploading ?
          <div className="flex-column justifyContent--center alignItems--center">
            <Loader className="u-marginBottom--10" size="30" />
            <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">{checkingUpdateText}</span>
          </div>
        : null }
        {this.state.checkingForUpdateError &&
          <div className={`flex-column flex-auto ${this.state.uploadingAirgapFile || this.state.checkingForUpdates || isBundleUploading ? "u-marginTop--10" : ""}`}>
            <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">Error updating version:</p>
            <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">{this.state.checkingUpdateMessage}</p>
          </div>}
      </div>
    )
  }

  render() {
    const {
      app,
      match,
      makingCurrentVersionErrMsg,
      redeployVersionErrMsg,
      isBundleUploading
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
      uploadingAirgapFile,
      checkingUpdateMessage
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

    const downstream = app.downstreams.length && app.downstreams[0];
    const gitopsEnabled = downstream.gitops?.enabled;
    const currentDownstreamVersion = downstream?.currentVersion;

    // This is kinda hacky. This finds the equivalent downstream version because the midstream
    // version type does not contain metadata like version label or release notes.
    const currentMidstreamVersion = versionHistory.find(version => version.parentSequence === app.currentVersion.sequence) || app.currentVersion;
    const otherAvailableVersions = versionHistory.filter((i, idx) => idx !== 0);
    const isPastVersion = find(downstream?.pastVersions, { sequence: this.state.versionToDeploy?.sequence });
  
    let checkingUpdateTextShort = checkingUpdateMessage;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

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
        <div className="flex-column flex1">
          <div className="flex flex1 justifyContent--center">
            <div className="flex1 flex AppVersionHistory">
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

              <div className="flex-column flex1" style={{ maxWidth: "370px", marginRight: "20px" }}>
                <div className="TableDiff--Wrapper currentVersionCard--wrapper">
                  <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">{currentDownstreamVersion?.versionLabel ? "Currently deployed version" : "No current version deployed"}</p>
                  <div className="currentVersion--wrapper u-marginTop--10">
                    <div className="flex flex1">
                      {app?.iconUri &&
                        <div className="flex-auto u-marginRight--10">
                          <div className="watch-icon" style={{ backgroundImage: `url(${app?.iconUri})` }}></div>
                        </div>
                      }
                      <div className="flex1 flex-column">
                        <div className="flex alignItems--center u-marginTop--5">
                          <p className="u-fontSize--header2 u-fontWeight--bold u-textColor--primary"> {currentDownstreamVersion ? currentDownstreamVersion.versionLabel : "---"}</p>
                          <p className="u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10"> {currentDownstreamVersion ? `Sequence ${currentDownstreamVersion?.sequence}` : null}</p>
                        </div>
                        {currentDownstreamVersion?.deployedAt ? <p className="u-fontSize--small u-lineHeight--normal u-textColor--info u-fontWeight--medium u-marginTop--10">{currentDownstreamVersion?.status === "deploying" ? "Deploy started at" : "Deployed"} {Utilities.dateFormat(currentDownstreamVersion.deployedAt, "MM/DD/YY @ hh:mm a z")}</p> : null}
                        {currentDownstreamVersion ?
                          <div className="flex alignItems--center u-marginTop--10">
                            {currentDownstreamVersion?.releaseNotes &&
                              <div>
                                <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => this.showDownstreamReleaseNotes(currentDownstreamVersion?.releaseNotes)} data-tip="View release notes" />
                                <ReactTooltip effect="solid" className="replicated-tooltip" />
                              </div>}
                            <div>
                              <Link to={`/app/${app?.slug}/downstreams/${app.downstreams[0].cluster?.slug}/version-history/preflight/${currentDownstreamVersion?.sequence}`}
                                className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer"
                                data-tip="View preflight checks" />
                              <ReactTooltip effect="solid" className="replicated-tooltip" />
                            </div>
                            <div>
                              <span className="icon deployLogs--icon u-cursor--pointer" onClick={() => this.handleViewLogs(currentDownstreamVersion, currentDownstreamVersion?.status === "failed")} data-tip="View deploy logs" />
                              <ReactTooltip effect="solid" className="replicated-tooltip" />
                              {currentDownstreamVersion?.status === "failed" ? <span className="icon version-row-preflight-status-icon preflight-checks-failed-icon logs" /> : null}
                            </div>
                            {app.isConfigurable &&
                              <div>
                                <Link to={`/app/${app?.slug}/config/${app?.downstreams[0]?.currentVersion?.parentSequence}`} className="icon configEdit--icon u-cursor--pointer" data-tip="Edit config" />
                                <ReactTooltip effect="solid" className="replicated-tooltip" />
                              </div>}
                          </div> : null}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className={`TableDiff--Wrapper flex-column flex1 alignSelf--start ${gitopsEnabled ? "gitops-enabled" : ""}`}>
                <div className={`flex-column flex1 version ${showDiffOverlay ? "u-visibility--hidden" : ""}`}>
                {versionHistory.length >= 1 ?
                  <div>
                    <div>
                      <div className="flex justifyContent--spaceBetween u-marginBottom--15">
                        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header">Latest available version</p>
                        <div className="flex alignItems--center">
                          <div className="flex alignItems--center">
                            {app?.isAirgap && airgapUploader ?
                              <MountAware onMount={el => airgapUploader?.assignElement(el)}>
                                <div className="flex alignItems--center">
                                  <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
                                  <span className="replicated-link u-fontSize--small u-lineHeight--default">Upload new version</span>
                                </div>
                              </MountAware>
                            :
                            <div className="flex alignItems--center">
                              {checkingForUpdates && !this.props.isBundleUploading ?
                                <div className="flex alignItems--center u-marginRight--20">
                                  <Loader className="u-marginRight--5" size="15" />
                                  <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">{checkingUpdateMessage === "" ? "Checking for updates" : checkingUpdateTextShort}</span>
                                </div>
                              :
                                <div className="flex alignItems--center u-marginRight--20">
                                  <span className="icon clickable dashboard-card-check-update-icon u-marginRight--5" />
                                  <span className="replicated-link u-fontSize--small" onClick={this.onCheckForUpdates}>Check for update</span>
                                </div>
                              }
                              <span className="icon clickable dashboard-card-configure-update-icon u-marginRight--5" />
                              <span className="replicated-link u-fontSize--small" onClick={this.toggleAutomaticUpdatesModal}>Configure automatic updates</span>
                            </div>
                            }
                          </div>
                          {versionHistory.length > 1 && this.renderDiffBtn()}
                        </div>
                      </div>
                      <AppVersionHistoryRow
                        key={versionHistory[0].sequence}
                        app={this.props.app}
                        match={this.props.match}
                        history={this.props.history}
                        version={versionHistory[0]}
                        latestVersion={versionHistory[0]}
                        selectedDiffReleases={selectedDiffReleases}
                        nothingToCommit={gitopsEnabled && !versionHistory[0].commitUrl}
                        isChecked={!!checkedReleasesToDiff.find(diffRelease => diffRelease.parentSequence === versionHistory[0].parentSequence)}
                        isNew={secondsAgo(versionHistory[0].createdOn) < 10}
                        showDownstreamReleaseNotes={this.showDownstreamReleaseNotes}
                        renderSourceAndDiff={this.renderSourceAndDiff}
                        yamlErrorsDetails={this.yamlErrorsDetails(downstream, versionHistory[0])}
                        toggleShowDetailsModal={this.toggleShowDetailsModal}
                        gitopsEnabled={gitopsEnabled}
                        deployVersion={this.deployVersion}
                        handleViewLogs={this.handleViewLogs}
                        handleSelectReleasesToDiff={this.handleSelectReleasesToDiff}
                        redeployVersion={this.redeployVersion}
                      />
                    </div>
                    {uploadingAirgapFile || isBundleUploading || this.state.checkingForUpdateError || (app?.isAirgap && this.state.checkingForUpdates)  ?
                      <div className="u-marginTop--20 u-marginBottom--20">
                        {this.renderAirgapVersionUploading()}
                      </div>
                    : null}
                    {otherAvailableVersions.length > 0 &&
                      <div className="flex u-marginBottom--15 u-marginTop--30">
                        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy">Other available versions</p>
                      </div>
                    }
                    {otherAvailableVersions?.map((version) => {
                      const isChecked = !!checkedReleasesToDiff.find(diffRelease => diffRelease.parentSequence === version.parentSequence);
                      const isNew = secondsAgo(version.createdOn) < 10;
                      const nothingToCommit = gitopsEnabled && !version.commitUrl;
                      const yamlErrorsDetails = this.yamlErrorsDetails(downstream, version);
                      return (
                        <AppVersionHistoryRow
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
                    })}
                  </div>
                :
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
          {this.state.showNoChangesModal &&
            <Modal
              isOpen={true}
              onRequestClose={this.toggleNoChangesModal}
              contentLabel="No Changes"
              ariaHideApp={false}
              className="Modal DefaultSize"
            >
              <div className="Modal-body">
                <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">No changes to show</p>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">The <span className="u-fontWeight--bold">Upstream {this.state.releaseWithNoChanges.versionLabel}, Sequence {this.state.releaseWithNoChanges.sequence}</span> release was unable to generate a diff because the changes made do not affect any manifests that will be deployed. Only changes affecting the application manifest will be included in a diff.</p>
                <div className="flex u-paddingTop--10">
                  <button className="btn primary" onClick={this.toggleNoChangesModal}>Ok, got it!</button>
                </div>
              </div>
            </Modal>
          }
          {this.state.showAutomaticUpdatesModal &&
            <AutomaticUpdatesModal
              isOpen={this.state.showAutomaticUpdatesModal}
              onRequestClose={this.toggleAutomaticUpdatesModal}
              updateCheckerSpec={app?.updateCheckerSpec}
              semverAutoDeploy={app?.semverAutoDeploy}
              appSlug={app?.slug}
              gitopsEnabled={downstream?.gitops?.enabled}
              onAutomaticUpdatesConfigured={() => {
                this.toggleAutomaticUpdatesModal();
                this.props.updateCallback();
              }}
            />
          }
      </div>
    );
  }
}

export default withRouter(AppVersionHistory);
