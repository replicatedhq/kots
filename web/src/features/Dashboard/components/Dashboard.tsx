import React, { useEffect, useReducer, useRef } from "react";
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
import { Utilities } from "@src/utilities/utilities";
import { AirgapUploader } from "@src/utilities/airgapUploader";
import { useSelectedAppClusterDashboardWithIntercept } from "../api/useSelectedAppClusterDashboard";
import { useHistory, useRouteMatch } from "react-router-dom";
import { useLicenseWithIntercept } from "@features/App";
import { useNextAppVersionWithIntercept } from "../api/useNextAppVersion";

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
import {
  UpdateStatusResponse,
  useUpdateDownloadStatus,
} from "../api/getUpdateDownloadStatus";
import { useAppDownstream } from "../api/getAppDownstream";
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
  selectedSnapshotOption: SnapshotOption;
  showAppStatusModal: boolean;
  showAutomaticUpdatesModal: boolean;
  snapshotDifferencesModal: boolean;
  startingSnapshot: boolean;
  startSnapshotErr: boolean;
  startSnapshotErrorMsg: string;
  startSnapshotOptions: SnapshotOption[];
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
  const airgapUploader = useRef<AirgapUploader | null>(null);

  const timer = useRef<NodeJS.Timeout[]>([]);

  const onAppDownstreamSuccess = (data: Downstream) => {
    setState({ downstream: data });
    let timerId = setTimeout(() => {
      props.refreshAppData();
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

    if (props.updateCallback) {
      props.updateCallback();
    }
    refetchAppDownstream();
  };

  const onUpdateDownloadStatusError = (data: Error) => {
    setState({
      checkingForUpdates: false,
      checkingForUpdateError: true,
      checkingUpdateMessage: data.message,
    });
  };

  const { refetch: refetchUpdateDownloadStatus } = useUpdateDownloadStatus(
    onUpdateDownloadStatusSuccess,
    onUpdateDownloadStatusError,
    props.isBundleUploading
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
    if (props.app) {
      setWatchState(props.app);
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

  const onUploadComplete = () => {
    refetchUpdateDownloadStatus();
    setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      checkingForUpdates: true,
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
    airgapUploader.current?.upload(
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

    airgapUploader.current = new AirgapUploader(
      true,
      app.slug,
      onDropBundle,
      simultaneousUploads
    );
  };

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
          prometheusAddress:
            selectedAppClusterDashboardResponse.prometheusAddress,
          metrics: selectedAppClusterDashboardResponse.metrics,
        },
      });
    }
  }, [selectedAppClusterDashboardResponse]);

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
          let timerId = setTimeout(() => {
            setState({ noUpdatesAvalable: false });
          }, 3000);
          timer.current.push(timerId);
        } else {
          refetchUpdateDownloadStatus();
          setState({ checkingForUpdates: true });
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
                    airgapUploader={airgapUploader.current}
                    uploadingAirgapFile={uploadingAirgapFile}
                    airgapUploadError={airgapUploadError}
                    refetchData={props.updateCallback}
                    downloadCallback={refetchAppDownstream}
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
