import Icon from "@components/Icon";
import { Link } from "react-router-dom";

function navigateToConfiguration(props) {
  props.navigate("/snapshots/settings?configure=true");
}

export default function GettingStartedSnapshots(props) {
  const {
    isVeleroInstalled,
    startInstanceSnapshot,
    isApp,
    app,
    startManualSnapshot,
  } = props;

  return (
    <div className="flex flex-column card-item GettingStartedSnapshots--wrapper  alignItems--center">
      <Icon icon="snapshot-getstarted" size={50} />
      <p className="u-fontSize--jumbo2 u-fontWeight--bold u-lineHeight--more u-textColor--secondary u-marginTop--20">
        {" "}
        {isVeleroInstalled
          ? "No snapshots yet"
          : "Get started with Snapshots"}{" "}
      </p>
      {isApp ? (
        <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-textColor--bodyCopy">
          There have been no snapshots made for {app?.name} yet. You can
          manually trigger snapshots or you can set up automatic snapshots to be
          made on a custom schedule.{" "}
        </p>
      ) : isVeleroInstalled ? (
        <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-textColor--bodyCopy">
          Now that Velero is configured, you can start making snapshots. You can{" "}
          <Link to="/snapshots/settings" className="link u-fontSize--normal">
            create a schedule{" "}
          </Link>
          for automatic snapshots or you can trigger one manually whenever you’d
          like.
        </p>
      ) : (
        <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-textColor--bodyCopy">
          To start backing up your data and applications, you need to have{" "}
          <a
            href="https://velero.io/docs/v1.10/basic-install/"
            target="_blank"
            rel="noopener noreferrer"
            className="link u-fontSize--normal"
          >
            Velero
          </a>{" "}
          installed in the cluster and configured to connect with the cloud
          provider you want to send your backups to
        </p>
      )}
      <div className="flex justifyContent--cenyer u-marginTop--20">
        {isApp ? (
          <button
            className="btn primary blue"
            onClick={
              isVeleroInstalled
                ? startManualSnapshot
                : () => navigateToConfiguration(props)
            }
          >
            {" "}
            {isVeleroInstalled
              ? "Start a snapshot"
              : "Configure snapshot settings"}
          </button>
        ) : (
          <button
            className="btn primary blue"
            onClick={
              isVeleroInstalled
                ? startInstanceSnapshot
                : () => navigateToConfiguration(props)
            }
          >
            {" "}
            {isVeleroInstalled
              ? "Start a snapshot"
              : "Configure snapshot settings"}
          </button>
        )}
      </div>
    </div>
  );
}
