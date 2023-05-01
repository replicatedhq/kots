import React, { Component } from "react";
import { AppConfigRenderer } from "../../../components/AppConfigRenderer";
import { Link } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import classNames from "classnames";
import { KotsPageTitle } from "@components/Head";
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
import { Span } from "../../../styles/common";
import Icon from "@src/components/Icon";

// Types
import { App, KotsParams, Version } from "@types";
import { RouteComponentProps } from "react-router-dom";

type Props = {
  app: App;
  fromLicenseFlow: boolean;
  isHelmManaged: boolean;
  refreshAppData: () => void;
  refetchAppsList: () => void;
} & RouteComponentProps<KotsParams>;

// This was typed from the implementation of the component so it might be wrong
type ConfigGroup = {
  hidden: boolean;
  hasError: boolean;
  items: ConfigGroupItem[];
  name: string;
  title: string;
  when: "true" | "false";
};

interface ConfigGroupItemValidationErrors {
  item_errors: ConfigGroupItemValidationError[];
  name: string;
}

interface ConfigGroupItemValidationError {
  name: string;
  validation_errors: {
    message: string;
  }[];
}

type ConfigGroupItem = {
  default: string;
  error: string;
  hidden: boolean;
  name: string;
  required: boolean;
  title: string;
  type: string;
  validationError: string;
  value: string;
  when: "true" | "false";
};

type RequiredItems = string[];

type State = {
  activeGroups: string[];
  app: App | null;
  changed: boolean;
  configError: boolean;
  configGroups: ConfigGroup[];
  configLoading: boolean;
  displayErrorModal: boolean;
  downstreamVersion: Version | null;
  errorTitle: string;
  gettingConfigErrMsg: string;
  showValidationError: boolean;
  initialConfigGroups: ConfigGroup[];
  savingConfig: boolean;
  showHelmDeployModal: boolean;
  showNextStepModal: boolean;
};

const validationErrorMessage =
  "Error detected. Please use config nav to the left to locate and fix issues.";

class AppConfig extends Component<Props, State> {
  sidebarWrapper: HTMLElement;

  fetchController: AbortController | null;

  constructor(props: Props) {
    super(props);

    this.state = {
      activeGroups: [],
      app: null,
      changed: false,
      configError: false,
      configGroups: [],
      configLoading: false,
      displayErrorModal: false,
      downstreamVersion: null,
      errorTitle: "",
      gettingConfigErrMsg: "",
      showValidationError: false,
      initialConfigGroups: [],
      savingConfig: false,
      showHelmDeployModal: false,
      showNextStepModal: false,
    };

    this.handleConfigChange = debounce(this.handleConfigChange, 250);
    this.determineSidebarHeight = debounce(this.determineSidebarHeight, 250);
    this.sidebarWrapper = document.createElement("div");
    this.fetchController = null;
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

  componentDidUpdate(lastProps: Props, lastState: State) {
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
    // TODO: use a ref for this instead of setting HTMLElement.style
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
      // TODO: add error handling for when the element with this hash id is not found
      document.getElementById(hash)?.scrollIntoView();
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
          "Content-Type": "application/json",
        },
        credentials: "include",
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

    fetch(
      `${process.env.API_ENDPOINT}/app/${slug}/config/${sequence}${window.location.search}`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
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

  updateUrlWithErrorId = (requiredItems: RequiredItems) => {
    const { match, fromLicenseFlow } = this.props;
    const slug = this.getSlug();

    if (fromLicenseFlow) {
      this.props.history.push(
        `/${slug}/config${window.location.search}#${requiredItems[0]}-group`
      );
    } else if (match.params.sequence) {
      this.props.history.push(
        `/app/${slug}/config/${match.params.sequence}${window.location.search}#${requiredItems[0]}-group`
      );
    } else {
      this.props.history.push(
        `/app/${slug}/config${window.location.search}#${requiredItems[0]}-group`
      );
    }
  };

  markRequiredItems = (requiredItems: RequiredItems) => {
    const configGroups = this.state.configGroups;
    requiredItems.forEach((requiredItem) => {
      configGroups.forEach((configGroup) => {
        const item = configGroup.items.find((i) => i.name === requiredItem);
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
    this.setState({
      savingConfig: true,
      configError: false,
    });

    const { fromLicenseFlow, history, match, isHelmManaged } = this.props;
    const sequence = this.getSequence();
    const slug = this.getSlug();
    const createNewVersion =
      !fromLicenseFlow && match.params.sequence == undefined;

    fetch(`${process.env.API_ENDPOINT}/app/${slug}/config`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
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

          const validationErrors: ConfigGroupItemValidationErrors[] =
            result.validationErrors;
          const [newGroups, hasValidationError] =
            this.mergeConfigGroupsAndValidationErrors(
              this.state.configGroups,
              validationErrors
            );

          this.setState({
            configGroups: newGroups,
            showValidationError: hasValidationError,
          });
          if (result.error) {
            this.setState({
              configError: result.error,
              showValidationError: true,
            });
          }
          return;
        }

        if (this.props.refreshAppData) {
          await this.props.refreshAppData();
        }

        if (isHelmManaged) {
          this.setState({
            showHelmDeployModal: true,
          });
          return;
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

  isConfigChanged = (newGroups: ConfigGroup[]) => {
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

  getItemInConfigGroups = (
    configGroups: ConfigGroup[],
    itemName: string
  ): ConfigGroupItem | undefined => {
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

  // this runs on config update and when save is clicked but before the request is submitted
  // on update it uses the errors from the liveconfig endpoint
  // on save it's mostly used to find required field errors
  mergeConfigGroupsAndValidationErrors = (
    groups: ConfigGroup[],
    validationErrors: ConfigGroupItemValidationErrors[]
  ): [ConfigGroup[], boolean] => {
    let hasValidationError = false;

    const newGroups = groups?.map((group: ConfigGroup) => {
      const newGroup = { ...group };
      const configGroupValidationErrors = validationErrors?.find(
        (validationError) => validationError.name === group.name
      );

      // required errors are handled separately
      if (group?.items?.find((item) => item.error)) {
        newGroup.hasError = true;
      }

      if (configGroupValidationErrors) {
        newGroup.items = newGroup?.items?.map((item: ConfigGroupItem) => {
          const itemValidationError =
            configGroupValidationErrors?.item_errors?.find(
              (validationError) => validationError.name === item.name
            );

          if (itemValidationError) {
            item.validationError =
              itemValidationError?.validation_errors?.[0]?.message;
            newGroup.hasError = true;
            // if there is an error, then block form submission with state.hasValidationError
            if (!hasValidationError) {
              hasValidationError = true;
            }
          }
          return item;
        });
      }
      return newGroup;
    });
    return [newGroups, hasValidationError];
  };

  handleConfigChange = (groups: ConfigGroup[]) => {
    const sequence = this.getSequence();
    const slug = this.getSlug();

    // cancel current request (if any)
    if (this.fetchController) {
      this.fetchController.abort();
    }

    this.fetchController = new AbortController();
    const signal = this.fetchController.signal;

    fetch(
      `${process.env.API_ENDPOINT}/app/${slug}/liveconfig${window.location.search}`,
      {
        signal,
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        credentials: "include",
        method: "POST",
        body: JSON.stringify({ configGroups: groups, sequence: sequence }),
      }
    )
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
        const validationErrors: ConfigGroupItemValidationErrors[] =
          data.validationErrors;

        // track errors at the form level
        this.setState({ showValidationError: false });

        // merge validation errors and config group
        const [newGroups, hasValidationError] =
          this.mergeConfigGroupsAndValidationErrors(
            data.configGroups,
            validationErrors
          );

        this.setState({
          configError: hasValidationError,
          showValidationError: hasValidationError,
        });

        map(newGroups, (group) => {
          if (!group.items) {
            return;
          }
          group.items.forEach((newItem: ConfigGroupItem) => {
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

  isConfigReadOnly = (app: App) => {
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

  toggleActiveGroups = (name: string) => {
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

  navigateToUpdatedConfig = (app: App) => {
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
      showValidationError,
      showNextStepModal,
      configError,
      configLoading,
      gettingConfigErrMsg,
      displayErrorModal,
      errorTitle,
    } = this.state;
    const { fromLicenseFlow, match, isHelmManaged } = this.props;
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

    const urlParams = new URLSearchParams(window.location.search);

    let downstreamVersionLabel = downstreamVersion?.versionLabel;
    if (!downstreamVersionLabel) {
      // TODO: add error handling for this. empty string is not valid
      downstreamVersionLabel = urlParams.get("semver") || "";
    }

    const isPending = urlParams.get("isPending") === "true";

    let saveButtonText = fromLicenseFlow ? "Continue" : "Save config";
    if (isHelmManaged) {
      saveButtonText = "Generate Upgrade Command";
    }

    const sections = document.querySelectorAll(".observe-elements");

    const callback = (entries: IntersectionObserverEntry[]) => {
      entries.forEach(({ isIntersecting, target }) => {
        // find the group nav link that matches the current section in view
        const groupNav = document.querySelector(
          `#config-group-nav-${target.id}`
        );
        // find the active link in the group nav
        const activeLink = document.querySelector(".active-item");
        const hash = this.props.location.hash.slice(1);
        const activeLinkByHash = document.querySelector(`a[href='#${hash}']`);
        if (isIntersecting) {
          groupNav?.classList.add("is-active");
          // if your group is active, item will be active
          if (activeLinkByHash && groupNav?.contains(activeLinkByHash)) {
            activeLinkByHash.classList.add("active-item");
          }
        } else {
          // if the section is not in view, remove the highlight from the active link
          if (groupNav?.contains(activeLink) && activeLink) {
            activeLink.classList.remove("active-item");
          }
          // remove the highlight from the group nav link
          groupNav?.classList.remove("is-active");
        }
      });
    };

    const options = {
      root: document,
      // rootMargin is the amount of space around the root element that the intersection observer will look for intersections
      rootMargin: "20% 0% -75% 0%",
      // threshold: the proportion of the element that must be within the root bounds for it to be considered intersecting
      threshold: 0.15,
    };

    const observer = new IntersectionObserver(callback, options);

    sections.forEach((section) => {
      observer.observe(section);
    });

    return (
      <div className="flex flex-column u-paddingLeft--20 u-paddingBottom--20 u-paddingRight--20 alignItems--center">
        <KotsPageTitle pageName="Config" showAppSlug />
        {fromLicenseFlow && app && (
          <Span size="18" weight="bold" mt="30" ml="38">
            Configure {app.name}
          </Span>
        )}
        <div className="flex" style={{ gap: "20px" }}>
          <div
            id="configSidebarWrapper"
            className="config-sidebar-wrapper card-bg clickable"
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
                <div
                  key={`${i}-${group.name}-${group.title}`}
                  className={`side-nav-group ${
                    this.state.activeGroups.includes(group.name) ||
                    group.hasError
                      ? "group-open"
                      : ""
                  }`}
                  id={`config-group-nav-${group.name}`}
                >
                  <div
                    className="flex alignItems--center"
                    onClick={() => this.toggleActiveGroups(group.name)}
                  >
                    <div className="u-lineHeight--normal group-title u-fontSize--normal">
                      {group.title}
                    </div>
                    {/* adding the arrow-down classes, will rotate the icon when clicked */}
                    <Icon
                      icon="down-arrow"
                      className="darkGray-color clickable flex-auto u-marginLeft--5 arrow-down"
                      size={12}
                      style={{}}
                      color={""}
                      disableFill={false}
                      removeInlineStyle={false}
                    />
                  </div>
                  {group.items ? (
                    <div className="side-nav-items">
                      {group.items
                        ?.filter((item) => item.type !== "label")
                        ?.map((item, j) => {
                          const hash = this.props.location.hash.slice(1);
                          if (item.hidden || item.when === "false") {
                            return;
                          }
                          return (
                            <a
                              className={`u-fontSize--normal u-lineHeight--normal
                                ${
                                  item.validationError || item.error
                                    ? "has-error"
                                    : ""
                                }
                                ${
                                  hash === `${item.name}-group`
                                    ? "active-item"
                                    : ""
                                }`}
                              href={`#${item.name}-group`}
                              key={`${j}-${item.name}-${item.title}`}
                            >
                              {item.title}
                            </a>
                          );
                        })}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
          <div className="ConfigArea--wrapper">
            <UseIsHelmManaged>
              {({ data: isHelmManagedFromHook }) => {
                const { isError: saveError } = useSaveConfig({
                  appSlug: this.getSlug(),
                });

                const { download, clearError: clearDownloadError } =
                  useDownloadValues({
                    appSlug: this.getSlug(),
                    fileName: "values.yaml",
                    sequence: match.params.sequence,
                    versionLabel: downstreamVersionLabel,
                    isPending: isPending,
                  });

                return (
                  <>
                    {!isHelmManagedFromHook && (
                      <ConfigInfo
                        app={app}
                        match={this.props.match}
                        fromLicenseFlow={this.props.fromLicenseFlow}
                      />
                    )}
                    <div
                      className={classNames(
                        "ConfigOuterWrapper card-bg u-padding--15",
                        { "u-marginTop--20": fromLicenseFlow }
                      )}
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
                      <div className="flex alignItems--flexStart">
                        {savingConfig && (
                          <div className="u-paddingBottom--30">
                            <Loader size="30" />
                          </div>
                        )}
                        {!savingConfig && (
                          <div className="ConfigError--wrapper flex-column alignItems--flexStart">
                            {(configError ||
                              this.state.showValidationError) && (
                              <span className="u-textColor--error tw-mb-2 tw-text-xs">
                                {validationErrorMessage}
                              </span>
                            )}
                            <button
                              className="btn primary blue"
                              disabled={
                                showValidationError ||
                                (!changed && !fromLicenseFlow) ||
                                this.isConfigReadOnly(app)
                              }
                              onClick={this.handleSave}
                            >
                              {saveButtonText}
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                    {this.state.showHelmDeployModal && (
                      <>
                        <HelmDeployModal
                          appSlug={this.props?.app?.slug}
                          chartPath={this.props?.app?.chartPath || ""}
                          downloadClicked={download}
                          hideHelmDeployModal={() => {
                            this.setState({ showHelmDeployModal: false });
                            clearDownloadError();
                          }}
                          registryUsername={
                            this.props?.app?.credentials?.username || ""
                          }
                          registryPassword={
                            this.props?.app?.credentials?.password || ""
                          }
                          saveError={saveError}
                          showHelmDeployModal={true}
                          showDownloadValues={true}
                          subtitle="Follow the steps below to upgrade the release with your new values.yaml."
                          title={`Upgrade ${this.props?.app?.slug}`}
                          upgradeTitle="Upgrade release"
                          version={downstreamVersionLabel || ""}
                          namespace={this.props?.app?.namespace || ""}
                          downloadError={false}
                          revision={null}
                        />
                      </>
                    )}
                  </>
                );
              }}
            </UseIsHelmManaged>
          </div>
        </div>

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
      </div>
    );
  }
}

/* eslint-disable */
// @ts-ignore
const AppConfigWithRouter: any = withRouter(AppConfig);

export default AppConfigWithRouter;
/* eslint-enable */
