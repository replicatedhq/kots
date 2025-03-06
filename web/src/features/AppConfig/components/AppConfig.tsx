import { Component } from "react";
import { AppConfigRenderer } from "../../../components/AppConfigRenderer";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import classNames from "classnames";
import { KotsPageTitle } from "@components/Head";
import debounce from "lodash/debounce";
import find from "lodash/find";
import map from "lodash/map";
import Modal from "react-modal";
import Loader from "../../../components/shared/Loader";
import ErrorModal from "../../../components/modals/ErrorModal";
import ConfigInfo from "./ConfigInfo";

import "../../../scss/components/watches/WatchConfig.scss";
import { Utilities } from "../../../utilities/utilities";

import Icon from "@src/components/Icon";

// Types
import { App, KotsParams, Version } from "@types";

type Props = {
  location: ReturnType<typeof useLocation>;
  params: KotsParams;
  app: App;
  fromLicenseFlow: boolean;
  isEmbeddedCluster: boolean;
  refreshAppData: () => void;
  refetchApps: () => void;
  navigate: ReturnType<typeof useNavigate>;
  setCurrentStep: (step: number) => void;
  setNavbarConfigGroups: (ConfigGroup) => void;
  setActiveGroups: (ConfigGroup) => void;
};

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
  configErrorMessage: string;
  configGroups: ConfigGroup[];
  configLoading: boolean;
  displayErrorModal: boolean;
  downstreamVersion: Version | null;
  errorTitle: string;
  gettingConfigErrMsg: string;
  initialConfigGroups: ConfigGroup[];
  savingConfig: boolean;
  showConfigError: boolean;
  showNextStepModal: boolean;
  showValidationError: boolean;
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
      showConfigError: false,
      configErrorMessage: "",
      configGroups: [],
      configLoading: false,
      displayErrorModal: false,
      downstreamVersion: null,
      errorTitle: "",
      gettingConfigErrMsg: "",
      showValidationError: false,
      initialConfigGroups: [],
      savingConfig: false,
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
    this.props.setCurrentStep(2);
    const { app, navigate } = this.props;
    if (app && !app.isConfigurable) {
      // app not configurable - redirect
      navigate(`/app/${app.slug}`, { replace: true });
    }
    window.addEventListener("resize", this.determineSidebarHeight);

    if (!app) {
      this.fetchApp();
    } else {
      this.setState({ app });
    }
    this.getConfig();
  }

  componentDidUpdate(lastProps: Props, lastState: State) {
    const { params, location, app } = this.props;

    if (app && !app.isConfigurable) {
      // app not configurable - redirect
      this.props.navigate(`/app/${app.slug}`, { replace: true });
    }
    if (params.sequence !== lastProps.params.sequence) {
      this.getConfig();
    }
    if (
      this.state.configGroups &&
      this.state.configGroups !== lastState.configGroups
    ) {
      this.determineSidebarHeight();
    }
    // need to dig into this more
    if (location.hash !== lastProps.location.hash && location.hash) {
      // navigate to error if there is one
      if (this.state.showConfigError) {
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

  fetchApp = async (): Promise<App | undefined> => {
    try {
      const { slug } = this.props.params;
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
        return app;
      }
    } catch (err) {
      console.log(err);
    }
  };

  getConfig = async () => {
    const sequence = this.getSequence();
    const { slug } = this.props.params;

    this.setState({
      configLoading: true,
      gettingConfigErrMsg: "",
      showConfigError: false,
      configErrorMessage: "",
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
        if (this.props.isEmbeddedCluster) {
          this.props.setNavbarConfigGroups(data.configGroups);
        }
        if (this.props.location.hash.length > 0) {
          this.navigateToCurrentHash();
        } else {
          if (this.props.isEmbeddedCluster) {
            this.props.setActiveGroups([data.configGroups[0].name]);
          }
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
    const { params, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return 0;
    }
    if (params.sequence != undefined) {
      return parseInt(params.sequence);
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
    const { params, app, fromLicenseFlow } = this.props;
    if (fromLicenseFlow) {
      return params.slug;
    }
    return app?.slug;
  };

  updateUrlWithErrorId = (requiredItems: RequiredItems) => {
    const { params, fromLicenseFlow } = this.props;
    const { slug } = this.props.params;

    if (fromLicenseFlow) {
      this.props.navigate(
        `/${slug}/config${window.location.search}#${requiredItems[0]}-group`
      );
    } else if (params.sequence) {
      this.props.navigate(
        `/app/${slug}/config/${params.sequence}${window.location.search}#${requiredItems[0]}-group`
      );
    } else {
      this.props.navigate(
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
    this.setState({ configGroups, showConfigError: true }, () => {
      this.updateUrlWithErrorId(requiredItems);
    });
  };

  handleSave = async () => {
    this.setState({
      savingConfig: true,
      showConfigError: false,
      configErrorMessage: "",
    });

    const { fromLicenseFlow, navigate, params } = this.props;
    const sequence = this.getSequence();
    const { slug } = this.props.params;
    const createNewVersion = !fromLicenseFlow && params.sequence == undefined;

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
            this.setState({
              showConfigError: Boolean(result.error),
              configErrorMessage: result.error,
            });
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
          if (this.props.isEmbeddedCluster) {
            this.props.setNavbarConfigGroups(newGroups);
          }
          if (result.error) {
            this.setState({
              showConfigError: Boolean(result.error),
              configErrorMessage: result.error,
              showValidationError: true,
            });
          }
          return;
        }

        if (this.props.refreshAppData) {
          await this.props.refreshAppData();
        }

        if (fromLicenseFlow) {
          const app = await this.fetchApp();
          const hasPreflight = app?.hasPreflight;

          if (hasPreflight) {
            navigate(`/${slug}/preflight`);
          } else {
            await this.props.refetchApps();
            navigate(`/app/${slug}`);
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
          showConfigError: Boolean(err),
          configErrorMessage: err
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
    const { slug } = this.props.params;

    // cancel current request (if any)
    if (this.fetchController) {
      this.fetchController.abort();
    }

    this.setState({
      showConfigError: false,
      configErrorMessage: "",
    });

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
          this.setState({
            showConfigError: Boolean(res?.error),
            configErrorMessage: res?.error,
          });
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
        if (this.props.isEmbeddedCluster) {
          this.props.setNavbarConfigGroups({ newGroups, changed });
        }
      })
      .catch((error) => {
        if (error.name !== "AbortError") {
          console.log(error);
          this.setState({
            showConfigError: Boolean(error?.message),
            configErrorMessage: error?.message,
          });
        }
      });
  };

  handleDownloadFile = async (fileName: string) => {
    const sequence = this.getSequence();
    const { slug } = this.props.params;
    const url = `${process.env.API_ENDPOINT}/app/${slug}/config/${sequence}/${fileName}/download`;
    fetch(url, {
      method: "GET",
      headers: {
        "Content-Type": "application/octet-stream",
      },
      credentials: "include",
    })
      .then((response) => {
        if (!response.ok) {
          throw Error(response.statusText); // TODO: handle error
        }
        return response.blob();
      })
      .then((blob) => {
        const downloadURL = window.URL.createObjectURL(new Blob([blob]));
        const link = document.createElement("a");
        link.href = downloadURL;
        link.setAttribute("download", fileName);
        document.body.appendChild(link);
        link.click();
        link.parentNode?.removeChild(link);
      })
      .catch(function (error) {
        console.log(error); // TODO handle error
      });
  };

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
  };

  isConfigReadOnly = (app: App) => {
    const { params, isEmbeddedCluster } = this.props;
    if (!params.sequence) {
      return false;
    }
    if (!isEmbeddedCluster) {
      return false;
    }
    // in embedded cluster, past versions cannot be edited
    const isPastVersion = find(app.downstream?.pastVersions, {
      sequence: parseInt(params.sequence),
    });
    return !!isPastVersion;
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
    this.props.navigate(
      `/app/${app?.slug}/config/${pendingVersions[0].parentSequence}`
    );
  };

  render() {
    const {
      app,
      changed,
      showConfigError,
      configErrorMessage,
      configGroups,
      configLoading,
      displayErrorModal,
      downstreamVersion,
      errorTitle,
      gettingConfigErrMsg,
      savingConfig,
      showNextStepModal,
      showValidationError,
    } = this.state;
    const { fromLicenseFlow, params } = this.props;

    if (configLoading || !app) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const gitops = app.downstream?.gitops;
    const isNewVersion = !fromLicenseFlow && params.sequence == undefined;

    const urlParams = new URLSearchParams(window.location.search);

    let downstreamVersionLabel = downstreamVersion?.versionLabel;
    if (!downstreamVersionLabel) {
      // TODO: add error handling for this. empty string is not valid
      downstreamVersionLabel = urlParams.get("semver") || "";
    }

    let saveButtonText = fromLicenseFlow ? "Continue" : "Save config";

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
      <div className=" u-overflow--auto tw-font-sans tw-max-w-[1024px] tw-mx-auto">
        <KotsPageTitle pageName="Config" showAppSlug />
        <div className="tw-mt-8 tw-shadow-[0_1px_0_#c4c8ca]">
          <p className="tls-header tw-pb-8 tw-font-bold u-textColor--primary">
            Configure {app.name}
          </p>
        </div>
        <div className="flex flex1 tw-mb-10 tw-mt-8 tw-flex tw-flex-col tw-gap-4 card-bg">
          <div className="tw-flex tw-justify-center" style={{ gap: "20px" }}>
            {/*If this is the initial installation of an app on embedded cluster,*/}
            {/*do not show the config sidebar as the installation wizard already has one.*/}
            {!(
              Utilities.isInitialAppInstall(app) && this.props.isEmbeddedCluster
            ) && (
              <div
                id="configSidebarWrapper"
                className="config-sidebar-wrapper card-bg clickable"
                data-testid="config-sidebar-wrapper"
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
            )}
            <div
              data-testid="config-area"
              className="ConfigArea--wrapper !tw-pt-0"
            >
              <ConfigInfo
                app={app}
                fromLicenseFlow={this.props.fromLicenseFlow}
              />
              <div
                className={classNames(
                  "ConfigOuterWrapper card-bg u-padding--15"
                )}
              >
                <div className="ConfigInnerWrapper">
                  <AppConfigRenderer
                    appSlug={app.slug}
                    configSequence={params.sequence}
                    getData={this.handleConfigChange}
                    groups={configGroups}
                    handleDownloadFile={this.handleDownloadFile}
                    readonly={this.isConfigReadOnly(app)}
                  />
                </div>
                <div className="flex tw-items-center tw-w-full">
                  {savingConfig && (
                    <div className="u-paddingBottom--30">
                      <Loader size="30" />
                    </div>
                  )}
                  {!savingConfig && (
                    <div className="ConfigError--wrapper tw-flex tw-items-center tw-justify-between !tw-w-full">
                      {(showConfigError || this.state.showValidationError) && (
                        <span className="u-textColor--error tw-mb-2 tw-text-xs">
                          {configErrorMessage || validationErrorMessage}
                        </span>
                      )}
                      <button
                        className="btn primary blue tw-ml-auto"
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
            </div>{" "}
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
            <div
              className="Modal-body"
              data-testid="config-next-step-modal-gitops"
            >
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
            <div className="Modal-body" data-testid="config-next-step-modal">
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
