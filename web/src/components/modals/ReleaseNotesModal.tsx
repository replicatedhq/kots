import Modal from "react-modal";
import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";

const ReleaseNotesModal = ({
  releaseNotes,
  hideReleaseNotes,
}: {
  releaseNotes: Object | null;
  hideReleaseNotes: () => void;
}) => {
  return (
    <Modal
      isOpen={!!releaseNotes}
      onRequestClose={hideReleaseNotes}
      contentLabel="Release Notes"
      ariaHideApp={false}
      className="Modal MediumSize"
    >
      <div className="flex-column">
        <MarkdownRenderer className="is-kotsadm" id="markdown-wrapper">
          {releaseNotes || ""}
        </MarkdownRenderer>
      </div>
      <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
        <button className="btn primary" onClick={hideReleaseNotes}>
          Close
        </button>
      </div>
    </Modal>
  );
};

export default ReleaseNotesModal;
