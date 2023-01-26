import React from "react";
import Modal from "react-modal";

interface SnapshotDifferencesModalProps {
  snapshotDifferencesModal: boolean;
  toggleSnapshotDifferencesModal: () => void;
}

export default function SnapshotDifferencesModal(
  props: SnapshotDifferencesModalProps
) {
  const { snapshotDifferencesModal, toggleSnapshotDifferencesModal } = props;

  return (
    <Modal
      isOpen={snapshotDifferencesModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => {
        toggleSnapshotDifferencesModal();
      }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal DefaultSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <div className="flex flex-column justifyContent--center alignItems--center ">
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              {" "}
              Full snapshots (instance){" "}
            </p>
            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              {" "}
              Full snapshots back up the admin console and all
              application data. They can be used for partial restorations, like
              application roll back, or full disaster recovery restorations
              over the same instance or into a new cluster.{" "}
            </p>
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10 u-marginTop--10">
              {" "}
              Partial snapshots (application)
            </p>
            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              Partial snapshots back up application volumes and
              application manifests. They do not back up the admin console or
              the application metadata. They can be used before deploying a 
              new version, in case of needed roll back, but they are not
              suitable for full disaster recovery.
            </p>
          </div>
        </div>
        <div className="flex-auto u-marginTop--20">
          <button
            className="btn primary blue"
            onClick={toggleSnapshotDifferencesModal}
          >
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
}
