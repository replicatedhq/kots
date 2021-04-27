import * as React from "react";
import Modal from "react-modal";



class ConfigureGraphsModal extends React.Component {
  render() {
    const {
      showConfigureGraphs,
      toggleConfigureGraphs,
      updatePromValue,
      promValue,
      savingPromValue,
      savingPromError,
      onPromValueChange
    } = this.props;

    return (
      <Modal
          isOpen={showConfigureGraphs}
          onRequestClose={toggleConfigureGraphs}
          shouldReturnFocusAfterClose={false}
          contentLabel="Configure prometheus value"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body flex-column flex1">
            <h2 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--10">Configure graphs</h2>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">To see graphs and metrics, provide the address of your Prometheus installation.<br />This must be resolvable from the Admin Console installation.</p>
            <h3 className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-marginBottom--10">Prometheus endpoint</h3>
            <div className="EditWatchForm flex-column">
              <input
                type="text"
                className="Input u-marginBottom--20"
                placeholder="https://prometheus.default.svc.cluster.local:9090"
                value={promValue}
                onChange={onPromValueChange}
              />
              <div className="flex justifyContent--flexEnd alignItems--center u-marginTop--20">
                {savingPromError && <span className="u-textColor--error u-fontSize--normal u-marginRight--10 u-fontWeight--bold">{savingPromError}</span>}
                <button
                  type="button"
                  onClick={toggleConfigureGraphs}
                  className="btn secondary force-gray u-marginRight--20">
                  Cancel
                </button>
                <button
                  disabled={savingPromValue}
                  onClick={updatePromValue}
                  className="btn primary lightBlue">
                  {
                    savingPromValue
                      ? "Saving"
                      : "Save"
                  }
                </button>
              </div>
            </div>
          </div>
        </Modal>
    );
  }
}

export default ConfigureGraphsModal;
