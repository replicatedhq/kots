import React from "react";
import Modal from "react-modal";

export default function ShowDetailsModal(props) {
  const { displayShowDetailsModal, toggleShowDetailsModal, yamlErrorDetails, deployView, showDeployWarningModal, showSkipModal, forceDeploy } = props;

  return (
    <Modal
      isOpen={displayShowDetailsModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => { toggleShowDetailsModal({}); }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal MediumSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-lineHeight--normal u-marginBottom--more"> Invalid files in your application </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal">
            Your application can be deployed, but the files with the errors will not be included. </p>

          <div className="u-marginTop--20">
            <p className="u-fontSize--large u-fontWeight--bold u-color--tuna u-lineHeight--normal u-borderBottom--gray darker u-paddingBottom--10"> The following files contain errors </p>
            {yamlErrorDetails?.map((err, i) => (
              <div className="flex flex1 alignItems--center u-borderBottom--gray darker u-paddingTop--10 u-paddingBottom--10" key={i}>
                <div className="flex">
                  <span className="icon invalid-yaml-icon" />
                </div>
                <div className="flex flex-column u-marginLeft--10">
                  <span className="u-fontSize--large u-fontWeight--bold u-color--tuna u-lineHeight--normal"> {err.path} </span>
                  <span className="u-fontSize--small u-fontWeight--medium u-color--red u-lineHeight--normal"> error: {err.error} </span>
                </div>
              </div>
            )
            )}
          </div>
          {deployView ?
            <div className="flex justifyContent--flexStart u-marginTop--20">
              <button className="btn primary blue" onClick={() => { (showDeployWarningModal || showSkipModal) ? toggleShowDetailsModal() : forceDeploy() }}>Deploy</button>
              <button className="btn secondary u-marginLeft--20" onClick={() => { toggleShowDetailsModal() }}>Cancel</button>
            </div>
            :
            <div className="flex justifyContent--flexStart u-marginTop--20">
              <button
                className="btn primary blue"
                onClick={() => { toggleShowDetailsModal() }}
              >
                Ok, got it!
          </button>
            </div>}
        </div>
      </div>
    </Modal>
  );
}