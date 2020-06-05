import React from "react";

export default function SnapshotInstallationBox(props) {
  const { snapshotSettings, hideCheckVeleroButton, fetchSnapshotSettings, renderNotVeleroMessage } = props;

  return (
    <div className="flex1 flex-column">
      <div className="CheckVelero--wrapper flex1 flex-column justifyContent--center">
        <p className="u-color--tundora u-fontSize--large u-fontWeight--bold">Check Velero installation</p>
        {!hideCheckVeleroButton ?
          <div className="u-marginTop--12">
            <button className="btn secondary blue" onClick={() => fetchSnapshotSettings(true)}>Check for Velero</button>
          </div>
          : renderNotVeleroMessage()
        }
        {snapshotSettings?.veleroVersion !== "" ?
          <span className="flex alignItems--center u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--tuna"><span className="icon checkmark-icon u-marginRight--5" />Velero is installed on your cluster</span> : null}
      </div>
      <div className={`${snapshotSettings?.isVeleroRunning ? "u-display--none" : "flex u-marginTop--20"}`}>
        <div className="flex u-marginRight--20">
          <span className="icon redWarningIcon" />
        </div>
        <div className="flex flex-column">
          <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Velero is not running </p>
          <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
            Velero has been detected, but it's not running successfully. To continue configuring and using snapshots Velero has to be running reliably.
        <a href="https://kots.io/kotsadm/snapshots/troubleshooting/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
          </p>
        </div>
      </div>
      <div className={`${snapshotSettings?.veleroVersion !== "" && snapshotSettings?.resticVersion === "" ? "flex u-marginTop--20" : "u-display--none"}`}>
        <div className="flex u-marginRight--20">
          <span className="icon redWarningIcon" />
        </div>
        <div className="flex flex-column">
          <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Restic integration not found </p>
          <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
            The Admin Console requires the Velero restic integration to use Snapshots, but it was not found. Please install the Velero restic integration to continue.
        <a href="https://kots.io/kotsadm/snapshots/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
          </p>
        </div>
      </div>
      <div className={`${snapshotSettings?.veleroVersion !== "" && snapshotSettings?.resticVersion !== "" && !snapshotSettings?.isResticRunning ? "flex u-marginTop--20" : "u-display--none"}`}>
        <div className="flex u-marginRight--20">
          <span className="icon redWarningIcon" />
        </div>
        <div className="flex flex-column">
          <p className="u-color--chestnut u-fontSize--larger u-fontWeight--bold"> Restic is not working </p>
          <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
            Velero and the restic integration have been detected, but restic is not running successfully. To continue configuring and using snapshots Restic has to be running reliably.
        <a href="https://kots.io/kotsadm/snapshots/restic-troubleshooting/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">Get help</a>
          </p>
        </div>
      </div>
    </div>
  );
}
