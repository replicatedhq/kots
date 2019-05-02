import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import { Utilities } from "../../../utilities/utilities";
import "../../../scss/components/watches/NotificationsModal.scss"

export class SetUpNotificationsModal extends React.Component {
  constructor() {
    super();
    this.state = {
      type: "email"
    }
  }

  onClickClose = () => {
    this.props.toggleNotificationsModal();
  }

  handleTypeClick = (type) => {
    this.setState({ type });
  }

  onConfigureClick = () => {
    const { type } = this.state;
    if (type === "email") {
      this.props.toggleEmailModal();
    } else {
      this.props.toggleWebhookModal();
    }
  }

  render() {
    const {
      show,
      appName
    } = this.props;
    const { type } = this.state;

    return (
      <Modal
        isOpen={show}
        onRequestClose={this.onClickClose.bind(this)}
        shouldReturnFocusAfterClose={false}
        contentLabel="Install Application Modal"
        ariaHideApp={false}
        className="SelectNotificationModal--wrapper Modal SmallSize"
      >
        <div className="Modal-body flex flex-column flex1 u-overflow--auto">
          <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Create a new integration for {appName}</h2>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Ship can notify you when updates are available via email, custom webhook url or we can even automatically make a PR to your GitHub repository.</p>
          <div className="u-flexMobileReflow">
            <div className="integration-type-wrapper flex1">
              <div className={`flex flex1 flex-column choose-integration-type alignItems--center ${type === "email" && "is-active"}`} onClick={() => this.handleTypeClick("email")}>
                <span className="icon integration-icon-email u-pointerEvents--none u-marginBottom--10 u-marginTop--10"></span>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginTop-5 u-marginBottom--10">Send an email</p>
              </div>
            </div>
            <div className="integration-type-wrapper flex1">
              <div className={`flex flex1 flex-column choose-integration-type alignItems--center ${type === "webhook" && "is-active"}`} onClick={() => this.handleTypeClick("webhook")}>
                <span className="icon integration-icon-webhook u-pointerEvents--none u-marginBottom--10 u-marginTop--10"></span>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginTop-5 u-marginBottom--10">Webhook URL</p>
              </div>
            </div>
          </div>
          <div className="u-textAlign--center u-marginTop--30">
            <button onClick={() => this.onConfigureClick()} className="btn primary green" disabled={!type.length}>Configure {type.length > 1 && Utilities.toTitleCase(type)}</button>
          </div>
        </div>
      </Modal>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
)(SetUpNotificationsModal);
