import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import IdentityProviders from "@src/components/identity/IdentityProviders";

import "@src/scss/components/identity/IdentityManagement.scss";

class AppIdentityServiceSettings extends Component {
  render() {
    const { app } = this.props;

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
       <KotsPageTitle pageName="Airgap Settings" showAppSlug />
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
