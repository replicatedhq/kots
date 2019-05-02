import * as React from "react";

export default class NoWatches extends React.Component {

  handleNavigateToStateFile = () => {
    this.props.history.push("/watch/create/state");
  }

  handleNavigateToShipUrl = () => {
    this.props.history.push("/watch/create/init");
  }

  render() {
    return (
      <div className="flex1 flex-column empty-message-wrapper">
        <div className="get-started-wrapper u-textAlign--center">
          <div className="icon ship-login-icon"></div>
          <p className="u-lineHeight--normal u-fontSize--largest u-color--tuna u-fontWeight--bold u-marginBottom--10">Get started with Replicated Ship</p>
          <p className="u-lineHeight--normal u-fontSize--large u-color--dustyGray u-fontWeight--medium">To add your first application we need to generate and locate the applications state file. To get the ball rolling you need to <a href="https://github.com/replicatedhq/ship" target="_blank" rel="noopener noreferrer" className="replicated-link">download Replicated Ship</a>. If you’ve already done this, begin at step&nbsp;two.</p>
          <div className="get-started-steps flex">
            <div className="step flex1 flex">
              <div className="step-number">1</div>
              <div className="flex1">
                <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-marginBottom--10">Download Replicated Ship</p>
                <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">Once you've <a href="https://github.com/replicatedhq/ship" target="_blank" rel="noopener noreferrer" className="replicated-link">downloaded Replicated Ship</a> you will run the <code>ship.init</code> command. This will take you through the apps configuration.</p>
              </div>
            </div>
            <div className="step flex1 flex">
              <div className="step-number flex-auto">2</div>
              <div className="flex1">
                <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-marginBottom--10">Locate your state file</p>
                <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">After you’ve completed the configuration you will need to find your state file. This can be found in <code>.ship/state.json</code></p>
              </div>
            </div>
          </div>
          <p className="u-lineHeight--normal u-fontSize--large u-color--dustyGray u-fontWeight--medium">After you have your state.json file ready to go, we’ll add it here so that you can begin making automatic PR’s into your organizations repos completing the CI/CD pipeline for 3rd-party&nbsp;applications.</p>
          <div className="u-marginTop--30">
            <button type="button" className="btn primary" onClick={this.handleNavigateToStateFile}>I’m ready to add my state file</button>
            {" "}
            <button type="button" className="btn primary" onClick={this.handleNavigateToShipUrl}>Let's try the new stuff</button>
          </div>
        </div>
      </div>
    );
  }
}
