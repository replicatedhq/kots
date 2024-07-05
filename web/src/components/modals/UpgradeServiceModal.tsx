import Icon from "@components/Icon";
import Loader from "@components/shared/Loader";
import Modal from "react-modal";

const UpgradeServiceModal = ({
  shouldShowUpgradeServiceModal,
  onRequestClose,
  isStartingUpgradeService,
  upgradeServiceStatus,
  appSlug,
  iframeRef,
  onLoad,
  isIframeLoading,
}) => {
  return (
    <Modal
      isOpen={shouldShowUpgradeServiceModal}
      onRequestClose={() => onRequestClose()}
      contentLabel="KOTS Upgrade Service Modal"
      ariaHideApp={false}
      className="Modal UpgradeServiceModal"
      shouldCloseOnOverlayClick={false}
    >
      <div className="tw-h-full tw-flex">
        <button
          style={{
            border: "none",
            background: "none",
            cursor: "pointer",
          }}
          className="tw-pt-4 tw-top-0 tw-right-6 tw-absolute tw-overflow-auto"
        >
          <Icon icon="close" onClick={() => onRequestClose()} size={15} />
        </button>
        {isStartingUpgradeService ? (
          <div className="flex-column flex1 alignItems--center justifyContent--center tw-mt-4 tw-gap-4">
            <span className="u-fontWeight--bold">{upgradeServiceStatus}</span>
            <Loader size="60" />
          </div>
        ) : (
          <>
            {isIframeLoading && (
              <div className="tw-w-full flex-column flex1 alignItems--center justifyContent--center tw-gap-4">
                <span className="u-fontWeight--bold">Loading...</span>
                <Loader size="60" />
              </div>
            )}

            <iframe
              src={`/upgrade-service/app/${appSlug}`}
              title="KOTS Upgrade Service"
              width="100%"
              height="100%"
              allowFullScreen={true}
              id="upgrade-service-iframe"
              ref={iframeRef}
              onLoad={onLoad}
            />
          </>
        )}
      </div>
    </Modal>
  );
};

export default UpgradeServiceModal;
