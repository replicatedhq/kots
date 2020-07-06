import React from "react";
import SnapshotInstallationBox from "./SnapshotInstallationBox";

export default function ConfigureSnapshots(props) {
  const { snapshotSettings, hideCheckVeleroButton, fetchSnapshotSettings, renderNotVeleroMessage, isLicenseUpload, history } = props;

  return (
    <div className="flex1 flex-column AppSnapshotsEmptyState--wrapper">
      {isLicenseUpload && <div className="u-fontSize--normal u-fontWeight--medium u-color--royalBlue u-cursor--pointer" onClick={() => history.goBack()}>
        <span className="icon clickable backArrow-icon u-marginRight--10" style={{ verticalAlign: "0" }} />
            Back to license upload
        </div>}
      <p className={`u-fontSize--largest u-fontWeight--bold u-color--tundora u-marginBottom--10 ${isLicenseUpload && "u-marginTop--12"}`}>Configure application snapshots</p>
      <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
        In order to configure and use Snapshots (backup and restore), please install <a href="https://kots.io/kotsadm/snapshots/" target="_blank" rel="noopener noreferrer" className="replicated-link">Velero</a> to the cluster. Once Velero is installed, click the button below and the Admin Console will verify the installation and begin configuring Snapshots.
          </p>
      <div className="flex u-marginTop--40">
        <div className="InstallVelero--wrapper flex flex-column u-marginRight--30">
          <p className="u-color--tundora u-fontSize--large u-fontWeight--bold">To install Velero</p>
          <div className="flex1 flex-column u-marginBottom--30">
            <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray u-marginTop--20"><span className="circleNumberGray u-marginRight--10"> 1 </span>Install the CLI on your machine by <a href="https://kots.io/kotsadm/snapshots/basic-install/" target="_blank" rel="noopener noreferrer" className="replicated-link u-marginLeft--5">following these instructions</a> </p>
            <div className="flex flex1 u-marginTop--20">
              <div className="flex">
                <span className="circleNumberGray u-marginRight--10"> 2 </span>
              </div>
              <div className="flex flex-column">
                <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray"> Run the commands from the instructions for your cloud provider </p>
                <div className="flex flexWrap--wrap" style={{ width: "500px" }}>
                  <a href="https://github.com/vmware-tanzu/velero-plugin-for-aws#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions">
                    <span style={{ width: "130px" }}> <span className="icon awsIcon u-cursor--pointer u-marginRight--5" />Amazon AWS </span>
                    <span className="icon external-link-icon u-cursor--pointer justifyContent--flexEnd u-marginLeft--30" /></a>
                  <a href="https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions">
                    <span style={{ width: "130px" }}> <span className="icon azureIcon u-cursor--pointer u-marginRight--5" />Microsoft Azure </span>
                    <span className="icon external-link-icon u-cursor--pointer u-marginLeft--30" /></a>
                  <a href="https://github.com/vmware-tanzu/velero-plugin-for-gcp#setup" target="_blank" rel="noopener noreferrer" className="snapshotOptions">
                    <span style={{ width: "130px" }}>  <span className="icon googleCloudIcon u-cursor--pointer u-marginRight--5" />Google Cloud </span>
                    <span className="icon external-link-icon u-cursor--pointer u-marginLeft--30" /></a>
                  <a href="https://kots.io/kotsadm/snapshots/supported-providers/" target="_blank" rel="noopener noreferrer" className="snapshotOptions">
                    <span style={{ width: "130px" }}>  <span className="icon cloudIcon u-cursor--pointer u-marginRight--5" /> Other provider  </span>
                    <span className="icon external-link-icon u-cursor--pointer u-marginLeft--30" /></a>
                </div>
              </div>
            </div>
            <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-color--dustyGray u-marginTop--20"> With all providers, you must install using the  <span className="inline-code u-marginLeft--5 u-marginRight--5"> --use-restic </span>  flag for snapshots to work. </p>
          </div>
        </div>
        <SnapshotInstallationBox
          fetchSnapshotSettings={fetchSnapshotSettings}
          renderNotVeleroMessage={renderNotVeleroMessage}
          snapshotSettings={snapshotSettings}
          hideCheckVeleroButton={hideCheckVeleroButton}
        />
      </div>
    </div>
  );
}
