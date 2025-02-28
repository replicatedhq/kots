import classNames from "classnames";
import { useEffect, useReducer } from "react";
import Modal from "react-modal";
import { Link, useNavigate, useParams } from "react-router-dom";
import ReactTooltip from "react-tooltip";

import EditConfigIcon from "@components/shared/EditConfigIcon";
import { useSelectedApp } from "@features/App";
import VersionDiff from "@features/VersionDiff/VersionDiff";
import AirgapUploadProgress from "@src/components/AirgapUploadProgress";
import Icon from "@src/components/Icon";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import Loader from "@src/components/shared/Loader";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import DeployWarningModal from "@src/components/shared/modals/DeployWarningModal";
import SkipPreflightsModal from "@src/components/shared/modals/SkipPreflightsModal";
import MountAware from "@src/components/shared/MountAware";
import { AirgapUploader } from "@src/utilities/airgapUploader";
import { Repeater } from "@src/utilities/repeater";
import {
  getPreflightResultState,
  getReadableGitOpsProviderName,
  secondsAgo,
  Utilities,
} from "@src/utilities/utilities";
import {
  AvailableUpdate,
  Downstream,
  KotsParams,
  Metadata,
  Version,
  VersionDownloadStatus,
  VersionStatus,
} from "@types";
import { useNextAppVersionWithIntercept } from "../api/useNextAppVersion";
import AvailableUpdateCard from "./AvailableUpdateCard";
import DashboardGitOpsCard from "./DashboardGitOpsCard";

import "@src/scss/components/watches/DashboardCard.scss";

type Props = {
  adminConsoleMetadata: Metadata | null;
  airgapUploader: AirgapUploader | null;
  airgapUploadError: string | null;
  checkingForUpdates: boolean;
  checkingForUpdateError: boolean;
  checkingUpdateText: string;
  currentVersion: Version | null;
  downloadCallback: () => void;
  downstream: Downstream | null;
  isBundleUploading: boolean;
  links?: string[];
  makeCurrentVersion: (
    slug: string,
    versionToDeploy: Version,
    isSkipPreflights: boolean,
    continueWithFailedPreflights: boolean
  ) => void;
  // TODO:  fix this misspelling
  noUpdatesAvalable: boolean;
  onCheckForUpdates: () => void;
  onProgressError: (airgapUploadError: string) => Promise<void>;
  redeployVersion: (slug: string, version: Version | null) => void;
  refetchData: () => void;
  showAutomaticUpdatesModal: () => void;
  uploadingAirgapFile: boolean;
  uploadProgress: number;
  uploadResuming: boolean;
  uploadSize: number;
  viewAirgapUploadError: () => void;
  showUpgradeStatusModal: boolean;
};

type State = {
  availableUpdates: AvailableUpdate[];
  confirmType: string;
  deployView: boolean;
  displayConfirmDeploymentModal: boolean;
  displayKotsUpdateModal: boolean;
  displayShowDetailsModal: boolean;
  firstSequence: string;
  secondSequence: string;
  isFetchingAvailableUpdates: boolean;
  isRedeploy: boolean;
  isSkipPreflights: boolean;
  kotsUpdateChecker: Repeater;
  kotsUpdateError: string | null;
  kotsUpdateMessage: string | null;
  kotsUpdateRunning: boolean;
  kotsUpdateStatus: VersionStatus | null;
  latestDeployableVersionErrMsg: string;
  logs: null | string;
  logsLoading: boolean;
  numOfRemainingVersions: number;
  numOfSkippedVersions: number;
  releaseNotes: string;
  releaseWithErr: Version | null;
  releaseWithNoChanges: Version | null;
  selectedAction: string;
  selectedSequence: number;
  selectedTab: string | null;
  showDeployWarningModal: boolean;
  showDiffErrModal: boolean;
  showDiffModal: boolean;
  showLogsModal: boolean;
  showNoChangesModal: boolean;
  showReleaseNotes: boolean;
  showSkipModal: boolean;
  versionDownloadStatuses: {
    [x: number]: VersionDownloadStatus;
  };
  versionFailing: boolean;
  versionToDeploy: Version | null;
  viewLogsErrMsg: string;
  yamlErrorDetails: string[];
};

const DashboardVersionCard = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      availableUpdates: [],
      confirmType: "",
      deployView: false,
      displayConfirmDeploymentModal: false,
      displayKotsUpdateModal: false,
      displayShowDetailsModal: false,
      firstSequence: "",
      isFetchingAvailableUpdates: false,
      isSkipPreflights: false,
      isRedeploy: false,
      kotsUpdateChecker: new Repeater(),
      kotsUpdateError: null,
      kotsUpdateMessage: null,
      kotsUpdateRunning: false,
      kotsUpdateStatus: null,
      latestDeployableVersionErrMsg: "",
      logs: null,
      logsLoading: false,
      numOfRemainingVersions: 0,
      numOfSkippedVersions: 0,
      releaseNotes: "",
      releaseWithErr: null,
      releaseWithNoChanges: null,
      secondSequence: "",
      selectedAction: "",
      selectedSequence: -1,
      selectedTab: null,
      showDiffErrModal: false,
      showDiffModal: false,
      showDeployWarningModal: false,
      showLogsModal: false,
      showNoChangesModal: false,
      showReleaseNotes: false,
      showSkipModal: false,
      versionDownloadStatuses: {},
      versionFailing: false,
      versionToDeploy: null,
      viewLogsErrMsg: "",
      yamlErrorDetails: [],
    }
  );
  const navigate = useNavigate();
  const params = useParams<KotsParams>();
  const selectedApp = useSelectedApp();
  const {
    data: newAppVersionWithInterceptData,
    error: latestDeployableVersionErrMsg,
    refetch: refetchNextAppVersionWithIntercept,
  } = useNextAppVersionWithIntercept();
  const { latestDeployableVersion } = newAppVersionWithInterceptData || {};

  const fetchAvailableUpdates = async () => {
    const appSlug = params.slug;
    setState({ isFetchingAvailableUpdates: true });
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
      setState({ isFetchingAvailableUpdates: false });
      return;
    }
    const response = await res.json();

    setState({
      isFetchingAvailableUpdates: false,
      availableUpdates: response.updates,
    });
    return response;
  };

  // moving this out of the state because new repeater instances were getting created
  // and it doesn't really affect the UI
  const versionDownloadStatusJobs: {
    [key: number]: Repeater;
  } = {};

  useEffect(() => {
    if (props.links && props.links.length > 0) {
      setState({ selectedAction: props.links[0] });
    }
    if (props.adminConsoleMetadata?.isEmbeddedCluster) {
      fetchAvailableUpdates();
    }
  }, []);

  useEffect(() => {
    if (
      props.links &&
      props.links.length > 0 &&
      props.links[0] !== state.selectedAction
    ) {
      setState({ selectedAction: props.links[0] });
    }
  }, [props.links]);

  useEffect(() => {
    if (state.showDiffModal === false && location.search !== "") {
      const splitSearch = location.search.split("/");
      setState({
        showDiffModal: true,
        firstSequence: splitSearch[1],
        secondSequence: splitSearch[2],
      });
    }
  }, [location.search]);

  useEffect(() => {
    if (props.showUpgradeStatusModal) {
      // if an upgrade is in progress, we don't want to show an error
      return;
    }

    if (latestDeployableVersionErrMsg instanceof Error) {
      setState({
        latestDeployableVersionErrMsg: `Failed to get latest deployable version: ${latestDeployableVersionErrMsg.message}`,
      });
      return;
    }

    if (latestDeployableVersionErrMsg) {
      setState({
        latestDeployableVersionErrMsg:
          "Something went wrong, please try again.",
      });
    } else {
      setState({
        latestDeployableVersionErrMsg: "",
      });
    }
  }, [latestDeployableVersionErrMsg]);

  useEffect(() => {
    refetchNextAppVersionWithIntercept();
  }, [props.downstream]);

  const closeViewDiffModal = () => {
    if (location.search) {
      navigate(location.pathname, { replace: true });
    }
    setState({ showDiffModal: false });
  };

  const hideLogsModal = () => {
    setState({
      showLogsModal: false,
    });
  };

  const renderLogsTabs = () => {
    const { logs, selectedTab } = state;
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
              data-testid={`logs-tab-${tab}`}
              key={tab}
              onClick={() => setState({ selectedTab: tab })}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  };

  const handleViewLogs = async (
    version: Version | null,
    isFailing: boolean
  ) => {
    if (!version) {
      return;
    }
    try {
      let clusterId = selectedApp?.downstream?.cluster?.id;

      setState({
        logsLoading: true,
        showLogsModal: true,
        viewLogsErrMsg: "",
        versionFailing: false,
      });

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`,
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
        setState({
          logs: response.logs,
          selectedTab,
          logsLoading: false,
          viewLogsErrMsg: "",
          versionFailing: isFailing,
        });
      } else {
        setState({
          logsLoading: false,
          viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
        setState({
          logsLoading: false,
          viewLogsErrMsg: `Failed to view logs: ${err.message}`,
        });
      } else {
        setState({
          logsLoading: false,
          viewLogsErrMsg: "Something went wrong, please try again.",
        });
      }
    }
  };

  const getCurrentVersionStatus = (version: Version | null) => {
    if (version?.status === "deployed" || version?.status === "pending") {
      return (
        <span className="status-tag success flex-auto">
          Currently {version?.status.replace("_", " ")} version
        </span>
      );
    } else if (version?.status === "failed") {
      return (
        <div className="flex alignItems--center">
          <span className="status-tag failed flex-auto u-marginRight--10">
            Deploy Failed
          </span>
          <span
            className="link u-fontSize--small"
            onClick={() => handleViewLogs(version, true)}
          >
            View deploy logs
          </span>
        </div>
      );
    } else if (version?.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-fontWeight--medium">
          <Loader
            className="flex alignItems--center u-marginRight--5"
            size="16"
          />
          Deploying
        </span>
      );
    } else {
      return (
        <span className="status-tag unknown flex-atuo">
          {" "}
          {Utilities.toTitleCase(version?.status).replace("_", " ")}{" "}
        </span>
      );
    }
  };

  const toggleDiffErrModal = (release?: Version) => {
    setState({
      showDiffErrModal: !state.showDiffErrModal,
      releaseWithErr: !state.showDiffErrModal && release ? release : null,
    });
  };

  const toggleNoChangesModal = (version?: Version) => {
    setState({
      showNoChangesModal: !state.showNoChangesModal,
      releaseWithNoChanges:
        !state.showNoChangesModal && version ? version : null,
    });
  };

  const toggleShowDetailsModal = (
    yamlErrorDetails: string[],
    selectedSequence: number
  ) => {
    setState({
      displayShowDetailsModal: !state.displayShowDetailsModal,
      deployView: false,
      yamlErrorDetails,
      selectedSequence,
    });
  };

  const getPreflightState = (version: Version) => {
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
      preflightSkipped: version?.preflightSkipped,
    };
  };
  const showReleaseNotes = (releaseNotes: string) => {
    setState({
      showReleaseNotes: true,
      releaseNotes: releaseNotes,
    });
  };

  const hideReleaseNotes = () => {
    setState({
      showReleaseNotes: false,
      releaseNotes: "",
    });
  };

  const renderReleaseNotes = (version: Version | null) => {
    if (!version?.releaseNotes) {
      return null;
    }
    return (
      <div className="u-marginRight--10">
        <Icon
          icon="release-notes"
          size={24}
          className="clickable"
          data-tip="View release notes"
          onClick={() => showReleaseNotes(version.releaseNotes)}
        />
        <ReactTooltip effect="solid" className="replicated-tooltip" />
      </div>
    );
  };

  const renderPreflights = (version: Version | null) => {
    const { currentVersion } = props;
    if (!version) {
      return null;
    }
    if (version.status === "pending_download") {
      return null;
    }
    if (version.status === "pending_config") {
      return null;
    }

    const preflightState = getPreflightState(version);
    let checksStatusText;
    if (preflightState.preflightsFailed) {
      checksStatusText = "Checks failed";
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Checks passed with warnings";
    }

    return (
      <div className="u-position--relative">
        {version.status === "pending_preflight" ? (
          <div className="u-marginLeft--10 u-position--relative">
            <Loader size="30" />
            <p className="checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium">
              Running checks
            </p>
          </div>
        ) : preflightState.preflightState !== "" ? (
          <>
            <Link
              to={`/app/${selectedApp?.slug}/downstreams/${selectedApp?.downstream?.cluster?.slug}/version-history/preflight/${version?.sequence}`}
              className="u-position--relative"
              data-tip="View preflight checks"
            >
              <Icon icon="preflight-checks" size={20} className="clickable" />
              {preflightState.preflightsFailed ||
              preflightState.preflightState === "warn" ? (
                <>
                  {preflightState.preflightsFailed ? (
                    <Icon
                      icon={"warning-circle-filled"}
                      size={12}
                      className="version-row-preflight-status-icon error-color"
                    />
                  ) : preflightState.preflightState === "warn" ? (
                    <Icon
                      icon={"warning"}
                      size={12}
                      className="version-row-preflight-status-icon warning-color"
                    />
                  ) : (
                    ""
                  )}
                  <p
                    className={`checks-running-text u-fontSize--small u-lineHeight--normal u-fontWeight--medium
                    } ${
                      preflightState.preflightsFailed
                        ? "err"
                        : preflightState.preflightState === "warn"
                        ? "warning"
                        : ""
                    }
                     ${
                       !selectedApp && currentVersion?.status === "deploying"
                         ? "without-btns"
                         : ""
                     }`}
                  >
                    {checksStatusText}
                  </p>
                </>
              ) : null}
            </Link>
            <ReactTooltip effect="solid" className="replicated-tooltip" />
          </>
        ) : null}
      </div>
    );
  };

  const finalizeDeployment = async (
    continueWithFailedPreflights: boolean,
    redeploy: boolean
  ) => {
    const { versionToDeploy, isSkipPreflights } = state;
    setState({ displayConfirmDeploymentModal: false, confirmType: "" });
    if (redeploy && params?.slug) {
      await props.redeployVersion(params.slug, versionToDeploy);
    }
    if (versionToDeploy && params?.slug) {
      await props.makeCurrentVersion(
        params.slug,
        versionToDeploy,
        isSkipPreflights,
        continueWithFailedPreflights
      );
      setState({ versionToDeploy: null, isRedeploy: false });

      if (props.refetchData) {
        props.refetchData();
      }
    } else {
      throw new Error("No version to deploy");
    }
  };

  const deployVersion = (
    version: Version | null,
    force = false,
    continueWithFailedPreflights = false,
    redeploy = false
  ) => {
    const clusterSlug = selectedApp?.downstream?.cluster?.slug;
    if (!clusterSlug) {
      return;
    }

    if (!force) {
      if (version?.yamlErrors) {
        setState({
          displayShowDetailsModal: !state.displayShowDetailsModal,
          deployView: true,
          versionToDeploy: version,
          yamlErrorDetails: version.yamlErrors,
        });
        return;
      }
      if (version?.status === "pending_preflight") {
        setState({
          showSkipModal: true,
          versionToDeploy: version,
          isSkipPreflights: true,
        });
        return;
      }
      if (version?.preflightResult && version.status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult);
        const preflightState = getPreflightResultState(preflightResults);
        if (preflightState === "fail") {
          setState({
            showDeployWarningModal: true,
            versionToDeploy: version,
          });
          return;
        }
      }

      // prompt to make sure user wants to deploy
      setState({
        displayConfirmDeploymentModal: true,
        versionToDeploy: version,
        isRedeploy: redeploy,
      });
      return;
    } else {
      // force deploy is set to true so finalize the deployment
      finalizeDeployment(continueWithFailedPreflights, redeploy);
    }
  };

  const renderCurrentVersion = () => {
    const { currentVersion } = props;

    return (
      <div className="flex1 flex-column">
        <div className="flex">
          <div className="flex-column">
            <div className="flex alignItems--center u-marginBottom--5">
              <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title">
                {currentVersion?.versionLabel || currentVersion?.appTitle}
              </p>
              <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">
                Sequence {currentVersion?.sequence}
              </p>
            </div>
            <div data-testid="current-version-status">{getCurrentVersionStatus(currentVersion)}</div>
            <div className="flex alignItems--center u-marginTop--10">
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
                {currentVersion?.status === "failed"
                  ? "---"
                  : `${
                      currentVersion?.status === "deploying"
                        ? "Deploy started at"
                        : "Deployed"
                    } ${Utilities.dateFormat(
                      currentVersion?.deployedAt,
                      "MM/DD/YY @ hh:mm a z"
                    )}`}
              </p>
            </div>
          </div>
          <div className="flex alignItems--center u-paddingLeft--20">
            <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default u-marginRight--5">
              {currentVersion?.source}
            </p>
          </div>
          <div className="flex flex1 alignItems--center justifyContent--flexEnd">
            {renderReleaseNotes(currentVersion)}
            {renderPreflights(currentVersion)}
            <EditConfigIcon version={currentVersion} isPending={false} />
            {selectedApp ? (
              <div className="u-marginLeft--10">
                <span
                  onClick={() =>
                    handleViewLogs(
                      currentVersion,
                      currentVersion?.status === "failed"
                    )
                  }
                  data-tip="View deploy logs"
                >
                  <Icon icon="view-logs" size={22} className="clickable" />
                </span>
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </div>
            ) : null}
            {currentVersion?.status === "deploying" ? null : (
              <div className="flex-column justifyContent--center u-marginLeft--10">
                <button
                  className="secondary blue btn"
                  onClick={() =>
                    deployVersion(currentVersion, false, false, true)
                  }
                >
                  Redeploy
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  const getVersionDiffSummary = (version: Version) => {
    if (!version.diffSummary || version.diffSummary === "") {
      return null;
    }
    try {
      return JSON.parse(version.diffSummary);
    } catch (err) {
      throw err;
    }
  };

  const renderDiff = (version: Version) => {
    const downstream = selectedApp?.downstream;
    const diffSummary = getVersionDiffSummary(version);
    const hasDiffSummaryError =
      version.diffSummaryError && version.diffSummaryError.length > 0;

    if (hasDiffSummaryError) {
      return (
        <div className="flex flex1 alignItems--center u-marginTop--5">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
            Unable to generate diff{" "}
            <span className="link" onClick={() => toggleDiffErrModal(version)}>
              Why?
            </span>
          </span>
        </div>
      );
    } else if (diffSummary) {
      return (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--5">
          {diffSummary.filesChanged > 0 ? (
            <div className="DiffSummary u-marginRight--10">
              <span className="files">
                {diffSummary.filesChanged} files changed{" "}
              </span>
              {!downstream?.gitops?.isConnected && (
                <Link
                  className="u-fontSize--small link u-marginLeft--5"
                  to={`${location.pathname}?diff/${props.currentVersion?.sequence}/${version.parentSequence}`}
                >
                  View diff
                </Link>
              )}
            </div>
          ) : (
            <div className="DiffSummary">
              <span className="files">
                No changes to show.{" "}
                <span
                  className="link"
                  onClick={() => toggleNoChangesModal(version)}
                >
                  Why?
                </span>
              </span>
            </div>
          )}
        </div>
      );
    }
  };

  const renderYamlErrors = (version: Version) => {
    if (!version.yamlErrors) {
      return null;
    }
    return (
      <div className="flex alignItems--center u-marginTop--5">
        <Icon icon="warning-circle-filled" size={16} className="error-color" />
        <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5 u-textColor--error">
          {version.yamlErrors?.length} Invalid file
          {version.yamlErrors?.length !== 1 ? "s" : ""}{" "}
        </span>
        <span
          className="link u-marginLeft--5 u-fontSize--small"
          onClick={() =>
            toggleShowDetailsModal(version.yamlErrors, version.sequence)
          }
        >
          {" "}
          See details{" "}
        </span>
      </div>
    );
  };

  const onForceDeployClick = (continueWithFailedPreflights = false) => {
    setState({
      showSkipModal: false,
      showDeployWarningModal: false,
      displayShowDetailsModal: false,
    });
    const versionToDeploy = state.versionToDeploy;
    if (versionToDeploy) {
      deployVersion(versionToDeploy, true, continueWithFailedPreflights);
    } else {
      throw new Error("No version to deploy");
    }
  };

  const actionButtonStatus = (version: Version) => {
    const isDeploying = version.status === "deploying";
    const isDownloading =
      state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion;
    const isPendingDownload = version.status === "pending_download";
    const needsConfiguration = version.status === "pending_config";
    const canUpdateKots =
      version.needsKotsUpgrade &&
      !props.adminConsoleMetadata?.isAirgap &&
      !props.adminConsoleMetadata?.isKurl;

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
    }
    if (needsConfiguration) {
      return "Configure";
    } else {
      if (canUpdateKots) {
        return "Upgrade";
      } else {
        return "Deploy";
      }
    }
  };

  const updateVersionDownloadStatus = (version: Version) => {
    return new Promise<void>((resolve, reject) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/sequence/${version?.parentSequence}/task/updatedownload`,
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
            versionDownloadStatusJobs[version.sequence].stop();

            setState({
              versionDownloadStatuses: {
                ...state.versionDownloadStatuses,
                [version.sequence]: {
                  downloadingVersion: false,
                  downloadingVersionMessage: response.currentMessage,
                  downloadingVersionError: response.status === "failed",
                },
              },
            });

            if (props.refetchData) {
              props.refetchData();
            }
            if (props.downloadCallback) {
              props.downloadCallback();
            }
          } else {
            setState({
              versionDownloadStatuses: {
                ...state.versionDownloadStatuses,
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

  const downloadVersion = (version: Version) => {
    if (!versionDownloadStatusJobs?.hasOwnProperty(version.sequence)) {
      versionDownloadStatusJobs[version.sequence] = new Repeater();
    }

    setState({
      versionDownloadStatuses: {
        ...state.versionDownloadStatuses,
        [version.sequence]: {
          downloadingVersion: true,
          downloadingVersionMessage: "",
          downloadingVersionError: false,
        },
      },
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/sequence/${version.parentSequence}/download`,
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
          setState({
            versionDownloadStatuses: {
              ...state.versionDownloadStatuses,
              [version.sequence]: {
                downloadingVersion: false,
                downloadingVersionMessage: response.error,
                downloadingVersionError: true,
              },
            },
          });
          return;
        }
        versionDownloadStatusJobs[version.sequence].start(
          () => updateVersionDownloadStatus(version),
          1000
        );
      })
      .catch((err) => {
        console.log(err);
        setState({
          versionDownloadStatuses: {
            ...state.versionDownloadStatuses,
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

  const renderGitopsVersionAction = (version: Version) => {
    const downstream = selectedApp?.downstream;
    const nothingToCommit =
      downstream?.gitops?.isConnected && !version?.commitUrl;

    if (version.status === "pending_download") {
      const isDownloading =
        state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion;
      return (
        <div className="flex flex1 alignItems--center justifyContent--flexEnd">
          {renderReleaseNotes(version)}
          <button
            className="btn secondary blue u-marginLeft--10"
            disabled={isDownloading}
            onClick={() => downloadVersion(version)}
          >
            {isDownloading ? "Downloading" : "Download"}
          </button>
        </div>
      );
    }
    if (version.gitDeployable === false) {
      return (
        <div className={nothingToCommit ? "u-opacity--half" : ""}>
          Nothing to commit
        </div>
      );
    }
    if (!version.commitUrl) {
      return (
        <div className="flex flex1 alignItems--center justifyContent--flexEnd">
          <span className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal">
            No commit URL found
          </span>
          <Icon
            icon="info-circle-outline"
            size={16}
            className="gray-color u-marginLeft--5"
            data-tip="This version may have been created before GitOps was enabled"
          />
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
  };

  const isActionButtonDisabled = (version: Version) => {
    if (state.versionDownloadStatuses?.[version.sequence]?.downloadingVersion) {
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
  };

  const getKotsUpdateStatus = () => {
    // TODO: handle with both resolve and reject or use async/await
    return new Promise<void>((resolve) => {
      fetch(
        `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/task/update-admin-console`,
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
            state.kotsUpdateChecker.stop();
            window.location.reload();
          }

          const response = await res.json();
          if (response.status === "successful") {
            window.location.reload();
          } else {
            setState({
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
          setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "waiting",
            kotsUpdateMessage: "Waiting for pods to restart...",
            kotsUpdateError: null,
          });
          resolve();
        });
    });
  };

  const upgradeAdminConsole = (version: Version) => {
    setState({
      displayKotsUpdateModal: true,
      kotsUpdateRunning: true,
      kotsUpdateStatus: null,
      kotsUpdateMessage: null,
      kotsUpdateError: null,
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/sequence/${version.parentSequence}/update-console`,
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
          setState({
            kotsUpdateRunning: false,
            kotsUpdateStatus: "failed",
            kotsUpdateError: response.error,
          });
          return;
        }
        state.kotsUpdateChecker.start(getKotsUpdateStatus, 1000);
      })
      .catch((err) => {
        console.log(err);
        setState({
          kotsUpdateRunning: false,
          kotsUpdateStatus: "failed",
          kotsUpdateError:
            err?.message || "Something went wrong, please try again.",
        });
      });
  };

  const renderVersionAction = (version: Version) => {
    const downstream = selectedApp?.downstream;

    if (downstream?.gitops?.isConnected) {
      return renderGitopsVersionAction(version);
    }

    const needsConfiguration = version.status === "pending_config";
    const isPendingDownload = version.status === "pending_download";
    const isSecondaryActionBtn = needsConfiguration || isPendingDownload;

    return (
      <div className="flex flex1 alignItems--center justifyContent--flexEnd">
        {renderReleaseNotes(version)}
        {renderPreflights(version)}
        <EditConfigIcon version={version} isPending={true} />
        <div className="flex-column justifyContent--center u-marginLeft--10">
          <button
            className={classNames("btn", {
              "secondary blue": isSecondaryActionBtn,
              "primary blue": !isSecondaryActionBtn,
            })}
            disabled={isActionButtonDisabled(version)}
            onClick={() => {
              if (needsConfiguration) {
                navigate(
                  `/app/${selectedApp?.slug}/config/${version.sequence}`
                );
                return;
              }
              if (version.needsKotsUpgrade) {
                upgradeAdminConsole(version);
                return;
              }
              if (isPendingDownload) {
                downloadVersion(version);
                return;
              }
              deployVersion(version);
            }}
          >
            <span
              key={version.nonDeployableCause}
              data-tip-disable={!isActionButtonDisabled(version)}
              data-tip={version.nonDeployableCause}
              data-for="disable-deployment-tooltip"
            >
              {actionButtonStatus(version)}
            </span>
          </button>
          <ReactTooltip effect="solid" id="disable-deployment-tooltip" />
        </div>
      </div>
    );
  };

  const renderVersionDownloadStatus = (version: Version) => {
    const { versionDownloadStatuses } = state;

    if (!versionDownloadStatuses.hasOwnProperty(version.sequence)) {
      // user hasn't tried to re-download the version yet, show last known download status if exists
      if (version.downloadStatus) {
        return (
          <div className="flex alignItems--center justifyContent--flexEnd">
            <span
              className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
                version.downloadStatus.status === "failed"
                  ? "u-textColor--error"
                  : ""
              }`}
            >
              {version.downloadStatus.message}
            </span>
          </div>
        );
      }
      return null;
    }

    const status = versionDownloadStatuses[version.sequence];

    return (
      <div className="flex alignItems--center justifyContent--flexEnd">
        {status.downloadingVersion && (
          <Loader className="u-marginRight--5" size="15" />
        )}
        <span
          className={`u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default ${
            status.downloadingVersionError ? "u-textColor--error" : ""
          }`}
        >
          {status.downloadingVersionMessage
            ? status.downloadingVersionMessage
            : status.downloadingVersion
            ? "Downloading"
            : ""}
        </span>
      </div>
    );
  };

  const shouldRenderUpdateProgress = () => {
    if (props.uploadingAirgapFile) {
      return true;
    }
    if (props.isBundleUploading) {
      return true;
    }
    if (props.checkingForUpdateError) {
      return true;
    }
    if (props.airgapUploadError) {
      return true;
    }
    if (selectedApp?.isAirgap && props.checkingForUpdates) {
      return true;
    }
    return false;
  };

  const renderUpdateProgress = () => {
    const {
      checkingForUpdateError,
      checkingUpdateText,
      isBundleUploading,
      uploadingAirgapFile,
      checkingForUpdates,
      airgapUploadError,
    } = props;

    let updateText;
    if (airgapUploadError) {
      updateText = (
        <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--small u-textColor--error u-fontWeight--medium">
          Error uploading bundle
          <span
            className="link u-textDecoration--underlineOnHover u-marginLeft--5"
            onClick={props.viewAirgapUploadError}
          >
            See details
          </span>
        </p>
      );
    } else if (checkingForUpdateError) {
      updateText = (
        <div className="flex-column flex-auto u-marginTop--10">
          <p className="u-fontSize--normal u-marginBottom--5 u-textColor--error u-fontWeight--medium">
            Error updating version:
          </p>
          <p className="u-fontSize--small u-textColor--error u-lineHeight--normal u-fontWeight--medium">
            {checkingUpdateText}
          </p>
        </div>
      );
    } else if (uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          appSlug={selectedApp?.slug}
          total={props.uploadSize}
          progress={props.uploadProgress}
          resuming={props.uploadResuming}
          onProgressError={props.onProgressError}
          smallSize={true}
        />
      );
    } else if (isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          appSlug={selectedApp?.slug}
          unkownProgress={true}
          onProgressError={props.onProgressError}
          smallSize={true}
        />
      );
    } else if (checkingForUpdates) {
      let shortText = checkingUpdateText;
      if (shortText && shortText.length > 65) {
        shortText = shortText.slice(0, 65) + "...";
      }
      updateText = (
        <div className="flex-column justifyContent--center alignItems--center">
          <Loader className="u-marginBottom--10" size="30" />
          <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--normal u-lineHeight--default">
            {shortText}
          </span>
        </div>
      );
    }

    return (
      <div className="VersionCard-content--wrapper card-item u-marginTop--15">
        {updateText}
      </div>
    );
  };

  const renderBottomSection = () => {
    if (shouldRenderUpdateProgress()) {
      return renderUpdateProgress();
    }

    if (state.latestDeployableVersionErrMsg) {
      return (
        <div className="error-block-wrapper u-marginTop--20 u-marginBottom--10 flex flex1">
          <span className="u-textColor--error">
            {state.latestDeployableVersionErrMsg}
          </span>
        </div>
      );
    }

    if (!latestDeployableVersion) {
      return null;
    }

    const downstream = props.downstream;
    const downstreamSource = latestDeployableVersion?.source;
    const gitopsIsConnected = downstream?.gitops?.isConnected;
    const isNew = secondsAgo(latestDeployableVersion?.createdOn) < 10;

    return (
      <div className="u-marginTop--20">
        <p className="u-fontSize--normal u-lineHeight--normal u-textColor--info u-fontWeight--medium">
          New version available
        </p>
        {gitopsIsConnected && (
          <div className="gitops-enabled-block u-fontSize--small u-fontWeight--medium flex alignItems--center u-textColor--header u-marginTop--10">
            <span
              className={`icon gitopsService--${downstream?.gitops?.provider} u-marginRight--10`}
            />
            GitOps is enabled for this application. Versions are tracked{" "}
            {selectedApp?.isAirgap ? "at" : "on"}&nbsp;
            <a
              target="_blank"
              rel="noopener noreferrer"
              href={downstream?.gitops?.uri}
              className="link"
            >
              {selectedApp?.isAirgap
                ? downstream?.gitops?.uri
                : getReadableGitOpsProviderName(downstream?.gitops?.provider)}
            </a>
          </div>
        )}
        <div className="VersionCard-content--wrapper card-item u-marginTop--15">
          <div
            className={`flex ${
              isNew && !selectedApp?.isAirgap ? "is-new" : ""
            }`}
          >
            <div className="flex-column">
              <div className="flex alignItems--center">
                <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title">
                  {latestDeployableVersion.versionLabel ||
                    latestDeployableVersion.title}
                </p>
                <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-marginLeft--10">
                  Sequence {latestDeployableVersion.sequence}
                </p>
                {latestDeployableVersion.isRequired && (
                  <span className="status-tag required u-marginLeft--10">
                    {" "}
                    Required{" "}
                  </span>
                )}
              </div>
              <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
                {" "}
                Released{" "}
                {Utilities.dateFormat(
                  latestDeployableVersion?.createdOn,
                  "MM/DD/YY @ hh:mm a z"
                )}{" "}
              </p>
              {renderDiff(latestDeployableVersion)}
              {renderYamlErrors(latestDeployableVersion)}
            </div>
            <div className="flex alignItems--center u-paddingLeft--20">
              <p className="u-fontSize--small u-fontWeight--bold u-textColor--lightAccent u-lineHeight--default">
                {downstreamSource}
              </p>
            </div>
            {renderVersionAction(latestDeployableVersion)}
          </div>
          {renderVersionDownloadStatus(latestDeployableVersion)}
        </div>
        {(state.numOfSkippedVersions > 0 ||
          state.numOfRemainingVersions > 0) && (
          <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--more u-textColor--info u-marginTop--10">
            {state.numOfSkippedVersions > 0
              ? `${state.numOfSkippedVersions} version${
                  state.numOfSkippedVersions > 1 ? "s" : ""
                } will be skipped in upgrading to ${
                  latestDeployableVersion.versionLabel
                }. `
              : ""}
            {state.numOfRemainingVersions > 0
              ? "Additional versions are available after you deploy this required version."
              : ""}
          </p>
        )}
      </div>
    );
  };

  const {
    currentVersion,
    checkingForUpdates,
    checkingUpdateText,
    isBundleUploading,
    airgapUploader,
  } = props;

  const gitopsIsConnected = props.downstream?.gitops?.isConnected;

  let checkingUpdateTextShort = checkingUpdateText;
  if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
    checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
  }

  const renderKotsUpgradeStatus =
    state.kotsUpdateStatus && !state.kotsUpdateMessage;
  let shortKotsUpdateMessage = state.kotsUpdateMessage;
  if (shortKotsUpdateMessage && shortKotsUpdateMessage.length > 60) {
    shortKotsUpdateMessage = shortKotsUpdateMessage.substring(0, 60) + "...";
  }

  if (gitopsIsConnected) {
    return (
      <DashboardGitOpsCard
        gitops={props.downstream?.gitops}
        isAirgap={selectedApp?.isAirgap}
        appSlug={selectedApp?.slug}
        checkingForUpdates={checkingForUpdates}
        latestConfigSequence={
          selectedApp?.downstream?.pendingVersions[0]?.parentSequence
        }
        isBundleUploading={isBundleUploading}
        checkingUpdateText={checkingUpdateText}
        checkingUpdateTextShort={checkingUpdateTextShort}
        noUpdatesAvalable={props.noUpdatesAvalable}
        onCheckForUpdates={props.onCheckForUpdates}
        showAutomaticUpdatesModal={props.showAutomaticUpdatesModal}
      />
    );
  }

  return (
    <div className="flex-column flex1 dashboard-card card-bg" data-testid="dashboard-version-card">
      <div className="flex flex1 justifyContent--spaceBetween alignItems--center u-marginBottom--10">
        <p className="card-title">Version</p>
        {!props.adminConsoleMetadata?.isEmbeddedCluster && (
          <div className="flex alignItems--center">
            {selectedApp?.isAirgap && airgapUploader ? (
              <MountAware
                onMount={(el: Element) =>
                  props.airgapUploader?.assignElement(el)
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
                {checkingForUpdates && !isBundleUploading ? (
                  <div className="flex alignItems--center u-marginRight--20">
                    <Loader className="u-marginRight--5" size="15" />
                    <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                      {checkingUpdateText === ""
                        ? "Checking for updates"
                        : checkingUpdateTextShort}
                    </span>
                  </div>
                ) : props.noUpdatesAvalable ? (
                  <div className="flex alignItems--center u-marginRight--20">
                    <span className="u-textColor--info u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                      Already up to date
                    </span>
                  </div>
                ) : (
                  <div className="flex alignItems--center u-marginRight--20 link">
                    <Icon
                      icon="check-update"
                      size={18}
                      className="clickable u-marginRight--5"
                    />
                    <span
                      className="u-fontSize--small"
                      onClick={props.onCheckForUpdates}
                    >
                      Check for update
                    </span>
                  </div>
                )}
                <div className="flex alignItems--center u-marginRight--20 link">
                  <Icon
                    icon="schedule-sync"
                    size={18}
                    className=" clickable u-marginRight--5"
                  />
                  <span
                    className="u-fontSize--small u-lineHeight--default"
                    onClick={props.showAutomaticUpdatesModal}
                  >
                    Configure automatic updates
                  </span>
                </div>
              </div>
            )}
          </div>
        )}
        {props.adminConsoleMetadata?.isEmbeddedCluster && (
          <div className="flex alignItems--center">
            {!state.isFetchingAvailableUpdates && (
              <span
                className="flex-auto flex alignItems--center link u-fontSize--small"
                onClick={() => fetchAvailableUpdates()}
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
            )}
            {state.isFetchingAvailableUpdates && (
              <div className="flex alignItems--center">
                <Loader className="u-marginRight--5" size="15" />
                <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                  Checking for updates
                </span>
              </div>
            )}
          </div>
        )}
      </div>
      {currentVersion?.deployedAt ? (
        <div className="VersionCard-content--wrapper card-item">
          {renderCurrentVersion()}
        </div>
      ) : (
        <div className="no-deployed-version u-textAlign--center">
          <p className="u-fontWeight--medium u-fontSize--normal u-textColor--bodyCopy">
            {" "}
            No version has been deployed{" "}
          </p>
        </div>
      )}
      {props.adminConsoleMetadata?.isEmbeddedCluster &&
        state.availableUpdates?.length > 0 && (
          <AvailableUpdateCard
            updates={state.availableUpdates}
            showReleaseNotes={showReleaseNotes}
            appSlug={params.slug}
          />
        )}
      {renderBottomSection()}
      <div className="u-marginTop--10">
        <Link
          to={`/app/${selectedApp?.slug}/version-history`}
          className="link u-fontSize--small"
        >
          See all versions
          <Icon
            icon="next-arrow"
            size={10}
            className="has-arrow u-marginLeft--5"
          />
        </Link>
      </div>
      {state.showReleaseNotes && (
        <Modal
          isOpen={state.showReleaseNotes}
          onRequestClose={hideReleaseNotes}
          contentLabel="Release Notes"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="flex-column">
            <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
              {state.releaseNotes || "No release notes for this version"}
            </MarkdownRenderer>
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={hideReleaseNotes}>
              Close
            </button>
          </div>
        </Modal>
      )}
      {state.showLogsModal && (
        <ShowLogsModal
          showLogsModal={state.showLogsModal}
          hideLogsModal={hideLogsModal}
          viewLogsErrMsg={state.viewLogsErrMsg}
          versionFailing={state.versionFailing}
          troubleshootUrl={`/app/${selectedApp?.slug}/troubleshoot`}
          logs={state.logs}
          selectedTab={state.selectedTab}
          logsLoading={state.logsLoading}
          renderLogsTabs={renderLogsTabs()}
        />
      )}
      {state.showDiffErrModal && (
        <Modal
          isOpen={true}
          onRequestClose={() => toggleDiffErrModal()}
          contentLabel="Unable to Get Diff"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              Unable to generate a file diff for release
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              The{" "}
              <span className="u-fontWeight--bold">
                {/* // TODO: add better error handling */}
                Upstream {state.releaseWithErr?.versionLabel || ""}, Sequence{" "}
                {state.releaseWithErr?.sequence || ""}
              </span>{" "}
              release was unable to generate a diff because the following error:
            </p>
            <div className="error-block-wrapper u-marginBottom--30 flex flex1">
              <span className="u-textColor--error">
                {state.releaseWithErr?.diffSummaryError || ""}
              </span>
            </div>
            <div className="flex u-marginBottom--10">
              <button
                className="btn primary"
                onClick={() => toggleDiffErrModal()}
              >
                Ok, got it!
              </button>
            </div>
          </div>
        </Modal>
      )}
      {state.showNoChangesModal && (
        <Modal
          isOpen={true}
          onRequestClose={() => toggleNoChangesModal()}
          contentLabel="No Changes"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              No changes to show
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              The{" "}
              <span className="u-fontWeight--bold">
                Upstream {state.releaseWithNoChanges?.versionLabel}, Sequence{" "}
                {state.releaseWithNoChanges?.sequence}
              </span>{" "}
              release was unable to generate a diff because the changes made do
              not affect any manifests that will be deployed. Only changes
              affecting the application manifest will be included in a diff.
            </p>
            <div className="flex u-paddingTop--10">
              <button
                className="btn primary"
                onClick={() => toggleNoChangesModal()}
              >
                Ok, got it!
              </button>
            </div>
          </div>
        </Modal>
      )}
      {state.displayConfirmDeploymentModal && (
        <Modal
          isOpen={true}
          onRequestClose={() =>
            setState({
              displayConfirmDeploymentModal: false,
              versionToDeploy: null,
              isRedeploy: false,
            })
          }
          contentLabel="Confirm deployment"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              {state.isRedeploy ? "Redeploy" : "Deploy"}{" "}
              {state.versionToDeploy?.versionLabel} (Sequence{" "}
              {state.versionToDeploy?.sequence})?
            </p>
            <div className="flex u-paddingTop--10">
              <button
                className="btn secondary blue"
                onClick={() =>
                  setState({
                    displayConfirmDeploymentModal: false,
                    versionToDeploy: null,
                    isRedeploy: false,
                  })
                }
              >
                Cancel
              </button>
              <button
                className="u-marginLeft--10 btn primary"
                onClick={() => finalizeDeployment(false, state.isRedeploy)}
              >
                Yes, {state.isRedeploy ? "Redeploy" : "Deploy"}
              </button>
            </div>
          </div>
        </Modal>
      )}
      {state.displayKotsUpdateModal && (
        <Modal
          isOpen={true}
          onRequestClose={() => setState({ displayKotsUpdateModal: false })}
          contentLabel="Upgrade is in progress"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="Modal-body u-textAlign--center">
            <div className="flex-column justifyContent--center alignItems--center">
              <p className="u-fontSize--large u-textColor--primary u-lineHeight--bold u-marginBottom--10">
                Upgrading...
              </p>
              <Loader className="flex alignItems--center" size="32" />
              {renderKotsUpgradeStatus ? (
                <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                  {state.kotsUpdateStatus}
                </p>
              ) : null}
              {state.kotsUpdateMessage ? (
                <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                  {shortKotsUpdateMessage}
                </p>
              ) : null}
            </div>
          </div>
        </Modal>
      )}
      {state.displayShowDetailsModal && (
        <ShowDetailsModal
          displayShowDetailsModal={state.displayShowDetailsModal}
          toggleShowDetailsModal={toggleShowDetailsModal}
          yamlErrorDetails={state.yamlErrorDetails}
          deployView={state.deployView}
          forceDeploy={onForceDeployClick}
          showDeployWarningModal={state.showDeployWarningModal}
          showSkipModal={state.showSkipModal}
          slug={params.slug}
          sequence={state.selectedSequence}
        />
      )}
      {state.showDeployWarningModal && (
        <DeployWarningModal
          showDeployWarningModal={state.showDeployWarningModal}
          hideDeployWarningModal={() =>
            setState({ showDeployWarningModal: false })
          }
          onForceDeployClick={onForceDeployClick}
        />
      )}
      {state.showSkipModal && (
        <SkipPreflightsModal
          showSkipModal={true}
          hideSkipModal={() => setState({ showSkipModal: false })}
          onForceDeployClick={onForceDeployClick}
        />
      )}
      {state.showDiffModal && (
        <Modal
          isOpen={true}
          onRequestClose={closeViewDiffModal}
          contentLabel="Release Diff Modal"
          ariaHideApp={false}
          className="Modal DiffViewerModal"
        >
          <div className="DiffOverlay" data-testid="diff-overlay">
            <VersionDiff
              slug={params.slug}
              firstSequence={state.firstSequence}
              secondSequence={state.secondSequence}
              hideBackButton={true}
              onBackClick={closeViewDiffModal}
              app={selectedApp}
            />
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={closeViewDiffModal}>
              Close
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
};

// TODO: remove default export
export default DashboardVersionCard;
