import React, { Component } from "react";
import classNames from "classnames";
import { withRouter, Link } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import Helmet from "react-helmet";
import dayjs from "dayjs";
import MonacoEditor from "react-monaco-editor";
import relativeTime from "dayjs/plugin/relativeTime";
import Dropzone from "react-dropzone";
import Modal from "react-modal";
import moment from "moment";
import changeCase from "change-case";
import find from "lodash/find";
import Loader from "../shared/Loader";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DownstreamWatchVersionDiff from "@src/components/watches/DownstreamWatchVersionDiff";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import { getKotsDownstreamHistory, getKotsDownstreamOutput, getUpdateDownloadStatus } from "../../queries/AppsQueries";
import { checkForKotsUpdates } from "../../mutations/AppsMutations";
import { Utilities, isAwaitingResults, getPreflightResultState, getGitProviderDiffUrl, getCommitHashFromUrl } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import has from "lodash/has";

import "@src/scss/components/watches/WatchVersionHistory.scss";
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
    checkingUpdateText: "Checking for updates",
    errorCheckingUpdate: false,
    airgapUploadError: null,
    showDiffOverlay: false,
    firstSequence: 0,
    secondSequence: 0,
    updateChecker: new Repeater(),
    uploadTotal: 0,
    uploadSent: 0
  }

  componentDidMount() {
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

  componentWillUnmount() {
    this.state.updateChecker.stop();
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

  renderVersionSequence = version => {
    return (
      <div className="flex">
        {version.sequence}
        {version.releaseNotes &&
          <span className="replicated-link u-marginLeft--5" style={{ fontSize: 12, marginTop: 2 }} onClick={() => this.showDownstreamReleaseNotes(version.releaseNotes)}>Release notes</span>
        }
      </div>
    );
  }

  renderSourceAndDiff = version => {
    const { app } = this.props;
    const downstream = app.downstreams[0];
    const diffSummary = this.getVersionDiffSummary(version);
    return (
      <div>
        {diffSummary ? (
          diffSummary.filesChanged > 0 ?
            <div
              className="DiffSummary u-cursor--pointer"
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
        ) : <span>&nbsp;</span>}
      </div>
    );
  }

  renderVersionAction = version => {
    const { app } = this.props;
    const downstream = app.downstreams[0];

    if (downstream.gitops?.enabled) {
      if (version.gitDeployable === false) {
        return (<div>Nothing to commit</div>);
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

    if (downstream.currentVersion?.sequence == undefined) {
      // no current version found
      return (
        <button
          className="btn primary blue"
          onClick={() => this.deployVersion(version)}
        >
          Deploy
        </button>
      );
    }

    const isCurrentVersion = version.sequence === downstream.currentVersion?.sequence;
    const isPastVersion = find(downstream.pastVersions, { sequence: version.sequence });
    const showActions = !isPastVersion || app.allowRollback;

    return (
      <div>
        {showActions &&
          <button
            className={classNames("btn", { "secondary blue": isPastVersion, "primary blue": !isPastVersion })}
            disabled={isCurrentVersion}
            onClick={() => this.deployVersion(version)}
          >
            {isPastVersion ?
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
      <Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`}>
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
    if (isPastVersion && isPastVersion.status !== "failed") {
      return null;
    }
    const clusterSlug = downstream.cluster?.slug;

    let preflightBlock = null;
    if (version.status === "pending_preflight") {
      preflightBlock = (
        <span className="flex u-marginLeft--5 alignItems--center">
          <Loader size="20" />
        </span>);
    } else if (app.hasPreflight && version.status === "pending") {
      if (preflightsFailed) {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflight errors</Link>);
      } else {
        preflightBlock = (<Link to={`/app/${match.params.slug}/downstreams/${clusterSlug}/version-history/preflight/${version.sequence}`} className="replicated-link u-marginLeft--5 u-fontSize--small">View preflights</Link>);
      }
    }

    return (
      <div className="flex alignItems--center" style={{ position: "relative", top: "-2px" }}>
        <div className="flex alignItems--center">
          <div
            data-tip={`${version.title}-${version.sequence}`}
            data-for={`${version.title}-${version.sequence}`}
            className={classNames("icon", {
              "checkmark-icon": version.status === "deployed" || version.status === "merged" || version.status === "pending",
              "exclamationMark--icon": version.status === "opened",
              "grayCircleMinus--icon": version.status === "closed",
              "error-small": version.status === "failed" || preflightsFailed
            })}
          />
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
                  : version.status === "pending" || version.status === "pending_config"
                    ? "Ready to deploy"
                    : version.status
            ).replace("_", " ")}
          </span>
        </div>
        {preflightBlock}
        {version.status === "failed" &&
          <span className="replicated-link u-marginLeft--5 u-fontSize--small" onClick={() => this.handleViewLogs(version)}>View logs</span>
        }
      </div>
    );
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
    if (!force) {
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
    await this.props.makeCurrentVersion(match.params.slug, version.sequence, clusterSlug);
    await this.props.data.refetch();
    this.setState({ versionToDeploy: null });

    if (this.props.updateCallback) {
      this.props.updateCallback();
    }
  }

  onForceDeployClick = () => {
    this.setState({ showSkipModal: false, showDeployWarningModal: false });
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

  hideDiffOverlay = () => {
    this.setState({
      showDiffOverlay: false
    });
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

  handleViewLogs = async version => {
    const { match, app } = this.props;
    const clusterSlug = app.downstreams?.length && app.downstreams[0].cluster?.slug;
    if (clusterSlug) {
      this.setState({ logsLoading: true, showLogsModal: true });
      this.props.client.query({
        query: getKotsDownstreamOutput,
        fetchPolicy: "no-cache",
        variables: {
          appSlug: match.params.slug,
          clusterSlug: clusterSlug,
          sequence: version.sequence
        }
      }).then(result => {
        const logs = result.data.getKotsDownstreamOutput;
        const selectedTab = Object.keys(logs)[0];
        this.setState({ logs, selectedTab, logsLoading: false });
      }).catch(err => {
        console.log(err);
        this.setState({ logsLoading: false });
      });
    }
  }

  updateStatus = () => {
    return new Promise((resolve, reject) => {
      this.props.client.query({
        query: getUpdateDownloadStatus,
        fetchPolicy: "no-cache",
      }).then((res) => {

        this.setState({
          checkingForUpdates: true,
          checkingUpdateText: res.data.getUpdateDownloadStatus?.currentMessage,
        });

        if (res.data.getUpdateDownloadStatus.status !== "running" && !this.props.isBundleUploading) {
          this.state.updateChecker.stop();
          this.setState({
            checkingForUpdates: false,
            checkingForUpdateError: res.data.getUpdateDownloadStatus.status === "failed",
            checkingUpdateText: res.data.getUpdateDownloadStatus?.currentMessage
          });

          if (this.props.updateCallback) {
            this.props.updateCallback();
          }
          this.props.data.refetch();
        }

        resolve();

      }).catch((err) => {
        console.log("failed to get rewrite status", err);
        reject();
      });
    });
  }

  onCheckForUpdates = async () => {
    const { client, app } = this.props;

    this.setState({ checkingForUpdates: true, checkingForUpdateError: false, errorCheckingUpdate: false });

    await client.mutate({
      mutation: checkForKotsUpdates,
      variables: {
        appId: app.id,
      }
    }).catch((err) => {
      this.setState({ errorCheckingUpdate: true });
      console.log(err);
    }).finally(() => {
      this.state.updateChecker.start(this.updateStatus, 1000);
    });
  }

  onDropBundle = async files => {
    this.setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: null
    });

    this.props.toggleIsBundleUploading(true);

    const formData = new FormData();
    formData.append("file", files[0]);
    formData.append("appId", this.props.app.id);

    const url = `${window.env.API_ENDPOINT}/kots/airgap/update`;
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);

    xhr.setRequestHeader("Authorization", Utilities.getToken())
    xhr.upload.onprogress = event => {
      const total = event.total;
      const sent = event.loaded;

      this.setState({
        uploadSent: sent,
        uploadTotal: total,
      });
    }

    xhr.upload.onerror = () => {
      this.setState({
        uploadingAirgapFile: false,
        checkingForUpdates: false,
        uploadSent: 0,
        uploadTotal: 0,
        airgapUploadError: "Error uploading bundle, please try again"
      });
      this.props.toggleIsBundleUploading(false);
    }

    xhr.onloadend = async () => {
      const response = xhr.response;
      if (xhr.status === 202) {
        this.state.updateChecker.start(this.updateStatus, 1000);
        this.setState({
          uploadingAirgapFile: false
        });
      } else {
        this.setState({
          uploadingAirgapFile: false,
          checkingForUpdates: false,
          airgapUploadError: `Error uploading airgap bundle: ${response}`
        });
      }
      this.props.toggleIsBundleUploading(false);
    }

    xhr.send(formData);
  }

  onProgressError = async (airgapUploadError) => {
    Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
      if (airgapUploadError.includes(errorString)) {
        airgapUploadError = message;
      }
    });
    this.setState({
      airgapUploadError,
      checkingForUpdates: false,
      uploadSent: 0,
      uploadTotal: 0
    });
  }

  renderDiffBtn = () => {
    const { app, data } = this.props;
    const {
      showDiffOverlay,
      selectedDiffReleases,
      checkedReleasesToDiff,
    } = this.state;
    const downstream = app.downstreams.length && app.downstreams[0];
    const gitopsEnabled = downstream.gitops?.enabled;
    const versionHistory = data?.getKotsDownstreamHistory?.length ? data.getKotsDownstreamHistory : [];
    return (
      versionHistory.length && selectedDiffReleases ?
        <div className="flex">
          <button className="btn secondary gray small u-marginRight--10" onClick={this.onCloseReleasesToDiff}>Cancel</button>
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

  render() {
    const {
      app,
      data,
      match,
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
      checkingForUpdates,
      checkingUpdateText,
      errorCheckingUpdate,
      airgapUploadError,
      showDiffOverlay,
      firstSequence,
      secondSequence,
      uploadingAirgapFile,
      uploadTotal,
      uploadSent
    } = this.state;

    if (!app) {
      return null;
    }

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    // only render loader if there is no app yet to avoid flickering
    if (data?.loading && !data?.getKotsDownstreamHistory?.length) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    let updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">Last checked {dayjs(app.lastUpdateCheck).fromNow()}</p>;
    if (airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">{airgapUploadError}</p>;
    } else if (uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          total={uploadTotal}
          sent={uploadSent}
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
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error checking for updates, please try again</p>
    } else if (checkingForUpdates) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">{checkingUpdateTextShort}</p>
    } else if (!app.lastUpdateCheck) {
      updateText = null;
    }

    const showAirgapUI = app.isAirgap && !isBundleUploading;
    const showOnlineUI = !app.isAirgap && !checkingForUpdates;
    const downstream = app.downstreams.length && app.downstreams[0];
    const gitopsEnabled = downstream.gitops?.enabled;
    const currentDownstreamVersion = downstream?.currentVersion;
    const versionHistory = data?.getKotsDownstreamHistory?.length ? data.getKotsDownstreamHistory : [];

    if (isAwaitingResults(versionHistory)) {
      data?.startPolling(2000);
    } else if (has(data, "stopPolling")) {
      data?.stopPolling();
    }


    return (
      <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{`${app.name} Version History`}</title>
        </Helmet>
        <div className="flex flex-auto alignItems--center justifyContent--center u-marginTop--10 u-marginBottom--30">
          <div className="upstream-version-box-wrapper flex">
            <div className="flex flex1">
              {app.iconUri &&
                <div className="flex-auto u-marginRight--10">
                  <div className="watch-icon" style={{ backgroundImage: `url(${app.iconUri})` }}></div>
                </div>
              }
              <div className="flex1 flex-column">
                <p className="u-fontSize--34 u-fontWeight--bold u-color--tuna">
                  {app.currentVersion ? app.currentVersion.title : "---"}
                </p>
                <p className="u-fontSize--large u-fontWeight--medium u-marginTop--5 u-color--nevada">{app.currentVersion ? "Current upstream version" : "No deployments have been made"}</p>
                <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">
                  {app?.currentVersion?.deployedAt && `Released on ${dayjs(app.currentVersion.deployedAt).format("MMMM D, YYYY")}`}
                  {app?.currentVersion?.releaseNotes && <span className={classNames("release-notes-link", { "u-paddingLeft--5": app?.currentVersion?.deployedAt })} onClick={this.showReleaseNotes}>Release Notes</span>}
                </p>
              </div>
            </div>
            {!app.cluster &&
              <div className="flex-auto flex-column alignItems--center justifyContent--center">
                {checkingForUpdates && !isBundleUploading
                  ? <Loader size="32" />
                  : showAirgapUI
                    ?
                    <Dropzone
                      className="Dropzone-wrapper"
                      accept=".airgap"
                      onDropAccepted={this.onDropBundle}
                      multiple={false}
                    >
                      <button className="btn secondary blue">Upload new version</button>
                    </Dropzone>
                    : showOnlineUI ?
                      <button className="btn secondary blue" onClick={this.onCheckForUpdates}>Check for updates</button>
                      : null
                }
                {updateText}
              </div>
            }
          </div>
        </div>
        {this.state.checkingForUpdateError &&
          <div className="flex-column flex-auto u-marginBottom--30">
            <div className="checking-update-error-wrapper">
              <p className="u-color--chestnut u-fontSize--normal u-lineHeight--normal">{this.state.checkingUpdateText}</p>
            </div>
          </div>
        }
        <div className="flex-column flex1">
          <div className="flex flex1">
            <div className="flex1 flex-column alignItems--center">
              {/* Active downstream */}
              {!gitopsEnabled && currentDownstreamVersion &&
                <div className="TableDiff--Wrapper u-marginBottom--30">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10 u-lineHeight--normal">Deployed version</p>
                  <div className={`VersionHistoryDeploymentRow active-deploy-row ${currentDownstreamVersion.status} flex flex-auto`}>
                    <div className="flex-column flex1 u-paddingRight--20">
                      <div>
                        <p className="u-fontSize--normal u-color--dustyGray">Environment: <span className="u-fontWeight--bold u-color--tuna">{changeCase.title(downstream.name)}</span></p>
                        <p className="u-fontSize--small u-marginTop--10 u-color--dustyGray">Received: <span className="u-fontWeight--bold u-color--tuna">{moment(currentDownstreamVersion.createdOn).format("MM/DD/YY @ hh:mm a")}</span></p>
                      </div>
                      <div className="flex flex1 u-marginTop--15">
                        <p className="u-fontSize--normal u-color--dustyGray">Upstream: <span className="u-fontWeight--bold u-color--tuna">{currentDownstreamVersion.title}</span></p>
                        <div className="u-fontSize--normal u-color--dustyGray u-marginLeft--20 flex">Sequence: <span className="u-fontWeight--bold u-color--tuna u-marginLeft--5">{this.renderVersionSequence(currentDownstreamVersion)}</span></div>
                      </div>
                    </div>
                    <div className="flex-column flex1">
                      <div>
                        <p className="u-fontSize--normal u-color--dustyGray">Source: <span className="u-fontWeight--bold u-color--tuna">{currentDownstreamVersion.source}</span></p>
                        <div className="u-fontSize--small u-marginTop--10 u-color--dustyGray">{this.renderSourceAndDiff(currentDownstreamVersion)}</div>
                      </div>
                      <div className="flex flex1 u-fontSize--normal u-color--dustyGray u-marginTop--15">Status:<span className="u-marginLeft--5">{gitopsEnabled ? this.renderViewPreflights(currentDownstreamVersion) : this.renderVersionStatus(currentDownstreamVersion)}</span></div>
                    </div>
                    <div className="flex-column flex1 alignItems--flexEnd">
                      <div className="flex alignItems--center">
                        <button className="btn secondary" onClick={() => this.handleViewLogs(currentDownstreamVersion)}>View logs</button>
                        {app.isConfigurable && <Link className="btn secondary blue u-marginLeft--10" to={`/app/${match.params.slug}/config`}>Edit config</Link>}
                      </div>
                      <p className="u-fontSize--normal u-color--dustyGray u-marginTop--15">Deployed: <span className="u-fontWeight--bold u-color--tuna">{moment(currentDownstreamVersion.deployedAt).format("MM/DD/YY @ hh:mm a")}</span></p>
                    </div>
                  </div>
                </div>
              }

              <div className="TableDiff--Wrapper flex-column flex1">
                <div className="flex justifyContent--spaceBetween u-borderBottom--gray darker u-paddingBottom--10">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-lineHeight--normal">All versions</p>
                  {versionHistory.length > 1 && this.renderDiffBtn()}
                </div>
                {/* Downstream version history */}
                {versionHistory.length >= 1 ? versionHistory.map((version) => {
                  const isChecked = !!checkedReleasesToDiff.find(diffRelease => diffRelease.parentSequence === version.parentSequence);
                  return (
                    <div
                      key={version.sequence}
                      className={classNames(`VersionHistoryDeploymentRow ${version.status} flex flex-auto`, { "overlay": selectedDiffReleases, "selected": isChecked })}
                      onClick={() => selectedDiffReleases && this.handleSelectReleasesToDiff(version, !isChecked)}
                    >
                      {selectedDiffReleases && <div className={classNames("checkbox u-marginRight--20", { "checked": isChecked })} />}
                      <div className="flex-column flex1 u-paddingRight--20">
                        <div>
                          <p className="u-fontSize--normal u-color--dustyGray">Environment: <span className="u-fontWeight--bold u-color--tuna">{changeCase.title(downstream.name)}</span></p>
                          <p className="u-fontSize--small u-marginTop--10 u-color--dustyGray">Received: <span className="u-fontWeight--bold u-color--tuna">{moment(version.createdOn).format("MM/DD/YY @ hh:mm a")}</span></p>
                        </div>
                        <div className="flex flex1 u-marginTop--15">
                          <p className="u-fontSize--normal u-color--dustyGray">Upstream: <span className="u-fontWeight--bold u-color--tuna">{version.title}</span></p>
                          <div className="u-fontSize--normal u-color--dustyGray u-marginLeft--20 flex">Sequence: <span className="u-fontWeight--bold u-color--tuna u-marginLeft--5">{this.renderVersionSequence(version)}</span></div>
                        </div>
                      </div>
                      <div className="flex-column flex1">
                        <div>
                          <p className="u-fontSize--normal u-color--dustyGray">Source: <span className="u-fontWeight--bold u-color--tuna">{version.source}</span></p>
                          <div className="u-fontSize--small u-marginTop--10 u-color--dustyGray">{this.renderSourceAndDiff(version)}</div>
                        </div>
                        <div className="flex flex1 u-fontSize--normal u-color--dustyGray u-marginTop--15">Status: <span className="u-marginLeft--5">{gitopsEnabled ? this.renderViewPreflights(version) : this.renderVersionStatus(version)}</span></div>
                      </div>
                      <div className="flex-column flex1 alignItems--flexEnd">
                        <div>
                          {this.renderVersionAction(version)}
                        </div>
                        <p className="u-fontSize--normal u-color--dustyGray u-marginTop--15">Deployed: <span className="u-fontWeight--bold u-color--tuna">{version.deployedAt ? moment(version.deployedAt).format("MM/DD/YY @ hh:mm a") : "N/A"}</span></p>
                      </div>
                    </div>
                  );
                }) :
                  <div className="flex-column flex1 alignItems--center justifyContent--center">
                    <p className="u-fontSize--large u-fontWeight--bold u-color--tuna">No versions have been deployed.</p>
                  </div>
                }

                {/* Diff overlay */}
                {showDiffOverlay &&
                  <div className="DiffOverlay">
                    <DownstreamWatchVersionDiff
                      slug={match.params.slug}
                      firstSequence={firstSequence}
                      secondSequence={secondSequence}
                      onBackClick={this.hideDiffOverlay}
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
              {app?.currentVersion?.releaseNotes || "No release notes for this version"}
            </MarkdownRenderer>
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={this.hideReleaseNotes}>Close</button>
          </div>
        </Modal>

        <Modal
          isOpen={showLogsModal}
          onRequestClose={this.hideLogsModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="View logs"
          ariaHideApp={false}
          className="Modal logs-modal"
        >
          <div className="Modal-body flex flex1">
            {!logs || !selectedTab || logsLoading ? (
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <Loader size="60" />
              </div>
            ) : (
                <div className="flex-column flex1">
                  {logs.renderError ?
                    <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                      <MonacoEditor
                        language="json"
                        value={logs.renderError}
                        height="100%"
                        width="100%"
                        options={{
                          readOnly: true,
                          contextmenu: false,
                          minimap: {
                            enabled: false
                          },
                          scrollBeyondLastLine: false,
                        }}
                      />
                    </div>
                    :
                    <div className="flex-column flex1">
                      {this.renderLogsTabs()}
                      <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                        <MonacoEditor
                          language="json"
                          value={logs[selectedTab]}
                          height="100%"
                          width="100%"
                          options={{
                            readOnly: true,
                            contextmenu: false,
                            minimap: {
                              enabled: false
                            },
                            scrollBeyondLastLine: false,
                          }}
                        />
                      </div>
                    </div>
                  }
                  <div className="u-marginTop--20 flex">
                    <button type="button" className="btn primary" onClick={this.hideLogsModal}>Ok, got it!</button>
                  </div>
                </div>
              )}
          </div>
        </Modal>

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
          className="Modal LargeSize"
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
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(getKotsDownstreamHistory, {
    skip: ({ app }) => {
      return !app.downstreams || !app.downstreams.length;
    },
    options: ({ match, app }) => {
      const downstream = app.downstreams[0];
      return {
        variables: {
          upstreamSlug: match.params.slug,
          clusterSlug: downstream.cluster.slug,
        },
        fetchPolicy: "no-cache"
      }
    }
  }),
)(AppVersionHistory);
