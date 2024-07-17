import { useEffect, useReducer, useRef } from "react";
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
import Modal from "react-modal";
import { Repeater } from "@src/utilities/repeater";
import { Utilities } from "@src/utilities/utilities";
import { AirgapUploader } from "@src/utilities/airgapUploader";
import { useSelectedAppClusterDashboardWithIntercept } from "../api/useSelectedAppClusterDashboard";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { useLicenseWithIntercept } from "@features/App";
import { useNextAppVersionWithIntercept } from "../api/useNextAppVersion";

import "@src/scss/components/watches/Dashboard.scss";
import "@src/../node_modules/react-vis/dist/style";
import { Paragraph } from "@src/styles/common";
// Types
import {
  App,
  AppLicense,
  DashboardActionLink,
  DashboardResponse,
  Downstream,
  Metadata,
  ResourceStates,
  Version,
} from "@types";
import {
  UpdateStatusResponse,
  useUpdateDownloadStatus,
} from "../api/getUpdateDownloadStatus";
import { useAppDownstream } from "../api/getAppDownstream";
import { useAirgapConfig } from "../api/getAirgapConfig";
import { Updates, useCheckForUpdates } from "../api/getUpdates";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host",
};

type Props = {
  adminConsoleMetadata: Metadata | null;
};

type OutletContext = {
  app: App;
  cluster: {
    // TODO: figure out if this is actually a "" | number- maybe just go with number
    id: "" | number;
  };
  isBundleUploading: boolean;
  isEmbeddedCluster: boolean;
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
  toggleIsBundleUploading: (isUploading: boolean) => void;
  updateCallback: () => void | null;
  showUpgradeStatusModal: boolean;
};

// TODO:  update these strings so that they are not nullable (maybe just set default to "")
type State = {
  activeChart: string | null;
  airgapUpdateError: string;
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
  getAppDashboardJob: Repeater;
  gettingAppErrMsg: string;
  gettingAppLicenseErrMsg: string;
  iconUri: string;
  loadingApp: boolean;
  links: DashboardActionLink[];
  noUpdatesAvalable: boolean;
  showAppStatusModal: boolean;
  showAutomaticUpdatesModal: boolean;
  uploadProgress: number;
  uploadResuming: boolean;
  uploadSize: number;
  uploadingAirgapFile: boolean;
  viewAirgapUpdateError: boolean;
  viewAirgapUploadError: boolean;
  slowLoader: boolean;
  lastUpdated: number;
  lastUpdatedDate: Date;
};

const Dashboard = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      activeChart: null,
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
        embeddedClusterState: "",
      },
      currentVersion: null,
      displayErrorModal: false,
      downstream: null,
      getAppDashboardJob: new Repeater(),
      gettingAppErrMsg: "",
      gettingAppLicenseErrMsg: "",
      iconUri: "",
      loadingApp: false,
      links: [],
      // TODO: fix misspelling of available
      noUpdatesAvalable: false,
      showAppStatusModal: false,
      showAutomaticUpdatesModal: false,
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadResuming: false,
      uploadSize: 0,
      viewAirgapUpdateError: false,
      viewAirgapUploadError: false,
      slowLoader: false,
      lastUpdated: 0,
      lastUpdatedDate: new Date(),
    }
  );

  const navigate = useNavigate();

  const {
    app,
    isBundleUploading,
    isEmbeddedCluster,
    isVeleroInstalled,
    makeCurrentVersion,
    ping,
    redeployVersion,
    refreshAppData,
    toggleIsBundleUploading,
    updateCallback,
    showUpgradeStatusModal,
  }: OutletContext = useOutletContext();
  const params = useParams();
  const airgapUploader = useRef<AirgapUploader | null>(null);

  const timer = useRef<NodeJS.Timeout[]>([]);

  const onAppDownstreamSuccess = (data: Downstream) => {
    setState({ downstream: data });
    let timerId = setTimeout(() => {
      refreshAppData();
    }, 2000);
    timer.current.push(timerId);
  };

  const onAppDownstreamError = (data: { message: string }) => {
    setState({
      loadingApp: false,
      gettingAppErrMsg: data.message,
      displayErrorModal: true,
    });
  };

  const { refetch: refetchAppDownstream } = useAppDownstream(
    onAppDownstreamSuccess,
    onAppDownstreamError
  );

  const setWatchState = (newAppState: App) => {
    setState({
      currentVersion: newAppState.downstream?.currentVersion,
      appName:
        newAppState.downstream?.currentVersion?.appTitle || newAppState.name,
      iconUri:
        newAppState.downstream?.currentVersion?.appIconUri ||
        newAppState.iconUri,
      downstream: newAppState.downstream,
      links: newAppState.downstream?.links,
    });
  };

  const {
    data: licenseWithInterceptResponse,
    error: licenseWithInterceptError,
    refetch: getAppLicense,
    isSlowLoading: isSlowLoadingLicense,
  } = useLicenseWithIntercept();

  const onUpdateDownloadStatusSuccess = (data: UpdateStatusResponse) => {
    setState({
      checkingForUpdateError: data.status === "failed",
      checkingForUpdates: data.status === "running",
      checkingUpdateMessage: data.currentMessage,
    });
    getAppLicense();

    if (updateCallback) {
      updateCallback();
    }
    refetchAppDownstream();
  };

  const onUpdateDownloadStatusError = (data: Error) => {
    if (showUpgradeStatusModal) {
      // if an upgrade is in progress, we don't want to show an error
      return;
    }

    setState({
      checkingForUpdates: false,
      checkingForUpdateError: true,
      checkingUpdateMessage: data.message,
    });
  };

  const { refetch: refetchUpdateDownloadStatus } = useUpdateDownloadStatus(
    onUpdateDownloadStatusSuccess,
    onUpdateDownloadStatusError,
    isBundleUploading
  );

  useEffect(() => {
    if (!licenseWithInterceptResponse) {
      setState({ appLicense: null, gettingAppLicenseErrMsg: "" });
      return;
    }

    if (licenseWithInterceptResponse.success) {
      setState({
        appLicense: licenseWithInterceptResponse.license,
        gettingAppLicenseErrMsg: "",
      });
      return;
    }
    if (licenseWithInterceptResponse.error) {
      setState({
        appLicense: null,
        gettingAppLicenseErrMsg: licenseWithInterceptResponse.error,
      });
      return;
    }
    if (licenseWithInterceptError instanceof Error) {
      setState({ gettingAppLicenseErrMsg: licenseWithInterceptError?.message });
      return;
    }

    setState({
      gettingAppLicenseErrMsg: "Something went wrong, please try again.",
    });
  }, [licenseWithInterceptResponse]);

  useEffect(() => {
    if (app) {
      setWatchState(app);
    }
  }, [app]);

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
    toggleIsBundleUploading(false);
  };

  const onUploadComplete = () => {
    refetchUpdateDownloadStatus();
    setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      checkingForUpdates: true,
    });
    toggleIsBundleUploading(false);
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

    toggleIsBundleUploading(true);

    const processParams = {
      appId: app?.id,
    };

    // TODO: remove after adding type to airgap uploader
    // eslint-disable-next-line
    // @ts-ignore
    airgapUploader.current?.upload(
      processParams,
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

  const toggleAppStatusModal = () => {
    setState({ showAppStatusModal: !state.showAppStatusModal });
  };

  const goToTroubleshootPage = () => {
    navigate(`/app/${params.slug}/troubleshoot`);
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
      if (s === "failed") {
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
      if (s === "success") {
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

  const onAirgapConfigSuccess = (simultaneousUploads: Number) => {
    airgapUploader.current = new AirgapUploader(
      true,
      app.slug,
      onDropBundle,
      simultaneousUploads
    );
  };

  const { refetch: getAirgapConfig } = useAirgapConfig(onAirgapConfigSuccess);

  const {
    data: selectedAppClusterDashboardResponse,
    isSlowLoading: isSlowLoadingSelectedAppClusterDashboard,
  } = useSelectedAppClusterDashboardWithIntercept({ refetchInterval: 2000 });

  const { isSlowLoading: isSlowLoadingNextAppVersion } =
    useNextAppVersionWithIntercept();

  // show slow loader if any of the apis are slow loading
  useEffect(() => {
    // since this is new and we may need to debug it, leaving these in for now
    // console.log("isSlowLoadingLicense", isSlowLoadingLicense);
    // console.log("isSlowLoadingNextAppVersion", isSlowLoadingNextAppVersion);
    // console.log(
    //   "isSlowLoadingSelectedAppClusterDashboard",
    //   isSlowLoadingSelectedAppClusterDashboard
    // );
    if (
      !state.slowLoader &&
      (isSlowLoadingLicense ||
        isSlowLoadingNextAppVersion ||
        isSlowLoadingSelectedAppClusterDashboard)
    ) {
      setState({ slowLoader: true });
      return;
    }
    if (
      state.slowLoader &&
      !isSlowLoadingLicense &&
      !isSlowLoadingNextAppVersion &&
      !isSlowLoadingSelectedAppClusterDashboard
    ) {
      setState({ slowLoader: false });
    }
  }, [
    isSlowLoadingLicense,
    isSlowLoadingNextAppVersion,
    isSlowLoadingSelectedAppClusterDashboard,
  ]);

  useEffect(() => {
    if (selectedAppClusterDashboardResponse) {
      setState({
        dashboard: {
          appStatus: selectedAppClusterDashboardResponse.appStatus,
          embeddedClusterState:
            selectedAppClusterDashboardResponse.embeddedClusterState,
          prometheusAddress:
            selectedAppClusterDashboardResponse.prometheusAddress,
          metrics: selectedAppClusterDashboardResponse.metrics,
        },
        lastUpdated: 0,
        lastUpdatedDate: new Date(),
      });
    }
  }, [selectedAppClusterDashboardResponse]);

  useEffect(() => {
    const interval = setInterval(() => {
      const prevDate: Date = state.lastUpdatedDate;
      const now: Date = new Date();
      const diffMs = now.getTime() - prevDate.getTime();
      const diffMins = diffMs / 1000 / 60;
      setState({ lastUpdated: diffMins });
    }, 1000);
    return () => {
      clearInterval(interval);
    };
  }, [state.lastUpdatedDate]);

  useEffect(() => {
    if (app?.isAirgap && !airgapUploader.current) {
      getAirgapConfig();
    }

    if (app) {
      setWatchState(app);
      getAppLicense();
    }
    return () => {
      timer.current.forEach((time: NodeJS.Timeout) => {
        clearTimeout(time);
      });
    };
  }, []);

  const onSuccess = (response: Updates) => {
    if (response.availableUpdates === 0) {
      setState({
        checkingForUpdates: false,
        noUpdatesAvalable: true,
      });
      getAppLicense();
      let timerId = setTimeout(() => {
        setState({ noUpdatesAvalable: false });
      }, 3000);
      timer.current.push(timerId);
    } else {
      refetchUpdateDownloadStatus();
      setState({ checkingForUpdates: true });
    }
  };

  const onError = (err: Error) => {
    if (showUpgradeStatusModal) {
      // if an upgrade is in progress, we don't want to show an error
      return;
    }

    setState({
      checkingForUpdateError: true,
      checkingForUpdates: false,
      checkingUpdateMessage: err?.message
        ? err?.message
        : "There was an error checking for updates.",
    });
    getAppLicense();
  };
  const { refetch: checkForUpdates } = useCheckForUpdates(onSuccess, onError);

  const onCheckForUpdates = async () => {
    setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
    });
    checkForUpdates();
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
                  <p className="u-fontSize--30 u-textColor--primary u-fontWeight--bold break-word">
                    {appName}
                  </p>
                  <AppStatus
                    appStatus={appStatus?.state}
                    onViewAppStatusDetails={toggleAppStatusModal}
                    links={links}
                    app={app}
                    hasStatusInformers={hasStatusInformers}
                    embeddedClusterState={state.dashboard.embeddedClusterState}
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
                    airgapUploader={airgapUploader.current}
                    uploadingAirgapFile={uploadingAirgapFile}
                    airgapUploadError={airgapUploadError}
                    refetchData={updateCallback}
                    downloadCallback={refetchAppDownstream}
                    uploadProgress={state.uploadProgress}
                    uploadSize={state.uploadSize}
                    uploadResuming={state.uploadResuming}
                    makeCurrentVersion={makeCurrentVersion}
                    redeployVersion={redeployVersion}
                    onProgressError={onProgressError}
                    onCheckForUpdates={() => onCheckForUpdates()}
                    isBundleUploading={isBundleUploading}
                    checkingForUpdateError={state.checkingForUpdateError}
                    viewAirgapUploadError={() => toggleViewAirgapUploadError()}
                    showAutomaticUpdatesModal={showAutomaticUpdatesModal}
                    noUpdatesAvalable={state.noUpdatesAvalable}
                    showUpgradeStatusModal={showUpgradeStatusModal}
                    adminConsoleMetadata={props.adminConsoleMetadata}
                  />
                </div>

                <div className="flex1 flex-column u-paddingLeft--15">
                  {app.allowSnapshots && isVeleroInstalled ? (
                    <div className="u-marginBottom--30">
                      <DashboardSnapshotsCard
                        url={params.url}
                        app={app}
                        ping={ping}
                        isSnapshotAllowed={
                          app.allowSnapshots && isVeleroInstalled
                        }
                        isEmbeddedCluster={isEmbeddedCluster}
                      />
                    </div>
                  ) : null}
                  <DashboardLicenseCard
                    appLicense={appLicense}
                    app={app}
                    syncCallback={() => getAppLicense()}
                    refetchLicense={getAppLicense}
                    gettingAppLicenseErrMsg={state.gettingAppLicenseErrMsg}
                  />
                </div>
              </div>
              {!isEmbeddedCluster && (
                <div className="u-marginTop--30 flex flex1">
                  <DashboardGraphsCard
                    prometheusAddress={state.dashboard?.prometheusAddress}
                    metrics={state.dashboard?.metrics}
                  />
                </div>
              )}
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
                <p className="tw-text-xs tw-pt-2">
                  Last Updated:{" "}
                  {state.lastUpdated < 1
                    ? "less than a minute ago"
                    : state.lastUpdated === 1
                    ? "1 minute ago"
                    : `${Math.round(state.lastUpdated)} minutes ago`}
                </p>
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
                              className={`ResourceStateText status-tag u-fontSize--normal ${resource.state}`}
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
                refreshAppData();
              }}
            />
          )}
        </div>
      )}
    </>
  );
};

export { Dashboard };
