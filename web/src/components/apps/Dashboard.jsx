import React, { useState, useEffect } from "react";
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

const updateCheckerRepeater = new Repeater();
const getAppDashboardJobRepeater = new Repeater();
const fetchAppDownloadstreamJobRepeater = new Repeater();

const Dashboard = ({
  app,
  cluster: clusterProp,
  refreshAppData,
  updateCallback,
  toggleIsBundleUploading,
  history,
  match,
  makeCurrentVersion,
  redeployVersion,
  isHelmManaged,
  ping,
  snapshotInProgressApps,
  isVeleroInstalled,
  isBundleUploading,
  onUploadNewVersion,
}) => {
  const [appName, setAppName] = useState("");
  const [iconUri, setIconUri] = useState("");
  const [currentVersion, setCurrentVersion] = useState({});
  const [downstream, setDownstream] = useState([]);
  const [links, setLinks] = useState([]);
  const [checkingForUpdates, setCheckingForUpdates] = useState(false);
  const [checkingUpdateMessage, setCheckingUpdateMessage] = useState(
    "Checking for updates"
  );
  const [checkingForUpdateError, setCheckingForUpdateError] = useState(false);
  const [appLicense, setAppLicense] = useState(null);
  const [activeChart, setActiveChart] = useState(null);
  const [crosshairValues, setCrosshairValues] = useState([]);
  const [noUpdatesAvalable, setNoUpdatesAvalable] = useState(false);
  const [updateChecker, setUpdateChecker] = useState(updateCheckerRepeater);
  const [uploadingAirgapFile, setUploadingAirgapFile] = useState(false);
  const [airgapUploader, setAirgapUploader] = useState(null);
  const [airgapUploadError, setAirgapUploadError] = useState(null);
  const [viewAirgapUploadError, setViewAirgapUploadError] = useState(false);
  const [viewAirgapUpdateError, setViewAirgapUpdateError] = useState(false);
  const [airgapUpdateError, setAirgapUpdateError] = useState("");
  const [startSnapshotErr, setStartSnapshotErr] = useState(false);
  const [snapshotError, setSnapshotError] = useState(false);
  const [startSnapshotErrorMsg, setStartSnapshotErrorMsg] = useState("");
  const [showAutomaticUpdatesModalState, setShowAutomaticUpdatesModalState] =
    useState(false);
  const [showAppStatusModal, setShowAppStatusModal] = useState(false);
  const [dashboard, setDashboard] = useState({
    appStatus: null,
    metrics: [],
    prometheusAddress: "",
  });
  const [getAppDashboardJob, setGetAppDashboardJob] = useState(
    getAppDashboardJobRepeater
  );
  const [fetchAppDownstreamJob, setFetchAppDownstreamJob] = useState(
    fetchAppDownloadstreamJobRepeater
  );
  const [gettingAppLicenseErrMsg, setGettingAppLicenseErrMsg] = useState("");
  const [startSnapshotOptions, setStartSnapshotOptions] = useState([
    { option: "partial", name: "Start a Partial snapshot" },
    { option: "full", name: "Start a Full snapshot" },
    { option: "learn", name: "Learn about the difference" },
  ]);
  const [selectedSnapshotOption, setSelectedSnapshotOption] = useState({
    option: "full",
    name: "Start a Full snapshot",
  });
  const [snapshotDifferencesModal, setSnapshotDifferencesModal] =
    useState(false);
  const [gettingAppErrMsg, setGettingAppErrMsg] = useState("");
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadSize, setUploadSize] = useState(0);
  const [uploadResuming, setUploadResuming] = useState(false);
  const [loadingApp, setLoadingApp] = useState(false);
  const [displayErrorModal, setDisplayErrorModal] = useState(false);
  const [startingSnapshot, setStartingSnapshot] = useState(false);
  const [cluster, setCluster] = useState(clusterProp);

  useEffect(() => {
    if (app) {
      setWatchState(app);
      getAppLicense(app);
    }
  }, [app]);

  useEffect(() => {
    setCluster(clusterProp);
  }, [clusterProp]);

  const setWatchState = (app) => {
    setAppName(app.name);
    setIconUri(app.iconUri);
    setCurrentVersion(app?.downstream.currentVersion);
    setDownstream(app.downstream);
    setLinks(app.links);
  };

  const getAppLicense = async (app) => {
    await fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/license`, {
      method: "GET",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
      .then(async (res) => {
        if (!res.ok) {
          setGettingAppLicenseErrMsg(body.error);
          return;
        }

        const body = await res.json();
        if (body === null) {
          setAppLicense({});
          setGettingAppLicenseErrMsg("");
        } else if (body.success) {
          setAppLicense(body.license);
          setGettingAppLicenseErrMsg("");
        } else if (body.error) {
          setAppLicense({});
          setGettingAppLicenseErrMsg(body.error);
        }
      })
      .catch((err) => {
        console.log(err);
        setGettingAppLicenseErrMsg(
          err
            ? `Error while getting the license: ${err.message}`
            : "Something went wrong, please try again."
        );
      });
  };

  const getAirgapConfig = async () => {
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

    setAirgapUploader(
      new AirgapUploader(true, app.slug, onDropBundle, simultaneousUploads)
    );
  };

  useEffect(() => {
    if (app?.isAirgap && !airgapUploader) {
      getAirgapConfig();
    }

    updateChecker.start(updateStatus, 1000);
    getAppDashboardJob.start(getAppDashboard, 2000);
    if (app) {
      setWatchState(app);
      getAppLicense(app);
    }
    return () => {
      updateChecker.stop();
      getAppDashboardJob.stop();
      fetchAppDownstreamJob.stop();
    };
  }, []);

  const getAppDashboard = () => {
    return new Promise((resolve, reject) => {
      // this function is in a repeating callback that terminates when
      // the promise is resolved

      // TODO: use react-query to refetch this instead of the custom repeater
      if (!app) {
        return;
      }

      if (cluster?.id == "" && isHelmManaged === true) {
        // TODO: use a callback to update the state in the parent component
        setCluster({ ...cluster, id: 0 });
        return;
      }

      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/cluster/${cluster?.id}/dashboard`,
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
          setDashboard({
            appStatus: response.appStatus,
            prometheusAddress: response.prometheusAddress,
            metrics: response.metrics,
          });

          resolve();
        })
        .catch((err) => {
          console.log(err);
          reject(err);
        });
    });
  };

  const onCheckForUpdates = async () => {
    setCheckingForUpdates(true);
    setCheckingForUpdateError(false);

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
          setCheckingForUpdates(false);
          setNoUpdatesAvalable(true);
          setTimeout(() => {
            setNoUpdatesAvalable(false);
          }, 3000);
        } else {
          updateChecker.start(updateStatus, 1000);
        }
      })
      .catch((err) => {
        console.log(err);
        setCheckingForUpdateError(true);
        setCheckingForUpdates(false);
        setCheckingUpdateMessage("Your license is expired.");
      });
  };

  const hideAutomaticUpdatesModal = () => {
    setShowAutomaticUpdatesModalState(false);
  };

  const showAutomaticUpdatesModal = () => {
    setShowAutomaticUpdatesModalState(true);
  };

  const fetchAppDownstream = async () => {
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
          fetchAppDownstreamJob.stop();
        }
        setDownstream(app.downstream);
        // wait a couple of seconds to avoid any race condiditons with the update checker then refetch the app to ensure we have the latest everything
        // this is hacky and I hate it but it's just building up more evidence in my case for having the FE be able to listen to BE envents
        // if that was in place we would have no need for this becuase the latest version would just be pushed down.
        setTimeout(() => {
          refreshAppData();
        }, 2000);
      } else {
        setLoadingApp(false);
        setGettingAppErrMsg(`Unexpected status code: ${res.status}`);
        setDisplayErrorModal(true);
      }
    } catch (err) {
      console.log(err);
      setLoadingApp(false);
      setGettingAppErrMsg(
        err ? err.message : "Something went wrong, please try again."
      );
      setDisplayErrorModal(true);
    }
  };

  const startFetchAppDownstreamJob = () => {
    fetchAppDownstreamJob.start(fetchAppDownstream, 2000);
  };

  const updateStatus = () => {
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

          if (response.status !== "running" && !isBundleUploading) {
            updateChecker.stop();

            setCheckingForUpdates(false);
            setCheckingUpdateMessage(response.currentMessage);
            setCheckingForUpdateError(response.status === "failed");

            if (updateCallback) {
              updateCallback();
            }
            startFetchAppDownstreamJob();
          } else {
            setCheckingForUpdates(true);
            setCheckingUpdateMessage(response.currentMessage);
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
    });
  };

  const onDropBundle = async () => {
    setUploadingAirgapFile(true);
    setCheckingForUpdates(true);
    setAirgapUpdateError(null);
    setUploadProgress(0);
    setUploadSize(0);
    setUploadResuming(false);

    toggleIsBundleUploading(true);

    const params = {
      appId: app?.id,
    };
    airgapUploader.upload(
      params,
      onUploadProgress,
      onUploadError,
      onUploadComplete
    );
  };

  const onUploadProgress = (progress, size, resuming = false) => {
    setUploadProgress(progress);
    setUploadSize(size);
    setUploadResuming(resuming);
  };

  const onUploadError = (message) => {
    setUploadingAirgapFile(false);
    setCheckingForUpdates(false);
    setUploadProgress(0);
    setUploadSize(0);
    setUploadResuming(false);
    setAirgapUpdateError(message || "Error uploading bundle, please try again");
    toggleIsBundleUploading(false);
  };

  const onUploadComplete = () => {
    updateChecker.start(updateStatus, 1000);
    setUploadingAirgapFile(false);
    setUploadProgress(0);
    setUploadSize(0);
    setUploadResuming(false);
    toggleIsBundleUploading(false);
  };

  const onProgressError = async (airgapUploadErrorReceived) => {
    Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
      if (airgapUploadErrorReceived.includes(errorString)) {
        airgapUploadErrorReceived = message;
      }
    });
    setUploadingAirgapFile(false);
    setAirgapUpdateError(airgapUploadErrorReceived);
    setCheckingForUpdates(false);
    setUploadProgress(0);
    setUploadSize(0);
    setUploadResuming(false);
  };

  const toggleViewAirgapUploadError = () => {
    setViewAirgapUploadError(!viewAirgapUploadError);
  };

  const toggleViewAirgapUpdateError = (err) => {
    setAirgapUpdateError(!viewAirgapUpdateError ? err : "");
    setViewAirgapUploadError(!viewAirgapUpdateError);
  };

  const startASnapshot = (option) => {
    setStartingSnapshot(true);
    setSnapshotError(false);
    setStartSnapshotErrorMsg("");

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
            setStartingSnapshot(false);
            history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          setStartingSnapshot(false);
          ping();
          option === "full"
            ? history.push("/snapshots")
            : history.push(`/snapshots/partial/${app.slug}`);
        } else {
          const body = await result.json();
          setStartingSnapshot(false);
          setStartSnapshotErr(true);
          setStartSnapshotErrorMsg(body.error);
        }
      })
      .catch((err) => {
        console.log(err);
        setStartSnapshotErrorMsg(
          err ? err.message : "Something went wrong, please try again."
        );
      });
  };

  const onSnapshotOptionChange = (selectedSnapshotOption) => {
    if (selectedSnapshotOption.option === "learn") {
      setSnapshotDifferencesModal(true);
    } else {
      startASnapshot(selectedSnapshotOption.option);
    }
  };

  const toggleSnaphotDifferencesModal = () => {
    setSnapshotDifferencesModal(!snapshotDifferencesModal);
  };

  const onSnapshotOptionClick = () => {
    startASnapshot(selectedSnapshotOption.option);
  };

  const toggleAppStatusModal = () => {
    setShowAppStatusModal(!showAppStatusModal);
  };

  const goToTroubleshootPage = () => {
    history.push(`${match.url}/troubleshoot`);
  };

  const getAppResourcesByState = () => {
    const appStatus = dashboard?.appStatus;
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

  const checkStatusInformers = () => {
    const appResourcesByState = getAppResourcesByState();
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

  const appResourcesByState = getAppResourcesByState();
  const hasStatusInformers = checkStatusInformers();

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
                    appStatus={dashboard?.appStatus?.state}
                    url={match.url}
                    onViewAppStatusDetails={toggleAppStatusModal}
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
                    url={match.url}
                    checkingForUpdates={checkingForUpdates}
                    checkingUpdateText={checkingUpdateText}
                    onDropBundle={onDropBundle}
                    airgapUploader={airgapUploader}
                    uploadingAirgapFile={uploadingAirgapFile}
                    airgapUploadError={airgapUploadError}
                    refetchData={updateCallback}
                    downloadCallback={startFetchAppDownstreamJob}
                    uploadProgress={uploadProgress}
                    uploadSize={uploadSize}
                    uploadResuming={uploadResuming}
                    makeCurrentVersion={makeCurrentVersion}
                    redeployVersion={redeployVersion}
                    onProgressError={onProgressError}
                    onCheckForUpdates={() => onCheckForUpdates()}
                    onUploadNewVersion={() => onUploadNewVersion()}
                    isBundleUploading={isBundleUploading}
                    checkingForUpdateError={checkingForUpdateError}
                    viewAirgapUploadError={() => toggleViewAirgapUploadError()}
                    viewAirgapUpdateError={(err) =>
                      toggleViewAirgapUpdateError(err)
                    }
                    showAutomaticUpdatesModal={showAutomaticUpdatesModal}
                    noUpdatesAvalable={noUpdatesAvalable}
                    isHelmManaged={isHelmManaged}
                  />
                </div>
                <div className="flex1 flex-column u-paddingLeft--15">
                  {app.allowSnapshots && isVeleroInstalled ? (
                    <div className="u-marginBottom--30">
                      <DashboardSnapshotsCard
                        url={match.url}
                        app={app}
                        ping={ping}
                        isSnapshotAllowed={
                          app.allowSnapshots && isVeleroInstalled
                        }
                        isVeleroInstalled={isVeleroInstalled}
                        startASnapshot={startASnapshot}
                        startSnapshotOptions={startSnapshotOptions}
                        startSnapshotErr={startSnapshotErr}
                        startSnapshotErrorMsg={startSnapshotErrorMsg}
                        snapshotInProgressApps={snapshotInProgressApps}
                        selectedSnapshotOption={selectedSnapshotOption}
                        onSnapshotOptionChange={onSnapshotOptionChange}
                        onSnapshotOptionClick={onSnapshotOptionClick}
                      />
                    </div>
                  ) : null}
                  <DashboardLicenseCard
                    appLicense={appLicense}
                    app={app}
                    syncCallback={() => getAppLicense(app)}
                    gettingAppLicenseErrMsg={gettingAppLicenseErrMsg}
                  />
                </div>
              </div>
              <div className="u-marginTop--30 flex flex1">
                <DashboardGraphsCard
                  prometheusAddress={dashboard?.prometheusAddress}
                  metrics={dashboard?.metrics}
                  appSlug={app.slug}
                  clusterId={cluster?.id}
                  isHelmManaged={isHelmManaged}
                />
              </div>
            </div>
          </div>
          {viewAirgapUploadError && (
            <Modal
              isOpen={viewAirgapUploadError}
              onRequestClose={toggleViewAirgapUploadError}
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
                    {airgapUploadError}
                  </p>
                </div>
                <button
                  type="button"
                  className="btn primary u-marginTop--15"
                  onClick={toggleViewAirgapUploadError}
                >
                  Ok, got it!
                </button>
              </div>
            </Modal>
          )}
          {viewAirgapUpdateError && (
            <Modal
              isOpen={viewAirgapUpdateError}
              onRequestClose={toggleViewAirgapUpdateError}
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
                    {airgapUpdateError}
                  </p>
                </div>
                <button
                  type="button"
                  className="btn primary u-marginTop--15"
                  onClick={toggleViewAirgapUpdateError}
                >
                  Ok, got it!
                </button>
              </div>
            </Modal>
          )}
          {showAppStatusModal && (
            <Modal
              isOpen={showAppStatusModal}
              onRequestClose={toggleAppStatusModal}
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
                    onClick={toggleAppStatusModal}
                  >
                    Ok, got it!
                  </button>
                  <button
                    type="button"
                    className="btn secondary blue u-marginLeft--10"
                    onClick={goToTroubleshootPage}
                  >
                    Troubleshoot
                  </button>
                </div>
              </div>
            </Modal>
          )}
          {showAutomaticUpdatesModal && (
            <AutomaticUpdatesModal
              isOpen={showAutomaticUpdatesModalState}
              onRequestClose={hideAutomaticUpdatesModal}
              updateCheckerSpec={app.updateCheckerSpec}
              autoDeploy={app.autoDeploy}
              appSlug={app.slug}
              isSemverRequired={app?.isSemverRequired}
              gitopsIsConnected={downstream?.gitops?.isConnected}
              onAutomaticUpdatesConfigured={() => {
                hideAutomaticUpdatesModal();
                refreshAppData();
              }}
            />
          )}
          {snapshotDifferencesModal && (
            <SnapshotDifferencesModal
              snapshotDifferencesModal={snapshotDifferencesModal}
              toggleSnapshotDifferencesModal={toggleSnaphotDifferencesModal}
            />
          )}
        </div>
      )}
    </>
  );
};

export default withRouter(Dashboard);
