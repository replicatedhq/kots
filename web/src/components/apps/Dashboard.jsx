import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import get from "lodash/get";
import sortBy from "lodash/sortBy";
import Loader from "../shared/Loader";
import DashboardVersionCard from "./DashboardVersionCard";
import AppStatus from "./AppStatus";
import DashboardLicenseCard from "./DashboardLicenseCard";
import DashboardSnapshotsCard from "./DashboardSnapshotsCard";
import DashboardGraphsCard from "./DashboardGraphsCard";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";
import Modal from "react-modal";
import { Repeater } from "../../utilities/repeater";
import { Utilities, isAwaitingResults } from "../../utilities/utilities";
import { AirgapUploader } from "../../utilities/airgapUploader";

import "../../scss/components/watches/Dashboard.scss";
import "../../../node_modules/react-vis/dist/style";
import { Paragraph } from "../../styles/common";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host",
};

class Dashboard extends Component {
  state = {
    appName: "",
    iconUri: "",
    currentVersion: {},
    downstream: [],
    links: [],
    checkingForUpdates: false,
    checkingUpdateMessage: "Checking for updates",
    checkingForUpdateError: false,
    appLicense: null,
    activeChart: null,
    crosshairValues: [],
    noUpdatesAvalable: false,
    updateChecker: new Repeater(),
    uploadingAirgapFile: false,
    airgapUploadError: null,
    viewAirgapUploadError: false,
    viewAirgapUpdateError: false,
    airgapUpdateError: "",
    startSnapshotErrorMsg: "",
    showAutomaticUpdatesModal: false,
    showAppStatusModal: false,
    dashboard: {
      appStatus: null,
      metrics: [],
      prometheusAddress: "",
    },
    getAppDashboardJob: new Repeater(),
    fetchAppDownstreamJob: new Repeater(),
    gettingAppLicenseErrMsg: "",
    startSnapshotOptions: [
      { option: "partial", name: "Start a Partial snapshot" },
      { option: "full", name: "Start a Full snapshot" },
      { option: "learn", name: "Learn about the difference" },
    ],
    selectedSnapshotOption: { option: "full", name: "Start a Full snapshot" },
    snapshotDifferencesModal: false,
  };

  setWatchState = (app) => {
    this.setState({
      appName: app.name,
      iconUri: app.iconUri,
      currentVersion: app.downstream?.currentVersion,
      downstream: app.downstream,
      links: app.downstream?.links,
    });
  };

  componentDidUpdate(lastProps) {
    const { app } = this.props;
    if (app !== lastProps.app && app) {
      this.setWatchState(app);
      this.getAppLicense(app);
    }
  }

  getAppLicense = async (app) => {
    await fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/license`, {
      method: "GET",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
      .then(async (res) => {
        if (!res.ok) {
          this.setState({ gettingAppLicenseErrMsg: body.error });
          return;
        }

        const body = await res.json();
        if (body === null) {
          this.setState({ appLicense: {}, gettingAppLicenseErrMsg: "" });
        } else if (body.success) {
          this.setState({
            appLicense: body.license,
            gettingAppLicenseErrMsg: "",
          });
        } else if (body.error) {
          this.setState({
            appLicense: {},
            gettingAppLicenseErrMsg: body.error,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          gettingAppLicenseErrMsg: err
            ? `Error while getting the license: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  componentDidMount() {
    const { app } = this.props;

    if (app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig();
    }

    this.state.updateChecker.start(this.updateStatus, 1000);
    this.state.getAppDashboardJob.start(this.getAppDashboard, 2000);
    if (app) {
      this.setWatchState(app);
      this.getAppLicense(app);
    }
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
    this.state.getAppDashboardJob.stop();
    this.state.fetchAppDownstreamJob.stop();
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

  getAppDashboard = () => {
    return new Promise((resolve, reject) => {
      // this function is in a repeating callback that terminates when
      // the promise is resolved

      // TODO: use react-query to refetch this instead of the custom repeater
      if (!this.props.app) {
        return;
      }

      if (this.props.cluster?.id == "" && this.props.isHelmManaged === true) {
        // TODO: use a callback to update the state in the parent component
        this.props.cluster.id = 0;
      }

      fetch(
        `${process.env.API_ENDPOINT}/app/${this.props.app?.slug}/cluster/${this.props.cluster?.id}/dashboard`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "GET",
        }
      )
        .then(async (res) => {
          if (!res.ok && res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          const response = await res.json();
          this.setState({
            dashboard: {
              appStatus: response.appStatus,
              prometheusAddress: response.prometheusAddress,
              metrics: response.metrics,
            },
          });
          resolve();
        })
        .catch((err) => {
          console.log(err);
          reject(err);
        });
    });
  };

  onCheckForUpdates = async () => {
    const { app } = this.props;

    this.setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
    });

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        const response = await res.json();
        if (response.availableUpdates === 0) {
          this.setState({
            checkingForUpdates: false,
            noUpdatesAvalable: true,
          });
          setTimeout(() => {
            this.setState({ noUpdatesAvalable: false });
          }, 3000);
        } else {
          this.state.updateChecker.start(this.updateStatus, 1000);
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          checkingForUpdateError: true,
          checkingForUpdates: false,
          checkingUpdateMessage: "Your license is expired.",
        });
      });
  };

  hideAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: false,
    });
  };

  showAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: true,
    });
  };

  fetchAppDownstream = async () => {
    const { app } = this.props;
    if (!app) {
      return;
    }

    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/app/${app.slug}`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        if (!isAwaitingResults(app.downstream.pendingVersions)) {
          this.state.fetchAppDownstreamJob.stop();
        }
        this.setState({
          downstream: app.downstream,
        });
        // wait a couple of seconds to avoid any race condiditons with the update checker then refetch the app to ensure we have the latest everything
        // this is hacky and I hate it but it's just building up more evidence in my case for having the FE be able to listen to BE envents
        // if that was in place we would have no need for this becuase the latest version would just be pushed down.
        setTimeout(() => {
          this.props.refreshAppData();
        }, 2000);
      } else {
        this.setState({
          loadingApp: false,
          gettingAppErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
      }
    } catch (err) {
      console.log(err);
      this.setState({
        loadingApp: false,
        gettingAppErrMsg: err
          ? err.message
          : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  startFetchAppDownstreamJob = () => {
    this.state.fetchAppDownstreamJob.start(this.fetchAppDownstream, 2000);
  };

  updateStatus = () => {
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
            this.state.updateChecker.stop();

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed",
            });

            if (this.props.updateCallback) {
              this.props.updateCallback();
            }
            this.startFetchAppDownstreamJob();
          } else {
            this.setState({
              checkingForUpdates: true,
              checkingUpdateMessage: response.currentMessage,
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
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
    this.state.updateChecker.start(this.updateStatus, 1000);
    this.setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.toggleIsBundleUploading(false);
  };

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
  };

  toggleViewAirgapUploadError = () => {
    this.setState({ viewAirgapUploadError: !this.state.viewAirgapUploadError });
  };

  toggleViewAirgapUpdateError = (err) => {
    this.setState({
      viewAirgapUpdateError: !this.state.viewAirgapUpdateError,
      airgapUpdateError: !this.state.viewAirgapUpdateError ? err : "",
    });
  };

  startASnapshot = (option) => {
    const { app } = this.props;
    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
    });

    let url =
      option === "full"
        ? `${process.env.API_ENDPOINT}/snapshot/backup`
        : `${process.env.API_ENDPOINT}/app/${app.slug}/snapshot/backup`;

    fetch(url, {
      method: "POST",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
      .then(async (result) => {
        if (!result.ok && result.status === 409) {
          const res = await result.json();
          if (res.kotsadmRequiresVeleroAccess) {
            this.setState({
              startingSnapshot: false,
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          this.setState({
            startingSnapshot: false,
          });
          this.props.ping();
          option === "full"
            ? this.props.history.push("/snapshots")
            : this.props.history.push(`/snapshots/partial/${app.slug}`);
        } else {
          const body = await result.json();
          this.setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: body.error,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          startSnapshotErrorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  onSnapshotOptionChange = (selectedSnapshotOption) => {
    if (selectedSnapshotOption.option === "learn") {
      this.setState({ snapshotDifferencesModal: true });
    } else {
      this.startASnapshot(selectedSnapshotOption.option);
    }
  };

  toggleSnaphotDifferencesModal = () => {
    this.setState({
      snapshotDifferencesModal: !this.state.snapshotDifferencesModal,
    });
  };

  onSnapshotOptionClick = () => {
    const { selectedSnapshotOption } = this.state;
    this.startASnapshot(selectedSnapshotOption.option);
  };

  toggleAppStatusModal = () => {
    this.setState({ showAppStatusModal: !this.state.showAppStatusModal });
  };

  goToTroubleshootPage = () => {
    this.props.history.push(`${this.props.match.url}/troubleshoot`);
  };

  getAppResourcesByState = () => {
    const appStatus = this.state.dashboard?.appStatus;
    if (!appStatus?.resourceStates?.length) {
      return {};
    }

    const resourceStates = appStatus?.resourceStates;
    const statesMap = {};

    for (let i = 0; i < resourceStates.length; i++) {
      const resourceState = resourceStates[i];
      if (!statesMap.hasOwnProperty(resourceState.state)) {
        statesMap[resourceState.state] = [];
      }
      statesMap[resourceState.state].push(resourceState);
    }

    // sort resources so that the order doesn't change while polling (since we show live data)
    Object.keys(statesMap).forEach((state) => {
      statesMap[state] = sortBy(statesMap[state], (resource) => {
        const fullResourceName = `${resource?.namespace}/${resource?.kind}/${resource?.name}`;
        return fullResourceName;
      });
    });

    // sort the available states to show them in the correct order
    const allStates = Object.keys(statesMap);
    const sortedStates = sortBy(allStates, (s) => {
      if (s === "missing") {
        return 1;
      }
      if (s === "unavailable") {
        return 2;
      }
      if (s === "degraded") {
        return 3;
      }
      if (s === "updating") {
        return 4;
      }
      if (s === "ready") {
        return 5;
      }
    });

    return {
      statesMap,
      sortedStates,
    };
  };

  checkStatusInformers = () => {
    const appResourcesByState = this.getAppResourcesByState();
    const { statesMap, sortedStates } = appResourcesByState;
    return sortedStates?.every((state) => {
      return statesMap[state]?.every((resource) => {
        const { kind, name, namespace } = resource;
        if (kind === "EMPTY" && name === "EMPTY" && namespace === "EMPTY") {
          return false;
        }
        return true;
      });
    });
  };

  render() {
    const {
      appName,
      iconUri,
      currentVersion,
      downstream,
      links,
      checkingForUpdates,
      checkingUpdateMessage,
      uploadingAirgapFile,
      airgapUploadError,
      appLicense,
    } = this.state;

    const { app, isBundleUploading, isVeleroInstalled } = this.props;

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

    const appResourcesByState = this.getAppResourcesByState();
    const hasStatusInformers = this.checkStatusInformers();

    return (
      <>
        {!app && (
          <div className="flex-column flex1 alignItems--center justifyContent--center">
            <Loader size="60" />
          </div>
        )}
        {app && (
          <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
            <Helmet>
              <title>{appName}</title>
            </Helmet>
            <div className="Dashboard flex flex-auto justifyContent--center alignSelf--center alignItems--center">
              <div className="flex1 flex-column">
                <div className="flex flex1 alignItems--center">
                  <div className="flex flex-auto">
                    <div
                      style={{ backgroundImage: `url(${iconUri})` }}
                      className="Dashboard--appIcon u-position--relative"
                    />
                  </div>
                  <div className="u-marginLeft--20">
                    <p className="u-fontSize--30 u-textColor--primary u-fontWeight--bold">
                      {appName}
                    </p>
                    <AppStatus
                      appStatus={this.state.dashboard?.appStatus?.state}
                      url={this.props.match.url}
                      onViewAppStatusDetails={this.toggleAppStatusModal}
                      links={links}
                      app={app}
                      hasStatusInformers={hasStatusInformers}
                    />
                  </div>
                </div>

                <div className="u-marginTop--30 flex flex1 u-width--full">
                  <div className="flex1 u-paddingRight--15">
                    <DashboardVersionCard
                      currentVersion={currentVersion}
                      downstream={downstream}
                      app={app}
                      url={this.props.match.url}
                      checkingForUpdates={checkingForUpdates}
                      checkingUpdateText={checkingUpdateText}
                      onDropBundle={this.onDropBundle}
                      airgapUploader={this.state.airgapUploader}
                      uploadingAirgapFile={uploadingAirgapFile}
                      airgapUploadError={airgapUploadError}
                      refetchData={this.props.updateCallback}
                      downloadCallback={this.startFetchAppDownstreamJob}
                      uploadProgress={this.state.uploadProgress}
                      uploadSize={this.state.uploadSize}
                      uploadResuming={this.state.uploadResuming}
                      makeCurrentVersion={this.props.makeCurrentVersion}
                      redeployVersion={this.props.redeployVersion}
                      onProgressError={this.onProgressError}
                      onCheckForUpdates={() => this.onCheckForUpdates()}
                      onUploadNewVersion={() => this.onUploadNewVersion()}
                      isBundleUploading={isBundleUploading}
                      checkingForUpdateError={this.state.checkingForUpdateError}
                      viewAirgapUploadError={() =>
                        this.toggleViewAirgapUploadError()
                      }
                      viewAirgapUpdateError={(err) =>
                        this.toggleViewAirgapUpdateError(err)
                      }
                      showAutomaticUpdatesModal={this.showAutomaticUpdatesModal}
                      noUpdatesAvalable={this.state.noUpdatesAvalable}
                      isHelmManaged={this.props.isHelmManaged}
                    />
                  </div>
                  <div className="flex1 flex-column u-paddingLeft--15">
                    {app.allowSnapshots && isVeleroInstalled ? (
                      <div className="u-marginBottom--30">
                        <DashboardSnapshotsCard
                          url={this.props.match.url}
                          app={app}
                          ping={this.props.ping}
                          isSnapshotAllowed={
                            app.allowSnapshots && isVeleroInstalled
                          }
                          isVeleroInstalled={isVeleroInstalled}
                          startASnapshot={this.startASnapshot}
                          startSnapshotOptions={this.state.startSnapshotOptions}
                          startSnapshotErr={this.state.startSnapshotErr}
                          startSnapshotErrorMsg={
                            this.state.startSnapshotErrorMsg
                          }
                          snapshotInProgressApps={
                            this.props.snapshotInProgressApps
                          }
                          selectedSnapshotOption={
                            this.state.selectedSnapshotOption
                          }
                          onSnapshotOptionChange={this.onSnapshotOptionChange}
                          onSnapshotOptionClick={this.onSnapshotOptionClick}
                        />
                      </div>
                    ) : null}
                    <DashboardLicenseCard
                      appLicense={appLicense}
                      app={app}
                      syncCallback={() => this.getAppLicense(this.props.app)}
                      gettingAppLicenseErrMsg={
                        this.state.gettingAppLicenseErrMsg
                      }
                    />
                  </div>
                </div>
                <div className="u-marginTop--30 flex flex1">
                  <DashboardGraphsCard
                    prometheusAddress={this.state.dashboard?.prometheusAddress}
                    metrics={this.state.dashboard?.metrics}
                    appSlug={app.slug}
                    clusterId={this.props.cluster?.id}
                    isHelmManaged={this.props.isHelmManaged}
                  />
                </div>
              </div>
            </div>
            {this.state.viewAirgapUploadError && (
              <Modal
                isOpen={this.state.viewAirgapUploadError}
                onRequestClose={this.toggleViewAirgapUploadError}
                contentLabel="Error uploading airgap bundle"
                ariaHideApp={false}
                className="Modal"
              >
                <div className="Modal-body">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary">
                    Error uploading airgap buundle
                  </p>
                  <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
                    <p className="u-fontSize--normal u-textColor--error">
                      {this.state.airgapUploadError}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="btn primary u-marginTop--15"
                    onClick={this.toggleViewAirgapUploadError}
                  >
                    Ok, got it!
                  </button>
                </div>
              </Modal>
            )}
            {this.state.viewAirgapUpdateError && (
              <Modal
                isOpen={this.state.viewAirgapUpdateError}
                onRequestClose={this.toggleViewAirgapUpdateError}
                contentLabel="Error updating airgap version"
                ariaHideApp={false}
                className="Modal"
              >
                <div className="Modal-body">
                  <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary">
                    Error updating version
                  </p>
                  <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
                    <p className="u-fontSize--normal u-textColor--error">
                      {this.state.airgapUpdateError}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="btn primary u-marginTop--15"
                    onClick={this.toggleViewAirgapUpdateError}
                  >
                    Ok, got it!
                  </button>
                </div>
              </Modal>
            )}
            {this.state.showAppStatusModal && (
              <Modal
                isOpen={this.state.showAppStatusModal}
                onRequestClose={this.toggleAppStatusModal}
                ariaHideApp={false}
                className="Modal DefaultSize"
              >
                <div className="Modal-body">
                  <Paragraph size="16" weight="bold">
                    Resource status
                  </Paragraph>
                  <div
                    className="u-marginTop--10 u-marginBottom--10 u-overflow--auto"
                    style={{ maxHeight: "50vh" }}
                  >
                    {appResourcesByState?.sortedStates?.map((state, i) => (
                      <div key={i}>
                        <p className="u-fontSize--normal u-color--mutedteal u-fontWeight--bold u-marginTop--20">
                          {Utilities.toTitleCase(state)}
                        </p>
                        {appResourcesByState?.statesMap[state]?.map(
                          (resource, i) => (
                            <div key={`${resource?.name}-${i}`}>
                              <p
                                className={`ResourceStateText u-fontSize--normal ${resource.state}`}
                              >
                                {resource?.namespace}/{resource?.kind}/
                                {resource?.name}
                              </p>
                            </div>
                          )
                        )}
                      </div>
                    ))}
                  </div>
                  <div className="flex alignItems--center u-marginTop--30">
                    <button
                      type="button"
                      className="btn primary"
                      onClick={this.toggleAppStatusModal}
                    >
                      Ok, got it!
                    </button>
                    <button
                      type="button"
                      className="btn secondary blue u-marginLeft--10"
                      onClick={this.goToTroubleshootPage}
                    >
                      Troubleshoot
                    </button>
                  </div>
                </div>
              </Modal>
            )}
            {this.state.showAutomaticUpdatesModal && (
              <AutomaticUpdatesModal
                isOpen={this.state.showAutomaticUpdatesModal}
                onRequestClose={this.hideAutomaticUpdatesModal}
                updateCheckerSpec={app.updateCheckerSpec}
                autoDeploy={app.autoDeploy}
                appSlug={app.slug}
                isSemverRequired={app?.isSemverRequired}
                gitopsIsConnected={downstream?.gitops?.isConnected}
                onAutomaticUpdatesConfigured={() => {
                  this.hideAutomaticUpdatesModal();
                  this.props.refreshAppData();
                }}
                isHelmManaged={this.props.isHelmManaged}
              />
            )}
            {this.state.snapshotDifferencesModal && (
              <SnapshotDifferencesModal
                snapshotDifferencesModal={this.state.snapshotDifferencesModal}
                toggleSnapshotDifferencesModal={
                  this.toggleSnaphotDifferencesModal
                }
              />
            )}
          </div>
        )}
      </>
    );
  }
}

export default withRouter(Dashboard);
