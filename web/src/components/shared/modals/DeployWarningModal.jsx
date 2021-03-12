import * as React from "react";
import Modal from "react-modal";


class DeployWarningModal extends React.Component {
  render() {
    const {
      showDeployWarningModal,
      hideDeployWarningModal,
      onForceDeployClick
    } = this.props;
		
    return (
      <Modal
      isOpen={showDeployWarningModal}
      onRequestClose={hideDeployWarningModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Skip preflight checks"
      ariaHideApp={false}
      className="Modal"
    >
      <div className="Modal-body">
        <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
          Preflight checks for this version are currently failing. Are you sure you want to make this the current version?
        </p>
        <div className="u-marginTop--10 flex">
          <button
            onClick={() => onForceDeployClick(true)}
            type="button"
            className="btn blue primary"
          >
            Deploy this version
          </button>
          <button
            onClick={hideDeployWarningModal}
            type="button"
            className="btn secondary u-marginLeft--20"
          >
            Cancel
          </button>
        </div>
      </div>
    </Modal>
    );
  }
}

export default DeployWarningModal;