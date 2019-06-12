import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "../../../utilities/utilities";
import "../../../scss/components/watches/NotificationsModal.scss";
import { createNotification, updateNotification } from "../../../mutations/NotificationMutations";
import { getNotification } from "../../../queries/WatchQueries";
import Loader from "../Loader";

export class SetUpEmailModal extends React.Component {
  constructor() {
    super();
    this.state = {
      recipient: "",
      isLoading: true,
      emailError: false,
      emailToastMessage: false
    };
  }

  validateEmail(email) {
    let canSubmit = true;
    if (!Utilities.isEmailValid(email)) {
      this.setState({
        emailError: true,
        emailToastMessage: "Please use a valid email address."
      })
      canSubmit = false;
    }
    if (isEmpty(email)) {
      this.setState({
        emailError: true,
        emailToastMessage: "Email field can't be blank."
      })
      canSubmit = false;
    }
    return canSubmit;
  }

  onSaveClick = () => {
    const { recipient } = this.state;
    const { watchId, notificationId } = this.props;

    const email = {
      recipientAddress: recipient,
    };

    const isValid = this.validateEmail(recipient);

    if (notificationId) {
      if (isValid) {
        this.props.updateNotification(watchId, notificationId, null, email)
          .then(() => {
            this.setState({ isLoading: true });
            if (typeof this.props.submitCallback === "function") {
              this.props.submitCallback();
            }
            this.props.toggle();
          })
          .catch(() => {
            this.setState({ isLoading: false });
          });
      }
    } else {
      if (isValid) {
        this.props.createNotification(watchId, null, email)
          .then(() => {
            this.setState({ isLoading: true });
            if (typeof this.props.submitCallback === "function") {
              this.props.submitCallback();
            }
            this.props.toggle();
          })
          .catch(() => {
            this.setState({ isLoading: false });
          });
      }
    }
  }

  componentDidUpdate(lastprops) {
    if (lastprops.show !== this.props.show && this.props.show) {
      this.setState({
        recipient: "",
        emailError: false,
        emailToastMessage: false,
        isLoading: false,
      });
    }
  }

  componentDidMount() {
    if (this.props.notificationId) {
      this.props.client.query({
        query: getNotification,
        variables: { notificationId: this.props.notificationId }
      })
      .then((res) => {
        const resValue = res.data.getNotification.email.recipientAddress === "placeholder" ? "" : res.data.getNotification.email.recipientAddress;
        this.setState({ recipient: resValue, isLoading: false });
      })
      .catch();
    }
  }

  onChange = (field, ev) => {
    const state = this.state;
    state[field] = ev.target.value;
    this.setState(state);
  }

  render() {
    const {
      emailError,
      emailToastMessage
    } = this.state;

    const {
      show,
      toggle,
      appName
    } = this.props;

    const content = this.state.isLoading ?
      (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      ) :
      (
        <div className="Modal-body flex flex-column flex1 u-overflow--auto">
          <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Configure email notifications for {appName}</h2>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Ship will send an email when an update is available.</p>
          <div className="Form">
            <div className="flex flex1 u-marginBottom--30">
              <div className="flex flex1 flex-column u-marginRight--10">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Recipient Address</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">What's the address to send the email to?</p>
                {emailToastMessage &&
                  <p className="u-fontSize--small u-color--chestnut u-marginBottom--5">{emailToastMessage}</p>
                }
                <input value={this.state.recipient} onChange={this.onChange.bind(this, "recipient")} type="text" className={`Input ${emailError ? "has-error" : "valid"}`} placeholder="someone@company.com" />
              </div>
            </div>
            <div className="flex flex1 justifyContent--flexEnd u-marginTop--20">
              <div className="flex flex1 justifyContent--flexEnd u-marginTop--20">
                <button onClick={this.onSaveClick.bind(this)} className="btn primary">Schedule Emails</button>
              </div>
            </div>
          </div>
        </div>
      );

    return (
      <Modal
        isOpen={show}
        onRequestClose={toggle}
        shouldReturnFocusAfterClose={false}
        contentLabel="Install Application Modal"
        ariaHideApp={false}
        className="SetUpAutoPRsModal--wrapper Modal DefaultSize"
      >
        {content}
      </Modal>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(createNotification, {
    props: ({ mutate }) => ({
      createNotification: (watchId, webhook, email) => mutate({ variables: { watchId, webhook, email }})
    })
  }),
  graphql(updateNotification, {
    props: ({ mutate }) => ({
      updateNotification: (watchId, notificationId, webhook, email) => mutate({ variables: { watchId, notificationId, webhook, email }})
    })
  })
)(SetUpEmailModal);
