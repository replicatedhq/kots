import React from "react";
import Modal from "react-modal";

export default function ErrorModal(props) {
  const { errorModal, toggleErrorModal, errMsg, tryAgain, loading, err } = props;

  return (
    <Modal
      isOpen={errorModal}
      shouldReturnFocusAfterClose={false}
      onRequestClose={() => { toggleErrorModal({}); }}
      ariaHideApp={false}
      contentLabel="Modal"
      className="Modal DefaultSize"
    >
      <div className="Modal-body">
        <div className="flex flex-column">
          <div className="flex flex1 justifyContent--center alignItems--center ">
            <span className="icon redWarningIcon flex-auto" />
            <div className="flex flex-column u-marginLeft--10">
              <p className="u-color--chestnut u-fontSize--normal u-fontWeight--bold u-lineHeight--normal">{err}</p>
              <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium u-lineHeight--normal">{errMsg}</p>
            </div>
            {tryAgain && typeof tryAgain === "function" &&
              <div className="flex-auto u-marginLeft--20">
                <button
                  className="btn primary blue"
                  onClick={tryAgain}
                  disabled={loading}
                >
                  {loading ? "Trying..." : "Try again"}
                </button>
              </div>
            }
          </div>
        </div>
      </div>
    </Modal>
  );
}