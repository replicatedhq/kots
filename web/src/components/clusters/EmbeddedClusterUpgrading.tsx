import { useEffect } from "react";
import { Utilities } from "../../utilities/utilities";
import Loader from "@components/shared/Loader";

interface Props {
  refetchAppsList: () => Promise<any>;
  clusterState: string;
  connectionTerminated: boolean;
  setTerminatedState: (terminated: boolean) => void;
}

const EmbeddedClusterUpgrading = (props: Props) => {
  const ping = async () => {
    await fetch(`${process.env.API_ENDPOINT}/ping`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then(async (res) => {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
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
      } else {
        props.refetchAppsList().then((apps) => {
          if (!Utilities.shouldShowClusterUpgradeModal(apps)) {
            window.location.reload();
          }
        });
      }
    }, 10000);
    return () => clearInterval(interval);
  }, []);

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
          The API cannot be reached because the cluster is updating. Stay on this
          page to automatically reconnect when the update is complete.
        </p>
      ) : (
        <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">
          The page will automatically refresh when the update is complete.<br/><br/>
          {props.clusterState !== "Installed" && (
            `Status: ${Utilities.humanReadableClusterState(props.clusterState)}`
          )}
        </p>
      )}
    </div>
  );
};

export default EmbeddedClusterUpgrading;
