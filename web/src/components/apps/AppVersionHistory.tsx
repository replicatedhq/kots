import { ChangeEvent, Component, Fragment, createRef } from "react";
import { Link } from "react-router-dom";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import find from "lodash/find";
import isEmpty from "lodash/isEmpty";
import get from "lodash/get";
import MountAware from "../shared/MountAware";
import Loader from "../shared/Loader";
import VersionDiff from "@src/features/VersionDiff/VersionDiff";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import AirgapUploadProgress from "@src/features/Dashboard/components/AirgapUploadProgress";
import ErrorModal from "../modals/ErrorModal";
import { AppVersionHistoryRow } from "@features/AppVersionHistory/AppVersionHistoryRow";
import DeployWarningModal from "../shared/modals/DeployWarningModal";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";
import SkipPreflightsModal from "../shared/modals/SkipPreflightsModal";
import {
  getCommitHashFromUrl,
  getGitProviderDiffUrl,
  getPreflightResultState,
  isAwaitingResults,
  secondsAgo,
  Utilities,
} from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import { AirgapUploader } from "../../utilities/airgapUploader";
import ReactTooltip from "react-tooltip";
import Pager from "../shared/Pager";
import { KotsPageTitle } from "@components/Head";

import "@src/scss/components/apps/AppVersionHistory.scss";
import { DashboardGitOpsCard } from "@features/Dashboard";
import Icon from "../Icon";
import { App, AvailableUpdate, Version, VersionDownloadStatus } from "@types";
import { RouterProps, withRouter } from "@src/utilities/react-router-utilities";
import PreflightIcon from "@features/App/PreflightIcon";
import ReleaseNotesModal from "@components/modals/ReleaseNotesModal";
import DiffErrorModal from "@components/modals/DiffErrorModal";
import ConfirmDeploymentModal from "@components/modals/ConfirmDeploymentModal";
import DisplayKotsUpdateModal from "@components/modals/DisplayKotsUpdateModal";
import NoChangesModal from "@components/modals/NoChangesModal";
import UpgradeServiceModal from "@components/modals/UpgradeServiceModal";

dayjs.extend(relativeTime);

type Release = {
  sequence?: number;
  versionLabel?: string;
};

type ReleaseWithError = {
  diffSummaryError?: string;
  sequence: number;
  title?: string;
};

type Props = {
  outletContext: {
    adminConsoleMetadata: {
      isAirgap: boolean;
      isKurl: boolean;
      isEmbeddedCluster: boolean;
    };
    app: App;
    displayErrorModal: boolean;
    isBundleUploading: boolean;
    isEmbeddedCluster: boolean;
    makeCurrentVersion: (
      slug: string,
      version: Version | null,
      isSkipPreflights: boolean,
      continueWithFailedPreflights: boolean
    ) => void;
    makingCurrentRelease: boolean;
    makingCurrentVersionErrMsg: string;
    redeployVersion: (slug: string, version: Version | null) => void;
    redeployVersionErrMsg: string;
    resetMakingCurrentReleaseErrorMessage: () => void;
    resetRedeployErrorMessage: () => void;
    refreshAppData: () => void;
    toggleErrorModal: () => void;
    toggleIsBundleUploading: (isUploading: boolean) => void;
    updateCallback: () => void;
  };
} & RouterProps;

type State = {
  airgapUploader: AirgapUploader | null;
  airgapUploadError: string;
  availableUpdates: AvailableUpdate[] | null;
  appUpdateChecker: Repeater;
  checkedReleasesToDiff: Version[];
  checkingForUpdateError: boolean;
  checkingForUpdates: boolean;
  checkingUpdateMessage: string;
  confirmType: string;
  currentPage: Number;
  deployView: boolean;
  diffHovered: boolean;
  displayConfirmDeploymentModal: boolean;
  displayErrorModal: boolean;
  displayKotsUpdateModal: boolean;
  displayShowDetailsModal: boolean;
  errorMsg: string;
  errorTitle: string;
  firstSequence: Number | string;
  isFetchingAvailableUpdates: boolean;
  isStartingUpgradeService: boolean;
  isSkipPreflights: boolean;
  kotsUpdateChecker: Repeater;
  kotsUpdateError: Object | undefined;
  kotsUpdateMessage: string;
  kotsUpdateRunning: boolean;
  kotsUpdateStatus: string;
  loadingPage: boolean;
  loadingVersionHistory: boolean;
  logs: Object | null;
  logsLoading: boolean;
  noUpdateAvailiableText: string;
  numOfRemainingVersions: number;
  numOfSkippedVersions: number;
  pageSize: Number;
  preflightState: {
    preflightsFailed: boolean;
    preflightState: string;
  } | null;
  releaseNotes: Object | null;
  releaseWithErr: ReleaseWithError | null | undefined;
  releaseWithNoChanges: Release | null | undefined;
  secondSequence: Number | string;
  selectedDiffReleases: boolean;
  selectedSequence: Number;
  selectedTab: Object | null;
  showAutomaticUpdatesModal: boolean;
  showDeployWarningModal: boolean;
  showDiffErrModal: boolean;
  showDiffOverlay: boolean;
  showLogsModal: boolean;
  showNoChangesModal: boolean;
  showSkipModal: boolean;
  totalCount: Number;
  updatesAvailable: boolean;
  uploadingAirgapFile: boolean;
  uploadProgress: Number;
  uploadResuming: boolean;
  uploadSize: Number;
  upgradeServiceChecker: Repeater;
  upgradeServiceStatus: string;
  versionDownloadStatuses: {
    [x: number]: VersionDownloadStatus;
  };
  versionHistory: Version[];
  versionHistoryJob: Repeater;
  versionToDeploy: Version | null;
  viewLogsErrMsg: string;
  yamlErrorDetails: string[];
  shouldShowUpgradeServiceModal: boolean;
  upgradeService: {
    versionLabel?: string;
    isLoading?: boolean;
    error?: string;
  } | null;
};

class AppVersionHistory extends Component<Props, State> {
  iframeRef: any;
  constructor(props: Props) {
    super(props);
    this.state = {
      airgapUploader: null,
      airgapUploadError: "",
      availableUpdates: null,
      appUpdateChecker: new Repeater(),
      checkedReleasesToDiff: [],
      checkingForUpdateError: false,
      checkingForUpdates: false,
      checkingUpdateMessage: "Checking for updates",
      confirmType: "",
      currentPage: 0,
      deployView: false,
      diffHovered: false,
      displayConfirmDeploymentModal: false,
      displayErrorModal: false,
      displayKotsUpdateModal: false,
      displayShowDetailsModal: false,
      errorMsg: "",
      errorTitle: "",
      firstSequence: 0,
      isFetchingAvailableUpdates: false,
      isStartingUpgradeService: false,
      isSkipPreflights: false,
      kotsUpdateChecker: new Repeater(),
      kotsUpdateError: undefined,
      kotsUpdateMessage: "",
      kotsUpdateRunning: false,
      kotsUpdateStatus: "",
      loadingPage: false,
      loadingVersionHistory: true,
      logs: null,
      logsLoading: false,
      noUpdateAvailiableText: "",
      numOfRemainingVersions: 0,
      numOfSkippedVersions: 0,
      pageSize: 20,
      preflightState: null,
      releaseNotes: null,
      releaseWithErr: { title: "", sequence: 0, diffSummaryError: "" },
      releaseWithNoChanges: { versionLabel: "", sequence: 0 },
      secondSequence: 0,
      selectedDiffReleases: false,
      selectedSequence: -1,
      selectedTab: null,
      showAutomaticUpdatesModal: false,
      showDeployWarningModal: false,
      showDiffErrModal: false,
      showDiffOverlay: false,
      showLogsModal: false,
      showNoChangesModal: false,
      showSkipModal: false,
      totalCount: 0,
      updatesAvailable: false,
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadResuming: false,
      uploadSize: 0,
      upgradeServiceChecker: new Repeater(),
      upgradeServiceStatus: "",
      versionDownloadStatuses: {},
      versionHistory: [],
      versionHistoryJob: new Repeater(),
      versionToDeploy: null,
      viewLogsErrMsg: "",
      yamlErrorDetails: [],
      shouldShowUpgradeServiceModal: false,
      upgradeService: {},
    };
    this.iframeRef = createRef<HTMLIFrameElement>();
  }

  // moving this out of the state because new repeater instances were getting created
  // and it doesn't really affect the UI
  versionDownloadStatusJobs: { [key: number]: Repeater } = {};

  _mounted: boolean | undefined;

  handleIframeMessage = (event) => {
    if (event.data.message === "closeUpgradeModal") {
      this.setState({ shouldShowUpgradeServiceModal: false });
      this.props.navigate(`/app/${this.props.params.slug}`);
    }
  };

  componentDidMount() {
    this.getPreflightState(
      this.props.outletContext.app.downstream.currentVersion
    );
    const urlParams = new URLSearchParams(window.location.search);
    const pageNumber = urlParams.get("page");
    if (pageNumber) {
      this.setState({ currentPage: parseInt(pageNumber) });
    } else {
      this.props.navigate(`${this.props.location.pathname}?page=0`);
    }

    this.fetchKotsDownstreamHistory();
    this.props.outletContext.refreshAppData();
    if (this.props.outletContext.app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig();
    }

    // check if there are any updates in progress
    this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);

    const url = window.location.pathname;
    const { params } = this.props;
    if (url.includes("/diff")) {
      const firstSequence = params.firstSequence;
      const secondSequence = params.secondSequence;
      if (firstSequence !== undefined && secondSequence !== undefined) {
        // undefined because a sequence can be zero!
        this.setState({ showDiffOverlay: true, firstSequence, secondSequence });
      }
    }

    if (this.props.outletContext.isEmbeddedCluster) {
      this.fetchAvailableUpdates();
    }
  }

  componentDidUpdate = async (lastProps: Props) => {
    if (this.state.shouldShowUpgradeServiceModal && this.iframeRef.current) {
      window.addEventListener("message", this.handleIframeMessage);
    }

    if (
      lastProps.params.slug !== this.props.params.slug ||
      lastProps.outletContext.app.id !== this.props.outletContext.app.id
    ) {
      this.fetchKotsDownstreamHistory();
    }
    if (
      this.props.outletContext.app.downstream.pendingVersions.length > 0 &&
      this.state.updatesAvailable === false
    ) {
      this.setState({ updatesAvailable: true });
    }
    if (
      this.props.outletContext.app.downstream.pendingVersions.length === 0 &&
      this.state.updatesAvailable === true
    ) {
      this.setState({ updatesAvailable: false });
    }
  };

  componentWillUnmount() {
    this.state.appUpdateChecker.stop();
    this.state.versionHistoryJob.stop();
    for (const j in this.versionDownloadStatusJobs) {
      this.versionDownloadStatusJobs[j].stop();
    }
    this._mounted = false;
    window.removeEventListener("message", this.handleIframeMessage);
  }

  fetchAvailableUpdates = async () => {
    const appSlug = this.props.params.slug;
    this.setState({ isFetchingAvailableUpdates: true });
    const res = await fetch(
      `${process.env.API_ENDPOINT}/app/${appSlug}/updates`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      }
    );
    if (!res.ok) {
      this.setState({ isFetchingAvailableUpdates: false });
      return;
    }
    const response = await res.json();

    this.setState({
      isFetchingAvailableUpdates: false,
      availableUpdates: response.updates,
    });
    return response;
  };

  fetchKotsDownstreamHistory = async () => {
    const appSlug = this.props.params.slug;

    this.setState({
      loadingVersionHistory: true,
      errorTitle: "",
      errorMsg: "",
      displayErrorModal: false,
    });

    try {
      const { currentPage, pageSize } = this.state;
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${appSlug}/versions?currentPage=${currentPage}&pageSize=${pageSize}&pinLatestDeployable=true`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          method: "GET",
          credentials: "include",
        }
      );
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

      if (isAwaitingResults(versionHistory) && this._mounted) {
        this.state.versionHistoryJob.start(
          this.fetchKotsDownstreamHistory,
          2000
        );
      } else {
        this.state.versionHistoryJob.stop();
      }

      this.setState({
        loadingVersionHistory: false,
        versionHistory: versionHistory,
        numOfSkippedVersions: response.numOfSkippedVersions,
        numOfRemainingVersions: response.numOfRemainingVersions,
        totalCount: response.totalCount,
      });
    } catch (err) {
      this.setState({
        loadingVersionHistory: false,
        errorTitle: "Failed to get version history",
        errorMsg: err
          ? (err as Error).message
          : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  setPageSize = (e: ChangeEvent<HTMLSelectElement>) => {
    this.setState(
      { pageSize: parseInt(e.target.value), currentPage: 0 },
      () => {
        this.fetchKotsDownstreamHistory();
        this.props.navigate(`${this.props.location.pathname}?page=0`);
      }
    );
  };

  getAirgapConfig = async () => {
    const { app } = this.props.outletContext;
    const configUrl = `${process.env.API_ENDPOINT}/app/${app.slug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
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

  onDropBundle = async () => {
    this.setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: "",
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });

    this.props.outletContext.toggleIsBundleUploading(true);

    const params = {
      appId: this.props.outletContext.app?.id,
    };
    this.state.airgapUploader?.upload(
      params,
      this.onUploadProgress,
      this.onUploadError,
      this.onUploadComplete
    );
  };

  onUploadProgress = (progress: number, size: number, resuming = false) => {
    this.setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  };

  onUploadError = (message: String) => {
    this.setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError:
        (message as string) || "Error uploading bundle, please try again",
    });
    this.props.outletContext.toggleIsBundleUploading(false);
  };

  onUploadComplete = () => {
    this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);
    this.setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.outletContext.toggleIsBundleUploading(false);
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  showReleaseNotes = (notes: string) => {
    this.setState({
      releaseNotes: notes,
    });
  };

  hideReleaseNotes = () => {
    this.setState({
      releaseNotes: null,
    });
  };

  toggleDiffErrModal = (release?: ReleaseWithError) => {
    this.setState({
      showDiffErrModal: !this.state.showDiffErrModal,
      releaseWithErr: !this.state.showDiffErrModal ? release : null,
    });
  };

  toggleAutomaticUpdatesModal = () => {
    this.setState({
      showAutomaticUpdatesModal: !this.state.showAutomaticUpdatesModal,
    });
  };

  toggleNoChangesModal = (version?: Version) => {
    this.setState({
      showNoChangesModal: !this.state.showNoChangesModal,
      releaseWithNoChanges: !this.state.showNoChangesModal ? version : {},
    });
  };

  renderLogsTabs = () => {
    const { logs, selectedTab } = this.state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs
          .filter((tab) => tab !== "renderError")
          .map((tab) => (
            <div
              className={`tab-item blue ${tab === selectedTab && "is-active"}`}
              key={tab}
              onClick={() => this.setState({ selectedTab: tab })}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  };

  downloadVersion = (version: Version) => {
    const { app } = this.props.outletContext;

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
        },
      },
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/download`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "POST",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          this.setState({
            versionDownloadStatuses: {
              ...this.state.versionDownloadStatuses,
              [version.sequence]: {
                downloadingVersion: false,
                downloadingVersionMessage: response.error,
                downloadingVersionError: true,
              },
            },
          });
          return;
        }
        this.versionDownloadStatusJobs[version.sequence].start(
          () => this.updateVersionDownloadStatus(version),
          1000
        );
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          versionDownloadStatuses: {
            ...this.state.versionDownloadStatuses,
            [version.sequence]: {
              downloadingVersion: false,
              downloadingVersionMessage:
                err?.message || "Something went wrong, please try again.",
              downloadingVersionError: true,
            },
          },
        });
      });
  };

  upgradeAdminConsole = (version: Version) => {
    const { app } = this.props.outletContext;

    this.setState({
      displayKotsUpdateModal: true,
      kotsUpdateRunning: true,
      kotsUpdateStatus: "",
      kotsUpdateMessage: "",
      kotsUpdateError: undefined,
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${app.slug}/sequence/${version.parentSequence}/update-console`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "POST",
      }
    )
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
          kotsUpdateError:
            err?.message || "Something went wrong, please try again.",
        });
      });
  };

  getKotsUpdateStatus = () => {
    const { app } = this.props.outletContext;

    return new Promise<void>((resolve) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app.slug}/task/update-admin-console`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      )
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
  };

  updateVersionDownloadStatus = (version: Version) => {
    const { app } = this.props.outletContext;

    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/sequence/${version?.parentSequence}/task/updatedownload`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      )
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
                  downloadingVersionError: response.status === "failed",
                },
              },
            });

            if (this.props.outletContext.updateCallback) {
              this.props.outletContext.updateCallback();
            }
            this.fetchKotsDownstreamHistory();
          } else {
            this.setState({
              versionDownloadStatuses: {
                ...this.state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: true,
                  downloadingVersionMessage: response.currentMessage,
                },
              },
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get version download status", err);
          reject();
        });
    });
  };

  deployVersion = (
    version: Version | null,
    force = false,
    continueWithFailedPreflights = false
  ) => {
    const { app } = this.props.outletContext;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }
    if (!force && version !== null) {
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
          isSkipPreflights: true,
        });
        return;
      }
      if (version?.preflightResult && version.status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult as string);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState === "fail") {
          this.setState({
            showDeployWarningModal: true,
            versionToDeploy: version,
          });
          return;
        }
      }
      // prompt to make sure user wants to deploy
      this.setState({
        displayConfirmDeploymentModal: true,
        versionToDeploy: version,
        confirmType: "deploy",
      });
      return;
    } else {
      // force deploy is set to true so finalize the deployment
      this.finalizeDeployment(continueWithFailedPreflights);
    }
  };

  finalizeDeployment = async (continueWithFailedPreflights: boolean) => {
    const { updateCallback } = this.props.outletContext;
    const { versionToDeploy, isSkipPreflights } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await this.props.outletContext.makeCurrentVersion(
      this.props.params.slug,
      versionToDeploy,
      isSkipPreflights,
      continueWithFailedPreflights
    );
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  redeployVersion = (version: Version) => {
    const { app } = this.props.outletContext;
    const clusterSlug = app.downstream.cluster?.slug;
    if (!clusterSlug) {
      return;
    }

    this.setState({
      displayConfirmDeploymentModal: true,
      confirmType: "redeploy",
      versionToDeploy: version,
    });
  };

  finalizeRedeployment = async () => {
    const { updateCallback } = this.props.outletContext;
    const { versionToDeploy } = this.state;
    this.setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    await this.props.outletContext.redeployVersion(
      this.props.params.slug,
      versionToDeploy
    );
    await this.fetchKotsDownstreamHistory();
    this.setState({ versionToDeploy: null });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  };

  onForceDeployClick = (continueWithFailedPreflights = false) => {
    this.setState({
      showSkipModal: false,
      showDeployWarningModal: false,
      displayShowDetailsModal: false,
    });
    const versionToDeploy = this.state.versionToDeploy;
    this.deployVersion(versionToDeploy, true, continueWithFailedPreflights);
  };

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false,
    });
  };

  hideDeployWarningModal = () => {
    this.setState({
      showDeployWarningModal: false,
    });
  };

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false,
    });
  };

  hideDiffOverlay = (closeReleaseSelect: boolean) => {
    this.setState({
      showDiffOverlay: false,
    });
    if (closeReleaseSelect) {
      this.onCloseReleasesToDiff();
    }
  };

  hideConfirmDeploymentModal = () => {
    this.setState({
      displayConfirmDeploymentModal: false,
      confirmType: "",
      versionToDeploy: null,
    });
  };

  onSelectReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: true,
      diffHovered: false,
    });
  };

  onCloseReleasesToDiff = () => {
    this.setState({
      selectedDiffReleases: false,
      checkedReleasesToDiff: [],
      diffHovered: false,
      showDiffOverlay: false,
    });
  };

  onCheckForUpdates = async () => {
    const { app } = this.props.outletContext;

    this.setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
      checkingUpdateMessage: "",
    });

    fetch(`${process.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          const text = await res.text();
          this.setState({
            checkingForUpdateError: true,
            checkingForUpdates: false,
            checkingUpdateMessage: text,
          });
          return;
        }
        this.props.outletContext.refreshAppData();
        const response = await res.json();

        if (response.availableUpdates === 0) {
          if (
            !find(this.state.versionHistory, {
              parentSequence: response.currentAppSequence,
            })
          ) {
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
                noUpdateAvailiableText: "",
              });
            }, 3000);
          }
        } else {
          this.state.appUpdateChecker.start(this.getAppUpdateStatus, 1000);
        }
      })
      .catch((err) => {
        this.setState({
          checkingForUpdateError: true,
          checkingForUpdates: false,
          checkingUpdateMessage: String(err),
        });
      });
  };

  onCheckForUpgradeServiceStatus = async () => {
    const { app } = this.props.outletContext;

    this.setState({ isStartingUpgradeService: true });
    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/task/upgrade-service`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      )
        .then(async (res) => {
          const response = await res.json();
          if (response.status !== "starting") {
            this.state.upgradeServiceChecker.stop();
            this.setState({
              isStartingUpgradeService: false,
            });
            if (response.status === "failed") {
              this.setState({
                shouldShowUpgradeServiceModal: false,
                upgradeService: {
                  isLoading: false,
                  error: response.currentMessage,
                },
              });
            }
          } else {
            this.setState({
              isStartingUpgradeService: true,
              upgradeServiceStatus: response.currentMessage,
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get upgrade service status", err);
          reject();
        });
    });
  };

  getAppUpdateStatus = () => {
    const { app } = this.props.outletContext;

    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${app?.slug}/task/updatedownload`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      )
        .then(async (res) => {
          const response = await res.json();

          if (
            response.status !== "running" &&
            !this.props.outletContext.isBundleUploading
          ) {
            this.state.appUpdateChecker.stop();

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed",
            });

            if (this.props.outletContext.updateCallback) {
              this.props.outletContext.updateCallback();
            }
            this.fetchKotsDownstreamHistory();
          } else {
            this.setState({
              checkingForUpdates: true,
              checkingUpdateMessage: response.currentMessage,
            });
          }
          resolve();
        })
        .catch((err) => {
          console.log("failed to get update status", err);
          reject();
        });
    });
  };

  handleViewLogs = async (version: Version | null, isFailing: boolean) => {
    try {
      const { app } = this.props.outletContext;
      let clusterId = app.downstream.cluster?.id;
      this.setState({
        logsLoading: true,
        showLogsModal: true,
        viewLogsErrMsg: "",
      });

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${app.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`,
        {
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      );
      if (res.ok && res.status === 200) {
        const response = await res.json();
        let selectedTab;
        if (isFailing) {
          selectedTab = Utilities.getDeployErrorTab(response.logs);
        } else {
          selectedTab = Object.keys(response.logs)[0];
        }
        this.setState({
          logs: response.logs,
          selectedTab,
          logsLoading: false,
          viewLogsErrMsg: "",
        });
      } else {
        this.setState({
          logsLoading: false,
          viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      this.setState({
        logsLoading: false,
        viewLogsErrMsg: err
          ? `Failed to view logs: ${(err as Error).message}`
          : "Something went wrong, please try again.",
      });
    }
  };

  renderDiffBtn = () => {
    const { app } = this.props.outletContext;
    const { showDiffOverlay, selectedDiffReleases, checkedReleasesToDiff } =
      this.state;
    const downstream = app?.downstream;
    const gitopsIsConnected = downstream.gitops?.isConnected;
    const versionHistory = this.state.versionHistory?.length
      ? this.state.versionHistory
      : [];
    return versionHistory.length && selectedDiffReleases ? (
      <div className="flex u-marginLeft--20">
        <button
          className="btn secondary small u-marginRight--10"
          onClick={this.onCloseReleasesToDiff}
        >
          Cancel
        </button>
        <button
          className="btn primary small blue"
          disabled={checkedReleasesToDiff.length !== 2 || showDiffOverlay}
          onClick={() => {
            if (gitopsIsConnected) {
              const { firstHash, secondHash } = this.getDiffCommitHashes();
              if (firstHash && secondHash) {
                const diffUrl = getGitProviderDiffUrl(
                  downstream.gitops?.uri,
                  downstream.gitops?.provider,
                  firstHash,
                  secondHash
                );
                window.open(diffUrl, "_blank");
              }
            } else {
              const { firstSequence, secondSequence } = this.getDiffSequences();
              this.setState({
                showDiffOverlay: true,
                firstSequence,
                secondSequence,
              });
            }
          }}
        >
          Diff versions
        </button>
      </div>
    ) : (
      <div
        className="flex-auto flex alignItems--center u-marginLeft--20"
        onClick={this.onSelectReleasesToDiff}
      >
        <Icon
          icon="diff-icon"
          size={21}
          className="clickable"
          color={""}
          style={{}}
          disableFill={false}
          removeInlineStyle={false}
        />
        <span className="u-fontSize--small link u-marginLeft--5">
          Diff versions
        </span>
      </div>
    );
  };

  handleSelectReleasesToDiff = (
    selectedRelease: Version,
    isChecked: boolean
  ) => {
    if (isChecked) {
      this.setState({
        checkedReleasesToDiff: (
          [{ ...selectedRelease, isChecked }] as Version[]
        )
          .concat(this.state.checkedReleasesToDiff)
          .slice(0, 2),
      });
    } else {
      this.setState({
        checkedReleasesToDiff: this.state.checkedReleasesToDiff.filter(
          (release: Version) =>
            release.parentSequence !== selectedRelease.parentSequence
        ),
      });
    }
  };

  getDiffSequences = () => {
    let firstSequence = 0,
      secondSequence = 0;

    const { checkedReleasesToDiff } = this.state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort(
        (r1: Version, r2: Version) => r1.parentSequence - r2.parentSequence
      );
      firstSequence = checkedReleasesToDiff[0].parentSequence;
      secondSequence = checkedReleasesToDiff[1].parentSequence;
    }

    return {
      firstSequence,
      secondSequence,
    };
  };

  getDiffCommitHashes = () => {
    let firstCommitUrl = "",
      secondCommitUrl = "";

    const { checkedReleasesToDiff } = this.state;
    if (checkedReleasesToDiff.length === 2) {
      checkedReleasesToDiff.sort(
        (r1, r2) => r1.parentSequence - r2.parentSequence
      );
      firstCommitUrl = checkedReleasesToDiff[0].commitUrl;
      secondCommitUrl = checkedReleasesToDiff[1].commitUrl;
    }

    return {
      firstHash: getCommitHashFromUrl(firstCommitUrl),
      secondHash: getCommitHashFromUrl(secondCommitUrl),
    };
  };

  toggleShowDetailsModal = (
    yamlErrorDetails: string[],
    selectedSequence: number
  ) => {
    this.setState({
      displayShowDetailsModal: !this.state.displayShowDetailsModal,
      deployView: false,
      yamlErrorDetails,
      selectedSequence,
    });
  };

  shouldRenderUpdateProgress = () => {
    if (this.state.uploadingAirgapFile) {
      return true;
    }
    if (this.props.outletContext.isBundleUploading) {
      return true;
    }
    if (this.state.checkingForUpdateError) {
      return true;
    }
    if (this.state.airgapUploadError) {
      return true;
    }
    if (
      this.props.outletContext.app?.isAirgap &&
      this.state.checkingForUpdates
    ) {
      return true;
    }
    return false;
  };

  renderUpdateProgress = () => {
    const { app } = this.props.outletContext;

    if (!this.shouldRenderUpdateProgress()) {
      return null;
    }

    let updateText;
    if (this.state.airgapUploadError) {
      updateText = (
        <p className="u-marginTop--10 u-fontSize--small u-textColor--error u-fontWeight--medium">
          {this.state.airgapUploadError}
        </p>
      );
    } else if (this.state.checkingForUpdateError) {
      updateText = (
        <div className="flex-column flex-auto u-marginTop--10">
          <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">
            Error updating version:
          </p>
          <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">
            {this.state.checkingUpdateMessage}
          </p>
        </div>
      );
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
    } else if (this.props.outletContext.isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          appSlug={app.slug}
          unkownProgress={true}
          onProgressError={undefined}
          smallSize={true}
        />
      );
    } else if (app?.isAirgap && this.state.checkingForUpdates) {
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
      updateText = (
        <div className="flex-column justifyContent--center alignItems--center">
          <Loader className="u-marginBottom--10" size="30" />
          <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">
            {checkingUpdateText}
          </span>
        </div>
      );
    }

    return (
      <div className="u-marginTop--20 u-marginBottom--20">{updateText}</div>
    );
  };

  renderAllVersions = () => {
    // This is kinda hacky. This finds the equivalent downstream version because the midstream
    // version type does not contain metadata like version label or release notes.
    let allVersions = this.state.versionHistory;

    // exclude pinned version
    if (this.state.updatesAvailable) {
      allVersions = this.state.versionHistory?.slice(1);
    }

    if (!allVersions?.length) {
      return null;
    }

    const { currentPage, pageSize, totalCount, loadingPage } = this.state;

    return (
      <div className="TableDiff--Wrapper card-bg">
        <div className="flex u-marginBottom--15 justifyContent--spaceBetween">
          <p className="u-fontSize--normal u-fontWeight--medium card-title">
            All versions
          </p>
          <div className="flex flex-auto alignItems--center">
            <span className="flex-auto u-marginRight--5 u-fontSize--small card-title u-lineHeight--normal u-fontWeight--medium">
              Results per page:
            </span>
            <select className="Select" onChange={(e) => this.setPageSize(e)}>
              <option value="20">20</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </div>
        </div>
        {allVersions?.map((version, index) =>
          this.renderAppVersionHistoryRow(version, index)
        )}
        <Pager
          pagerType="releases"
          currentPage={currentPage}
          pageSize={pageSize}
          totalCount={totalCount}
          loading={loadingPage}
          currentPageLength={allVersions.length}
          goToPage={this.onGotoPage}
        />
      </div>
    );
  };

  onGotoPage = (page: Number, ev: { preventDefault: () => void }) => {
    ev.preventDefault();
    this.setState({ currentPage: page, loadingPage: true }, async () => {
      this.props.navigate(`${this.props.location.pathname}?page=${page}`);
      await this.fetchKotsDownstreamHistory();
      this.setState({ loadingPage: false });
    });
  };

  deployButtonStatus = (version: Version) => {
    const app = this.props.outletContext.app;
    const downstream = app?.downstream;

    const isCurrentVersion =
      version.sequence === downstream.currentVersion?.sequence;
    const isDeploying = version.status === "deploying";

    const isPastVersion = find(downstream.pastVersions, {
      sequence: version.sequence,
    });
    const needsConfiguration = version.status === "pending_config";
    const isRollback = isPastVersion && version.deployedAt && app.allowRollback;
    const isRedeploy =
      isCurrentVersion &&
      (version.status === "failed" || version.status === "deployed");
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !this.props.outletContext.adminConsoleMetadata?.isAirgap &&
      !this.props.outletContext.adminConsoleMetadata?.isKurl;

    if (needsConfiguration) {
      return "Configure";
    } else if (downstream?.currentVersion?.sequence == undefined) {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    } else if (isRedeploy) {
      return "Redeploy";
    } else if (isRollback) {
      return "Rollback";
    } else if (isDeploying) {
      return "Deploying";
    } else if (isCurrentVersion) {
      return "Deployed";
    } else {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    }
  };

  startUpgraderService = (update: AvailableUpdate) => {
    this.setState({
      upgradeService: {
        versionLabel: update.versionLabel,
        isLoading: true,
        error: "",
      },
    });
    const appSlug = this.props.params.slug;
    fetch(`${process.env.API_ENDPOINT}/app/${appSlug}/start-upgrade-service`, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({
        versionLabel: update.versionLabel,
        updateCursor: update.updateCursor,
        channelId: update.channelId,
      }),
      credentials: "include",
      method: "POST",
    })
      .then(async (res) => {
        if (res.ok) {
          this.state.upgradeServiceChecker.start(
            this.onCheckForUpgradeServiceStatus,
            1000
          );
          this.setState({
            shouldShowUpgradeServiceModal: true,
            upgradeService: {
              versionLabel: update.versionLabel,
              isLoading: false,
            },
          });
          return;
        }
        const text = await res.json();
        if (!res.ok) {
          console.log("failed to init upgrade service", text);
          this.setState({
            upgradeService: {
              isLoading: false,
              error: text.error,
              versionLabel: update.versionLabel,
            },
          });
        }
      })
      .catch((err) => {
        console.log(err);
      });

    this._mounted = true;
  };

  renderAvailableUpdates = (updates: AvailableUpdate[]) => {
    return (
      <>
        <p className="u-fontSize--normal u-fontWeight--medium card-title tw-pb-4">
          Available Updates
        </p>
        <div className="tw-flex tw-flex-col tw-gap-2 tw-max-h-[275px] tw-overflow-auto">
          {updates.map((update, index) => (
            <div key={index}>
              <div className="tw-h-10 tw-bg-white tw-p-4 tw-flex tw-justify-between tw-items-center tw-rounded">
                <div className="flex-column">
                  <div className="flex alignItems--center">
                    <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title ">
                      {update.versionLabel}
                    </p>
                    {update.isRequired && (
                      <span className="status-tag required u-marginLeft--10">
                        {" "}
                        Required{" "}
                      </span>
                    )}
                  </div>
                  {update.upstreamReleasedAt && (
                    <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
                      {" "}
                      Released{" "}
                      <span className="u-fontWeight--bold">
                        {Utilities.dateFormat(
                          update.upstreamReleasedAt,
                          "MM/DD/YY @ hh:mm a z"
                        )}
                      </span>
                    </p>
                  )}
                </div>
                <div className="flex alignItems--center">
                  {update?.releaseNotes && (
                    <>
                      <Icon
                        icon="release-notes"
                        size={24}
                        onClick={() =>
                          this.showReleaseNotes(update?.releaseNotes)
                        }
                        data-tip="View release notes"
                        className="u-marginRight--5 clickable"
                      />
                      <ReactTooltip
                        effect="solid"
                        className="replicated-tooltip"
                      />
                    </>
                  )}
                  <button
                    className={"btn tw-ml-2 primary blue"}
                    onClick={() => this.startUpgraderService(update)}
                    disabled={!update.isDeployable}
                  >
                    <span
                      key={update.nonDeployableCause}
                      data-tip-disable={update.isDeployable}
                      data-tip={update.nonDeployableCause}
                      data-for="disable-deployment-tooltip"
                    >
                      Deploy
                    </span>
                  </button>
                  <ReactTooltip
                    effect="solid"
                    id="disable-deployment-tooltip"
                  />
                </div>
              </div>
              {this.state.upgradeService?.error &&
                this.state.upgradeService?.versionLabel ===
                  update.versionLabel && (
                  <div className="tw-my-4">
                    <span className="u-fontSize--small u-textColor--error u-fontWeight--bold">
                      {this.state.upgradeService.error}
                    </span>
                  </div>
                )}
            </div>
          ))}
        </div>
      </>
    );
  };

  renderAppVersionHistoryRow = (version: Version, index?: number) => {
    if (
      !version ||
      isEmpty(version) ||
      (this.state.selectedDiffReleases && version.status === "pending_download")
    ) {
      // non-downloaded versions can't be diffed
      return null;
    }

    const downstream = this.props.outletContext.app.downstream;
    const gitopsIsConnected = downstream?.gitops?.isConnected;
    const nothingToCommit = gitopsIsConnected && !version.commitUrl;
    const isChecked = !!this.state.checkedReleasesToDiff.find(
      (diffRelease) => diffRelease.parentSequence === version.parentSequence
    );
    const isNew = secondsAgo(version.createdOn) < 10;
    let newPreflightResults = false;
    if (version.preflightResultCreatedAt) {
      newPreflightResults = secondsAgo(version.preflightResultCreatedAt) < 12;
    }
    return (
      <Fragment key={index}>
        <AppVersionHistoryRow
          navigate={this.props.navigate}
          adminConsoleMetadata={this.props.outletContext.adminConsoleMetadata}
          isEmbeddedCluster={this.props.outletContext.isEmbeddedCluster}
          deployVersion={this.deployVersion}
          downloadVersion={this.downloadVersion}
          gitopsEnabled={gitopsIsConnected}
          handleSelectReleasesToDiff={this.handleSelectReleasesToDiff}
          handleViewLogs={this.handleViewLogs}
          isChecked={isChecked}
          isDownloading={
            this.state.versionDownloadStatuses?.[version.sequence]
              ?.downloadingVersion
          }
          isNew={isNew}
          key={version.sequence}
          newPreflightResults={newPreflightResults}
          nothingToCommit={nothingToCommit}
          onWhyNoGeneratedDiffClicked={(rowVersion: Version) =>
            this.toggleNoChangesModal(rowVersion)
          }
          onWhyUnableToGeneratedDiffClicked={(rowVersion: Version) =>
            this.toggleDiffErrModal(rowVersion)
          }
          onViewDiffClicked={(
            firstSequence: number,
            secondSequence: number
          ) => {
            const url = location.pathname;
            this.props.navigate(
              `${url}/diff/${firstSequence}/${secondSequence}`,
              {
                replace: true,
              }
            );
            this.setState({
              showDiffOverlay: true,
              firstSequence,
              secondSequence,
            });
          }}
          redeployVersion={this.redeployVersion}
          selectedDiffReleases={this.state.selectedDiffReleases}
          showReleaseNotes={this.showReleaseNotes}
          showVersionPreviousDownloadStatus={
            // user hasn't tried to re-download the version yet,
            // show last known download status if exists
            !this.state.versionDownloadStatuses.hasOwnProperty(
              version.sequence
            ) && Boolean(version.downloadStatus)
          }
          showVersionDownloadingStatus={
            this.state.versionDownloadStatuses.hasOwnProperty(
              version.sequence
            ) && Boolean(this.state.versionDownloadStatuses?.[version.sequence])
          }
          toggleShowDetailsModal={this.toggleShowDetailsModal}
          upgradeAdminConsole={this.upgradeAdminConsole}
          version={version}
          versionDownloadStatus={
            this.state.versionDownloadStatuses?.[version.sequence]
          }
          versionHistory={this.state.versionHistory}
        />
      </Fragment>
    );
  };

  getPreflightState = (version: Version) => {
    let preflightsFailed = false;
    let preflightState = "";
    if (version?.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      preflightState = getPreflightResultState(preflightResult);
      preflightsFailed = preflightState === "fail";

      this.setState({
        preflightState: {
          preflightsFailed,
          preflightState,
        },
      });
    }
    return {};
  };

  render() {
    const {
      app,
      makingCurrentVersionErrMsg,
      redeployVersionErrMsg,
      resetRedeployErrorMessage,
      resetMakingCurrentReleaseErrorMessage,
    } = this.props.outletContext;

    const {
      airgapUploader,
      checkingForUpdates,
      checkingUpdateMessage,
      displayErrorModal,
      errorMsg,
      errorTitle,
      firstSequence,
      loadingVersionHistory,
      logs,
      logsLoading,
      releaseNotes,
      secondSequence,
      selectedTab,
      showDeployWarningModal,
      showDiffOverlay,
      showLogsModal,
      showSkipModal,
      versionHistory,
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

    const downstream = app?.downstream;
    const gitopsIsConnected = downstream.gitops?.isConnected;
    const currentDownstreamVersion = downstream?.currentVersion;
    const iconUri = currentDownstreamVersion?.appIconUri || app?.iconUri;
    const isPastVersion = find(downstream?.pastVersions, {
      sequence: this.state.versionToDeploy?.sequence,
    });

    let checkingUpdateTextShort = checkingUpdateMessage;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    const renderKotsUpgradeStatus =
      this.state.kotsUpdateStatus && !this.state.kotsUpdateMessage;
    let shortKotsUpdateMessage = this.state.kotsUpdateMessage;
    if (shortKotsUpdateMessage && shortKotsUpdateMessage.length > 60) {
      shortKotsUpdateMessage = shortKotsUpdateMessage.substring(0, 60) + "...";
    }

    let sequenceLabel = "Sequence";

    let pendingVersion;
    if (this.state.updatesAvailable) {
      pendingVersion = versionHistory[0];
    }

    const renderVersionLabel = () => {
      let shorten = "";
      let isTruncated = false;
      if (currentDownstreamVersion && currentDownstreamVersion?.versionLabel) {
        const { versionLabel } = currentDownstreamVersion;
        if (versionLabel.length > 18) {
          shorten = `${versionLabel.slice(0, 15)}...`;
          isTruncated = true;
        } else {
          shorten = versionLabel;
        }
      } else {
        shorten = "---";
      }

      return (
        <div className="tw-flex tw-items-center">
          <p className="u-fontSize--header2 u-fontWeight--bold card-item-title u-marginTop--5 u-position--relative">
            {shorten}
          </p>{" "}
          {isTruncated && (
            <>
              <Icon
                icon="info"
                size={16}
                data-tip={currentDownstreamVersion?.versionLabel || ""}
              />
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </>
          )}
        </div>
      );
    };

    return (
      <div className="flex flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <KotsPageTitle pageName="Version History" showAppSlug />
        <div className="flex-column flex1">
          <div className="flex flex1 justifyContent--center">
            <div className="flex1 flex AppVersionHistory">
              {makingCurrentVersionErrMsg && (
                <ErrorModal
                  errorModal={true}
                  err="Failed to deploy version"
                  errMsg={makingCurrentVersionErrMsg}
                  showDismissButton={true}
                  toggleErrorModal={resetMakingCurrentReleaseErrorMessage}
                />
              )}
              {redeployVersionErrMsg && (
                <ErrorModal
                  errorModal={true}
                  err="Failed to redeploy version"
                  errMsg={redeployVersionErrMsg}
                  showDismissButton={true}
                  toggleErrorModal={resetRedeployErrorMessage}
                />
              )}
              {!gitopsIsConnected && !showDiffOverlay && (
                <div
                  className="flex-column flex1"
                  style={{ maxWidth: "370px", marginRight: "20px" }}
                >
                  <div className="card-bg TableDiff--Wrapper currentVersionCard--wrapper">
                    <p className="u-fontSize--large card-title u-fontWeight--bold">
                      {currentDownstreamVersion?.versionLabel
                        ? "Currently deployed version"
                        : "No current version deployed"}
                    </p>
                    <div className="currentVersion--wrapper card-item u-marginTop--10">
                      <div className="flex flex1">
                        {iconUri && (
                          <div className="flex-auto u-marginRight--10">
                            <div
                              className="watch-icon"
                              style={{
                                backgroundImage: `url(${iconUri})`,
                              }}
                            ></div>
                          </div>
                        )}
                        <div className="flex1 flex-column ">
                          {renderVersionLabel()}
                          <p className="u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
                            {" "}
                            {currentDownstreamVersion
                              ? `${sequenceLabel} ${currentDownstreamVersion?.sequence}`
                              : null}
                          </p>
                          {currentDownstreamVersion?.deployedAt ? (
                            <p className="u-fontSize--small u-lineHeight--normal u-textColor--info u-fontWeight--medium u-marginTop--10">
                              {currentDownstreamVersion?.status === "deploying"
                                ? "Deploy started at"
                                : "Deployed"}{" "}
                              {Utilities.dateFormat(
                                currentDownstreamVersion.deployedAt,
                                "MM/DD/YY @ hh:mm a z"
                              )}
                            </p>
                          ) : null}
                          {currentDownstreamVersion ? (
                            <div className="flex alignItems--center u-marginTop--10">
                              {currentDownstreamVersion?.releaseNotes && (
                                <div className="u-marginRight--5">
                                  <Icon
                                    icon="release-notes"
                                    className="clickable"
                                    size={24}
                                    onClick={() =>
                                      this.showReleaseNotes(
                                        currentDownstreamVersion?.releaseNotes
                                      )
                                    }
                                    data-tip="View release notes"
                                    color={""}
                                    style={{}}
                                    disableFill={false}
                                    removeInlineStyle={false}
                                  />
                                  <ReactTooltip
                                    effect="solid"
                                    className="replicated-tooltip"
                                  />
                                </div>
                              )}
                              {app ? (
                                <div>
                                  <span
                                    onClick={() =>
                                      this.handleViewLogs(
                                        currentDownstreamVersion,
                                        currentDownstreamVersion?.status ===
                                          "failed"
                                      )
                                    }
                                    data-tip="View deploy logs"
                                  >
                                    <Icon
                                      icon="view-logs"
                                      size={22}
                                      className="clickable"
                                      color={""}
                                      style={{}}
                                      disableFill={false}
                                      removeInlineStyle={false}
                                    />
                                  </span>
                                  <ReactTooltip
                                    effect="solid"
                                    className="replicated-tooltip"
                                  />
                                </div>
                              ) : null}
                              <PreflightIcon
                                app={app}
                                version={downstream.currentVersion}
                                showText={false}
                                preflightState={this.state.preflightState}
                                className={"tw-mx-1"}
                                newPreflightResults={true}
                              />
                              {app.isConfigurable && (
                                <div>
                                  <Link
                                    to={`/app/${app?.slug}/config/${app?.downstream?.currentVersion?.parentSequence}`}
                                    data-tip="Edit config"
                                  >
                                    <Icon
                                      icon="edit-config"
                                      size={22}
                                      color={""}
                                      style={{}}
                                      className={""}
                                      disableFill={false}
                                      removeInlineStyle={false}
                                    />
                                  </Link>
                                  <ReactTooltip
                                    effect="solid"
                                    className="replicated-tooltip"
                                  />
                                </div>
                              )}
                            </div>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              <div
                className={`flex-column flex1 alignSelf--start ${
                  gitopsIsConnected ? "gitops-enabled" : ""
                }`}
              >
                <div
                  className={`flex-column flex1 version ${
                    showDiffOverlay ? "u-visibility--hidden" : ""
                  }`}
                >
                  {(versionHistory.length === 0 && gitopsIsConnected) ||
                  versionHistory?.length > 0 ||
                  (this.state.availableUpdates &&
                    this.state.availableUpdates?.length > 0) ? (
                    <div>
                      {gitopsIsConnected && (
                        <div
                          style={{ maxWidth: "1030px" }}
                          className="u-width--full u-marginBottom--30"
                        >
                          <DashboardGitOpsCard
                            gitops={downstream?.gitops}
                            isAirgap={app?.isAirgap}
                            appSlug={app?.slug}
                            checkingForUpdates={checkingForUpdates}
                            latestConfigSequence={
                              versionHistory[0]?.parentSequence
                            }
                            isBundleUploading={
                              this.props.outletContext.isBundleUploading
                            }
                            checkingUpdateText={checkingUpdateMessage}
                            checkingUpdateTextShort={checkingUpdateTextShort}
                            onCheckForUpdates={this.onCheckForUpdates}
                            showAutomaticUpdatesModal={
                              this.toggleAutomaticUpdatesModal
                            }
                          />
                        </div>
                      )}

                      {!gitopsIsConnected &&
                        !this.props.outletContext.isEmbeddedCluster && (
                          <div className="TableDiff--Wrapper card-bg u-marginBottom--30">
                            <div className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--15">
                              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--info">
                                {this.state.updatesAvailable
                                  ? "New version available"
                                  : ""}
                              </p>
                              <div className="flex alignItems--center">
                                <div className="flex alignItems--center">
                                  {app?.isAirgap && airgapUploader ? (
                                    <MountAware
                                      onMount={(el: Element) =>
                                        airgapUploader?.assignElement(el)
                                      }
                                    >
                                      <div className="flex alignItems--center">
                                        <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
                                        <span className="link u-fontSize--small u-lineHeight--default">
                                          Upload new version
                                        </span>
                                      </div>
                                    </MountAware>
                                  ) : (
                                    <div className="flex alignItems--center">
                                      {checkingForUpdates &&
                                      !this.props.outletContext
                                        .isBundleUploading ? (
                                        <div className="flex alignItems--center u-marginRight--20">
                                          <Loader
                                            className="u-marginRight--5"
                                            size="15"
                                          />
                                          <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                                            {checkingUpdateMessage === ""
                                              ? "Checking for updates"
                                              : checkingUpdateTextShort}
                                          </span>
                                        </div>
                                      ) : (
                                        <div className="flex alignItems--center u-marginRight--20">
                                          <span
                                            className="flex-auto flex alignItems--center link u-fontSize--small"
                                            onClick={this.onCheckForUpdates}
                                          >
                                            <Icon
                                              icon="check-update"
                                              size={16}
                                              className="clickable u-marginRight--5"
                                              color={""}
                                              style={{}}
                                              disableFill={false}
                                              removeInlineStyle={false}
                                            />
                                            Check for update
                                          </span>
                                        </div>
                                      )}
                                      <span
                                        className="flex-auto flex alignItems--center link u-fontSize--small"
                                        onClick={
                                          this.toggleAutomaticUpdatesModal
                                        }
                                      >
                                        <Icon
                                          icon="schedule-sync"
                                          size={16}
                                          className="clickable u-marginRight--5"
                                          color={""}
                                          style={{}}
                                          disableFill={false}
                                          removeInlineStyle={false}
                                        />
                                        Configure automatic updates
                                      </span>
                                    </div>
                                  )}
                                </div>
                                {versionHistory.length > 1 && !gitopsIsConnected
                                  ? this.renderDiffBtn()
                                  : null}
                              </div>
                            </div>
                            {pendingVersion ? (
                              this.renderAppVersionHistoryRow(pendingVersion)
                            ) : (
                              <div className="card-item flex-column flex1 u-marginTop--20 u-marginBottom--10 alignItems--center justifyContent--center">
                                <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--10">
                                  Application up to date.
                                </p>
                              </div>
                            )}
                            {(this.state.numOfSkippedVersions > 0 ||
                              this.state.numOfRemainingVersions > 0) && (
                              <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--more u-textColor--info u-marginTop--10">
                                {this.state.numOfSkippedVersions > 0
                                  ? `${
                                      this.state.numOfSkippedVersions
                                    } version${
                                      this.state.numOfSkippedVersions > 1
                                        ? "s"
                                        : ""
                                    } will be skipped in upgrading to ${
                                      versionHistory[0].versionLabel
                                    }. `
                                  : ""}
                                {this.state.numOfRemainingVersions > 0
                                  ? "Additional versions are available after you deploy this required version."
                                  : ""}
                              </p>
                            )}
                          </div>
                        )}

                      {this.props.outletContext.isEmbeddedCluster && (
                        <>
                          {this.state.isFetchingAvailableUpdates ? (
                            <div className="TableDiff--Wrapper card-bg u-marginBottom--30">
                              <div className="flex-column flex1 alignItems--center justifyContent--center">
                                <Loader size="60" />
                              </div>
                            </div>
                          ) : (
                            this.state.availableUpdates &&
                            this.state.availableUpdates.length > 0 &&
                            this.props.outletContext.isEmbeddedCluster && (
                              <div className="TableDiff--Wrapper card-bg u-marginBottom--30">
                                {this.renderAvailableUpdates(
                                  this.state.availableUpdates
                                )}
                              </div>
                            )
                          )}
                        </>
                      )}
                      {versionHistory?.length > 0 && (
                        <>
                          {this.renderUpdateProgress()}
                          {this.renderAllVersions()}
                        </>
                      )}
                    </div>
                  ) : (
                    <div className="flex-column flex1 alignItems--center justifyContent--center">
                      <p className="u-fontSize--large u-fontWeight--bold u-textColor--primary">
                        No versions have been deployed.
                      </p>
                    </div>
                  )}
                </div>

                {/* Diff overlay */}
                {showDiffOverlay && (
                  <div className="DiffOverlay">
                    <VersionDiff
                      slug={this.props.params?.slug}
                      firstSequence={firstSequence}
                      secondSequence={secondSequence}
                      onBackClick={this.hideDiffOverlay}
                      hideBackButton={false}
                      app={this.props.outletContext.app}
                    />
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* MODALS */}
        <ShowLogsModal
          showLogsModal={showLogsModal}
          hideLogsModal={this.hideLogsModal}
          viewLogsErrMsg={this.state.viewLogsErrMsg}
          logs={logs}
          selectedTab={selectedTab}
          logsLoading={logsLoading}
          renderLogsTabs={this.renderLogsTabs()}
        />

        <DeployWarningModal
          showDeployWarningModal={showDeployWarningModal}
          hideDeployWarningModal={this.hideDeployWarningModal}
          onForceDeployClick={this.onForceDeployClick}
          showAutoDeployWarning={
            isPastVersion &&
            this.props.outletContext.app?.autoDeploy !== "disabled"
          }
          confirmType={this.state.confirmType}
        />

        <SkipPreflightsModal
          showSkipModal={showSkipModal}
          hideSkipModal={this.hideSkipModal}
          onForceDeployClick={this.onForceDeployClick}
        />

        <ReleaseNotesModal
          releaseNotes={releaseNotes}
          hideReleaseNotes={this.hideReleaseNotes}
        />

        <DiffErrorModal
          showDiffErrModal={this.state.showDiffErrModal}
          toggleDiffErrModal={() => this.toggleDiffErrModal()}
          releaseWithErr={this.state.releaseWithErr}
        />

        <ConfirmDeploymentModal
          displayConfirmDeploymentModal={
            this.state.displayConfirmDeploymentModal
          }
          hideConfirmDeploymentModal={this.hideConfirmDeploymentModal}
          confirmType={this.state.confirmType}
          versionToDeploy={this.state.versionToDeploy}
          outletContext={this.props.outletContext}
          finalizeDeployment={this.finalizeDeployment}
          isPastVersion={isPastVersion}
          finalizeRedeployment={this.finalizeRedeployment}
        />

        <DisplayKotsUpdateModal
          displayKotsUpdateModal={this.state.displayKotsUpdateModal}
          onRequestClose={() =>
            this.setState({ displayKotsUpdateModal: false })
          }
          renderKotsUpgradeStatus={renderKotsUpgradeStatus}
          kotsUpdateStatus={this.state.kotsUpdateStatus}
          shortKotsUpdateMessage={shortKotsUpdateMessage}
          kotsUpdateMessage={this.state.kotsUpdateMessage}
        />

        <ShowDetailsModal
          displayShowDetailsModal={this.state.displayShowDetailsModal}
          toggleShowDetailsModal={this.toggleShowDetailsModal}
          yamlErrorDetails={this.state.yamlErrorDetails}
          deployView={this.state.deployView}
          forceDeploy={this.onForceDeployClick}
          showDeployWarningModal={this.state.showDeployWarningModal}
          showSkipModal={this.state.showSkipModal}
          slug={this.props.params.slug}
          sequence={this.state.selectedSequence}
        />

        {errorMsg && (
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={errorMsg}
            appSlug={this.props.params?.slug}
          />
        )}

        <NoChangesModal
          showNoChangesModal={this.state.showNoChangesModal}
          toggleNoChangesModal={this.toggleNoChangesModal}
          releaseWithNoChanges={this.state.releaseWithNoChanges}
        />

        {this.state.showAutomaticUpdatesModal && (
          <AutomaticUpdatesModal
            appSlug={app?.slug}
            autoDeploy={app?.autoDeploy}
            gitopsIsConnected={downstream?.gitops?.isConnected}
            isOpen={this.state.showAutomaticUpdatesModal}
            isSemverRequired={app?.isSemverRequired}
            onAutomaticUpdatesConfigured={() => {
              this.toggleAutomaticUpdatesModal();
              this.props.outletContext.updateCallback();
            }}
            onRequestClose={this.toggleAutomaticUpdatesModal}
            updateCheckerSpec={app?.updateCheckerSpec}
          />
        )}

        <UpgradeServiceModal
          shouldShowUpgradeServiceModal={
            this.state.shouldShowUpgradeServiceModal
          }
          isStartingUpgradeService={this.state.isStartingUpgradeService}
          upgradeServiceStatus={this.state.upgradeServiceStatus}
          appSlug={app?.slug}
          onRequestClose={() =>
            this.setState({ shouldShowUpgradeServiceModal: false })
          }
          iframeRef={this.iframeRef}
        />
      </div>
    );
  }
}

export default withRouter(AppVersionHistory);
