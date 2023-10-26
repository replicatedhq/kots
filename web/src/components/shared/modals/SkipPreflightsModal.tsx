import Modal from "react-modal";

interface Props {
  showSkipModal: boolean;
  hideSkipModal: () => void;
  onIgnorePreflightsAndDeployClick?: () => void;
  // TODO: remove this parameter
  onForceDeployClick?: (continueWithFailedPreflights: boolean) => void;
}

export default function SkipPreflightsModal(props: Props) {
  const {
    showSkipModal,
    hideSkipModal,
    onIgnorePreflightsAndDeployClick,
    onForceDeployClick,
  } = props;

  return (
    <Modal
      isOpen={showSkipModal}
      onRequestClose={hideSkipModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Ignore preflight checks"
      ariaHideApp={false}
      className="Modal PreflightModal"
    >
      <div className="Modal-body">
        <div className="flex flex-column justifyContent--center alignItems--center">
          <span className="icon yellowWarningIcon" />
          <p className="u-fontSize--jumbo2 u-fontWeight--bold u-lineHeight--medium u-textColor--warning u-marginTop--20">
            {" "}
            Ignoring Preflights is NOT Recommended{" "}
          </p>
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginTop--12 u-textAlign--center">
            Preflight checks help ensure your current environment matches the
            requirements necessary for the application deployment to be
            successful.
          </p>
          <div className="u-marginTop--30 flex flex-column">
            <button
              type="button"
              className="btn blue primary"
              onClick={hideSkipModal}
            >
              Wait for Preflights to finish
            </button>
            {onForceDeployClick ? (
              <span
                className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={() => onForceDeployClick(false)}
              >
                Ignore Preflights and deploy
              </span>
            ) : (
              <span
                className="u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={onIgnorePreflightsAndDeployClick}
              >
                Ignore Preflights and deploy
              </span>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}
