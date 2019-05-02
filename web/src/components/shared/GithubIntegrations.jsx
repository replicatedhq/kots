import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
//import PropTypes from "prop-types";
import findIndex from "lodash/findIndex";
import DeleteIntegrationModal from "./modals/DeleteIntegrationModal";
import CardEmptyState from "../watches/WatchCard/CardEmptyState";
import { enableNotification } from "../../mutations/NotificationMutations";
import GitHubIntegration from "../watches/GitHubIntegration";

export class GitHubIntegrations extends React.Component {
  static propTypes = {}

  state = {
    integrationToManage: null,
    displayPRsModal: false,
    gitHubIntegrations: [],
    displayDeleteIntegrationModal: false,
    integrationToDeleteId: "",
    integrationPath: "",
    currentVersion: "",
    isPending: false
  };

  componentDidMount() {
    const { location } = this.props;
    if (this.props.gitHubIntegrations) {
      this.setState({
        gitHubIntegrations: this.props.gitHubIntegrations
      });
    }
    if (location.search.length) {
      if (location.search === "?configure") {
        this.togglePRsModal();
      }
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.gitHubIntegrations !== lastProps.gitHubIntegrations) {
      this.setState({
        gitHubIntegrations: this.props.gitHubIntegrations
      });
    }
  }

  refreshGithubIntegrations = (id, enabled) => {
    const nextState = this.state;
    const i = findIndex(nextState.gitHubIntegrations, { id });
    nextState.gitHubIntegrations[i].enabled = enabled;
    this.setState({ nextState });
  }

  handleEnableToggle = (e, enabled) => {
    const { name } = e.target;
    const { id } = this.props.watch;
    this.props.enableNotification(id, name, !enabled)
      .then(async (data) => {
        const { enabled } = data.enableNotification;
        this.refreshGithubIntegration(name, enabled);
      })
      .catch((err) => {
        console.log(err);
      })
  }

  handleManageIntegration = (e, { id, org, repo, branch, rootPath }) => {
    const integrationToManage = { id, org, repo, branch, rootPath };
    this.setState({ integrationToManage });
    this.togglePRsModal();
  }

  togglePRsModal = () => {
    this.setState({ displayPRsModal: !this.state.displayPRsModal })
  }

  toggleDeleteIntegrationModal = (id, path, pending) => {
    if (this.state.displayDeleteIntegrationModal) {
      this.setState({
        displayDeleteIntegrationModal: false,
        integrationToDeleteId: "",
        integrationPath: "",
        isPending: false
      })
    } else {
      this.setState({
        integrationToDeleteId: id,
        integrationPath: path,
        displayDeleteIntegrationModal: true,
        isPending: pending
      })
    }
  }

  render() {
    const { displayDeleteIntegrationModal, integrationToDeleteId, integrationPath  } = this.state;
    const { integrationCallback, className, title, watch } = this.props;
    const { gitHubIntegrations, isPending } = this.state;

    return (
      <div className={`installed-watch-github flex-column u-width--full ${className || ""}`}>
        {gitHubIntegrations.length && title ? <p className="uppercase-title">{title}</p> : null}
        { gitHubIntegrations.length ?
          <div className="integrations u-overflow--auto flex flex1">
            {gitHubIntegrations.length ? gitHubIntegrations.map((gi, i) => (
              <GitHubIntegration
                key={i}
                {...gi}
                handleEnableToggle={this.handleEnableToggle}
                handleManage={this.handleManageIntegration}
                toggleDeleteIntegrationModal={this.toggleDeleteIntegrationModal}
                handleModalClose={this.props.handleModalClose}
                slug={watch.slug}
              />
            )) : null}
            <div className="add-new-integration u-position--relative flex flex-column alignItems--center justifyContent--center u-cursor--pointer" onClick={this.togglePRsModal}>
              <span className="icon integration-card-icon-github-add-new u-marginBottom--10 u-cursor--pointer"></span>
              <p className="u-fontSize--small replicated-link">Add More</p>
            </div>
          </div> :
          <CardEmptyState
            watchName={watch.watchName}
            watchSlug={watch.slug}
            toggleModal={this.togglePRsModal}
          />
        }
        <DeleteIntegrationModal 
          displayDeleteIntegrationModal={displayDeleteIntegrationModal}
          toggleDeleteIntegrationModal={this.toggleDeleteIntegrationModal}
          integrationToDeleteId={integrationToDeleteId}
          integrationToDeletePath={integrationPath}
          isPending={isPending}
          submitCallback={() => { integrationCallback(); }}
        />
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(enableNotification, {
    props: ({ mutate }) => ({
      enableNotification: (watchId, notificationId, enabled) => mutate({ variables: { watchId, notificationId, enabled } })
    })
  })
)(GitHubIntegrations);
