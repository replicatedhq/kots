import React, { Component } from "react";
import { AppConfigRenderer } from "../../../components/AppConfigRenderer";
import { withRouter, Link } from "react-router-dom";
import PropTypes from "prop-types";
import classNames from "classnames";
import Helmet from "react-helmet";
import debounce from "lodash/debounce";
import find from "lodash/find";
import map from "lodash/map";
import Modal from "react-modal";
import Loader from "../../../components/shared/Loader";
import ErrorModal from "../../../components/modals/ErrorModal";
import { HelmDeployModal } from "../../../components/shared/modals/HelmDeployModal";
import {
  UseIsHelmManaged,
  useDownloadValues,
  useSaveConfig,
} from "../../../components/hooks";
import ConfigInfo from "./ConfigInfo";

import "../../../scss/components/watches/WatchConfig.scss";
import { Utilities } from "../../../utilities/utilities";
import { Flex, Span } from "../../../styles/common";
import {
  SideNavWrapper,
  SideNavGroup,
  GroupTitle,
  SideNavItems,
} from "../styles";

class AppConfig extends Component {
  static propTypes = {
    app: PropTypes.object,
  };

  constructor(props) {
    super(props);

    this.state = {
      configLoading: false,
      gettingConfigErrMsg: "",
      errorTitle: "",
      initialConfigGroups: [],
      configGroups: [],
      savingConfig: false,
      changed: false,
      showNextStepModal: false,
      activeGroups: [],
      configError: "",
      app: null,
      displayErrorModal: false,
    };

    this.handleConfigChange = debounce(this.handleConfigChange, 250);
    this.determineSidebarHeight = debounce(this.determineSidebarHeight, 250);
  }

  componentWillUnmount() {
    window.removeEventListener("resize", this.determineSidebarHeight);
  }

  componentDidMount() {
    const { app, history } = this.props;
    if (app && !app.isConfigurable) {
      // app not configurable - redirect
      history.replace(`/app/${app.slug}`);
    }
    window.addEventListener("resize", this.determineSidebarHeight);

    if (!this.props.app) {
      this.getApp();
    }
    this.getConfig();
  }

  componentDidUpdate(lastProps, lastState) {
    const { match, location } = this.props;

    if (this.state.app && !this.state.app.isConfigurable) {
      // app not configurable - redirect
      this.props.history.replace(`/app/${this.state.app.slug}`);
    }
    if (match.params.sequence !== lastProps.match.params.sequence) {
      this.getConfig();
    }
    if (
      this.state.configGroups &&
      this.state.configGroups !== lastState.configGroups
    ) {
      this.determineSidebarHeight();
    }
    if (location.hash !== lastProps.location.hash && location.hash) {
      // navigate to error if there is one
      if (this.state.configError) {
        const hash = location.hash.slice(1);
        const element = document.getElementById(hash);
        if (element) {
          element.scrollIntoView();
        }
      }
    }
  }

  determineSidebarHeight = () => {
    const windowHeight = window.innerHeight;
    const sidebarEl = this.sidebarWrapper;
    if (sidebarEl) {
      sidebarEl.style.maxHeight = `${windowHeight - 225}px`;
    }
  };

  navigateToCurrentHash = () => {
    const hash = this.props.location.hash.slice(1);
    // slice `-group` off the end of the hash
    const slicedHash = hash.slice(0, -6);
    let activeGroupName = null;
    this.state.configGroups.map((group) => {
      const activeItem = find(group.items, ["name", slicedHash]);
      if (activeItem) {
        activeGroupName = group.name;
      }
    });

    if (activeGroupName) {
      this.setState({ activeGroups: [activeGroupName], configLoading: false });
      document.getElementById(hash).scrollIntoView();
    }
  };

  getApp = async () => {
    if (this.props.app) {
      return;
    }

    try {
      const { slug } = this.props.match.params;
      const res = await fetch(`${process.env.API_ENDPOINT}/app/${slug}`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        this.setState({ app });
      }
    } catch (err) {
      console.log(err);
    }
  };

  getConfig = async () => {
    const sequence = this.getSequence();
    const slug = this.getSlug();

    this.setState({
      configLoading: true,
      gettingConfigErrMsg: "",
      configError: false,
    });

    fetch(`${process.env.API_ENDPOINT}/app/${slug}/config/${sequence}`, {
      method: "GET",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
    })
      .then(async (response) => {
        if (!response.ok) {
          const res = await response.json();
          throw new Error(res.error);
        }
        const data = await response.json();
        this.setState({
          configGroups: data.configGroups,
          downstreamVersion: data.downstreamVersion,
          changed: false,
          configLoading: false,
        });
        if (this.props.location.hash.length > 0) {
          this.navigateToCurrentHash();
        } else {
          this.setState({
            activeGroups: [data.configGroups[0].name],
            configLoading: false,
            gettingConfigErrMsg: "",
          });
        }
      })
      .catch((err) => {
        this.setState({
          configLoading: false,
          errorTitle: `Failed to get config data`,
          displayErrorModal: true,
          gettingConfigErrMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  getSequence = () => {
    const { match, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return 0;
    }
    if (match.params.sequence != undefined) {
      return parseInt(match.params.sequence);
    }

    // check is current deployed config latest
    const currentDeployedSequence =
      app?.downstream?.currentVersion?.parentSequence;
    if (currentDeployedSequence != undefined) {
      return currentDeployedSequence;
    } else {
      return app?.currentSequence;
    }
  };

  getSlug = () => {
    const { match, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return match.params.slug;
    }
    return app?.slug;
  };

  updateUrlWithErrorId = (requiredItems) => {
    const { match, fromLicenseFlow } = this.props;
    const slug = this.getSlug();

    if (fromLicenseFlow) {
      this.props.history.push(`/${slug}/config#${requiredItems[0]}-group`);
    } else if (match.params.sequence) {
      this.props.history.push(
        `/app/${slug}/config/${match.params.sequence}#${requiredItems[0]}-group`
      );
    } else {
      this.props.history.push(`/app/${slug}/config#${requiredItems[0]}-group`);
    }
  };

  markRequiredItems = (requiredItems) => {
    const configGroups = this.state.configGroups;
    requiredItems.forEach((requiredItem) => {
      configGroups.forEach((configGroup) => {
        const item = configGroup.items.find(
          (item) => item.name === requiredItem
        );
        if (item) {
          item.error = "This item is required";
        }
      });
    });
    this.setState({ configGroups, configError: true }, () => {
      this.updateUrlWithErrorId(requiredItems);
    });
  };

  handleSave = async () => {
    this.setState({ savingConfig: true, configError: "" });

    const { fromLicenseFlow, history, match } = this.props;
    const sequence = this.getSequence();
    const slug = this.getSlug();
    const createNewVersion =
      !fromLicenseFlow && match.params.sequence == undefined;

    fetch(`${process.env.API_ENDPOINT}/app/${slug}/config`, {
      method: "PUT",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        configGroups: this.state.configGroups,
        sequence,
        createNewVersion,
      }),
    })
      .then((res) => res.json())
      .then(async (result) => {
        this.setState({ savingConfig: false });

        if (!result.success) {
          if (result.requiredItems?.length) {
            this.markRequiredItems(result.requiredItems);
          }
          if (result.error) {
            this.setState({ configError: result.error });
          }
          return;
        }

        if (this.props.refreshAppData) {
          await this.props.refreshAppData();
        }

        if (fromLicenseFlow) {
          const hasPreflight = this.state.app?.hasPreflight;
          if (hasPreflight) {
            history.replace(`/${slug}/preflight`);
          } else {
            if (this.props.refetchAppsList) {
              await this.props.refetchAppsList();
            }
            history.replace(`/app/${slug}`);
          }
        } else {
          this.setState({
            savingConfig: false,
            changed: false,
            showNextStepModal: true,
          });
        }
      })
      .catch((err) => {
        this.setState({
          savingConfig: false,
          configError: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  isConfigChanged = (newGroups) => {
    const { initialConfigGroups } = this.state;
    for (let g = 0; g < newGroups.length; g++) {
      const group = newGroups[g];
      if (!group.items) {
        continue;
      }
      for (let i = 0; i < group.items.length; i++) {
        const newItem = group.items[i];
        const oldItem = this.getItemInConfigGroups(
          initialConfigGroups,
          newItem.name
        );
        if (!oldItem || oldItem.value !== newItem.value) {
          return true;
        }
      }
    }
    return false;
  };

  getItemInConfigGroups = (configGroups, itemName) => {
    let foundItem;
    map(configGroups, (group) => {
      map(group.items, (item) => {
        if (item.name === itemName) {
          foundItem = item;
        }
      });
    });
    return foundItem;
  };

  handleConfigChange = (groups) => {
    const sequence = this.getSequence();
    const slug = this.getSlug();

    // cancel current request (if any)
    if (this.fetchController) {
      this.fetchController.abort();
    }

    this.fetchController = new AbortController();
    const signal = this.fetchController.signal;

    fetch(`${process.env.API_ENDPOINT}/app/${slug}/liveconfig`, {
      signal,
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      method: "POST",
      body: JSON.stringify({ configGroups: groups, sequence: sequence }),
    })
      .then(async (response) => {
        if (!response.ok) {
          if (response.status == 401) {
            Utilities.logoutUser();
            return;
          }
          const res = await response.json();
          this.setState({ configError: res?.error });
          return;
        }

        const data = await response.json();
        const oldGroups = this.state.configGroups;
        const newGroups = data.configGroups;
        map(newGroups, (group) => {
          if (!group.items) {
            return;
          }
          group.items.forEach((newItem) => {
            if (newItem.type === "password") {
              const oldItem = this.getItemInConfigGroups(
                oldGroups,
                newItem.name
              );
              if (oldItem) {
                newItem.value = oldItem.value;
              }
            }
          });
        });
        const changed = this.isConfigChanged(newGroups);
        this.setState({ configGroups: newGroups, changed });
      })
      .catch((error) => {
        if (error.name !== "AbortError") {
          console.log(error);
          this.setState({ configError: error?.message });
        }
      });
  };

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
  };

  isConfigReadOnly = (app) => {
    const { match } = this.props;
    if (!match.params.sequence) {
      return false;
    }
    const sequence = parseInt(match.params.sequence);
    const isCurrentVersion =
      app.downstream?.currentVersion?.sequence === sequence;
    const isLatestVersion = app.currentSequence === sequence;
    const pendingVersion = find(app.downstream?.pendingVersions, {
      sequence: sequence,
    });
    return !isLatestVersion && !isCurrentVersion && !pendingVersion?.semver;
  };

  toggleActiveGroups = (name) => {
    let groupsArr = this.state.activeGroups;
    if (groupsArr.includes(name)) {
      let updatedGroupsArr = groupsArr.filter((n) => n !== name);
      this.setState({ activeGroups: updatedGroupsArr });
    } else {
      groupsArr.push(name);
      this.setState({ activeGroups: groupsArr });
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  navigateToUpdatedConfig = (app) => {
    this.setState({ showNextStepModal: false });

    const pendingVersions = app?.downstream?.pendingVersions;
    this.props.history.push(
      `/app/${app?.slug}/config/${pendingVersions[0].parentSequence}`
    );
  };

  render() {
    const {
      configGroups,
      downstreamVersion,
      savingConfig,
      changed,
      showNextStepModal,
      configError,
      configLoading,
      gettingConfigErrMsg,
      displayErrorModal,
      errorTitle,
    } = this.state;
    const { fromLicenseFlow, match } = this.props;
    const app = this.props.app || this.state.app;

    if (configLoading || !app) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const gitops = app.downstream?.gitops;
    const isNewVersion = !fromLicenseFlow && match.params.sequence == undefined;

    return (
      <Flex flex="1" direction="column" p="20" align="center">
        <Helmet>
          <title>{`${app.name} Config`}</title>
        </Helmet>
        {fromLicenseFlow && app && (
          <Span size="18" weight="bold" mt="30" ml="38">
            Configure {app.name}
          </Span>
        )}
        <Flex gap="20px">
          <SideNavWrapper
            id="configSidebarWrapper"
            ref={(wrapper) => (this.sidebarWrapper = wrapper)}
          >
            {configGroups?.map((group, i) => {
              if (
                group.title === "" ||
                group.title.length === 0 ||
                group.hidden ||
                group.when === "false"
              ) {
                return;
              }
              return (
                <SideNavGroup
                  key={`${i}-${group.name}-${group.title}`}
                  className={`${
                    this.state.activeGroups.includes(group.name)
                      ? "group-open"
                      : ""
                  }`}
                >
                  <Flex
                    align="center"
                    onClick={() => this.toggleActiveGroups(group.name)}
                  >
                    <GroupTitle fontSize="16" className="u-lineHeight--normal">
                      {group.title}
                    </GroupTitle>
                    <span className="icon u-darkDropdownArrow clickable flex-auto" />
                  </Flex>
                  {group.items ? (
                    <SideNavItems>
                      {group.items?.map((item, i) => {
                        const hash = this.props.location.hash.slice(1);
                        if (item.hidden || item.when === "false") {
                          return;
                        }
                        return (
                          <a
                            className={`u-fontSize--normal u-lineHeight--normal ${
                              hash === `${item.name}-group` ? "active-item" : ""
                            }`}
                            href={`#${item.name}-group`}
                            key={`${i}-${item.name}-${item.title}`}
                          >
                            {item.title}
                          </a>
                        );
                      })}
                    </SideNavItems>
                  ) : null}
                </SideNavGroup>
              );
            })}
          </SideNavWrapper>
          <div className="ConfigArea--wrapper">
            <UseIsHelmManaged>
              {({ data = {} }) => {
                const { isHelmManaged } = data;

                const {
                  mutate: saveConfig,
                  isLoading: isSaving,
                  isError: saveError,
                } = useSaveConfig({
                  appSlug: this.getSlug(),
                });

                const {
                  download,
                  clearError: clearDownloadError,
                  error: downloadError,
                  isDownloading,
                } = useDownloadValues({
                  appSlug: this.getSlug(),
                  fileName: "values.yaml",
                });

                const handleGenerateConfig = () => {
                  this.setState({
                    showHelmDeployModal: true,
                  });
                  saveConfig({
                    body: JSON.stringify({
                      configGroups: this.state.configGroups,
                      sequence: this.getSequence(),
                      createNewVersion:
                        !this.props.fromLicenseFlow &&
                        this.props.match.params.sequence == undefined,
                    }),
                  });
                };

                return (
                  <>
                    {!isHelmManaged && (
                      <ConfigInfo
                        app={app}
                        match={this.props.match}
                        fromLicenseFlow={this.props.fromLicenseFlow}
                      />
                    )}
                    <div
                      className={classNames(
                        "ConfigOuterWrapper u-paddingTop--30",
                        { "u-marginTop--20": fromLicenseFlow }
                      )}
                      style={{ width: "100%" }}
                    >
                      <div className="ConfigInnerWrapper">
                        <AppConfigRenderer
                          groups={configGroups}
                          getData={this.handleConfigChange}
                          readonly={this.isConfigReadOnly(app)}
                          configSequence={match.params.sequence}
                          appSlug={app.slug}
                        />
                      </div>
                    </div>
                    <div className="flex alignItems--flexStart">
                      {isHelmManaged && (
                        <div className="ConfigError--wrapper flex-column u-paddingBottom--30 alignItems--flexStart">
                          <button
                            className="btn primary blue"
                            disabled={isSaving}
                            onClick={handleGenerateConfig}
                          >
                            Generate Upgrade Command
                          </button>
                        </div>
                      )}
                      {!isHelmManaged && savingConfig && (
                        <div className="u-paddingBottom--30">
                          <Loader size="30" />
                        </div>
                      )}
                      {!isHelmManaged && !savingConfig && (
                        <div className="ConfigError--wrapper flex-column u-paddingBottom--30 alignItems--flexStart">
                          {configError && (
                            <span className="u-textColor--error u-marginBottom--20 u-fontWeight--bold">
                              {configError}
                            </span>
                          )}
                          <button
                            className="btn primary blue"
                            disabled={
                              (!changed && !fromLicenseFlow) ||
                              this.isConfigReadOnly(app)
                            }
                            onClick={this.handleSave}
                          >
                            {fromLicenseFlow ? "Continue" : "Save config"}
                          </button>
                        </div>
                      )}
                    </div>
                    {this.state.showHelmDeployModal && (
                      <>
                        <HelmDeployModal
                          appSlug={this.props?.app?.slug}
                          chartPath={this.props?.app?.chartPath || ""}
                          downloadClicked={download}
                          downloadError={downloadError}
                          isDownloading={isDownloading}
                          hideHelmDeployModal={() => {
                            this.setState({ showHelmDeployModal: false });
                            clearDownloadError();
                          }}
                          registryUsername={
                            this.props?.app?.credentials?.username
                          }
                          registryPassword={
                            this.props?.app?.credentials?.password
                          }
                          saveError={saveError}
                          showHelmDeployModal={true}
                          showDownloadValues={true}
                          subtitle="Follow the steps below to upgrade the release with your new values.yaml."
                          title={`Upgrade ${this.props?.app?.slug}`}
                          upgradeTitle="Upgrade release"
                          version={downstreamVersion?.versionLabel}
                          namespace={this.props?.app?.namespace}
                        />
                      </>
                    )}
                  </>
                );
              }}
            </UseIsHelmManaged>
          </div>
        </Flex>

        <Modal
          isOpen={showNextStepModal}
          onRequestClose={this.hideNextStepModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Next step"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          {gitops?.isConnected ? (
            <div className="Modal-body">
              {
                <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
                  The config for {app.name} has been updated. A new commit has
                  been made to the gitops repository with these changes. Please
                  head to the{" "}
                  <a
                    className="link"
                    target="_blank"
                    href={gitops?.uri}
                    rel="noopener noreferrer"
                  >
                    repo
                  </a>{" "}
                  to see the diff.
                </p>
              }
              <div className="flex justifyContent--flexEnd">
                <button
                  type="button"
                  className="btn blue primary"
                  onClick={this.hideNextStepModal}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          ) : (
            <div className="Modal-body">
              {isNewVersion ? (
                <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
                  The config for {app?.name} has been updated. A new version is
                  available on the version history page with these changes.
                </p>
              ) : (
                <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
                  The config for {app?.name} has been updated.
                </p>
              )}
              <div className="flex justifyContent--flexEnd">
                <button
                  type="button"
                  className="btn blue secondary u-marginRight--10"
                  onClick={() => this.navigateToUpdatedConfig(app)}
                >
                  Edit the latest config
                </button>
                <Link to={`/app/${app?.slug}/version-history`}>
                  <button type="button" className="btn blue primary">
                    {isNewVersion
                      ? "Go to new version"
                      : "Go to updated version"}
                  </button>
                </Link>
              </div>
            </div>
          )}
        </Modal>
        {gettingConfigErrMsg && (
          <ErrorModal
            errorModal={displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            err={errorTitle}
            errMsg={gettingConfigErrMsg}
            tryAgain={this.getConfig}
          />
        )}
      </Flex>
    );
  }
}

export default withRouter(AppConfig);
