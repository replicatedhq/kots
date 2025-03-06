import { Component } from "react";
import Modal from "react-modal";

class DeployWarningModal extends Component {
  render() {
    const {
      showDeployWarningModal,
      hideDeployWarningModal,
      onForceDeployClick,
    } = this.props;

    return (
      <Modal
        isOpen={showDeployWarningModal}
        onRequestClose={hideDeployWarningModal}
        shouldReturnFocusAfterClose={false}
        contentLabel="Skip preflight checks"
        ariaHideApp={false}
        className="Modal MediumSize"
      >
        <div className="Modal-body" data-testid="deploy-warning-modal">
          <p
            className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20"
            data-testid="deploy-warning-modal-text"
          >
            Preflight checks for this version are currently failing. Are you
            sure you want to make this the current version?
          </p>
          {this.props.showAutoDeployWarning && (
            <div className="info-box">
              <span className="u-fontSize--small u-textColor--info u-lineHeight--normal u-fontWeight--medium">
                You have automatic deploys enabled.{" "}
                {this.props.confirmType === "rollback"
                  ? "Rolling back to"
                  : "Deploying"}{" "}
                this version will disable automatic deploys. You can turn it
                back on after this version finishes deployment.
              </span>
            </div>
          )}
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
