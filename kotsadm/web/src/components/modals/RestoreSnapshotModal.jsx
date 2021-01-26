import React from "react";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";

export default function RestoreSnapshotModal(props) {
  const { restoreSnapshotModal, toggleRestoreModal, snapshotToRestore, handleRestoreSnapshot, restoringSnapshot, restoreErr, restoreErrorMsg, app, appSlugToRestore, appSlugMismatch, handleApplicationSlugChange } = props;

  return (
    <Modal
      isOpen={restoreSnapshotModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => { toggleRestoreModal({}); }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal MediumSize"
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
            Are you sure you want to restore {app?.name} to the following version?
      </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-lineHeight--normal">{snapshotToRestore?.name}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Captured on:</span> {Utilities.dateFormat(snapshotToRestore?.startedAt, "MM/DD/YY @ hh:mm a")}</p>
            </div>
          </div>
          <div className="flex flex1 u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal"> Restoring to this version will remove data and replace it with data from the restored version. During the restoration, your application will not be available and you will not be able to use the admin console. This action cannot be reversed. </p>
          </div>
          <div className="flex flex-column u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-lineHeight--normal"> Type your application slug to continue</p>

            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal">To confirm that you want to restore this snapshot, please type it's slug in the input as it appears below.</p>
            {appSlugMismatch ?
              <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">The app slug you entered does not match the current app slug</p>
              : null}
            <div className="u-marginTop--12 flex flex1">
              <span className="slugArrow flex justifyContent--center alignItems--center"> {app?.slug} </span>
              <input type="text" className="Input u-position--relative" style={{ textIndent: "200px", width: "70%"}} placeholder="type your slug" value={appSlugToRestore} onChange={(e) => { handleApplicationSlugChange(e) }} />
            </div>
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