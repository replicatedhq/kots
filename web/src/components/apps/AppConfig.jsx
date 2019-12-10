import React, { Component } from "react";
import { ShipConfigRenderer } from "@replicatedhq/ship-init";
import { compose, withApollo, graphql } from "react-apollo";
import { withRouter } from "react-router-dom";
import PropTypes from "prop-types";
import classNames from "classnames";
import debounce from "lodash/debounce";
import map from "lodash/map";

import Loader from "../shared/Loader";
import { getAppConfigGroups, getKotsApp, templateConfigGroups } from "../../queries/AppsQueries";
import { updateAppConfig, updateDownstreamsStatus } from "../../mutations/AppsMutations";

import "../../scss/components/watches/WatchConfig.scss";

class AppConfig extends Component {
  static propTypes = {
    app: PropTypes.object
  }

  constructor(props) {
    super(props);

    this.state = {
      initialConfigGroups: [],
      configGroups: [],
      savingConfig: false,
      changed: false
    }

    this.handleConfigChange = debounce(this.handleConfigChange, 250);
  }

  componentWillMount() {
    const { app, history } = this.props;
    if (app && !app.isConfigurable) { // app not configurable - redirect
      history.replace(`/app/${app.slug}`);
    }
  }

  componentDidUpdate(lastProps) {
    const { getAppConfigGroups } = this.props.getAppConfigGroups;
    if (getAppConfigGroups && getAppConfigGroups !== lastProps.getAppConfigGroups.getAppConfigGroups) {
      const initialConfigGroups = JSON.parse(JSON.stringify(getAppConfigGroups)); // quick deep copy
      this.setState({ configGroups: getAppConfigGroups, initialConfigGroups });
    }
    if (this.props.getKotsApp) {
      const { getKotsApp } = this.props.getKotsApp;
      if (getKotsApp && !getKotsApp.isConfigurable) { // app not configurable - redirect
        this.props.history.replace(`/app/${getKotsApp.slug}`);
      }
    }
  }

  getSequence = () => {
    const { match, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return 0;
    }
    if (match.params.sequence !== undefined) {
      return match.params.sequence;
    }
    return app.currentSequence;
  }

  getSlug = () => {
    const { match, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return match.params.slug;
    }
    return app.slug;
  }

  handleSave = async () => {
    this.setState({ savingConfig: true });

    const { fromLicenseFlow, history, getKotsApp } = this.props;
    const sequence = this.getSequence();
    const slug = this.getSlug();

    try {
      await this.props.updateAppConfig(slug, sequence, this.state.configGroups, !fromLicenseFlow);

      if (this.props.refreshAppData) {
        this.props.refreshAppData();
      }

      if (fromLicenseFlow) {
        const hasPreflight = getKotsApp?.getKotsApp?.hasPreflight;
        const status = hasPreflight ? "pending_preflight" : "deployed";
        await this.props.updateDownstreamsStatus(slug, sequence, status);
        if (hasPreflight) {
          history.replace("/preflight");
        } else {
          if (this.props.refetchListApps) {
            await this.props.refetchListApps();
          }
          history.replace(`/app/${slug}`);
        }
      } else {
        this.setState({ savingConfig: false, changed: false });
      }
    } catch(error) {
      console.log(error);
      this.setState({ savingConfig: false });
    }
  }

  isConfigChanged = newGroups => {
    const { initialConfigGroups } = this.state;
    for (let g = 0; g < newGroups.length; g++) {
      const group = newGroups[g];
      for (let i = 0; i < group.items.length; i++) {
        const newItem = group.items[i];
        const oldItem = this.getItemInConfigGroups(initialConfigGroups, newItem.name);
        if (!oldItem || oldItem.value !== newItem.value) {
          return true;
        }
      }
    }
    return false;
  }

  getItemInConfigGroups = (configGroups, itemName) => {
    let foundItem;
    map(configGroups, group => {
      map(group.items, item => {
        if (item.name === itemName) {
          foundItem = item;
        }
      });
    })
    return foundItem;
  }

  handleConfigChange = groups => {
    const sequence = this.getSequence();
    const slug = this.getSlug();

    this.props.client.query({
      query: templateConfigGroups,
      variables: {
        slug: slug,
        sequence: sequence,
        configGroups: groups
      },
      fetchPolicy: "no-cache"
    }).then(response => {
      const oldGroups = this.state.configGroups;
      const newGroups = response.data.templateConfigGroups;
      map(newGroups, group => {
        group.items.forEach(newItem => {
          if (newItem.type === "password") {
            const oldItem = this.getItemInConfigGroups(oldGroups, newItem.name);
            if (oldItem) {
              newItem.value = oldItem.value;
            }
          }
        });
      });
      const changed = this.isConfigChanged(newGroups);
      this.setState({ configGroups: newGroups, changed });
    }).catch((error) => {
      console.log(error);
    });
  }

  render() {
    const { configGroups, savingConfig, changed } = this.state;
    const { fromLicenseFlow, getKotsApp } = this.props;

    if (!configGroups.length || getKotsApp?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className={classNames("flex1 flex-column u-padding--20 alignItems--center u-overflow--auto")}>
        {fromLicenseFlow && getKotsApp?.getKotsApp && <span className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginTop--auto">Configure {getKotsApp.getKotsApp.name}</span>}
        <div className={classNames("ConfigOuterWrapper flex u-padding--15", { "u-marginTop--20": fromLicenseFlow } )}>
          <div className="ConfigInnerWrapper flex1 u-padding--15">
            <div className="flex1">
              <ShipConfigRenderer groups={configGroups} getData={this.handleConfigChange} />
            </div>
          </div>
        </div>
        <button className="btn secondary green u-marginTop--20 u-marginBottom--auto" disabled={savingConfig || (!changed && !fromLicenseFlow)} onClick={this.handleSave}>{savingConfig ? "Saving" : fromLicenseFlow ? "Continue" : "Save config"}</button>
      </div>
    )
  }
}

export default withRouter(compose(
  withApollo,
  withRouter,
  graphql(getAppConfigGroups, {
    name: "getAppConfigGroups",
    options: ({ match, app, fromLicenseFlow }) => {
      let sequence;
      if (fromLicenseFlow) {
        sequence = 0;
      } else if (match.params.sequence != undefined) {
        sequence = match.params.sequence;
      } else {
        sequence = app.currentSequence;
      }
      const slug = fromLicenseFlow ? match.params.slug : app.slug;
      return {
        variables: {
          slug,
          sequence,
        },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(getKotsApp, {
    name: "getKotsApp",
    skip: ({ app }) => !!app,
    options: ({ match }) => {
      const slug = match.params.slug;
      return {
        variables: {
          slug,
        },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(updateAppConfig, {
    props: ({ mutate }) => ({
      updateAppConfig: (slug, sequence, configGroups, createNewVersion) => mutate({ variables: { slug, sequence, configGroups, createNewVersion } })
    })
  }),
  graphql(updateDownstreamsStatus, {
    props: ({ mutate }) => ({
      updateDownstreamsStatus: (slug, sequence, status) => mutate({ variables: { slug, sequence, status } })
    })
  }),
)(AppConfig));
