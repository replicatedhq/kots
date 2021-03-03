import * as React from "react";
import Modal from "react-modal";


class DeployModal extends React.Component {
  render() {
    const {
      showSkipModal,
      hideSkipModal,
      onForceDeployClick
    } = this.props;
		
    return (
      <Modal
      isOpen={showSkipModal}
      onRequestClose={hideSkipModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Skip preflight checks"
      ariaHideApp={false}
      className="Modal SkipModal"
    >
      <div className="Modal-body">
        <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
          Preflight checks have not finished yet. Are you sure you want to deploy this version?
        </p>
        <div className="u-marginTop--10 flex">
          <button
            onClick={() => onForceDeployClick(false)}
            type="button"
            className="btn blue primary">
            Deploy this version
          </button>
          <button type="button" onClick={hideSkipModal} className="btn secondary u-marginLeft--20">Cancel</button>
        </div>
      </div>
    </Modal>
    );
  }
}

export default DeployModal;