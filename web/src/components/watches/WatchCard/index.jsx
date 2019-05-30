import * as React from "react";
import CardHeader from "./CardHeader";
import CardHeaderReplicatedApp from "./CardHeaderReplicatedApp";
import DeploymentClusters from "../DeploymentClusters";
import CardRightSideBar from "./CardRightSidebar";
import PropTypes from "prop-types";

export default class WatchCard extends React.Component {
  static propTypes = {
    item: PropTypes.object.isRequired,
    onEditContributorsClick: PropTypes.func.isRequired,
    onCardActionClick: PropTypes.func.isRequired,
    onEditApplication: PropTypes.func.isRequired,
  }

  state = {
    loadingEdit: false,
  };

  parseNotifications = (notifications) => {
    let parsed = {
      webhook: [],
      email: []
    };
    notifications.filter((i) => !i.isDefault).map(n => {
      if (n.webhook) {parsed.webhook.push(n.webhook);}
      if (n.email) {parsed.email.push(n.email);}
    })

    return parsed;
  };

  render() {
    const {
      item,
      submitCallback,
      onEditApplication,
      downloadingIds,
      handleAddNewClusterClick,
      toggleDeleteDeploymentModal,
      installLatestVersion
    } = this.props;
    const integrations = this.parseNotifications(item.notifications);

    let appType = "other";
    let parsedMetadata;

    try {
      parsedMetadata = JSON.parse(item.metadata);
      appType = parsedMetadata.applicationType;
    } catch {
      /* ignore */
    }

    let cardHeader;
    if (appType === "replicated.app") {
      let expirationDate = parsedMetadata.license.expiresAt;
      if (expirationDate === "0001-01-01T00:00:00Z") {
        expirationDate = null;
      }

      cardHeader = <CardHeaderReplicatedApp
        watchIntegrations={integrations}
        watch={item}
        onEditApplication={onEditApplication}
        expirationDate={expirationDate}
      />;
    } else {
      cardHeader = <CardHeader
        watchIntegrations={integrations}
        watch={item}
        onEditApplication={onEditApplication}
        />;
    }

    return (
      <div data-qa={`WatchCard--${item.id}`} className="installed-watch flex-column u-width--full">
        {cardHeader}
        <div className="installed-watch-body flex">
          <DeploymentClusters
            parentClusterName={item.watchName}
            childWatches={item.watches}
            handleAddNewCluster={handleAddNewClusterClick}
            onEditApplication={onEditApplication}
            toggleDeleteDeploymentModal={toggleDeleteDeploymentModal}
            installLatestVersion={installLatestVersion}
          />
          <CardRightSideBar
            watch={item}
            childWatches={item.watches}
            handleDownload={this.props.handleDownload}
            setWatchIdToDownload={this.props.setWatchIdToDownload}
            submitCallback={submitCallback}
            downloadingIds={downloadingIds}
          />
        </div>
      </div>
    );
  }
}
