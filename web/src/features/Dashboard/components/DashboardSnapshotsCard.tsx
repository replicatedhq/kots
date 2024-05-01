import { useEffect, useReducer } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import { Link, useNavigate } from "react-router-dom";
import find from "lodash/find";
import "@src/scss/components/watches/DashboardCard.scss";
import InlineDropdown from "@src/components/shared/InlineDropdown";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";
import Icon from "@src/components/Icon";
import { App, SnapshotSettings } from "@types";
import { usePrevious } from "@src/hooks/usePrevious";
import { useCreateSnapshot } from "../api/createSnapshot";
import { useSnapshotSettings } from "../api/getSnapshotSettings";
import { Utilities } from "@src/utilities/utilities";

const DESTINATIONS = [
  {
    value: "aws",
    label: "Amazon S3",
  },
  {
    value: "azure",
    label: "Azure Blob Storage",
  },
  {
    value: "gcp",
    label: "Google Cloud Storage",
  },
  {
    value: "other",
    label: "S3-Compatible Storage",
  },
  {
    value: "internal",
    label: "Internal Storage (Default)",
  },
  {
    value: "nfs",
    label: "Network File System (NFS)",
  },
  {
    value: "hostpath",
    label: "Host Path",
  },
];

type Props = {
  app: App;
  isSnapshotAllowed: boolean;
  isEmbeddedCluster: boolean;
  ping: (clusterId?: string) => void;
};

type State = {
  determiningDestination: boolean;
  isLoadingSnapshotSettings: boolean;
  kotsadmRequiresVeleroAccess: boolean;
  locationStr: string;
  minimalRBACKotsadmNamespace: string;
  readableName: string | undefined;
  selectedDestination: { value: string; label: string } | undefined;
  snapshotDifferencesModal: boolean;
  snapshotSettingsErr: boolean;
  snapshotSettingsErrMsg: string;
  startingSnapshot: boolean;
  startSnapshotErr: boolean;
  startSnapshotErrorMsg: string;
};

export const DashboardSnapshotsCard = (props: Props) => {
  const navigate = useNavigate();

  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      determiningDestination: false,
      isLoadingSnapshotSettings: false,
      kotsadmRequiresVeleroAccess: false,
      locationStr: "",
      minimalRBACKotsadmNamespace: "",
      readableName: "",
      selectedDestination: undefined,
      snapshotDifferencesModal: false,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      startingSnapshot: false,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
    }
  );
  const { app, ping, isSnapshotAllowed, isEmbeddedCluster } = props;
  const { selectedDestination } = state;

  const setCurrentProvider = (
    snapshotSettings: SnapshotSettings | undefined
  ) => {
    if (!snapshotSettings) {
      return;
    }
    const { store } = snapshotSettings;

    if (store?.aws) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "aws"]),
        locationStr: Utilities.snapshotLocationStr(store?.bucket, store?.path),
      });
    }

    if (store?.azure) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "azure"]),
        locationStr: Utilities.snapshotLocationStr(store?.bucket, store?.path),
      });
    }

    if (store?.gcp) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "gcp"]),
        locationStr: Utilities.snapshotLocationStr(store?.bucket, store?.path),
      });
    }

    if (store?.other) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "other"]),
        locationStr: Utilities.snapshotLocationStr(store?.bucket, store?.path),
      });
    }

    if (store?.internal) {
      return setState({
        determiningDestination: false,
        selectedDestination: find(DESTINATIONS, ["value", "internal"]),
      });
    }

    if (store?.fileSystem) {
      const { fileSystemConfig } = snapshotSettings;
      return setState({
        selectedDestination: fileSystemConfig?.hostPath
          ? find(DESTINATIONS, ["value", "hostpath"])
          : find(DESTINATIONS, ["value", "nfs"]),
        locationStr: fileSystemConfig?.hostPath
          ? fileSystemConfig?.hostPath.path
          : fileSystemConfig?.nfs?.path,
      });
    }

    // if nothing exists yet, we've determined default state is good
    let defaultDestination = "aws";
    if (isEmbeddedCluster) {
      defaultDestination = "other";
    }
    setState({
      determiningDestination: false,
      selectedDestination: find(DESTINATIONS, ["value", defaultDestination]),
    });
  };

  const onSnapshotSettingsSuccess = (result: SnapshotSettings) => {
    setState({
      kotsadmRequiresVeleroAccess: false,
      isLoadingSnapshotSettings: false,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
    });

    setCurrentProvider(result);
  };
  const onSnapshotSettingsError = (err: Error) => {
    setState({
      isLoadingSnapshotSettings: false,
      snapshotSettingsErr: true,
      snapshotSettingsErrMsg: err.message,
    });
  };

  const { data: snapshotSettings } = useSnapshotSettings(
    onSnapshotSettingsSuccess,
    onSnapshotSettingsError
  );
  const previousSnapshotSettings = usePrevious(snapshotSettings);

  const onCreateSnapshotSuccess = (data: {
    success?: boolean;
    option?: "full" | "partial" | undefined;
  }) => {
    setState({
      startingSnapshot: false,
    });
    ping();
    if (data.option === "full") {
      navigate("/snapshots");
    } else {
      navigate(`/snapshots/partial/${app.slug}`);
    }
  };

  const onCreateSnapshotError = (err: Error) => {
    setState({
      startingSnapshot: false,
      startSnapshotErr: true,
      startSnapshotErrorMsg: err.message,
    });
  };

  const { mutate: createSnapshot } = useCreateSnapshot(
    onCreateSnapshotSuccess,
    onCreateSnapshotError
  );

  const toggleSnaphotDifferencesModal = () => {
    setState({
      snapshotDifferencesModal: !state.snapshotDifferencesModal,
    });
  };

  useEffect(() => {
    if (snapshotSettings !== previousSnapshotSettings && snapshotSettings) {
      setCurrentProvider(snapshotSettings);
    }
  }, []);
  /// useEffects /////

  return (
    <div className="flex-column flex1 dashboard-card card-bg">
      <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
        <p className="card-title">
          {isEmbeddedCluster ? "Disaster Recovery" : "Snapshots"}
        </p>
        <div className="u-fontSize--small u-fontWeight--medium flex flex-auto alignItems--center">
          <Link
            className="link u-marginRight--20 flex alignItems--center"
            to="/snapshots/settings"
          >
            <Icon
              icon="settings-gear-outline"
              size={16}
              className="clickable u-marginRight--5"
            />
            {isEmbeddedCluster ? "Backup settings" : "Snapshot settings"}
          </Link>
          {!isEmbeddedCluster && (
            <>
              <Icon
                icon="schedule-update"
                size={16}
                className="clickable u-marginRight--5"
              />
              <InlineDropdown
                defaultDisplayText="Start snapshot"
                dropdownOptions={[
                  {
                    displayText: "Start a Partial snapshot",
                    onClick: () => createSnapshot("partial"),
                  },
                  {
                    displayText: "Start a Full snapshot",
                    onClick: () => createSnapshot("full"),
                  },
                  {
                    displayText: "Learn about the difference",
                    onClick: () => toggleSnaphotDifferencesModal(),
                  },
                ]}
              />
            </>
          )}
          {isEmbeddedCluster && (
            <Link
              className="link u-marginRight--20 flex alignItems--center"
              onClick={() => createSnapshot("full")}
              to=""
            >
              <Icon
                icon="schedule-update"
                size={16}
                className="clickable u-marginRight--5"
              />
              Start backup
            </Link>
          )}
        </div>
      </div>
      <div className="SnapshotsCard-content--wrapper u-marginTop--10 flex flex1">
        <div className="flex1">
          {!isEmbeddedCluster && (
            <>
              <span
                className={`status-dot ${
                  isSnapshotAllowed ? "u-color--success" : "u-color--warning"
                }`}
              />
              <span
                className={`u-fontSize--small u-fontWeight--medium ${
                  isSnapshotAllowed
                    ? "u-textColor--success"
                    : "u-textColor--warning"
                }`}
              >
                {isSnapshotAllowed ? "Enabled" : "Disabled"}
              </span>
            </>
          )}
          <div
            className={`flex alignItems--center ${
              !isEmbeddedCluster && "u-marginTop--10"
            }`}
          >
            <span
              className={`icon snapshotDestination--${selectedDestination?.value} u-marginRight--5`}
            />
            <p className="u-fontSize--normal u-fontWeight--medium card-item-title">
              {selectedDestination?.label}
            </p>
          </div>
          {selectedDestination?.value !== "internal" && (
            <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--10">
              {state.locationStr}
            </p>
          )}
        </div>
        <div className="flex-auto">
          <div className="u-color--taupe u-padding--10">
            <p></p>
          </div>
        </div>
      </div>
      <div className="u-marginTop--10">
        <Link to={`/snapshots`} className="link u-fontSize--small">
          See all {isEmbeddedCluster ? "backups" : "snapshots"}
          <Icon
            icon="next-arrow"
            size={10}
            className="has-arrow u-marginLeft--5"
          />
        </Link>
        <p className="tw-mt-2 u-textColor--error u-fontSize--normal u-lineHeight--normal">
          {state.startSnapshotErrorMsg}
        </p>
      </div>
      {state.snapshotDifferencesModal && (
        <SnapshotDifferencesModal
          snapshotDifferencesModal={state.snapshotDifferencesModal}
          toggleSnapshotDifferencesModal={toggleSnaphotDifferencesModal}
        />
      )}
    </div>
  );
};

/* eslint-disable */
// @ts-ignore
export default withRouter(DashboardSnapshotsCard) as any;
/* eslint-enable */
