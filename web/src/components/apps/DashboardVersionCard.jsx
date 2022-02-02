import React from "react";
import { Link, withRouter } from "react-router-dom";
import ReactTooltip from "react-tooltip"

import dayjs from "dayjs";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import Modal from "react-modal";
import AirgapUploadProgress from "../AirgapUploadProgress";
import Loader from "../shared/Loader";
import MountAware from "../shared/MountAware";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
import classNames from "classnames";

import { Utilities, getPreflightResultState, getDeployErrorTab, isAwaitingResults, secondsAgo } from "@src/utilities/utilities";

import "../../scss/components/watches/DashboardCard.scss";

class DashboardVersionCard extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      selectedAction: "",
      logsLoading: false,
      logs: null,
      selectedTab: null,
      displayConfirmDeploymentModal: false,
      showDiffModal: false,
      showNoChangesModal: false,
      releaseWithNoChanges: {},
      releaseWithErr: {},
      showDiffErrModal: false
    }
    this.cardTitleText = React.createRef();
  }

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.links !== lastProps.links && this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
    if (this.props.location.search !== lastProps.location.search && this.props.location.search !== "") {
      const splitSearch = this.props.location.search.split("/");
      this.setState({
        showDiffModal: true,
        firstSequence: splitSearch[1],
        secondSequence: splitSearch[2]
      });
    }
  }

  closeViewDiffModal = () => {
    if (this.props.location.search) {
      this.props.history.replace(location.pathname);
    }
    this.setState({ showDiffModal: false });
  }

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false
    });
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

  handleViewLogs = async (version, isFailing) => {
    try {
      const { app } = this.props;
      const clusterId = app.downstreams?.length && app.downstreams[0].cluster?.id;

      this.setState({ logsLoading: true, showLogsModal: true, viewLogsErrMsg: "", versionFailing: false });

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
        this.setState({ logs: response.logs, selectedTab, logsLoading: false, viewLogsErrMsg: "", versionFailing: isFailing });
      } else {
        this.setState({ logsLoading: false, viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}` });
      }
    } catch (err) {
      console.log(err)
      this.setState({ logsLoading: false, viewLogsErrMsg: err ? `Failed to view logs: ${err.message}` : "Something went wrong, please try again." });
    }
  }

  getCurrentVersionStatus = (version) => {
    if (version?.status === "deployed" || version?.status === "merged" || version?.status === "pending") {
      return <span className="status-tag success flex-auto">Currently {version?.status.replace("_", " ")} version</span>
    } else if (version?.status === "failed") {
      return (
        <div className="flex alignItems--center">
          <span className="status-tag failed flex-auto u-marginRight--10">Deploy Failed</span>
          <span className="replicated-link u-fontSize--small" onClick={() => this.handleViewLogs(version, true)}>View deploy logs</span>
        </div>
      );
    } else if (version?.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
          <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
        </span>);
    } else {
      return <span className="status-tag unknown flex-atuo"> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
    }
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

  toggleDiffErrModal = (release) => {
    this.setState({
      showDiffErrModal: !this.state.showDiffErrModal,
      releaseWithErr: !this.state.showDiffErrModal ? release : {}
    });
  }

  toggleNoChangesModal = (version) => {
    this.setState({
      showNoChangesModal: !this.state.showNoChangesModal,
      releaseWithNoChanges: !this.state.showNoChangesModal ? version: {}
    });
  }

  getPreflightState = (version) => {
    let preflightsFailed = false;
    let preflightState = "";
    if (version?.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      preflightState = getPreflightResultState(preflightResult);
      preflightsFailed = preflightState === "fail";
    }
    return {
      preflightsFailed,
      preflightState,
      preflightSkipped: version?.preflightSkipped
    };
  }

  renderCurrentVersion = () => {
    const { currentVersion, app } = this.props;
    const preflightState = this.getPreflightState(currentVersion);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed"
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings"
    }
    return (
      <div className="flex1 flex-column">
        <div className="flex">
          <div className="flex-column">
            <div className="flex alignItems--center u-marginBottom--5">
              <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">{currentVersion.versionLabel || currentVersion.title}</p>
              <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">Sequence {currentVersion.sequence}</p>
            </div>
            <div>{this.getCurrentVersionStatus(currentVersion)}</div>
            <div className="flex alignItems--center u-marginTop--10">
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">{currentVersion.status === "failed" ? "---" :  `${currentVersion.status === "deploying" ? "Deploy started at" : "Deployed"} ${Utilities.dateFormat(currentVersion?.deployedAt, "MM/DD/YY @ hh:mm a z")}`}</p>
            </div>
          </div>
          <div className="flex alignItems--center u-paddingLeft--20">
            <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">{currentVersion.source}</p>
          </div>
          <div className="flex flex1 alignItems--center justifyContent--flexEnd">
            {currentVersion?.releaseNotes &&
              <div>
                <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => this.showDownstreamReleaseNotes(currentVersion?.releaseNotes)} data-tip="View release notes" />
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </div>
            }
            <div>
            {currentVersion.status === "pending_preflight" ?
              <div className="u-marginRight--10 u-position--relative">
                <Loader size="30" />
                <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">Running checks</p>
              </div>
            :
            <div>
              <Link to={`/app/${app?.slug}/downstreams/${app?.downstreams[0].cluster?.slug}/version-history/preflight/${currentVersion?.sequence}`}
                className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer u-position--relative"
                data-tip="View preflight checks">
                  {preflightState.preflightsFailed || preflightState.preflightState === "warn" ?
                  <div>
                    <span className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed ? "preflight-checks-failed-icon" : preflightState.preflightState === "warn" ? "preflight-checks-warn-icon" : ""}`} />
                    <p className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed ? "err" : preflightState.preflightState === "warn" ? "warning" : ""}`}>{checksStatusText}</p>
                  </div>
                  : null}
                </Link>
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
            }
            </div>
            {app?.isConfigurable &&
              <div className="u-marginRight--10">
                <Link to={`/app/${app?.slug}/config/${currentVersion.sequence}`} className="icon configEdit--icon u-cursor--pointer" data-tip="Edit config" />
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </div>
            }
            <div>
              <span className="icon deployLogs--icon u-cursor--pointer" onClick={() => this.handleViewLogs(currentVersion, currentVersion?.status === "failed")} data-tip="View deploy logs" />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
            <div className="flex-column justifyContent--center">
              <button
                className="secondary blue btn u-marginLeft--10"
                disabled={currentVersion.status === "deploying"}
                onClick={() => this.deployVersion(currentVersion)}
              >
                {currentVersion.status === "deploying" ? "Redeploying" : "Redeploy"}
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  getUpdateTypeClassname = (updateType) => {
    if (updateType.includes("Upstream Update")) {
      return "upstream-update";
    }
    if (updateType.includes("Config Change")) {
      return "config-update";
    }
    if (updateType.includes("License Change")) {
      return "license-sync";
    }
    if (updateType.includes("Airgap Install") || updateType.includes("Airgap Update")) {
      return "airgap-install";
    }
    return "online-install";
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
    } else if (version.source === "Online Install") {
      return (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
          <span>Online Install</span>
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
                  <Link className="u-fontSize--small replicated-link u-marginLeft--5" to={`${this.props.location.pathname}?diff/${this.props.currentVersion?.sequence}/${version.parentSequence}`}>View diff</Link>
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

  onForceDeployClick = (continueWithFailedPreflights = false) => {
    this.setState({ showSkipModal: false, showDeployWarningModal: false, displayShowDetailsModal: false });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true, continueWithFailedPreflights);
  }

  showDownstreamReleaseNotes = (releaseNotes) => {
    this.setState({
      showDownstreamReleaseNotes: true,
      downstreamReleaseNotes: releaseNotes
    });
  }
  
  deployButtonStatus = (downstream, version) => {
    const isDeploying = version.status === "deploying";
    const needsConfiguration = version.status === "pending_config";
  
    if (needsConfiguration) {
      return "Configure";
    } else if (downstream?.currentVersion?.sequence == undefined) {
      return "Deploy";
    } else if (isDeploying) {
      return "Deploying";
    } else {
      return "Deploy";
    }
  }
  
  renderVersionAction = (version, nothingToCommit) => {
    const { app } = this.props;
    const downstream = app.downstreams[0];
    if (downstream.gitops?.enabled) {
      if (version.gitDeployable === false) {
        return (<div className={nothingToCommit && "u-opacity--half"}>Nothing to commit</div>);
      }
      if (!version.commitUrl) {
        return (
          <div className="flex flex1 alignItems--center justifyContent--flexEnd">
            <span className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal">No commit URL found</span>
            <span className="icon grayOutlineQuestionMark--icon u-marginLeft--5" data-tip="This version may have been created before Gitops was enabled" />
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </div>
        );
      }
      return (
        <button
          className="btn primary blue"
          onClick={() => window.open(version.commitUrl, "_blank")}
        >
          View
        </button>
      );
    }
  
    const needsConfiguration = version.status === "pending_config";
    const preflightState = this.getPreflightState(version);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed"
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings"
    }
    return (
      <div className="flex flex1 alignItems--center justifyContent--flexEnd">
          {version?.releaseNotes &&
            <div>
              <span className="icon releaseNotes--icon u-marginRight--10 u-cursor--pointer" onClick={() => this.showDownstreamReleaseNotes(version?.releaseNotes)} data-tip="View release notes" />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
          }
          {version.status === "pending_preflight" ?
            <div className="u-marginRight--10 u-position--relative">
              <Loader size="30" />
              <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">Running checks</p>
            </div>
            :
            <div>
              <Link to={`/app/${app?.slug}/downstreams/${app?.downstreams[0].cluster?.slug}/version-history/preflight/${version?.sequence}`}
                className="icon preflightChecks--icon u-marginRight--10 u-cursor--pointer u-position--relative"
                data-tip="View preflight checks">
                  {preflightState.preflightsFailed || preflightState.preflightState === "warn" ?
                  <div>
                    <span className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed ? "preflight-checks-failed-icon" : preflightState.preflightState === "warn" ? "preflight-checks-warn-icon" : ""}`} />
                    <p className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed ? "err" : preflightState.preflightState === "warn" ? "warning" : ""}`}>{checksStatusText}</p>
                  </div>
                  : null}
                </Link>
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
          }
          {app?.isConfigurable &&
            <div>
              <Link to={`/app/${app?.slug}/config/${version.sequence}`} className="icon configEdit--icon u-cursor--pointer" data-tip="Edit config" />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
          }
          <div className="flex-column justifyContent--center">
            <button
              className={classNames("btn u-marginLeft--10", { "secondary blue": needsConfiguration, "primary blue": !needsConfiguration })}
              disabled={version.status === "deploying"}
              onClick={needsConfiguration ? history.push(`/app/${app?.slug}/config/${version.sequence}`) : () => this.deployVersion(version)}
            >
              {this.deployButtonStatus(downstream, version, app)}
            </button>
          </div>
      </div>
    );
  }

  renderVersionAvailable = () => {
    const { app, downstream, checkingForUpdateError, checkingUpdateText, errorCheckingUpdate, isBundleUploading } = this.props;

    const showOnlineUI = !app.isAirgap;
    const nothingToCommit = downstream.gitops?.enabled && !downstream?.pendingVersions[0].commitUrl;
    const downstreamSource = downstream?.pendingVersions[0]?.source;

    let updateText;
    if (showOnlineUI && app.lastUpdateCheckAt) {
      updateText = <p className="u-marginTop--8 u-fontSize--smaller u-textColor--info u-marginTop--8">Last checked <span className="u-fontWeight--bold">{dayjs(app.lastUpdateCheckAt).fromNow()}</span></p>;
    } else if (this.props.airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">Error uploading bundle <span className="u-linkColor u-textDecoration--underlineOnHover" onClick={this.props.viewAirgapUploadError}>See details</span></p>
    } else if (this.props.uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          total={this.props.uploadSize}
          progress={this.props.uploadProgress}
          resuming={this.props.uploadResuming}
          onProgressError={this.props.onProgressError}
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
    } else if (errorCheckingUpdate) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">Error checking for updates, please try again</p>
    } else if (!app.lastUpdateCheckAt) {
      updateText = null;
    }

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 65) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 65) + "...";
    }

    return (
      <div>
        {!checkingForUpdateError && downstream?.pendingVersions?.length > 0 && (!isBundleUploading || !this.props.uploadingAirgapFile) ?
          <div className="flex">
            <div className="flex-column">
              <div className="flex alignItems--center">
                <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">{downstream?.pendingVersions[0].versionLabel || downstream?.pendingVersions[0].title}</p>
                <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">Sequence {downstream?.pendingVersions[0].sequence}</p>
              </div>
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5"> Released {Utilities.dateFormat(downstream?.pendingVersions[0]?.createdOn, "MM/DD/YY @ hh:mm a z")} </p>
              <div className="u-marginTop--5 flex flex-auto alignItems--center">
                {this.renderSourceAndDiff(downstream?.pendingVersions[0])}
              </div>
            </div>
            <div className="flex alignItems--center u-paddingLeft--20">
              <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">{downstreamSource}</p>
            </div>
              {this.renderVersionAction(downstream?.pendingVersions[0], nothingToCommit)}
          </div>
        : null}
        {!showOnlineUI && updateText}
        {app?.isAirgap && this.props.checkingForUpdates && !isBundleUploading ?
          <div className="flex-column justifyContent--center alignItems--center">
            <Loader className="u-marginBottom--10" size="30" />
            <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">{checkingUpdateTextShort}</span>
          </div>
        : null }
        {checkingForUpdateError &&
          <div className={`flex-column flex-auto ${this.props.uploadingAirgapFile || this.props.checkingForUpdates || isBundleUploading ? "u-marginTop--10" : ""}`}>
            <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">Error updating version:</p>
            <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">{checkingUpdateText}</p>
          </div>}
      </div>
    )
  }

  render() {
    const { app, downstream, currentVersion, checkingForUpdates, checkingUpdateText, uploadingAirgapFile, isBundleUploading, airgapUploader } = this.props;
    const { downstreamReleaseNotes } = this.state;
    const versionsToSkip = downstream?.pendingVersions?.length - 1;
    const gitopsEnabled = downstream?.gitops?.enabled;

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }
    const isNew = downstream?.pendingVersions ? secondsAgo(downstream?.pendingVersions[0]?.createdOn) < 10 : false;
    return (
      <div className="flex-column flex1 dashboard-card">
        <div className="flex flex1 justifyContent--spaceBetween alignItems--center u-marginBottom--10">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">Version</p>
          <div className="flex alignItems--center">
            {app?.isAirgap && airgapUploader ?
              <MountAware onMount={el => this.props.airgapUploader?.assignElement(el)}>
                <div className="flex alignItems--center">
                  <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
                  <span className="replicated-link u-fontSize--small u-lineHeight--default">Upload new version</span>
                </div>
              </MountAware>
            :
            <div className="flex alignItems--center">
              {checkingForUpdates && !isBundleUploading ?
                <div className="flex alignItems--center u-marginRight--20">
                  <Loader className="u-marginRight--5" size="15" />
                  <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">{checkingUpdateText === "" ? "Checking for updates" : checkingUpdateTextShort}</span>
                </div>
              : this.props.noUpdatesAvalable ?
                <div className="flex alignItems--center u-marginRight--20">
                  <span className="u-textColor--primary u-fontWeight--medium u-fontSize--small u-lineHeight--default">Already up to date</span>
                </div>
              :
                <div className="flex alignItems--center u-marginRight--20">
                  <span className="icon clickable dashboard-card-check-update-icon u-marginRight--5" />
                  <span className="replicated-link u-fontSize--small" onClick={this.props.onCheckForUpdates}>Check for update</span>
                </div>
              }
              <span className="icon clickable dashboard-card-configure-update-icon u-marginRight--5" />
              <span className="replicated-link u-fontSize--small u-lineHeight--default" onClick={this.props.showAutomaticUpdatesModal}>Configure automatic updates</span>
            </div>
            }
          </div>
        </div>
        {currentVersion?.deployedAt ?
          <div className="LicenseCard-content--wrapper">
            {this.renderCurrentVersion()}
          </div>
        :
          <div className="no-deployed-version u-textAlign--center">
            <p className="u-fontWeight--medium u-fontSize--normal u-textColor--bodyCopy"> No version has been deployed </p>
          </div>
        }
        {downstream?.pendingVersions?.length > 0 || uploadingAirgapFile || isBundleUploading || this.props.checkingForUpdateError || (app?.isAirgap && this.props.checkingForUpdates)  ?
          <div className="u-marginTop--20">
            {uploadingAirgapFile || isBundleUploading || this.props.checkingForUpdateError || (app?.isAirgap && this.props.checkingForUpdates) ? null :
              <p className="u-fontSize--normal u-lineHeight--normal u-textColor--header u-fontWeight--medium">{currentVersion?.deployedAt ? "Latest available version" : "Deploy latest available version"}</p>
            }
            {gitopsEnabled &&
              <div className="gitops-enabled-block u-fontSize--small u-fontWeight--medium flex alignItems--center u-textColor--header u-marginTop--10">
                <span className={`icon gitopsService--${downstream?.gitops?.provider} u-marginRight--10`}/>Gitops is enabled for this application. Versions are tracked {app?.isAirgap ? "at" : "on"}&nbsp;<a target="_blank" rel="noopener noreferrer" href={downstream?.gitops?.uri} className="replicated-link">{app.isAirgap ? downstream?.gitops?.uri : Utilities.toTitleCase(downstream?.gitops?.provider)}</a>
              </div>
            }
            <div className={`LicenseCard-content--wrapper u-marginTop--15 ${isNew && !app?.isAirgap ? "is-new" : ""}`}>
              {this.renderVersionAvailable()}
            </div>
            {versionsToSkip > 0 && <p className="u-fontSize--small u-fontWeight--medium u-textColor--header u-marginTop--10">{versionsToSkip} version{versionsToSkip > 1 && "s"} will be skipped in upgrading to {downstream?.pendingVersions[0]?.versionLabel}.</p>}
          </div>
        : null}
        <div className="u-marginTop--10">
          <Link to={`/app/${this.props.app?.slug}/version-history`} className="replicated-link has-arrow u-fontSize--small">See all versions</Link>
        </div>
        {this.state.showDownstreamReleaseNotes &&
          <Modal
            isOpen={this.state.showDownstreamReleaseNotes}
            onRequestClose={() => this.setState({ showDownstreamReleaseNotes: false })}
            contentLabel="Release Notes"
            ariaHideApp={false}
            className="Modal MediumSize"
          >
            <div className="flex-column">
              <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
                {downstreamReleaseNotes || "No release notes for this version"}
              </MarkdownRenderer>
            </div>
            <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
              <button className="btn primary" onClick={() => this.setState({ showDownstreamReleaseNotes: false })}>Close</button>
            </div>
          </Modal>
        }
        {this.state.showLogsModal &&
          <ShowLogsModal
            showLogsModal={this.state.showLogsModal}
            hideLogsModal={this.hideLogsModal}
            viewLogsErrMsg={this.state.viewLogsErrMsg}
            versionFailing={this.state.versionFailing}
            troubleshootUrl={`/app/${this.props.app?.slug}/troubleshoot`}
            logs={this.state.logs}
            selectedTab={this.state.selectedTab}
            logsLoading={this.state.logsLoading}
            renderLogsTabs={this.renderLogsTabs()}
          />}
          {this.state.showDiffErrModal &&
            <Modal
              isOpen={true}
              onRequestClose={this.toggleDiffErrModal}
              contentLabel="Unable to Get Diff"
              ariaHideApp={false}
              className="Modal MediumSize"
            >
              <div className="Modal-body">
                <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">Unable to generate a file diff for release</p>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">The <span className="u-fontWeight--bold">Upstream {this.state.releaseWithErr.versionLabel}, Sequence {this.state.releaseWithErr.sequence}</span> release was unable to generate a diff because the following error:</p>
                <div className="error-block-wrapper u-marginBottom--30 flex flex1">
                  <span className="u-textColor--error">{this.state.releaseWithErr.diffSummaryError}</span>
                </div>
                <div className="flex u-marginBottom--10">
                  <button className="btn primary" onClick={this.toggleDiffErrModal}>Ok, got it!</button>
                </div>
              </div>
            </Modal>
          }
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
          {this.state.displayConfirmDeploymentModal &&
            <Modal
              isOpen={true}
              onRequestClose={() => this.setState({ displayConfirmDeploymentModal: false, versionToDeploy: null })}
              contentLabel="Confirm deployment"
              ariaHideApp={false}
              className="Modal DefaultSize"
            >
              <div className="Modal-body">
                <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">Deploy {this.state.versionToDeploy?.versionLabel} (Sequence {this.state.versionToDeploy?.sequence})?</p>
                <div className="flex u-paddingTop--10">
                  <button className="btn secondary blue" onClick={() => this.setState({ displayConfirmDeploymentModal: false, versionToDeploy: null })}>Cancel</button>
                  <button className="u-marginLeft--10 btn primary" onClick={() => this.finalizeDeployment(false)}>Yes, deploy</button>
                </div>
              </div>
            </Modal>
          }
          {this.state.showDeployWarningModal &&
          <DeployWarningModal
            showDeployWarningModal={this.state.showDeployWarningModal}
            hideDeployWarningModal={() => this.setState({ showDeployWarningModal: false })}
            onForceDeployClick={this.onForceDeployClick}
          />}
          {this.state.showSkipModal &&
            <SkipPreflightsModal
              showSkipModal={true}
              hideSkipModal={() => this.setState({ showSkipModal: false })}
              onForceDeployClick={this.onForceDeployClick} 
            />
          }
          {this.state.showDiffModal && 
            <Modal
              isOpen={true}
              onRequestClose={this.closeViewDiffModal}
              contentLabel="Release Diff Modal"
              ariaHideApp={false}
              className="Modal DiffViewerModal"
            >
              <div className="DiffOverlay">
                <DownstreamWatchVersionDiff
                  slug={this.props.match.params.slug}
                  firstSequence={this.state.firstSequence}
                  secondSequence={this.state.secondSequence}
                  hideBackButton={true}
                  onBackClick={this.closeViewDiffModal}
                  app={this.props.app}
                />
              </div>
              <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
                <button className="btn primary" onClick={this.closeViewDiffModal}>Close</button>
              </div>
            </Modal>
          }
      </div>
    );
  }
}

export default withRouter(DashboardVersionCard)
