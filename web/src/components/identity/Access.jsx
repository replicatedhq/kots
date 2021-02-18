import React, { Component, Fragment } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { Helmet } from "react-helmet";

import withTheme from "@src/components/context/withTheme";
// import NotFound from "../static/NotFound";
// import SubNavBar from "@src/components/shared/SubNavBar";
// import ConfigureIngress from "@src/components/identity/ConfigureIngress";
import IdentityProviders from "@src/components/identity/IdentityProviders";

import "@src/scss/components/identity/IdentityManagement.scss";

class Access extends Component {
  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/access") {
      history.replace(`/access/identity-providers`);
      return;
    }
  }

  render() {
    // const {
    //   match,
    // } = this.props;


    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title> Access </title>
        </Helmet>
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          {/* TODO ===> THIS WILL COME LATER */}
          {/* <Fragment>
            <SubNavBar
              className="flex"
              isAccess={true}
              activeTab={match.params.tab}
            />
            <Switch>
              <Route exact path="/access/configure-ingress" render={() =>
                <ConfigureIngress />
              } />
              <Route exact path="/access/identity-providers" render={() =>
                <IdentityProviders isKurlEnabled={this.props.isKurlEnabled} />
              } />
              <Route component={NotFound} />
            </Switch>
          </Fragment> */}
          <IdentityProviders isKurlEnabled={this.props.isKurlEnabled} isGeoaxisSupported={this.props.isGeoaxisSupported} />
        </div>
      </div>
    );
  }
}

export default withTheme(withRouter(Access));
