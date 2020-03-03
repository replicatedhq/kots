import React, { Component } from "react";
import { ShipConfigRenderer } from "@replicatedhq/ship-init";
import { compose, withApollo, graphql } from "react-apollo";
import { withRouter, Link } from "react-router-dom";
import PropTypes from "prop-types";
import classNames from "classnames";
import debounce from "lodash/debounce";
import map from "lodash/map";
import Modal from "react-modal";
import Loader from "../shared/Loader";
import { getAppConfigGroups, getKotsApp, templateConfigGroups } from "../../queries/AppsQueries";
import { updateDownstreamsStatus } from "../../mutations/AppsMutations";

import "../../scss/components/watches/WatchConfig.scss";
import { Utilities } from "../../utilities/utilities";

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
      changed: false,
      showNextStepModal: false,
      savingConfigError: ""
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
    if (match.params.sequence != undefined) {
      return parseInt(match.params.sequence);
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

  markRequiredItems = requiredItems => {
    const configGroups = this.state.configGroups;
    requiredItems.forEach(requiredItem => {
      configGroups.forEach(configGroup => {
        const item = configGroup.items.find(item => item.name === requiredItem);
        if (item) {
          item.error = "This item is required";
        }
      });
    });
    this.setState({ configGroups });
  }

  handleSave = async () => {
    this.setState({ savingConfig: true, savingConfigError: "" });

    const { fromLicenseFlow, history, getKotsApp, match } = this.props;
    const sequence = this.getSequence();
    const slug = this.getSlug();
    const createNewVersion = !fromLicenseFlow && match.params.sequence == undefined;

    fetch(`${window.env.API_ENDPOINT}/app/${slug}/config`, {
      method: "PUT",
      headers: {
        "Authorization": `${Utilities.getToken()}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        configGroups: this.state.configGroups,
        sequence,
        createNewVersion,
      })
    })
      .then(res => res.json())
      .then(async (result) => {
        this.setState({ savingConfig: false });

        if (!result.success) {
          if (result.requiredItems?.length) {
            this.markRequiredItems(result.requiredItems);
          }
          if (result.error) {
            this.setState({ savingConfigError: result.error });
          }
          return;
        }

        if (this.props.refreshAppData) {
          this.props.refreshAppData();
        }

        if (fromLicenseFlow) {
          const hasPreflight = getKotsApp?.getKotsApp?.hasPreflight;
          if (hasPreflight) {
            history.replace("/preflight");
          } else {
            if (this.props.refetchListApps) {
              await this.props.refetchListApps();
            }
            history.replace(`/app/${slug}`);
          }
        } else {
          this.setState({ savingConfig: false, changed: false, showNextStepModal: true });
        }
      })
      .catch((err) => {
        console.error(err);
        this.setState({ savingConfig: false });
      });
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

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
  }

  render() {
    const { configGroups, savingConfig, changed, showNextStepModal, savingConfigError } = this.state;
    const { fromLicenseFlow, getKotsApp, match } = this.props;

    if (!configGroups.length || getKotsApp?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const app = this.props.app || getKotsApp?.getKotsApp;
    const gitops = app?.downstreams?.length && app.downstreams[0]?.gitops;
    const isNewVersion = !fromLicenseFlow && match.params.sequence == undefined;

    return (
      <div className={classNames("flex1 flex-column u-padding--20 alignItems--center u-overflow--auto")}>
        {fromLicenseFlow && app && <span className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginTop--auto">Configure {app.name}</span>}
        <div className={classNames("ConfigOuterWrapper flex u-padding--15", { "u-marginTop--20": fromLicenseFlow })}>
          <div className="ConfigInnerWrapper flex1 u-padding--15">
            <div className="flex1">
              <ShipConfigRenderer groups={configGroups} getData={this.handleConfigChange} />
            </div>
          </div>
        </div>
        {savingConfig ?
          <div className="u-marginTop--20 u-marginBottom--auto">
            <Loader size="30" />
          </div>
          :
          <div className="ConfigError--wrapper flex-column u-marginTop--20 u-marginBottom--auto alignItems--center">
            {savingConfigError && <span className="u-color--chestnut u-marginBottom--20 u-fontWeight--bold">{savingConfigError}</span>}
            <button className="btn secondary blue" disabled={!changed && !fromLicenseFlow} onClick={this.handleSave}>{fromLicenseFlow ? "Continue" : "Save config"}</button>
          </div>
        }

        <Modal
          isOpen={showNextStepModal}
          onRequestClose={this.hideNextStepModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Next step"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          {gitops?.enabled ?
            <div className="Modal-body">
              {<p className="u-fontSize--large u-color--tuna u-lineHeight--medium u-marginBottom--20">
                The config for {app.name} has been updated. A new commit has been made to the gitops repository with these changes. Please head to the <a className="link" target="_blank" href={gitops?.uri} rel="noopener noreferrer">repo</a> to see the diff.
              </p>}
              <div className="flex justifyContent--flexEnd">
                <button type="button" className="btn blue primary" onClick={this.hideNextStepModal}>Ok, got it!</button>
              </div>
            </div>
            :
            <div className="Modal-body">
              {isNewVersion ? 
                <p className="u-fontSize--large u-color--tuna u-lineHeight--medium u-marginBottom--20">
                  The config for {app?.name} has been updated. A new version is available on the version history page with these changes.
                </p>
                :
                <p className="u-fontSize--large u-color--tuna u-lineHeight--medium u-marginBottom--20">
                  The config for {app?.name} has been updated.
                </p>
              }
              <div className="flex justifyContent--flexEnd">
                <button type="button" className="btn blue secondary u-marginRight--10" onClick={this.hideNextStepModal}>Continue editing</button>
                <Link to={`/app/${app?.slug}/version-history`}>
                  <button type="button" className="btn blue primary">{isNewVersion ? "Go to new version" : "Go to updated version"}</button>
                </Link>
              </div>
            </div>
          }
        </Modal>
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
  graphql(updateDownstreamsStatus, {
    props: ({ mutate }) => ({
      updateDownstreamsStatus: (slug, sequence, status) => mutate({ variables: { slug, sequence, status } })
    })
  }),
)(AppConfig));
