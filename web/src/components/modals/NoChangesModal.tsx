import Modal from "react-modal";

const NoChangesModal = ({
  showNoChangesModal,
  toggleNoChangesModal,
  releaseWithNoChanges,
}: {
  showNoChangesModal: boolean;
  toggleNoChangesModal: () => void;
  releaseWithNoChanges: {
    versionLabel?: string;
    sequence?: number;
  };
}) => {
  return (
    <Modal
      isOpen={showNoChangesModal}
      onRequestClose={() => toggleNoChangesModal()}
      contentLabel="No Changes"
      ariaHideApp={false}
      className="Modal DefaultSize"
    >
      <div className="Modal-body">
        <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
          No changes to show
        </p>
        <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
          The{" "}
          {releaseWithNoChanges && (
            <span className="u-fontWeight--bold">
              Upstream {releaseWithNoChanges.versionLabel}, Sequence{" "}
              {releaseWithNoChanges.sequence}{" "}
            </span>
          )}
          release was unable to generate a diff because the changes made do not
          affect any manifests that will be deployed. Only changes affecting the
          application manifest will be included in a diff.
        </p>
        <div className="flex u-paddingTop--10">
          <button
            className="btn primary"
            onClick={() => toggleNoChangesModal()}
          >
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default NoChangesModal;
