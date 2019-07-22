import React from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { listNotificationsQuery } from "../../queries/WatchQueries";
import { enableNotification } from "../../mutations/NotificationMutations";
import { Utilities } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import SetUpNotificationsModal from "../shared/modals/SetUpNotificationsModal";
import SetUpWebhookModal from "../shared/modals/SetUpWebhookModal";
import SetUpEmailModal from "../shared/modals/SetUpEmailModal";
import IntegrationCard from "./IntegrationCard";

import "../../scss/components/watches/IntegrationCard.scss";
import "../../scss/components/watches/NotificationsModal.scss";

class DetailPageIntegrations extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      displayNotificationsModal: false,
      displayEmailModal: false,
      displayWebhookModal: false,
      notificationId: null,
      hasData: false
    }
  }

  toggleEmailModal = (id) => {
    const state = this.state;
    state.displayEmailModal = !state.displayEmailModal;
    state.notificationId = id || null;
    if (state.displayEmailModal) {
      state.displayNotificationsModal = false;
    }

    this.setState(state);
  }

  toggleWebhookModal = (id) => {
    const state = this.state;
    state.displayWebhookModal = !state.displayWebhookModal;
    state.notificationId = id || null
    if (state.displayWebhookModal) {
      state.displayNotificationsModal = false;
    }

    this.setState(state);
  }

  toggleNotificationsModal = () => {
    const state = this.state;
    state.displayNotificationsModal = !state.displayNotificationsModal;
    if (state.displayNotificationsModal) {
      state.displayEmailModal = false;
      state.displayWebhookModal = false;
    }

    this.setState(state);
  }

  toggleEnable = (id, val) => {
    const { watch } = this.props;
    const watchId = watch && watch.id || "";
    this.props.enableNotification(watchId, id, val);
  }

  determineModalToOpen = (type, id) => {
    switch (type) {
    case "webhook":
      return this.toggleWebhookModal(id);
    case "email":
      return this.toggleEmailModal(id);
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.watch !== lastProps.watch && this.props.watch) {
      this.props.client.query({
        query: listNotificationsQuery,
        variables: { watchId: this.props.watch.id },
        fetchPolicy: "no-cache"
      })
        .then((res) => {
          this.setState({ notifications: res.data.listNotifications });
        });
    }
  }

  componentDidMount() {
    if (this.props.watch) {
      this.props.client.query({
        query: listNotificationsQuery,
        variables: { watchId: this.props.watch.id }
      })
        .then((res) => {
          this.setState({ notifications: res.data.listNotifications });
        });
    }
  }

  render() {
    const { watch } = this.props;
    const {
      displayNotificationsModal,
      displayEmailModal,
      displayWebhookModal,
      notifications,
      notificationId
    } = this.state;

    const appIdSelected = watch && watch.id || "";
    const appName = watch && watch.watchName || "";

    if (!watch) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="flex-column flex1">
        <div className="flex-column flex-1-auto u-overflow--auto container">
          <div className="u-flexTabletReflow u-paddingBottom--20 integration-cards-wrapper flexWrap--wrap">
            {notifications && notifications.map((integration) => {
              const type = Utilities.getNotificationType(integration);
              if (type === "github") {return null}
              return (
                <div key={integration.id} className="integration-card-wrapper flex-auto u-paddingBottom--20">
                  <IntegrationCard
                    item={integration}
                    type={type}
                    toggleEnable={this.toggleEnable}
                    onEditClick={this.determineModalToOpen}
                  />
                </div>
              )
            })
            }
            <div className="add-new-integration u-position--relative flex-column alignItems--center justifyContent--center u-cursor--pointer" onClick={this.toggleNotificationsModal}>
              <span className="icon integration-card-icon-github-add-new u-marginBottom--10 u-cursor--pointer"></span>
              <p className="u-fontSize--small replicated-link">Add More</p>
            </div>
          </div>
        </div>
        {displayNotificationsModal &&
          <SetUpNotificationsModal
            show={displayNotificationsModal}
            appName={appName}
            toggleNotificationsModal={this.toggleNotificationsModal}
            togglePRsModal={this.togglePRsModal}
            toggleEmailModal={this.toggleEmailModal}
            toggleWebhookModal={this.toggleWebhookModal}
            appIdSelected={appIdSelected}
          />
        }
        {displayEmailModal &&
          <SetUpEmailModal
            show={displayEmailModal}
            appName={appName}
            toggle={this.toggleEmailModal}
            watchId={appIdSelected}
            notificationId={notificationId}
            submitCallback={() => {
              if (watch) {
                this.props.client.query({
                  query: listNotificationsQuery,
                  variables: { watchId: watch.id },
                  fetchPolicy: "no-cache"
                }).then((res) => {
                  this.setState({ notifications: res.data.listNotifications });
                });
              }
            }}
          />
        }
        {displayWebhookModal &&
          <SetUpWebhookModal
            show={displayWebhookModal}
            appName={appName}
            toggle={this.toggleWebhookModal}
            watchId={appIdSelected}
            notificationId={notificationId}
            submitCallback={() => {
              if (watch) {
                this.props.client.query({
                  query: listNotificationsQuery,
                  variables: { watchId: watch.id },
                  fetchPolicy: "no-cache"
                }).then((res) => {
                  this.setState({ notifications: res.data.listNotifications });
                });
              }
            }}
          />
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(enableNotification, {
    props: ({ mutate }) => ({
      enableNotification: (watchId, notificationId, enabled) => mutate({ variables: { watchId, notificationId, enabled } })
    })
  })
)(DetailPageIntegrations);
