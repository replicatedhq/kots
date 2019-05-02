import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import isEmpty from "lodash/isEmpty";
import "../../../scss/components/watches/NotificationsModal.scss";
import { createNotification, updateNotification } from "../../../mutations/NotificationMutations";
import { getNotification } from "../../../queries/WatchQueries";

export class SetUpWebhookModal extends React.Component {
  constructor() {
    super();
    this.state = {
      uri: "",
      webhookUriError: false,
      webhookToastMessage: false,
    }
  }

  validateWebhookUri(uri) {
    let canSubmit = true;
    if (isEmpty(uri)) {
      this.setState({
        webhookUriError: true,
        webhookToastMessage: "Please enter a webhook uri"
      })
      canSubmit = false;
    }
    return canSubmit;
  }

  onSaveClick = () => {
    const { uri } = this.state;
    const { watchId, notificationId } = this.props;

    const webhook = {
      uri: uri
    };

    const isValid = this.validateWebhookUri(uri);
    if (notificationId) {
      if (isValid) {
        this.props.updateNotification(watchId, notificationId, webhook, null)
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
        this.props.createNotification(watchId, webhook, null)
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
        uri: "",
        webhookUriError: false,
        webhookToastMessage: false
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
          const resValue = res.data.getNotification.webhook.uri === "placeholder" ? "" : res.data.getNotification.webhook.uri;
          this.setState({ uri: resValue });
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
      webhookUriError,
      webhookToastMessage
    } = this.state;

    const {
      show,
      toggle,
      appName,
    } = this.props;

    return (
      <Modal
        isOpen={show}
        onRequestClose={toggle}
        shouldReturnFocusAfterClose={false}
        contentLabel="Install Application Modal"
        ariaHideApp={false}
        className="SetUpAutoPRsModal--wrapper Modal DefaultSize"
      >
        <div className="Modal-body flex flex-column flex1 u-overflow--auto">
          <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Configure webhooks for {appName}</h2>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Ship will trigger a webhook when an update is available.</p>
          <div className="Form">
            <div className="u-flexMobileReflow">
              <div className="flex flex1">
                <div className="flex flex1 flex-column u-marginRight--10">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Webhook URI</p>
                  <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">What's the URI to send the webhook request to?</p>
                  {webhookToastMessage &&
                    <p className="u-fontSize--small u-color--chestnut u-marginBottom--5">{webhookToastMessage}</p>
                  }
                  <input value={this.state.uri} onChange={this.onChange.bind(this, "uri")} type="text" className={`Input ${webhookUriError ? "has-error" : "valid"}`} placeholder="Webhook URI" />
                </div>
              </div>
            </div>
            <div className="flex flex1 justifyContent--flexEnd u-marginTop--20">
              <button onClick={this.onSaveClick.bind(this)} className="btn primary">Create webhook</button>
            </div>
          </div>
        </div>
      </Modal>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(createNotification, {
    props: ({ mutate }) => ({
      createNotification: (watchId, webhook, email) => mutate({ variables: { watchId, webhook, email } })
    })
  }),
  graphql(updateNotification, {
    props: ({ mutate }) => ({
      updateNotification: (watchId, notificationId, webhook, email) => mutate({ variables: { watchId, notificationId, webhook, email } })
    })
  })
)(SetUpWebhookModal);
