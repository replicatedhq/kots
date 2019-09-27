import React, { Component } from "react";
import { ShipConfigRenderer } from "@replicatedhq/ship-init";
import { compose, withApollo, graphql } from "react-apollo";
import { withRouter } from "react-router-dom";
import PropTypes from "prop-types";

import Loader from "../shared/Loader";
import { getKotsConfigGroups } from "../../queries/AppsQueries";
import { updateAppConfig } from "../../mutations/AppsMutations";

import "../../scss/components/watches/WatchConfig.scss";

class AppConfig extends Component {
  static propTypes = {
    app: PropTypes.object.isRequired
  }

  constructor(props) {
    super(props);

    this.state = {
      configGroups: [],
      savingConfig: false
    }
  }

  componentDidUpdate(lastProps) {
    const { getKotsConfigGroups } = this.props.getKotsConfigGroups;
    if (getKotsConfigGroups && getKotsConfigGroups !== lastProps.getKotsConfigGroups.getKotsConfigGroups) {
      this.setState({ configGroups: getKotsConfigGroups });
    }
  }

  handleSave = () => {
    this.setState({ savingConfig: true });

    this.props.client.mutate({
      mutation: updateAppConfig,
      variables: {
        slug: this.props.app.slug,
        sequence: this.props.app.currentSequence,
        configGroups: this.state.configGroups,
      },
    })
      .then(() => {
        this.props.refreshAppData();
      })
      .catch((error) => {
        console.log(error);
      })
      .finally(() => {
        this.setState({ savingConfig: false });
      });
  }

  render() {
    const { configGroups, savingConfig } = this.state;

    if (!configGroups.length) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className="flex1 flex-column u-overflow--auto u-padding--20 justifyContent--flexStart alignItems--center">
        <div className="ConfigOuterWrapper flex u-padding--15" >
          <div className="ConfigInnerWrapper flex1 u-padding--15">
            <div className="flex1">
              <ShipConfigRenderer groups={configGroups} />
            </div>
          </div>
        </div>
        <button className="btn secondary green u-marginTop--20" disabled={savingConfig} onClick={this.handleSave}>{savingConfig ? "Saving" : "Save config"}</button>
      </div>
    )
  }
}

export default withRouter(compose(
  withApollo,
  withRouter,
  graphql(getKotsConfigGroups, {
    name: "getKotsConfigGroups",
    options: ({ app }) => ({
      variables: {
        slug: app.slug,
        sequence: app.currentSequence,
      },
      fetchPolicy: "no-cache"
    })
  }),
)(AppConfig));
