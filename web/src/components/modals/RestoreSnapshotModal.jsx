import React from "react";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";

export default function RestoreSnapshotModal(props) {
  const { restoreSnapshotModal, toggleRestoreModal, appTitle, snapshotToRestore, handleRestoreSnapshot, restoringSnapshot, restoreErr, restoreErrorMsg } = props;

  return (
    <Modal
      isOpen={restoreSnapshotModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => { toggleRestoreModal({}); }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal LargeSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--more">
            Restore from snapshot
      </p>
          {restoreErr ?
            <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{restoreErrorMsg}</p>
            : null}
          <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal">
            Are you sure you want to restore {appTitle} to the following version?
      </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-lineHeight--normal">{snapshotToRestore?.name}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Captured on:</span> {Utilities.dateFormat(snapshotToRestore?.started, "MMM D, YYYY h:mm A")}</p>
            </div>
            <div className="flex alignItems--center">
              <span className={`status-indicator ${snapshotToRestore?.status.toLowerCase()}`}>{snapshotToRestore?.status}</span>
            </div>
          </div>
          <div className="flex flex1 u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal"> Restoring to this version will remove all of the data in the {appTitle} namespace and will replace it with the data from the restored version. During the restoration your application will not be available and you will not be able to use the admin console. This action cannot be reversed.</p>
          </div>
          <div className="flex justifyContent--flexStart u-marginTop--20">
            <button
              className="btn secondary blue u-marginRight--10"
              onClick={() => { toggleRestoreModal({}); }}
            >
              Cancel
        </button>
            <button
              className="btn primary blue"
              onClick={() => { handleRestoreSnapshot(snapshotToRestore) }}
              disabled={restoringSnapshot}
            >
              {restoringSnapshot ? "Restoring from snapshot" : "Restore from snapshot"}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}