import Modal from "react-modal";
import Loader from "../shared/Loader";

const DisplayKotsUpdateModal = ({
  displayKotsUpdateModal,
  onRequestClose,
  renderKotsUpgradeStatus,
  kotsUpdateStatus,
  shortKotsUpdateMessage,
  kotsUpdateMessage,
}: {
  displayKotsUpdateModal: boolean;
  onRequestClose: () => void;
  renderKotsUpgradeStatus: boolean;
  kotsUpdateStatus: string;
  shortKotsUpdateMessage: string;
  kotsUpdateMessage: string;
}) => {
  return (
    <Modal
      isOpen={displayKotsUpdateModal}
      onRequestClose={onRequestClose}
      contentLabel="Upgrade is in progress"
      ariaHideApp={false}
      className="Modal DefaultSize"
    >
      <div className="Modal-body u-textAlign--center">
        <div className="flex-column justifyContent--center alignItems--center">
          <p className="u-fontSize--large u-textColor--primary u-lineHeight--bold u-marginBottom--10">
            Upgrading...
          </p>
          <Loader className="flex alignItems--center" size="32" />
          {renderKotsUpgradeStatus ? (
            <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              {kotsUpdateStatus}
            </p>
          ) : null}
          {kotsUpdateMessage ? (
            <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-marginBottom--10">
              {shortKotsUpdateMessage}
            </p>
          ) : null}
        </div>
      </div>
    </Modal>
  );
};

export default DisplayKotsUpdateModal;
