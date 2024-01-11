import { Component } from "react";
import IdentityProviders from "@src/components/identity/IdentityProviders";

import "@src/scss/components/identity/IdentityManagement.scss";

class AppIdentityServiceSettings extends Component {
  render() {
    const { app } = this.props;

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
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

export default AppIdentityServiceSettings;
