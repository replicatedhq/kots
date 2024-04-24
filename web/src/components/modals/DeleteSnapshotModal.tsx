import Modal from "react-modal";
import { Utilities } from "@src/utilities/utilities";
import { Snapshot } from "@src/types";

interface DeleteSnapshotModalProps {
  featureName: string;
  deleteSnapshotModal: boolean;
  toggleConfirmDeleteModal: (snapshot: Snapshot | {}) => void;
  snapshotToDelete: Snapshot;
  deletingSnapshot: boolean;
  handleDeleteSnapshot: (snapshot: Snapshot) => void;
  deleteErr: boolean;
  deleteErrorMsg: string;
}

export default function DeleteSnapshotModal(props: DeleteSnapshotModalProps) {
  const {
    featureName,
    deleteSnapshotModal,
    toggleConfirmDeleteModal,
    snapshotToDelete,
    deletingSnapshot,
    handleDeleteSnapshot,
    deleteErr,
    deleteErrorMsg,
  } = props;

  return (
    <Modal
      isOpen={deleteSnapshotModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => {
        toggleConfirmDeleteModal({});
      }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal DefaultSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
            Delete {featureName}
          </p>
          {deleteErr ? (
            <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
              {deleteErrorMsg}
            </p>
          ) : null}
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
            Are you sure you want to permanently delete a {featureName}? This
            action cannot be reversed.
          </p>
          <div className="flex flex1 justifyContent--spaceBetween u-marginTop--20">
            <div className="flex flex-column">
              <p className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-lineHeight--normal">
                {snapshotToDelete?.name}
              </p>
              <p className="u-fontSize--normal u-textColor--accent u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
                <span className="u-fontWeight--normal u-textColor--bodyCopy">
                  Captured on:
                </span>{" "}
                {Utilities.dateFormat(
                  snapshotToDelete?.startedAt,
                  "MM/DD/YY @ hh:mm a z"
                )}
              </p>
            </div>
            <div className="flex alignItems--center">
              <span
                className={`status-indicator ${snapshotToDelete?.status.toLowerCase()}`}
              >
                {snapshotToDelete?.status}
              </span>
            </div>
          </div>
          <div className="flex justifyContent--flexStart u-marginTop--20">
            <button
              className="btn secondary blue u-marginRight--10"
              onClick={() => {
                toggleConfirmDeleteModal({});
              }}
            >
              Cancel
            </button>
            <button
              className="btn primary red"
              onClick={() => {
                handleDeleteSnapshot(snapshotToDelete);
              }}
              disabled={deletingSnapshot}
            >
              {deletingSnapshot
                ? `Deleting ${featureName}`
                : `Delete ${featureName}`}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
