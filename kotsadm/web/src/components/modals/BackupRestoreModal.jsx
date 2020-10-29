import React from "react";
import Modal from "react-modal";

import CodeSnippet from "../shared/CodeSnippet";

import { Utilities } from "../../utilities/utilities";

export default function BackupRestoreModal(props) {
  const { restoreSnapshotModal, toggleRestoreModal, snapshotToRestore } = props;

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
          <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--more"> Restore from backup </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal">
            This is a full restore of the admin console, applications, application metadata, application config and your database. Any data not backed up will be lost and replaced with the data in this backup. </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20 snapshotDetails--wrapper">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-lineHeight--normal">{snapshotToRestore?.name}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--medium u-lineHeight--normal u-marginRight--20">{Utilities.dateFormat(snapshotToRestore?.startedAt, "MMM D YYYY @ hh:mm a")}</p>
            </div>
            <div className="flex flex1 justifyContent--flexEnd">
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--30 justifyContent--center flex alignItems--center"><span className="icon snapshot-volume-size-icon" /> {snapshotToRestore?.volumeSizeHuman} </p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center"><span className="icon snapshot-volume-icon" /> {snapshotToRestore?.volumeSuccessCount}/{snapshotToRestore?.volumeCount}</p>
            </div>
          </div>
          <div className="flex flex-column u-marginTop--20">
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal"> To start the restore, run this command on your cluster. </p>
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
            >
              {`kubectl kots restore create --from-backup ${snapshotToRestore?.name}`}
            </CodeSnippet>
          </div>
        </div>
        <div className="flex justifyContent--flexStart u-marginTop--20">
          <button className="btn primary" onClick={toggleRestoreModal}> Ok, got it! </button>
        </div>
      </div>
    </Modal>
  );
}