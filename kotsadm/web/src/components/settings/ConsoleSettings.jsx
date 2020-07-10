import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { Helmet } from "react-helmet";

import settingsSubNavConfig from "@src/config-ui/settingsSubNavConfig";
import withTheme from "@src/components/context/withTheme";
import { getKotsApp, listDownstreamsForApp } from "@src/queries/AppsQueries";
import { isVeleroInstalled } from "@src/queries/SnapshotQueries";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import ConsoleAuthentication from "./ConsoleAuthentication";
import ConsoleConfig from "./ConsoleConfig";
import ConsoleLogsWrapper from "./ConsoleLogsWrapper";
import ConsoleNetworking from "./ConsoleNetworking";
import ConsoleRegistry from "./ConsoleRegistry";
import ConsoleTroubleshootWrapper from "./ConsoleTroubleshootWrapper";
import ConsoleVersionHistory from "./ConsoleVersionHistory";
import GitOps from "../clusters/GitOps";
import Snapshots from "../snapshots/Snapshots";
import Redactors from "../redactors/Redactors";
import EditRedactor from "../redactors/EditRedactor";
import NotFound from "../static/NotFound";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";

import "../../scss/components/watches/WatchDetailPage.scss";
class ConsoleSettings extends Component {
  constructor(props) {
    super(props);
    this.state = {
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      watchToEdit: {},
      existingDeploymentClusters: [],
      displayDownloadCommandModal: false,
      isBundleUploading: false,
      sidebarOpen: false
    }
  }

  static defaultProps = {
    getKotsAppQuery: {
      loading: true
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  toggleSidebarState = (isOpen) => {
    this.setState({ sidebarOpen: isOpen })
  }

  render() {
    const {
      match,
      listApps,
      appName,
      isVeleroInstalled,
    } = this.props;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    const loading = false

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={listApps?.length > 1}
          sidebar={(
            <SideBar
              sidebarOpen={this.state.sidebarOpen}
              toggleSidebar={this.toggleSidebarState}
              items={listApps?.map((item, idx) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = match.params.slug;
                  sidebarItemNode = (
                    <KotsSidebarItem
                      sidebarOpen={this.state.sidebarOpen}
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      app={item} />
                  );
                } else if (item.helmName) {
                  sidebarItemNode = (
                    <HelmChartSidebarItem
                      key={idx}
                      sidebarOpen={this.state.sidebarOpen}
                      className={classNames({ selected: item.id === match.params.slug })}
                      helmChart={item} />
                  );
                }
                return sidebarItemNode;
              })}
            />
          )}>
          <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
            {loading
              ? centeredLoader
              : (
                <Fragment>
                  <SubNavBar
                    className="flex"
                    activeTab={match.params.tab || "authentication"}
                    items={settingsSubNavConfig}
                    isVeleroInstalled={isVeleroInstalled?.isVeleroInstalled}
                  />
                  <div className="u-paddingLeft--60">
                    <Switch>
                      <Route exact path="/settings/authentication" render={(props) => <ConsoleAuthentication {...props} />} />
                      <Route exact path={["/settings/logs", "/settings/logs/:logsTab"]} render={(props) => <ConsoleLogsWrapper {...props} />} />
                      <Route exact path="/settings/snapshots" render={(props) => <Snapshots {...props} />} />
                      <Route exact path="/settings/registry" render={(props) => <ConsoleRegistry {...props} />} />
                      <Route exact path="/settings/configuration" render={(props) => <ConsoleConfig {...props} />} />
                      <Route exact path="/settings/networking" render={(props) => <ConsoleNetworking {...props} />} />
                      <Route exact path="/settings/version-history" render={(props) => <ConsoleVersionHistory {...props} />} />
                      <Route exact path="/settings/gitops" render={(props) => <GitOps {...props} />} />
                      <Route exact path={["/settings/troubleshoot", "/settings/logs/:troubleshootTab"]} render={(props) => <ConsoleTroubleshootWrapper {...props} />} />
                      <Route exact path="/settings/troubleshoot/" render={(props) => <Redactors {...props} />} />
                      <Route exact path="/settings/troubleshoot/redactors/new" render={(props) => <EditRedactor {...props} />} />
                      <Route exact path="/settings/troubleshoot/redactors/:slug" render={(props) => <EditRedactor {...props} />} />
                      <Route component={NotFound} />
                    </Switch>
                  </div>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
      </div>
    );
  }
}

export { ConsoleSettings };
export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(getKotsApp, {
    name: "getKotsAppQuery",
    skip: props => {
      const { slug } = props.match.params;

      // Skip if no variables (user at "/app" URL)
      if (!slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
  graphql(listDownstreamsForApp, {
    name: "listDownstreamsForAppQuery",
    skip: props => {
      const { slug } = props.match.params;

      // Skip if no variables (user at "/app" URL)
      if (!slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
  graphql(isVeleroInstalled, {
    name: "isVeleroInstalled",
    options: {
      fetchPolicy: "no-cache"
    }
  }),
)(ConsoleSettings);
