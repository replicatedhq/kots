import React from "react";
import Modal from "react-modal";
import { Utilities, getReadableCronDescriptor } from "../../utilities/utilities";

export default class UpdateCheckerModal extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      updateCheckerSpec: props.updateCheckerSpec,
      submitUpdateCheckerSpecErr: ""
    };
  }

  onSubmitUpdateCheckerSpec = () => {
    const { updateCheckerSpec } = this.state;
    const { appSlug } = this.props;

    this.setState({
      submitUpdateCheckerSpecErr: ""
    });

    fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/updatecheckerspec`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "PUT",
      body: JSON.stringify({
        updateCheckerSpec: updateCheckerSpec,
      })
    })
      .then(async (res) => {
        if (!res.ok) {
          const text = await res.text();
          this.setState({
            submitUpdateCheckerSpecErr: text
          });
          return;
        }

        this.setState({
          submitUpdateCheckerSpecErr: ""
        });
        
        if (this.props.onUpdateCheckerSpecSubmitted) {
          this.props.onUpdateCheckerSpecSubmitted();
        }
      })
      .catch((err) => {
        this.setState({
          submitUpdateCheckerSpecErr: String(err)
        });
      });
  }

  getReadableCronExpression = () => {
    const { updateCheckerSpec } = this.state;
    try {
      const readable = getReadableCronDescriptor(updateCheckerSpec);
      if (readable.includes("undefined")) {
        return "";
      } else {
        return readable;
      }
    } catch(error) {
      return "";
    }
  }

  render() {
    const { isOpen, onRequestClose } = this.props;
    const { updateCheckerSpec, submitUpdateCheckerSpecErr } = this.state;

    const humanReadableCron = this.getReadableCronExpression(updateCheckerSpec);

    return (
      <Modal
        isOpen={isOpen}
        onRequestClose={onRequestClose}
        shouldReturnFocusAfterClose={false}
        contentLabel="Update Checker"
        ariaHideApp={false}
        className="Modal"
      >
        <div className="u-position--relative flex-column u-padding--20">
          <span className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--15">Configure update checker</span>
          <div className="info-box u-marginBottom--20">
            <span className="u-fontSize--small u-textAlign--center">
              You can enter <span className="u-fontWeight--bold u-color--tuna">@never</span> to disable scheduled update checks
            </span>
          </div>
          <div className="flex-column flex1 u-paddingLeft--5">
            <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Cron expression</p>
            <input
              type="text"
              className="Input u-marginBottom--5"
              placeholder="0 0 * * MON"
              value={updateCheckerSpec}
              onChange={(e) => this.setState({ updateCheckerSpec: e.target.value })}
            />
            {humanReadableCron && <span className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">{humanReadableCron}</span>}
            {submitUpdateCheckerSpecErr && <span className="u-color--chestnut u-fontSize--small u-fontWeight--bold u-marginTop--15">{submitUpdateCheckerSpecErr}</span>}
          </div>
          <div className="flex u-marginTop--20">
            <button className="btn primary blue" onClick={this.onSubmitUpdateCheckerSpec}>Update</button>
            <button className="btn secondary u-marginLeft--10" onClick={onRequestClose}>Cancel</button>
          </div>
        </div>
      </Modal>
    );
  }
}