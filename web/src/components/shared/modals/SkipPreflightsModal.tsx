import Modal from "react-modal";

interface Props {
  showSkipModal: boolean;
  hideSkipModal: () => void;
  onIgnorePreflightsAndDeployClick?: () => void;
  // TODO: remove this parameter
  onForceDeployClick?: (continueWithFailedPreflights: boolean) => void;
  isEmbeddedCluster?: boolean;
}

export default function SkipPreflightsModal(props: Props) {
  const {
    showSkipModal,
    hideSkipModal,
    onIgnorePreflightsAndDeployClick,
    onForceDeployClick,
    isEmbeddedCluster,
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
      <div className="Modal-body" data-testid="skip-preflights-modal">
        <div className="flex flex-column justifyContent--center alignItems--center">
          <span className="icon yellowWarningIcon" />
          <p className="u-fontSize--jumbo2 u-fontWeight--bold u-lineHeight--medium u-textColor--warning u-marginTop--20" data-testid="skip-preflights-modal-title">
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
              data-testid="wait-for-preflights-to-finish"
            >
              Wait for Preflights to finish
            </button>
            {onForceDeployClick ? (
              <span
                className="tw-text-center u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={() => onForceDeployClick(false)}
                data-testid="ignore-preflights-and-deploy"
              >
                Ignore Preflights {!isEmbeddedCluster && "and deploy"}
              </span>
            ) : (
              <span
                className="tw-text-center u-fontSize--normal u-fontWeight--medium u-textDecoration--underline u-textColor--bodyCopy u-marginTop--15 u-cursor--pointer"
                onClick={onIgnorePreflightsAndDeployClick}
                data-testid="ignore-preflights-and-deploy"
              >
                Ignore Preflights {!isEmbeddedCluster && "and deploy"}
              </span>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}
