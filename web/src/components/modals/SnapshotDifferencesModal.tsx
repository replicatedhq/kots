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
              Full snapshots{" "}
            </p>
            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              {" "}
              Full snapshots (Instances) back up the Admin Console and all
              application data. They can be used for partial restorations, such
              as application roll back, or full Disaster Recovery restorations
              (over the same instance or into a new cluster).{" "}
            </p>
            <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10 u-marginTop--10">
              {" "}
              Partial snapshots
            </p>
            <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal">
              Partial snapshots (Application) back up application volumes and
              application manifests; they do not back up the Admin Console. They
              can be used for capturing information before deploying a new
              release, in case of needed roll back, but they are not suitable
              for full disaster recovery.
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
