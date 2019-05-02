import * as React from "react";
import "../scss/components/watches/WatchedApps.scss";

export default class ComingSoon extends React.Component {

  render() {
    return (
      <div className="container flex-column flex1 u-overflow--auto">
        <div className="flex1 flex-column empty-message-wrapper">
          <div className="get-started-wrapper u-textAlign--center">
            <div className="icon ship-login-icon"></div>
            <p className="u-lineHeight--normal u-fontSize--largest u-color--tuna u-fontWeight--bold u-marginBottom--10">Coming Soon</p>
            <p className="u-lineHeight--normal u-fontSize--large u-color--dustyGray u-fontWeight--medium">The hosted Replicated Ship experience will be available for everyone soon. The good news is, you can download and use Replicated Ship today!</p>
            <p className="u-lineHeight--normal u-fontSize--large u-color--dustyGray u-fontWeight--medium u-marginTop--30">We’ll send you an email when the hosted service is ready for you to use. In the meantime, <a href="https://github.com/replicatedhq/ship" target="_blank" rel="noopener noreferrer" className="replicated-link">head&nbsp;over to GitHub</a> and checkout the project source.</p>
            <div className="u-marginTop--30">
              <a href="https://github.com/replicatedhq/ship" target="_blank" rel="noopener noreferrer" className="btn primary" onClick={this.handleNavigateToDownload}>Download Replicated Ship</a>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
