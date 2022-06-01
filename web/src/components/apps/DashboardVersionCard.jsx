import React from "react";
import { Link, withRouter } from "react-router-dom";
import ReactTooltip from "react-tooltip"
import DashboardGitOpsCard from "./DashboardGitOpsCard";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import Modal from "react-modal";
import AirgapUploadProgress from "../AirgapUploadProgress";
import Loader from "../shared/Loader";
import MountAware from "../shared/MountAware";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
import classNames from "classnames";

import { Utilities, getPreflightResultState, secondsAgo } from "@src/utilities/utilities";
import { Repeater } from "@src/utilities/repeater";

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
      displayShowDetailsModal: false,
      yamlErrorDetails: [],
      deployView: false,
      selectedSequence: -1,
      showDiffModal: false,
      showNoChangesModal: false,
      releaseWithNoChanges: {},
      releaseWithErr: {},
      showDiffErrModal: false,
      versionDownloadStatuses: {},
      kotsUpdateChecker: new Repeater(),
      kotsUpdateRunning: false,
      kotsUpdateStatus: undefined,
      kotsUpdateMessage: undefined,
      kotsUpdateError: undefined,
      latestDeployableVersion: null,
      numOfSkippedVersions: 0,
      numOfRemainingVersions: 0,
      latestDeployableVersionErrMsg: "",
    }
    this.cardTitleText = React.createRef();

    // moving this out of the state because new repeater instances were getting created
    // and it doesn't really affect the UI
    this.versionDownloadStatusJobs = {}
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
    if (lastProps.downstream !== this.props.downstream && this.props.isHelmManaged !== true) {
      this.getLatestDeployableVersion();
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
      const clusterId = app.downstream.cluster?.id;

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
      console.log(err);
      this.setState({ logsLoading: false, viewLogsErrMsg: err ? `Failed to view logs: ${err.message}` : "Something went wrong, please try again." });
    }
  }

  getLatestDeployableVersion = async () => {
    try {
      const { app } = this.props;

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/next-app-version`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });

      if (!res.ok) {
        const response = await res.json();
        this.setState({
          latestDeployableVersionErrMsg: response.error,
        });
        return;
      }

      const response = await res.json();
      this.setState({
        latestDeployableVersion: response.latestDeployableVersion,
        numOfSkippedVersions: response.numOfSkippedVersions,
        numOfRemainingVersions: response.numOfRemainingVersions,
        latestDeployableVersionErrMsg: "",
      });
    } catch (err) {
      console.log(err);
      this.setState({
        latestDeployableVersionErrMsg: err ? `Failed to get next deployable version: ${err.message}` : "Something went wrong, please try again.",
      });
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

  toggleShowDetailsModal = (yamlErrorDetails, selectedSequence) => {
    this.setState({ displayShowDetailsModal: !this.state.displayShowDetailsModal, deployView: false, yamlErrorDetails, selectedSequence });
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

  renderReleaseNotes = version => {
    if (!version?.releaseNotes) {
      return null;
    }
    return (
      <div>
        <span className="icon releaseNotes--icon u-cursor--pointer" onClick={() => this.showReleaseNotes(version?.releaseNotes)} data-tip="View release notes" />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  }

  renderPreflights = version => {
    if (!version) {
      return null;
    }
    if (version.status === "pending_download") {
      return null;
    }
    if (version.status === "pending_config") {
      return null;
    }

    const { app } = this.props;

    const preflightState = this.getPreflightState(version);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed"
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings"
    }

    return (
      <div>
        {version.status === "pending_preflight" ?
          <div className="u-marginLeft--10 u-position--relative">
            <Loader size="30" />
            <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">Running checks</p>
          </div>
        :
        <div>
          <Link to={`/app/${app?.slug}/downstreams/${app?.downstream.cluster?.slug}/version-history/preflight/${version?.sequence}`}
            className="icon preflightChecks--icon u-marginLeft--10 u-cursor--pointer u-position--relative"
            data-tip="View preflight checks">
              {preflightState.preflightsFailed || preflightState.preflightState === "warn" ?
              <div>
                <span className={`icon version-row-preflight-status-icon ${preflightState.preflightsFailed ? "preflight-checks-failed-icon" : preflightState.preflightState === "warn" ? "preflight-checks-warn-icon" : ""}`} />
                <p className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium ${preflightState.preflightsFailed ? "err" : preflightState.preflightState === "warn" ? "warning" : ""}`}>{checksStatusText}</p>
              </div>
              : null}
          </Link>
          <ReactTooltip effect="solid" className="replicated-tooltip" />
        </div>}
      </div>
    );
  }

  renderEditConfigIcon = (app, version) => {
    if (!app?.isConfigurable) {
      return null;
    }
    if (!version) {
      return null;
    }
    if (version.status === "pending_download") {
      return null;
    }
    if (version.status === "pending_config") {
      // action button will already be set to "Configure", no need to show edit config icon as well
      return null;
    }
    return (
      <div className="u-marginLeft--10">
        <Link to={`/app/${app?.slug}/config/${version.sequence}`} className="icon configEdit--icon u-cursor--pointer" data-tip="Edit config" />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  }

  renderCurrentVersion = () => {
    const { currentVersion, app } = this.props;
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
            {this.renderReleaseNotes(currentVersion)}
            {this.renderPreflights(currentVersion)}
            {this.renderEditConfigIcon(app, currentVersion)}
            <div className="u-marginLeft--10">
              <span className="icon deployLogs--icon u-cursor--pointer" onClick={() => this.handleViewLogs(currentVersion, currentVersion?.status === "failed")} data-tip="View deploy logs" />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </div>
            {currentVersion.status === "deploying" ? null : 
              <div className="flex-column justifyContent--center u-marginLeft--10">
                <button
                  className="secondary blue btn"
                  disabled={currentVersion.status === "deploying"}
                  onClick={() => this.deployVersion(currentVersion, false, false, true)}
                >
                  Redeploy
                </button>
              </div>
            }
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

  renderDiff = version => {
    const { app } = this.props;
    const downstream = app?.downstream;
    const diffSummary = this.getVersionDiffSummary(version);
    const hasDiffSummaryError = version.diffSummaryError && version.diffSummaryError.length > 0;

    if (hasDiffSummaryError) {
      return (
        <div className="flex flex1 alignItems--center u-marginTop--5">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">Unable to generate diff <span className="replicated-link" onClick={() => this.toggleDiffErrModal(version)}>Why?</span></span>
        </div>
      );
    } else if (diffSummary) {
      return (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5">
          {diffSummary.filesChanged > 0 ?
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
          }
        </div>
      );
    }
  }

  renderYamlErrors = (version) => {
    if (!version.yamlErrors) {
      return null;
    }
    return (
      <div className="flex alignItems--center u-marginTop--5">
        <span className="icon error-small" />
        <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-textColor--error">{version.yamlErrors?.length} Invalid file{version.yamlErrors?.length !== 1 ? "s" : ""} </span>
        <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => this.toggleShowDetailsModal(version.yamlErrors, version.sequence)}> See details </span>
      </div>
    )
  }

  deployVersion = (version, force = false, continueWithFailedPreflights = false, redeploy = false) => {
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
        isRedeploy: redeploy
      });
      return;
    } else { // force deploy is set to true so finalize the deployment
      this.finalizeDeployment(continueWithFailedPreflights, redeploy);
    }
  }

  finalizeDeployment = async (continueWithFailedPreflights, redeploy) => {
    const { match } = this.props;
    const { versionToDeploy, isSkipPreflights } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    if (redeploy) {
      await this.props.redeployVersion(match.params.slug, versionToDeploy);
    }
    await this.props.makeCurrentVersion(match.params.slug, versionToDeploy, isSkipPreflights, continueWithFailedPreflights);
    this.setState({ versionToDeploy: null, isRedeploy: false });

    if (this.props.refetchData) {
      this.props.refetchData();
    }
  }
  

  onForceDeployClick = (continueWithFailedPreflights = false) => {
    this.setState({ showSkipModal: false, showDeployWarningModal: false, displayShowDetailsModal: false });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true, continueWithFailedPreflights);
  }

  showReleaseNotes = (releaseNotes) => {
    this.setState({
      showReleaseNotes: true,
      releaseNotes: releaseNotes
    });
  }

  hideReleaseNotes = () => {
    this.setState({
      showReleaseNotes: false,
      releaseNotes: ""
    });
  }
  
  actionButtonStatus = version => {
    const isDeploying = version.status === "deploying";
    const isDownloading = this.state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion;
    const isPendingDownload = version.status === "pending_download";
    const needsConfiguration = version.status === "pending_config";
    const canUpdateKots = version.needsKotsUpgrade && !this.props.adminConsoleMetadata?.isAirgap && !this.props.adminConsoleMetadata?.isKurl;

    if (isDeploying) {
      return "Deploying";
    } else if (isDownloading) {
      return "Downloading";
    } else if (isPendingDownload) {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Download";
      }
    } if (needsConfiguration) {
      return "Configure";
    } else {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    }
  }

  renderGitopsVersionAction = version => {
    const { app } = this.props;
    const downstream = app?.downstream;
    const nothingToCommit = downstream?.gitops?.enabled && !version?.commitUrl;

    if (version.status === "pending_download") {
      const isDownloading = this.state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion;
      return (
        <div className="flex flex1 alignItems--center justifyContent--flexEnd">
          {this.renderReleaseNotes(version)}
          <button
            className="btn secondary blue u-marginLeft--10"
            disabled={isDownloading}
            onClick={() => this.downloadVersion(version)}
          >
            {isDownloading ? "Downloading" : "Download"}
          </button>
        </div>
      );
    }
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
      <div className="flex flex1 alignItems--center justifyContent--flexEnd">
        <button
          className="btn primary blue"
          onClick={() => window.open(version.commitUrl, "_blank")}
        >
          View
        </button>
      </div>
    );
  }
  
  renderVersionAction = version => {
    const { app } = this.props;
    const downstream = app?.downstream;

    if (downstream.gitops?.enabled) {
      return this.renderGitopsVersionAction(version);
    }

    const needsConfiguration = version.status === "pending_config";
    const isPendingDownload = version.status === "pending_download";
    const isSecondaryActionBtn = needsConfiguration || isPendingDownload;

    return (
      <div className="flex flex1 alignItems--center justifyContent--flexEnd">
        {this.renderReleaseNotes(version)}
        {this.renderPreflights(version)}
        {this.renderEditConfigIcon(app, version)}
        <div className="flex-column justifyContent--center u-marginLeft--10">
          <button
            className={classNames("btn", { "secondary blue": isSecondaryActionBtn, "primary blue": !isSecondaryActionBtn })}
            disabled={this.isActionButtonDisabled(version)}
            onClick={() => {
              if (needsConfiguration) {
                this.props.history.push(`/app/${app?.slug}/config/${version.sequence}`);
                return;
              }
              if (version.needsKotsUpgrade) {
                this.upgradeAdminConsole(version);
                return;
              }
              if (isPendingDownload) {
                this.downloadVersion(version);
                return;
              }
              this.deployVersion(version);
            }}
          >
            <span
              key={version.nonDeployableCause}
              data-tip-disable={!this.isActionButtonDisabled(version)}
              data-tip={version.nonDeployableCause}
              data-for="disable-deployment-tooltip"
            >
              {this.actionButtonStatus(version)}
            </span>
          </button>
          <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
        </div>
      </div>
    );
  }

  isActionButtonDisabled = version => {
    if (this.state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion) {
      return true;
    }
    if (version.status === "deploying") {
      return true;
    }
    if (version.status === "pending_config") {
      return false;
    }
    if (version.status === "pending_download") {
      return false;
    }
    return !version.isDeployable;
  }

  renderVersionDownloadStatus = version => {
    const { versionDownloadStatuses } = this.state;

    if (!versionDownloadStatuses.hasOwnProperty(version.sequence)) {
      // user hasn't tried to re-download the version yet, show last known download status if exists
      if (version.downloadStatus) {
        return (
          <div className="flex alignItems--center justifyContent--flexEnd">
            <span className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${version.downloadStatus.status === "failed" ? "u-textColor--error" : ""}`}>
              {version.downloadStatus.message}
            </span>
          </div>
        );
      }
      return null
    }

    const status = versionDownloadStatuses[version.sequence];

    return (
      <div className="flex alignItems--center justifyContent--flexEnd">
        {status.downloadingVersion && <Loader className="u-marginRight--5" size="15" />}
        <span className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${status.downloadingVersionError ? "u-textColor--error" : ""}`}>
          {status.downloadingVersionMessage ? status.downloadingVersionMessage : status.downloadingVersion ? "Downloading" : ""}
        </span>
      </div>
    );
  }

  upgradeAdminConsole = version => {
    const { app } = this.props;

    this.setState({
      displayKotsUpdateModal: true,
      kotsUpdateRunning: true,
      kotsUpdateStatus: undefined,
      kotsUpdateMessage: undefined,
      kotsUpdateError: undefined,
    });

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/update-console`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
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
          kotsUpdateError: err?.message || "Something went wrong, please try again.",
        });
      });
  }

  getKotsUpdateStatus = () => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
      fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/task/update-admin-console`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "GET",
    })
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
  }

  downloadVersion = version => {
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
        }
      },
    });
    
    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/download`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          this.setState({
            versionDownloadStatuses: {
              ...this.state.versionDownloadStatuses,
              [version.sequence]: {
                downloadingVersion: false,
                downloadingVersionMessage: response.error,
                downloadingVersionError: true
              }
            }
          });
          return;
        }
        this.versionDownloadStatusJobs[version.sequence].start(() => this.updateVersionDownloadStatus(version), 1000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          versionDownloadStatuses: {
            ...this.state.versionDownloadStatuses,
            [version.sequence]: {
              downloadingVersion: false,
              downloadingVersionMessage: err?.message || "Something went wrong, please try again.",
              downloadingVersionError: true
            }
          }
        });
      });
  }

  updateVersionDownloadStatus = version => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
      fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/sequence/${version?.parentSequence}/task/updatedownload`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
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
                  downloadingVersionError: response.status === "failed"
                }
              }
            });

            if (this.props.refetchData) {
              this.props.refetchData();
            }
            if (this.props.downloadCallback) {
              this.props.downloadCallback();
            }
          } else {
            this.setState({
              versionDownloadStatuses: {
                ...this.state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: true,
                  downloadingVersionMessage: response.currentMessage,
                }
              }
            });
          }
          resolve();
        }).catch((err) => {
          console.log("failed to get version download status", err);
          reject();
        });
    });
  }

  shouldRenderUpdateProgress = () => {
    if (this.props.uploadingAirgapFile) {
      return true;
    }
    if (this.props.isBundleUploading) {
      return true;
    }
    if (this.props.checkingForUpdateError) {
      return true;
    }
    if (this.props.airgapUploadError) {
      return true;
    }
    if (this.props.app?.isAirgap && this.props.checkingForUpdates) {
      return true;
    }
    return false;
  }

  renderUpdateProgress = () => {
    const {
      app,
      checkingForUpdateError,
      checkingUpdateText,
      isBundleUploading,
      uploadingAirgapFile,
      checkingForUpdates,
      airgapUploadError
    } = this.props;

    let updateText;
    if (airgapUploadError) {
      updateText = (
        <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--small u-textColor--error u-fontWeight--medium">
          Error uploading bundle
          <span className="u-linkColor u-textDecoration--underlineOnHover u-marginLeft--5" onClick={this.props.viewAirgapUploadError}>
            See details
          </span>
        </p>
      );
    } else if (checkingForUpdateError) {
      updateText = (
        <div className="flex-column flex-auto u-marginTop--10">
          <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">Error updating version:</p>
          <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">{checkingUpdateText}</p>
        </div>
      );
    } else if (uploadingAirgapFile) {
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
    } else if (checkingForUpdates) {
      let shortText = checkingUpdateText;
      if (shortText && shortText.length > 65) {
        shortText = shortText.slice(0, 65) + "...";
      }
      updateText = (
        <div className="flex-column justifyContent--center alignItems--center">
          <Loader className="u-marginBottom--10" size="30" />
          <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">{shortText}</span>
        </div>
      );
    }

    return (
      <div className="VersionCard-content--wrapper u-marginTop--15">
        {updateText}
      </div>
    );
  }

  renderBottomSection = () => {
    if (this.shouldRenderUpdateProgress()) {
      return this.renderUpdateProgress();
    }

    if (this.state.latestDeployableVersionErrMsg) {
      return (
        <div className="error-block-wrapper u-marginTop--20 u-marginBottom--10 flex flex1">
          <span className="u-textColor--error">{this.state.latestDeployableVersionErrMsg}</span>
        </div>
      );
    }

    const latestDeployableVersion = this.state.latestDeployableVersion;
    if (!latestDeployableVersion) {
      return null;
    }

    const app = this.props.app;
    const downstream = this.props.downstream;
    const downstreamSource = latestDeployableVersion?.source;
    const gitopsEnabled = downstream?.gitops?.enabled;
    const isNew = secondsAgo(latestDeployableVersion?.createdOn) < 10;

    return (
      <div className="u-marginTop--20">
        <p className="u-fontSize--normal u-lineHeight--normal u-textColor--header u-fontWeight--medium">New version available</p>
        {gitopsEnabled &&
          <div className="gitops-enabled-block u-fontSize--small u-fontWeight--medium flex alignItems--center u-textColor--header u-marginTop--10">
            <span className={`icon gitopsService--${downstream?.gitops?.provider} u-marginRight--10`}/>Gitops is enabled for this application. Versions are tracked {app?.isAirgap ? "at" : "on"}&nbsp;<a target="_blank" rel="noopener noreferrer" href={downstream?.gitops?.uri} className="replicated-link">{app.isAirgap ? downstream?.gitops?.uri : Utilities.toTitleCase(downstream?.gitops?.provider)}</a>
          </div>
        }
        <div className="VersionCard-content--wrapper u-marginTop--15">
          <div className={`flex ${isNew && !app?.isAirgap ? "is-new" : ""}`}>
            <div className="flex-column">
              <div className="flex alignItems--center">
                <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium u-textColor--primary">{latestDeployableVersion.versionLabel || latestDeployableVersion.title}</p>
                <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">Sequence {latestDeployableVersion.sequence}</p>
                {latestDeployableVersion.isRequired &&
                  <span className="status-tag required u-marginLeft--10"> Required </span>
                }
              </div>
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5"> Released {Utilities.dateFormat(latestDeployableVersion?.createdOn, "MM/DD/YY @ hh:mm a z")} </p>
              {this.renderDiff(latestDeployableVersion)}
              {this.renderYamlErrors(latestDeployableVersion)}
            </div>
            <div className="flex alignItems--center u-paddingLeft--20">
              <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">{downstreamSource}</p>
            </div>
            {this.renderVersionAction(latestDeployableVersion)}
          </div>
          {this.renderVersionDownloadStatus(latestDeployableVersion)}
        </div>
        {(this.state.numOfSkippedVersions > 0 || this.state.numOfRemainingVersions > 0) && (
          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--more u-textColor--header u-marginTop--10">
            {this.state.numOfSkippedVersions > 0 ? `${this.state.numOfSkippedVersions} version${this.state.numOfSkippedVersions > 1 ? "s" : ""} will be skipped in upgrading to ${latestDeployableVersion.versionLabel}. ` : ""}
            {this.state.numOfRemainingVersions > 0 ? "Additional versions are available after you deploy this required version." : ""}
          </p>
        )}
      </div>
    );
  }

  render() {
    const { app, currentVersion, checkingForUpdates, checkingUpdateText, isBundleUploading, airgapUploader } = this.props;

    const gitopsEnabled = this.props.downstream?.gitops?.enabled;

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    const renderKotsUpgradeStatus = this.state.kotsUpdateStatus && !this.state.kotsUpdateMessage;
    let shortKotsUpdateMessage = this.state.kotsUpdateMessage;
    if (shortKotsUpdateMessage && shortKotsUpdateMessage.length > 60) {
      shortKotsUpdateMessage = shortKotsUpdateMessage.substring(0, 60) + "...";
    }

    if (gitopsEnabled) {
      return (
        <DashboardGitOpsCard 
          gitops={this.props.downstream?.gitops}
          isAirgap={app?.isAirgap}
          appSlug={app?.slug}
          checkingForUpdates={checkingForUpdates}
          latestConfigSequence={app?.downstream?.pendingVersions[0]?.parentSequence}
          isBundleUploading={isBundleUploading}
          checkingUpdateText={checkingUpdateText}
          checkingUpdateTextShort={checkingUpdateTextShort}
          noUpdatesAvalable={this.props.noUpdatesAvalable}
          onCheckForUpdates={this.props.onCheckForUpdates}
          showAutomaticUpdatesModal={this.props.showAutomaticUpdatesModal}
        />
      );
    }

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
          <div className="VersionCard-content--wrapper">
            {this.renderCurrentVersion()}
          </div>
        :
          <div className="no-deployed-version u-textAlign--center">
            <p className="u-fontWeight--medium u-fontSize--normal u-textColor--bodyCopy"> No version has been deployed </p>
          </div>
        }
        {this.renderBottomSection()}
        <div className="u-marginTop--10">
          <Link to={`/app/${this.props.app?.slug}/version-history`} className="replicated-link has-arrow u-fontSize--small">See all versions</Link>
        </div>
        {this.state.showReleaseNotes &&
          <Modal
            isOpen={this.state.showReleaseNotes}
            onRequestClose={this.hideReleaseNotes}
            contentLabel="Release Notes"
            ariaHideApp={false}
            className="Modal MediumSize"
          >
            <div className="flex-column">
              <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
                {this.state.releaseNotes || "No release notes for this version"}
              </MarkdownRenderer>
            </div>
            <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
              <button className="btn primary" onClick={this.hideReleaseNotes}>Close</button>
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
              onRequestClose={() => this.setState({ displayConfirmDeploymentModal: false, versionToDeploy: null, isRedeploy: false })}
              contentLabel="Confirm deployment"
              ariaHideApp={false}
              className="Modal DefaultSize"
            >
              <div className="Modal-body">
                <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">{this.state.isRedeploy ? "Redeploy" : "Deploy"} {this.state.versionToDeploy?.versionLabel} (Sequence {this.state.versionToDeploy?.sequence})?</p>
                <div className="flex u-paddingTop--10">
                  <button className="btn secondary blue" onClick={() => this.setState({ displayConfirmDeploymentModal: false, versionToDeploy: null, isRedeploy: false })}>Cancel</button>
                  <button className="u-marginLeft--10 btn primary" onClick={() => this.finalizeDeployment(false, this.state.isRedeploy)}>Yes, {this.state.isRedeploy ? "Redeploy" : "Deploy"}</button>
                </div>
              </div>
            </Modal>
          }
          {this.state.displayKotsUpdateModal &&
            <Modal
              isOpen={true}
              onRequestClose={() => this.setState({ displayKotsUpdateModal: false })}
              contentLabel="Upgrade is in progress"
              ariaHideApp={false}
              className="Modal DefaultSize"
            >
              <div className="Modal-body u-textAlign--center">
                <div className="flex-column justifyContent--center alignItems--center">
                  <p className="u-fontSize--large u-textColor--primary u-lineHeight--bold u-marginBottom--10">Upgrading...</p>
                  <Loader className="flex alignItems--center" size="32" />
                  {renderKotsUpgradeStatus ?  
                    <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">{ this.state.kotsUpdateStatus }</p>
                    : null
                  }
                  {this.state.kotsUpdateMessage ?  
                    <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">{ shortKotsUpdateMessage }</p>
                    : null
                  }
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

          {this.state.showDeployWarningModal &&
            <DeployWarningModal
              showDeployWarningModal={this.state.showDeployWarningModal}
              hideDeployWarningModal={() => this.setState({ showDeployWarningModal: false })}
              onForceDeployClick={this.onForceDeployClick}
            />
          }
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
