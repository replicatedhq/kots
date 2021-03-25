import React from "react";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";

export default function DeleteSnapshotModal(props) {
  const { deleteSnapshotModal, toggleConfirmDeleteModal, snapshotToDelete, deletingSnapshot, handleDeleteSnapshot, deleteErr, deleteErrorMsg } = props;

  return (
    <Modal
      isOpen={deleteSnapshotModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => { toggleConfirmDeleteModal({}); }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal DefaultSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--more">
            Delete snapshot
              </p>
          {deleteErr ?
            <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{deleteErrorMsg}</p>
            : null}
          <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal">
            Are you sure you want do permanently delete a snapshot? This action cannot be reversed.
              </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-lineHeight--normal">{snapshotToDelete?.name}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Captured on:</span> {Utilities.dateFormat(snapshotToDelete?.startedAt, "MM/DD/YY @ hh:mm a z")}</p>
            </div>
            <div className="flex alignItems--center">
              <span className={`status-indicator ${snapshotToDelete?.status.toLowerCase()}`}>{snapshotToDelete?.status}</span>
            </div>
          </div>
          <div className="flex justifyContent--flexStart u-marginTop--20">
            <button
              className="btn secondary blue u-marginRight--10"
              onClick={() => { toggleConfirmDeleteModal({}); }}
            >
              Cancel
                </button>
            <button
              className="btn primary red"
              onClick={() => { handleDeleteSnapshot(snapshotToDelete) }}
              disabled={deletingSnapshot}
            >
              {deletingSnapshot ? "Deleting snapshot" : "Delete snapshot"}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}