import React, { useEffect, useReducer } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import { Link, useHistory } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import find from "lodash/find";
import "@src/scss/components/watches/DashboardCard.scss";
import InlineDropdown from "@src/components/shared/InlineDropdown";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";
import Icon from "@src/components/Icon";
import { App, KotsParams, Snapshot } from "@types";
import { RouteComponentProps } from "react-router-dom";
import { usePrevious } from "@src/hooks/usePrevious";
import { useCreateSnapshot } from "../api/createSnapshot";
import { useSnapshotSettings } from "../api/getSnapshotSettings";

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
    label: "Other S3-Compatible Storage",
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
type SnapshotSettings = {
  store: {
    aws: {
      region: string;
      bucket: string;
      accessKeyId: string;
      secretAccessKey: string;
    };
    azure: { accountName: string; accountKey: string; container: string };
    gcp: { bucket: string; projectId: string; serviceAccountKey: string };
    other: {
      endpoint: string;
      bucket: string;
      accessKeyId: string;
      secretAccessKey: string;
    };
    internal: { bucket: string };
    nfs: { server: string; path: string };
    hostpath: { path: string };
    bucket: string;
    path: string;
    fileSystem: string;
  };
  fileSystemConfig: {
    nfs: { server: string; path: string };
    hostPath: { path: string };
  };
};

type Props = {
  app: App;
  isSnapshotAllowed: boolean;
  ping: (clusterId?: string) => void;
} & RouteComponentProps<KotsParams>;

type State = {
  determiningDestination: boolean;
  isLoadingSnapshotSettings: boolean;
  kotsadmRequiresVeleroAccess: boolean;
  locationStr: string;
  minimalRBACKotsadmNamespace: string;
  readableName: string | undefined;
  selectedDestination: { value: string; label: string } | undefined;
  snapshotDifferencesModal: boolean;
  snapshotSettings: SnapshotSettings | null;
  snapshotSettingsErr: boolean;
  snapshotSettingsErrMsg: string;
  startingSnapshot: boolean;
  startSnapshotErr: boolean;
  startSnapshotErrorMsg: string;
};

export const DashboardSnapshotsCard = (props: Props) => {
  const history = useHistory();

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
      snapshotSettings: null,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      startingSnapshot: false,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
    }
  );
  const { app, ping, isSnapshotAllowed } = props;
  const { selectedDestination } = state;
  const previousSnapshotSettings = usePrevious(state.snapshotSettings);

  const setCurrentProvider = (snapshotSettings: Snapshot | undefined) => {
    console.log("snapshotSettings", snapshotSettings);
    if (!snapshotSettings) {
      return;
    }
    const { store } = snapshotSettings;

    if (store?.aws) {
      return setState({
        readableName: find(DESTINATIONS, ["value", "aws"])?.label,
        locationStr: `${store?.bucket}${store?.path ? `/${store?.path}` : ""}`,
      });
    }

    if (store?.azure) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "azure"]),
        locationStr: `${store?.bucket}${store?.path ? `/${store?.path}` : ""}`,
      });
    }

    if (store?.gcp) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "gcp"]),
        locationStr: `${store?.bucket}${store?.path ? `/${store?.path}` : ""}`,
      });
    }

    if (store?.other) {
      return setState({
        selectedDestination: find(DESTINATIONS, ["value", "other"]),
        locationStr: `${store?.bucket}${store?.path ? `/${store?.path}` : ""}`,
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
    setState({
      determiningDestination: false,
      selectedDestination: find(DESTINATIONS, ["value", "aws"]),
    });
  };

  const onCreateSnapshotSuccess = ({
    option,
  }: {
    success?: boolean;
    option: "full" | "partial";
  }) => {
    setState({
      startingSnapshot: false,
    });
    ping();
    if (option === "full") {
      history.push("/snapshots");
    } else {
      history.push(`/snapshots/partial/${app.slug}`);
    }
  };

  const onCreateSnapshotError = (err: Error) => {
    // need to figure out a way to handle 409 error code,
    // not important here since we render this component when velero is installed
    // will be useful if we use useCreateSnapshot hook in Snapshots.
    // if (err.status === 409) {
    //           if (res.kotsadmRequiresVeleroAccess) {
    //             setState({
    //               startingSnapshot: false,
    //             });
    //             props.history.replace("/snapshots/settings");
    //             return;
    //           }
    //         }
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

  const fetchSnapshotSettings = async () => {
    setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      minimalRBACKotsadmNamespace: "",
    });

    fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            setState({ isLoadingSnapshotSettings: false });
            // requires velero access so do something here to show that
            // openConfigureSnapshotsMinimalRBACModal(result.kotsadmRequiresVeleroAccess, result.kotsadmNamespace);
            return;
          }
        }

        const result = await res.json();
        setState({
          snapshotSettings: result,
          kotsadmRequiresVeleroAccess: false,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        });
        setCurrentProvider(result);
      })
      .catch((err) => {
        setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        });
      });
  };

  const onSuccess = (result: Snapshot) => {
    setState({
      snapshotSettings: result,
      kotsadmRequiresVeleroAccess: false,
      isLoadingSnapshotSettings: false,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
    });
    setCurrentProvider(result);
  };
  const onError = (err: Error) => {
    setState({
      isLoadingSnapshotSettings: false,
      snapshotSettingsErr: true,
      snapshotSettingsErrMsg: err.message,
    });
  };

  const { data: hookSnapshotSettings } = useSnapshotSettings(
    onSuccess,
    onError
  );

  const toggleSnaphotDifferencesModal = () => {
    setState({
      snapshotDifferencesModal: !state.snapshotDifferencesModal,
    });
  };

  /// useEffects /////
  useEffect(() => {
    fetchSnapshotSettings();
  }, []);

  useEffect(() => {
    if (
      state.snapshotSettings !== previousSnapshotSettings &&
      state.snapshotSettings
    ) {
      setCurrentProvider(state.snapshotSettings);
    }
  }, []);
  /// useEffects /////

  return (
    <div className="flex-column flex1 dashboard-card">
      <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
        <p className="card-title">Snapshots</p>
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
            Snapshot settings
          </Link>
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
        </div>
      </div>
      <div className="SnapshotsCard-content--wrapper u-marginTop--10 flex flex1">
        <div className="flex1">
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
          <div className="flex alignItems--center u-marginTop--10">
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
          See all snapshots
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
