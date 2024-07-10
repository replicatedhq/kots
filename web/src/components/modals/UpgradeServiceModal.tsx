import Icon from "@components/Icon";
import Loader from "@components/shared/Loader";
import { getBuildVersion } from "@src/utilities/utilities";
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
      <div className="tw-h-full tw-flex tw-flex-col tw-relative tw-overflow-hidden">
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
              <div className="tw-absolute tw-top-[45%] tw-w-full flex-column flex1 alignItems--center justifyContent--center tw-mt-4 tw-gap-4">
                <span className="u-fontWeight--bold">Almost done...</span>
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
              style={{ visibility: isIframeLoading ? "hidden" : "visible" }}
            />
          </>
        )}
        <div
          className="tw-flex tw-justify-start tw-m-4 tw-color-gray-400 tw-text-xs tw-invisible"
          id="kotsUpgradeVersion"
        >
          {getBuildVersion()}
        </div>
      </div>
    </Modal>
  );
};

export default UpgradeServiceModal;
