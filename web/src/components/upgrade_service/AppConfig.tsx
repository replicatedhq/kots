import { Component } from "react";
import { AppConfigRenderer } from "@src/components/AppConfigRenderer";
import { useLocation, useNavigate } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import classNames from "classnames";
import debounce from "lodash/debounce";
import find from "lodash/find";
import map from "lodash/map";
import Loader from "@src/components/shared/Loader";
import ErrorModal from "@src/components/modals/ErrorModal";

import "@src/scss/components/watches/WatchConfig.scss";

import Icon from "@src/components/Icon";

// Types
import { KotsParams } from "@types";

type Props = {
  location: ReturnType<typeof useLocation>;
  params: KotsParams;
  navigate: ReturnType<typeof useNavigate>;
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
  configErrorMessage: string;
  configGroups: ConfigGroup[];
  configLoading: boolean;
  displayErrorModal: boolean;
  errorTitle: string;
  gettingConfigErrMsg: string;
  initialConfigGroups: ConfigGroup[];
  showConfigError: boolean;
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
      showConfigError: false,
      configErrorMessage: "",
      configGroups: [],
      configLoading: false,
      displayErrorModal: false,
      errorTitle: "",
      gettingConfigErrMsg: "",
      showValidationError: false,
      initialConfigGroups: [],
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
    window.addEventListener("resize", this.determineSidebarHeight);
    this.getConfig();
  }

  componentDidUpdate(lastProps: Props, lastState: State) {
    const { location } = this.props;
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

  getConfig = async () => {
    const { slug } = this.props.params;

    this.setState({
      configLoading: true,
      gettingConfigErrMsg: "",
      showConfigError: false,
      configErrorMessage: "",
    });

    fetch(
      `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/config${window.location.search}`,
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
        if (!data.configGroups?.length) {
          throw new Error("No config data found");
        }
        this.setState({
          configGroups: data.configGroups,
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

  updateUrlWithErrorId = (requiredItems: RequiredItems) => {
    const { slug } = this.props.params;
    this.props.navigate(
      `/app/${slug}/config${window.location.search}#${requiredItems[0]}-group`
    );
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

  handleNext = async () => {
    const { slug } = this.props.params;
    const url = `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/deploy`;
    fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        isSkipPreflights: false,
        continueWithFailedPreflights: false,
      }),
      credentials: "include",
    })
      .then(async (response) => {
        console.log(await response.text())
      })
      .catch(function (error) {
        console.log(error); // TODO handle error
      });
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
      `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/liveconfig${window.location.search}`,
      {
        signal,
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        credentials: "include",
        method: "POST",
        body: JSON.stringify({ configGroups: groups }),
      }
    )
      .then(async (response) => {
        if (!response.ok) {
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
        this.setState({ configGroups: newGroups });
      })
      .catch((error) => {
        if (error?.name !== "AbortError") {
          console.log(error);
          this.setState({
            showConfigError: Boolean(error?.message),
            configErrorMessage: error?.message,
          });
        }
      });
  };

  handleDownloadFile = async (fileName: string) => {
    const { slug } = this.props.params;
    const url = `${process.env.API_ENDPOINT}/upgrade-service/app/${slug}/config/${fileName}/download${window.location.search}`;
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

  render() {
    const {
      showConfigError,
      configErrorMessage,
      configGroups,
      configLoading,
      displayErrorModal,
      errorTitle,
      gettingConfigErrMsg,
      showValidationError,
    } = this.state;

    const { params } = this.props;

    if (configLoading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
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
        <div className="tw-flex tw-flex-col tw-mx-48">
          <div className="tw-flex tw-justify-center" style={{ gap: "20px" }}>
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
              <div
                className={classNames(
                  "ConfigOuterWrapper card-bg u-padding--15"
                )}
              >
                <div className="ConfigInnerWrapper">
                  <AppConfigRenderer
                    groups={configGroups}
                    getData={this.handleConfigChange}
                    handleDownloadFile={this.handleDownloadFile}
                    appSlug={params.slug}
                  />
                </div>
                <div className="flex alignItems--flexStart">
                  <div className="ConfigError--wrapper flex-column alignItems--flexStart">
                    {(showConfigError || this.state.showValidationError) && (
                      <span className="u-textColor--error tw-mb-2 tw-text-xs">
                        {configErrorMessage || validationErrorMessage}
                      </span>
                    )}
                    <button
                      className="btn primary blue"
                      disabled={showValidationError}
                      onClick={this.handleNext}
                    >
                      Next
                    </button>
                  </div>
                </div>
              </div>
            </div>{" "}
          </div>
        </div>
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
