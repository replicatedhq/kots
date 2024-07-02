import Modal from "react-modal";

const DiffErrorModal = ({
  showDiffErrModal,
  toggleDiffErrModal,
  releaseWithErr,
}: {
  showDiffErrModal: boolean;
  toggleDiffErrModal: () => void;
  releaseWithErr: {
    title?: string;
    sequence?: number;
    diffSummaryError?: string;
  };
}) => {
  return (
    <Modal
      isOpen={showDiffErrModal}
      onRequestClose={() => toggleDiffErrModal()}
      contentLabel="Unable to Get Diff"
      ariaHideApp={false}
      className="Modal MediumSize"
    >
      <div className="Modal-body">
        <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
          Unable to generate a file diff for release
        </p>
        {releaseWithErr && (
          <>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              The release with the{" "}
              <span className="u-fontWeight--bold">
                Upstream {releaseWithErr.title}, Sequence{" "}
                {releaseWithErr.sequence}
              </span>{" "}
              was unable to generate a files diff because the following error:
            </p>
            <div className="error-block-wrapper u-marginBottom--30 flex flex1">
              <span className="u-textColor--error">
                {releaseWithErr.diffSummaryError}
              </span>
            </div>
          </>
        )}
        <div className="flex u-marginBottom--10">
          <button className="btn primary" onClick={() => toggleDiffErrModal()}>
            Ok, got it!
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default DiffErrorModal;
