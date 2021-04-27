import React from "react";
import Modal from "react-modal";
import { Link } from "react-router-dom";

export default function ErrorModal(props) {
  const { errorModal, toggleErrorModal, errMsg, tryAgain, loading, err, appSlug } = props;

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
        <div className="flex justifyContent--flexEnd">
          <span className="icon u-grayX-icon u-cursor--pointer" onClick={() => toggleErrorModal()} />
        </div>
        <div className="flex flex-column justifyContent--center alignItems--center">
          <span className="icon redWarningIcon flex-auto" />
          <div className="flex flex-column alignItems--center u-marginTop--10">
            <p className="u-textColor--error u-fontSize--largest u-fontWeight--bold u-lineHeight--normal">{err}</p>
            <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textAlign--center">{errMsg}</p>
          </div>
          <div className="flex u-marginTop--20">
            <Link to={appSlug ? `/app/${appSlug}` : "/"} className="btn secondary blue">Back to the dashboard</Link>
            {tryAgain && typeof tryAgain === "function" &&
              <div className="flex-auto u-marginLeft--10">
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