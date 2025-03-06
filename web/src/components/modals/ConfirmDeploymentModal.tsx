import { Version } from "@types";
import Modal from "react-modal";

const ConfirmDeploymentModal = ({
  displayConfirmDeploymentModal,
  hideConfirmDeploymentModal,
  confirmType,
  versionToDeploy,
  outletContext,
  finalizeDeployment,
  isPastVersion,
  finalizeRedeployment,
}: {
  displayConfirmDeploymentModal: boolean;
  hideConfirmDeploymentModal: () => void;
  confirmType: string;
  versionToDeploy: {
    versionLabel?: string;
    sequence: number;
  };
  outletContext: {
    app: {
      autoDeploy: string;
    };
  };
  finalizeDeployment: (continueWithFailedPreflights: boolean) => void;
  isPastVersion: Version;
  finalizeRedeployment: () => void;
}) => {
  return (
    <Modal
      isOpen={displayConfirmDeploymentModal}
      onRequestClose={() => hideConfirmDeploymentModal()}
      contentLabel="Confirm deployment"
      ariaHideApp={false}
      className="Modal DefaultSize"
    >
      <div className="Modal-body" data-testid="confirm-deployment-modal">
        <p
          className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10"
          data-testid="confirm-deployment-modal-text"
        >
          {confirmType === "rollback"
            ? "Rollback to"
            : confirmType === "redeploy"
            ? "Redeploy"
            : "Deploy"}{" "}
          {versionToDeploy?.versionLabel} (Sequence {versionToDeploy?.sequence}
          )?
        </p>
        {isPastVersion && outletContext.app?.autoDeploy !== "disabled" ? (
          <div className="info-box">
            <span className="u-fontSize--small u-textColor--info u-lineHeight--normal u-fontWeight--medium">
              You have automatic deploys enabled.{" "}
              {confirmType === "rollback"
                ? "Rolling back to"
                : confirmType === "redeploy"
                ? "Redeploying"
                : "Deploying"}{" "}
              this version will disable automatic deploys. You can turn it back
              on after this version finishes deployment.
            </span>
          </div>
        ) : null}
        <div className="flex u-paddingTop--10">
          <button
            className="btn secondary blue"
            onClick={() => hideConfirmDeploymentModal()}
          >
            Cancel
          </button>
          <button
            className="u-marginLeft--10 btn primary"
            onClick={
              confirmType === "redeploy"
                ? finalizeRedeployment
                : () => finalizeDeployment(false)
            }
          >
            Yes,{" "}
            {confirmType === "rollback"
              ? "rollback"
              : confirmType === "redeploy"
              ? "redeploy"
              : "deploy"}
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default ConfirmDeploymentModal;
