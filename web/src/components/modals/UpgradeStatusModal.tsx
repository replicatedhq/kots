import { useEffect } from "react";
import { Utilities } from "../../utilities/utilities";
import { Link } from "react-router-dom";
import Loader from "@components/shared/Loader";

interface Props {
  status: string;
  message: string;
  appSlug: string;
  refetchStatus: (appSlug: string) => Promise<void>;
  closeModal: () => void;
  connectionTerminated: boolean;
  setTerminatedState: (terminated: boolean) => void;
}

const UpgradeStatusModal = (props: Props) => {
  const ping = async () => {
    await fetch(`${process.env.API_ENDPOINT}/ping`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          throw new Error(`Unexpected status code: ${res.status}`);
        }
        props.setTerminatedState(false);
      })
      .catch(() => {
        props.setTerminatedState(true);
      });
  };

  useEffect(() => {
    const interval = setInterval(() => {
      if (props.connectionTerminated) {
        ping();
      } else if (props.status !== "upgrade-failed") {
        props.refetchStatus(props.appSlug);
      }
    }, 10000);
    return () => clearInterval(interval);
  }, [props.connectionTerminated, props.status]);

  if (props.status === "upgrade-failed") {
    return (
      <div className="u-padding--25 tw-flex tw-flex-col tw-justify-center tw-items-center">
        <span className="icon redWarningIcon flex-auto" />
        <div className="flex flex-column alignItems--center u-marginTop--10">
          <p className="u-textColor--error u-fontSize--largest u-fontWeight--bold u-lineHeight--normal">
            Upgrade failed
          </p>
          <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textAlign--center">
            {props.message}
          </p>
        </div>
        <div className="flex u-marginTop--20">
          <Link
            to={`/app/${props.appSlug}/troubleshoot`}
            className="btn secondary blue"
            onClick={props.closeModal}
          >
            Troubleshoot
          </Link>
          <button
            className="btn primary blue u-marginLeft--10"
            onClick={props.closeModal}
          >
            Ok, got it!
          </button>
        </div>
      </div>
    );
  }

  let status;
  if (props.status === "upgrading-cluster") {
    status = Utilities.humanReadableClusterState(props.message);
  } else if (props.status === "upgrading-app") {
    status = "Almost done";
  }

  return (
    <div className="Modal-body u-textAlign--center">
      <div className="flex u-marginTop--30 u-marginBottom--10 justifyContent--center">
        <Loader size="60" />
      </div>
      <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-userSelect--none">
        Cluster update in progress
      </h2>
      {props.connectionTerminated ? (
        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">
          The API cannot be reached because the cluster is updating. Stay on
          this page to automatically reconnect when the update is complete.
        </p>
      ) : (
        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">
          The page will automatically refresh when the update is complete.
          <br />
          <br />
          Status: {status}
        </p>
      )}
    </div>
  );
};

export default UpgradeStatusModal;
