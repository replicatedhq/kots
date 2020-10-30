import React from "react";
import { Link } from "react-router-dom";

function navigateToConfiguration(props) {
  props.history.push("/snapshots/settings?configure=true");
}

export default function GettingStartedSnapshots(props) {
  const { isVeleroInstalled, startInstanceSnapshot } = props;

  return (
    <div className="flex flex-column GettingStartedSnapshots--wrapper alignItems--center">
      <span className="icon snapshot-getstarted-icon" />
      <p className="u-fontSize--jumbo2 u-fontWeight--bold u-lineHeight--more u-color--tundora u-marginTop--20"> {isVeleroInstalled ? "No snapshots yet" : "Get started with Snapshots"} </p>
      {isVeleroInstalled ?
        <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-color--dustyGray">Now that everything is configured you can start making snapshots. You can <Link to="/snapshots/settings" className="replicated-link u-fontSize--normal">create a schedule </Link>for automatic snapshots or you can trigger one manually whenever you’d like.</p>
        :
        <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-color--dustyGray">To start backing up your data and applications you need to make sure you have <a href="https://kots.io/kotsadm/snapshots/" target="_blank" rel="noopener noreferrer" className="replicated-link u-fontSize--normal">Velero</a> installed to the cluster along with the CLI tools for the cloud provider you want to send your backups to.</p>
      }
      <div className="flex justifyContent--cenyer u-marginTop--20">
        <button className="btn primary blue" onClick={isVeleroInstalled ? startInstanceSnapshot : () => navigateToConfiguration(props)}> {isVeleroInstalled ? "Start a snapshot" : "Configure snapshot settings"}</button>
      </div>
    </div>
  )
}