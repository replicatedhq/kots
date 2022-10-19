import React, { useEffect, useReducer } from "react";
import { KotsPageTitle } from "@components/Head";
import get from "lodash/get";
import sortBy from "lodash/sortBy";
import Loader from "@src/components/shared/Loader";
import DashboardVersionCard from "./DashboardVersionCard";
import AppStatus from "./AppStatus";
import DashboardLicenseCard from "./DashboardLicenseCard";
import DashboardSnapshotsCard from "./DashboardSnapshotsCard";
import DashboardGraphsCard from "./DashboardGraphsCard";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";
import Modal from "react-modal";
import { Repeater } from "@src/utilities/repeater";
import { Utilities, isAwaitingResults } from "@src/utilities/utilities";
import { AirgapUploader } from "@src/utilities/airgapUploader";
import { useSelectedAppClusterDashboardWithIntercept } from "../api/useSelectedAppClusterDashboard";
import { useHistory, useRouteMatch } from "react-router-dom";
import { useLicenseWithIntercept } from "@features/App";

import "@src/scss/components/watches/Dashboard.scss";
import "@src/../node_modules/react-vis/dist/style";
import { Paragraph } from "@src/styles/common";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host",
};

// Types
import {
  App,
  AppLicense,
  Downstream,
  DashboardResponse,
  DashboardActionLink,
  ResourceStates,
  Version,
} from "@types";
//import LicenseTester from "./LicenseTester";

type Props = {
  app: App;
  cluster: {
    // TODO: figure out if this is actually a "" | number- maybe just go with number
    id: "" | number;
  };
  isBundleUploading: boolean;
  isHelmManaged: boolean;
  isVeleroInstalled: boolean;
  makeCurrentVersion: (
    slug: string,
    versionToDeploy: Version,
    isSkipPreflights: boolean,
    continueWithFailedPreflights: boolean
  ) => void;
  ping: (clusterId?: string) => void;
  redeployVersion: (
    upstreamSlug: string,
    version: Version | null
  ) => Promise<void>;
  refreshAppData: () => void;
  snapshotInProgressApps: string[];
  toggleIsBundleUploading: (isUploading: boolean) => void;
  updateCallback: () => void | null;
};

type SnapshotOption = {
  option: string;
  name: string;
};

// TODO:  update these strings so that they are not nullable (maybe just set default to "")
type State = {
  activeChart: string | null;
  airgapUpdateError: string;
  airgapUploader: AirgapUploader | null;
  airgapUploadError: string | null;
  appLicense: AppLicense | null;
  appName: string;
  checkingForUpdateError: boolean;
  checkingForUpdates: boolean;
  checkingUpdateMessage: string;
  currentVersion: Version | null;
  dashboard: DashboardResponse;
  displayErrorModal: boolean;
  downstream: Downstream | null;
  fetchAppDownstreamJob: Repeater;
  getAppDashboardJob: Repeater;
  gettingAppErrMsg: string;
  gettingAppLicenseErrMsg: string;
  iconUri: string;
  loadingApp: boolean;
  links: DashboardActionLink[];
  noUpdatesAvalable: boolean;
  selectedSnapshotOption: SnapshotOption;
  showAppStatusModal: boolean;
  showAutomaticUpdatesModal: boolean;
  snapshotDifferencesModal: boolean;
  startingSnapshot: boolean;
  startSnapshotErr: boolean;
  startSnapshotErrorMsg: string;
  startSnapshotOptions: SnapshotOption[];
  updateChecker: Repeater;
  uploadProgress: number;
  uploadResuming: boolean;
  uploadSize: number;
  uploadingAirgapFile: boolean;
  viewAirgapUpdateError: boolean;
  viewAirgapUploadError: boolean;
  slowLoader: boolean;
};

const Dashboard = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      activeChart: null,
      airgapUploader: null,
      airgapUpdateError: "",
      airgapUploadError: null,
      appLicense: null,
      appName: "",
      checkingForUpdateError: false,
      checkingForUpdates: false,
      checkingUpdateMessage: "Checking for updates",
      dashboard: {
        appStatus: null,
        metrics: [],
        prometheusAddress: "",
      },
      currentVersion: null,
      displayErrorModal: false,
      downstream: null,
      fetchAppDownstreamJob: new Repeater(),
      getAppDashboardJob: new Repeater(),
      gettingAppErrMsg: "",
      gettingAppLicenseErrMsg: "",
      iconUri: "",
      loadingApp: false,
      links: [],
      // TODO: fix misspelling of available
      noUpdatesAvalable: false,
      selectedSnapshotOption: { option: "full", name: "Start a Full snapshot" },
      showAppStatusModal: false,
      showAutomaticUpdatesModal: false,
      snapshotDifferencesModal: false,
      startingSnapshot: false,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
      startSnapshotOptions: [
        { option: "partial", name: "Start a Partial snapshot" },
        { option: "full", name: "Start a Full snapshot" },
        { option: "learn", name: "Learn about the difference" },
      ],
      updateChecker: new Repeater(),
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadResuming: false,
      uploadSize: 0,
      viewAirgapUpdateError: false,
      viewAirgapUploadError: false,
      slowLoader: false,
    }
  );

  const history = useHistory();
  const match = useRouteMatch();
  const { app, isBundleUploading, isVeleroInstalled } = props;

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
        const appResponse = await res.json();
        if (!isAwaitingResults(appResponse.downstream.pendingVersions)) {
          state.fetchAppDownstreamJob.stop();
        }
        setState({
          downstream: appResponse.downstream,
        });
        // wait a couple of seconds to avoid any race condiditons with the update checker then refetch the app to ensure we have the latest everything
        // this is hacky and I hate it but it's just building up more evidence in my case for having the FE be able to listen to BE envents
        // if that was in place we would have no need for this becuase the latest version would just be pushed down.
        setTimeout(() => {
          props.refreshAppData();
        }, 2000);
      } else {
        setState({
          loadingApp: false,
          gettingAppErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
      }
    } catch (err) {
      console.log(err);
      const errorMessage =
        err instanceof Error
          ? err.message
          : "Something went wrong, please try again.";
      setState({
        loadingApp: false,
        gettingAppErrMsg: errorMessage,
        displayErrorModal: true,
      });
    }
  };

  const startFetchAppDownstreamJob = () => {
    state.fetchAppDownstreamJob.start(fetchAppDownstream, 2000);
  };

  const setWatchState = (newAppState: App) => {
    setState({
      appName: newAppState.name,
      iconUri: newAppState.iconUri,
      currentVersion: newAppState.downstream?.currentVersion,
      downstream: newAppState.downstream,
      links: newAppState.downstream?.links,
    });
  };

  const { data: licenseWithInterceptResponse, refetch: getAppLicense, error: licenseWithInterceptError } =
    useLicenseWithIntercept();
  useEffect(() => {
    // if (!res.ok) {
    //   setState({ gettingAppLicenseErrMsg: body.error });
    //   return;
    // }
    if (!licenseWithInterceptResponse) {
      setState({ appLicense: null, gettingAppLicenseErrMsg: "" });
    } else if (licenseWithInterceptResponse.success) {
      setState({
        appLicense: licenseWithInterceptResponse.license,
        gettingAppLicenseErrMsg: "",
      });
    } else if (licenseWithInterceptResponse.error) {
      setState({
        appLicense: null,
        gettingAppLicenseErrMsg: licenseWithInterceptResponse.error,
      }) }else if (licenseWithInterceptError) {

        setState({ gettingAppLicenseErrMsg: licenseWithInterceptError?.message });

      };
    }
  }, [licenseWithInterceptResponse]);

  // const getAppLicense = async ({ slug }: { slug: string }) => {
  //   await fetch(`${process.env.API_ENDPOINT}/app/${slug}/license`, {
  //     method: "GET",
  //     headers: {
  //       Authorization: Utilities.getToken(),
  //       "Content-Type": "application/json",
  //     },
  //   })
  //     .then(async (res) => {
  //       const body = await res.json();
  //       if (!res.ok) {
  //         setState({ gettingAppLicenseErrMsg: body.error });
  //         return;
  //       }
  //       if (body === null) {
  //         setState({ appLicense: null, gettingAppLicenseErrMsg: "" });
  //       } else if (body.success) {
  //         setState({
  //           appLicense: body.license,
  //           gettingAppLicenseErrMsg: "",
  //         });
  //       } else if (body.error) {
  //         setState({
  //           appLicense: null,
  //           gettingAppLicenseErrMsg: body.error,
  //         });
  //       }
  //     })
  //     .catch((err) => {
  //       console.log(err);
  //       setState({
  //         gettingAppLicenseErrMsg: err
  //           ? `Error while getting the license: ${err.message}`
  //           : "Something went wrong, please try again.",
  //       });
  //     });
  // };

  useEffect(() => {
    if (props.app) {
      setWatchState(props.app);
      //  getAppLicense(props.app);
    }
  }, [props.app]);

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

  const onUploadError = (message: string) => {
    setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError: message || "Error uploading bundle, please try again",
    });
    props.toggleIsBundleUploading(false);
  };

  const updateStatus = (): Promise<void> => {
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

          if (response.status !== "running" && !props.isBundleUploading) {
            state.updateChecker.stop();

            setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed",
            });

            getAppLicense();
            if (props.updateCallback) {
              props.updateCallback();
            }
            startFetchAppDownstreamJob();
          } else {
            setState({
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

  const onUploadComplete = () => {
    state.updateChecker.start(updateStatus, 1000);
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
      airgapUploadError: null,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });

    props.toggleIsBundleUploading(true);

    const params = {
      appId: props.app?.id,
    };

    // TODO: remove after adding type to airgap uploader
    // eslint-disable-next-line
    // @ts-ignore
    state.airgapUploader?.upload(
      params,
      onUploadProgress,
      onUploadError,
      onUploadComplete
    );
  };

  const onProgressError = async (airgapUploadError: string) => {
    Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
      if (airgapUploadError.includes(errorString)) {
        airgapUploadError = message;
      }
    });
    setState({
      uploadingAirgapFile: false,
      airgapUploadError,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
  };

  const toggleViewAirgapUploadError = () => {
    setState({ viewAirgapUploadError: !state.viewAirgapUploadError });
  };

  const toggleViewAirgapUpdateError = (err?: string) => {
    setState({
      viewAirgapUpdateError: !state.viewAirgapUpdateError,
      airgapUpdateError: !state.viewAirgapUpdateError && err ? err : "",
    });
  };

  const startASnapshot = (option: string) => {
    setState({
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
            setState({
              startingSnapshot: false,
            });
            history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          setState({
            startingSnapshot: false,
          });
          props.ping();
          if (option === "full") {
            history.push("/snapshots");
          } else {
            history.push(`/snapshots/partial/${app.slug}`);
          }
        } else {
          const body = await result.json();
          setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: body.error,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        setState({
          startSnapshotErrorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  const onSnapshotOptionChange = (selectedSnapshotOption: SnapshotOption) => {
    if (selectedSnapshotOption.option === "learn") {
      setState({ snapshotDifferencesModal: true });
    } else {
      startASnapshot(selectedSnapshotOption.option);
    }
  };

  const toggleSnaphotDifferencesModal = () => {
    setState({
      snapshotDifferencesModal: !state.snapshotDifferencesModal,
    });
  };

  const onSnapshotOptionClick = () => {
    const { selectedSnapshotOption } = state;
    startASnapshot(selectedSnapshotOption.option);
  };

  const toggleAppStatusModal = () => {
    setState({ showAppStatusModal: !state.showAppStatusModal });
  };

  const goToTroubleshootPage = () => {
    history.push(`${match.url}/troubleshoot`);
  };

  const getAppResourcesByState = () => {
    const { appStatus } = state.dashboard;
    if (!appStatus?.resourceStates?.length) {
      return {};
    }

    const resourceStates = appStatus?.resourceStates;
    const statesMap: {
      [key: string]: ResourceStates[];
    } = {};

    for (let i = 0; i < resourceStates.length; i++) {
      const resourceState = resourceStates[i];
      if (!statesMap.hasOwnProperty(resourceState.state)) {
        statesMap[resourceState.state] = [];
      }
      statesMap[resourceState.state].push(resourceState);
    }

    // sort resources so that the order doesn't change while polling (since we show live data)
    Object.keys(statesMap).forEach((stateKey) => {
      statesMap[stateKey] = sortBy(statesMap[stateKey], (resource) => {
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
    return sortedStates?.every((sortedState) => {
      return statesMap[sortedState]?.every((resource) => {
        const { kind, name, namespace } = resource;
        if (kind === "EMPTY" && name === "EMPTY" && namespace === "EMPTY") {
          return false;
        }
        return true;
      });
    });
  };

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
  } = state;

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

  const { appStatus } = state.dashboard;

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

    setState({
      airgapUploader: new AirgapUploader(
        true,
        app.slug,
        onDropBundle,
        simultaneousUploads
      ),
    });
  };

  const { data: selectedAppClusterDashboardResponse } =
    useSelectedAppClusterDashboardWithIntercept({ refetchInterval: 2000 });

  useEffect(() => {
    if (selectedAppClusterDashboardResponse) {
      setState({
        dashboard: {
          appStatus: selectedAppClusterDashboardResponse.appStatus,
          prometheusAddress:
            selectedAppClusterDashboardResponse.prometheusAddress,
          metrics: selectedAppClusterDashboardResponse.metrics,
        },
      });
    }
  }, [selectedAppClusterDashboardResponse]);

  useEffect(() => {
    if (app?.isAirgap && !state.airgapUploader) {
      getAirgapConfig();
    }

    state.updateChecker.start(updateStatus, 1000);
    if (app) {
      setWatchState(app);
      getAppLicense();
    }
    return () => {
      state.updateChecker.stop();
      state.fetchAppDownstreamJob.stop();
    };
  }, []);

  const onCheckForUpdates = async () => {
    setState({
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
        getAppLicense();
        if (!res.ok) {
          const text = await res.text();
          setState({
            checkingForUpdateError: true,
            checkingForUpdates: false,
            checkingUpdateMessage: text
              ? text
              : "There was an error checking for updates.",
          });
          return;
        }

        const response = await res.json();
        if (response.availableUpdates === 0) {
          setState({
            checkingForUpdates: false,
            noUpdatesAvalable: true,
          });
          setTimeout(() => {
            setState({ noUpdatesAvalable: false });
          }, 3000);
        } else {
          state.updateChecker.start(updateStatus, 1000);
        }
      })
      .catch((err) => {
        setState({
          checkingForUpdateError: true,
          checkingForUpdates: false,
          checkingUpdateMessage: err?.message
            ? err?.message
            : "There was an error checking for updates.",
        });
      });
  };

  const hideAutomaticUpdatesModal = () => {
    setState({
      showAutomaticUpdatesModal: false,
    });
  };

  const showAutomaticUpdatesModal = () => {
    setState({
      showAutomaticUpdatesModal: true,
    });
  };

  return (
    <>
      {!app ||
        (state.slowLoader && (
          <div
            className="flex-column flex1 alignItems--center justifyContent--center"
            style={{
              position: "absolute",
              width: "100%",
              left: 0,
              right: 0,
              top: 0,
              bottom: 0,
              backgroundColor: "rgba(255,255,255,0.7",
              zIndex: 100,
            }}
          >
            <Loader size="60" />
          </div>
        ))}
      {app && (
        <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
          <KotsPageTitle pageName="Dashboard" showAppSlug />
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
                    appStatus={appStatus?.state}
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
                    checkingForUpdates={checkingForUpdates}
                    checkingUpdateText={checkingUpdateText}
                    airgapUploader={state.airgapUploader}
                    uploadingAirgapFile={uploadingAirgapFile}
                    airgapUploadError={airgapUploadError}
                    refetchData={props.updateCallback}
                    downloadCallback={startFetchAppDownstreamJob}
                    uploadProgress={state.uploadProgress}
                    uploadSize={state.uploadSize}
                    uploadResuming={state.uploadResuming}
                    makeCurrentVersion={props.makeCurrentVersion}
                    redeployVersion={props.redeployVersion}
                    onProgressError={onProgressError}
                    onCheckForUpdates={() => onCheckForUpdates()}
                    isBundleUploading={isBundleUploading}
                    checkingForUpdateError={state.checkingForUpdateError}
                    viewAirgapUploadError={() => toggleViewAirgapUploadError()}
                    showAutomaticUpdatesModal={showAutomaticUpdatesModal}
                    noUpdatesAvalable={state.noUpdatesAvalable}
                  />
                </div>

                <div className="flex1 flex-column u-paddingLeft--15">
                  {app.allowSnapshots && isVeleroInstalled ? (
                    <div className="u-marginBottom--30">
                      <DashboardSnapshotsCard
                        url={match.url}
                        app={app}
                        ping={props.ping}
                        isSnapshotAllowed={
                          app.allowSnapshots && isVeleroInstalled
                        }
                        isVeleroInstalled={isVeleroInstalled}
                        startASnapshot={startASnapshot}
                        startSnapshotOptions={state.startSnapshotOptions}
                        startSnapshotErr={state.startSnapshotErr}
                        startSnapshotErrorMsg={state.startSnapshotErrorMsg}
                        snapshotInProgressApps={props.snapshotInProgressApps}
                        selectedSnapshotOption={state.selectedSnapshotOption}
                        onSnapshotOptionChange={onSnapshotOptionChange}
                        onSnapshotOptionClick={onSnapshotOptionClick}
                      />
                    </div>
                  ) : null}
                  <DashboardLicenseCard
                    appLicense={appLicense}
                    app={app}
                    syncCallback={() => getAppLicense()}
                    refetchLicense={getAppLicense}
                    gettingAppLicenseErrMsg={state.gettingAppLicenseErrMsg}
                  >
                    {/* leaving this here as an example: please delete later */}
                    {/* <LicenseTester
                      appSlug={app.slug}
                      setLoader={(e: boolean) =>
                        setState({ slowLoader: e })
                      }
                    /> */}
                  </DashboardLicenseCard>
                </div>
              </div>
              <div className="u-marginTop--30 flex flex1">
                <DashboardGraphsCard
                  prometheusAddress={state.dashboard?.prometheusAddress}
                  metrics={state.dashboard?.metrics}
                  isHelmManaged={props.isHelmManaged}
                />
              </div>
            </div>
          </div>
          {state.viewAirgapUploadError && (
            <Modal
              isOpen={state.viewAirgapUploadError}
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
                    {state.airgapUploadError}
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
          {state.viewAirgapUpdateError && (
            <Modal
              isOpen={state.viewAirgapUpdateError}
              onRequestClose={() => toggleViewAirgapUpdateError()}
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
                    {state.airgapUpdateError}
                  </p>
                </div>
                <button
                  type="button"
                  className="btn primary u-marginTop--15"
                  onClick={() => toggleViewAirgapUpdateError()}
                >
                  Ok, got it!
                </button>
              </div>
            </Modal>
          )}
          {state.showAppStatusModal && (
            <Modal
              isOpen={state.showAppStatusModal}
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
                  {appResourcesByState?.sortedStates?.map((sortedState, i) => (
                    <div key={i}>
                      <p className="u-fontSize--normal u-color--mutedteal u-fontWeight--bold u-marginTop--20">
                        {Utilities.toTitleCase(sortedState)}
                      </p>
                      {appResourcesByState?.statesMap[sortedState]?.map(
                        (resource, j) => (
                          <div key={`${resource?.name}-${j}`}>
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
          {state.showAutomaticUpdatesModal && (
            <AutomaticUpdatesModal
              isOpen={state.showAutomaticUpdatesModal}
              onRequestClose={hideAutomaticUpdatesModal}
              updateCheckerSpec={app.updateCheckerSpec}
              autoDeploy={app.autoDeploy}
              appSlug={app.slug}
              isSemverRequired={app?.isSemverRequired}
              gitopsIsConnected={downstream?.gitops?.isConnected}
              onAutomaticUpdatesConfigured={() => {
                hideAutomaticUpdatesModal();
                props.refreshAppData();
              }}
              isHelmManaged={props.isHelmManaged}
            />
          )}
          {state.snapshotDifferencesModal && (
            <SnapshotDifferencesModal
              snapshotDifferencesModal={state.snapshotDifferencesModal}
              toggleSnapshotDifferencesModal={toggleSnaphotDifferencesModal}
            />
          )}
        </div>
      )}
    </>
  );
};

export { Dashboard };
