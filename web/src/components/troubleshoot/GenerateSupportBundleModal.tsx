import React from "react";
import Modal from "react-modal";

const GenerateSupportBundleModal = ({ isOpen, appTitle, toggleModal }) => {
  return (
    <Modal
      isOpen={isOpen}
      className="Modal generate-support-modal"
      shouldReturnFocusAfterClose={false}
      contentLabel="Connection terminated modal"
      onRequestClose={toggleModal}
      ariaHideApp={false}
    >
      <div className="u-padding--25" onClick={(e) => e.stopPropagation()}>
        <span className="u-fontWeight--medium card-title u-fontSize--larger">
          Generate a support bundle
        </span>
        <div className="analyze-modal">
          <span className="u-fontWeight--bold ">Analyze {appTitle}</span>
          <div className="flex analyze-content alignItems--center">
            <p style={{ maxWidth: "440px" }}>
              Collect logs, resources and other data from the running
              application and analyze them against a set of known problems in
              Sentry Enterprise. Logs, cluster info and other data will not
              leave your cluster.
            </p>

            <button
              type="button"
              className="btn primary"
              style={{ height: "30px" }}
            >
              Analyze {appTitle}
            </button>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default GenerateSupportBundleModal;
