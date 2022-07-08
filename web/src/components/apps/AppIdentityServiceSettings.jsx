import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import IdentityProviders from "@src/components/identity/IdentityProviders";

import "@src/scss/components/identity/IdentityManagement.scss";

class AppIdentityServiceSettings extends Component {
  render() {
    const { app } = this.props;

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>{`${app.name} Airgap settings`}</title>
        </Helmet>
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          <IdentityProviders
            isKurlEnabled={this.props.isKurlEnabled}
            isApplicationSettings={true}
            app={app}
          />
        </div>
      </div>
    );
  }
}

export default withRouter(AppIdentityServiceSettings);
