import * as React from "react";
import CardHeader from "./WatchCard/CardHeader";
import PropTypes from "prop-types";

export default class PendingWatchCard extends React.Component {
  static propTypes = {
    item: PropTypes.object.isRequired,
    onEditApplication: PropTypes.func.isRequired,
  }

  state = {
    loadingEdit: false,
  };

  render() {
    const {
      item,
      onEditApplication
    } = this.props;

    return (
      <div data-qa={`WatchCard--${item.id}`} className="installed-watch flex-column u-width--full">
        <CardHeader 
          watchIntegrations={{
            webhook: [],
            email: []
          }}
          watch={{
            id: item.id,
            watchName: item.title
          }}
          isPending={true}
          onEditApplication={onEditApplication}
        />
        <div className="installed-watch-body flex">
          <div className="flex-column u-width--full u-padding--20">
            <div className="PendingInstall--wrapper flex-column flex1 justifyContent--center">
              <div className="u-textAlign--center pending-text-wrapper">
                <p className="u-fontSize--normal u-color--tundora u-lineHeight--more u-fontWeight--medium">This application is available for you to install. Once you have installed this application, you will be able to deploy it to a cluster.</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
