import { ChangeEvent, PureComponent } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";
import { RouterProps, withRouter } from "@src/utilities/react-router-utilities";
import { Link } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import ErrorModal from "../modals/ErrorModal";
import NavBarDropdown from "./NavBarDropdown";

import "@src/scss/components/shared/NavBar.scss";
import { App } from "@types";

type Props = {
  appsList: App[];
  className?: string;
  errLoggingOut: string;
  fetchingMetadata: boolean;
  isGitOpsSupported: boolean;
  isIdentityServiceSupported: boolean;
  isKurlEnabled: boolean;
  isEmbeddedClusterEnabled: boolean;
  isEmbeddedClusterWaitingForNodes: boolean;
  isSnapshotsSupported: boolean;
  logo: string | null;
  onLogoutError: (message: string) => void;
  refetchAppsList: () => void;
} & RouterProps;

interface State {
  displayErrorModal: boolean;
  loggingOut: boolean;
  selectedTab: string;
}

export class NavBar extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      selectedTab: "",
      loggingOut: false,
      displayErrorModal: false,
    };
  }

  static propTypes = {
    refetchAppsList: PropTypes.func.isRequired,
  };

  handleLogOut = async (e: ChangeEvent) => {
    const { onLogoutError } = this.props;
    e.preventDefault();
    try {
      this.setState({ loggingOut: true, displayErrorModal: false });
      const res = await fetch(`${process.env.API_ENDPOINT}/logout`, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "POST",
      });
      if (!res.ok) {
        if (res.status === 401) {
          this.setState({ loggingOut: false, displayErrorModal: false });
          Utilities.logoutUser();
          return;
        }
        this.setState({ loggingOut: false, displayErrorModal: true });
        onLogoutError(`Unexpected status code: ${res.status}`);
      }
      if (res.ok && res.status === 204) {
        this.setState({ loggingOut: false, displayErrorModal: false });
        Utilities.logoutUser();
      }
    } catch (err) {
      console.log(err);
      const errorMessage =
        err instanceof Error ? err.message : "Something went wrong.";
      this.setState({ loggingOut: false, displayErrorModal: true });
      onLogoutError(errorMessage);
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  componentDidUpdate(lastProps: Props) {
    if (this.props.location?.pathname !== lastProps.location?.pathname) {
      this.setSelectedTab();
    }
  }

  componentDidMount() {
    this.setSelectedTab();
  }

  setSelectedTab = () => {
    let selectedTab = "";
    if (this.props.location?.pathname === "/gitops") {
      selectedTab = "gitops";
    } else if (this.props.location?.pathname.startsWith("/cluster")) {
      selectedTab = "cluster_management";
    } else if (this.props.location?.pathname.startsWith("/app")) {
      selectedTab = "dashboard";
    } else if (this.props.location?.pathname.startsWith("/snapshots")) {
      selectedTab = "snapshots";
    } else if (this.props.location?.pathname.startsWith("/access")) {
      selectedTab = "access";
    }
    this.setState({ selectedTab });
  };

  handleGoToGitOps = () => {
    if (this.props.location?.pathname !== "/gitops") {
      this.props.navigate("/gitops");
    }
  };

  handleGoToClusterManagement = () => {
    this.props.navigate("/cluster/manage");
  };

  handleAddNewApplication = () => {
    this.props.navigate("/upload-license");
  };

  handleGoToSnapshots = () => {
    this.props.navigate("/snapshots");
  };

  handleGoToAccess = () => {
    this.props.navigate("/access");
  };

  redirectToDashboard = () => {
    const { navigate, refetchAppsList } = this.props;
    refetchAppsList();
    navigate("/");
  };

  render() {
    const {
      className,
      fetchingMetadata,
      isKurlEnabled,
      isEmbeddedClusterEnabled,
      isEmbeddedClusterWaitingForNodes,
      isGitOpsSupported,
      isIdentityServiceSupported,
      appsList,
      logo,
      location,
      isSnapshotsSupported,
    } = this.props;
    const { selectedTab } = this.state;

    const pathname = location?.pathname.split("/") || "";
    let selectedApp;
    let appLogo;
    let licenseType;
    if (pathname.length > 2 && pathname[1] === "app") {
      selectedApp = appsList.find((app) => app.slug === pathname[2]);
      licenseType = selectedApp?.licenseType;
      appLogo =
        selectedApp?.downstream?.currentVersion?.appIconUri ||
        selectedApp?.iconUri;
    } else {
      appLogo = logo;
      licenseType = "";
    }

    let isInitialEmbeddedInstall = false;
    if (isEmbeddedClusterEnabled && appsList.length > 0) {
      isInitialEmbeddedInstall = Utilities.isInitialAppInstall(appsList[0]);
    }

    const isClusterScope =
      this.props.location?.pathname.includes("/clusterscope");
    return (
      <div
        className={classNames("NavBarWrapper", className, {
          "cluster-scope": isClusterScope,
        })}
      >
        <div className="flex flex-auto u-height--full">
          <div className="flex alignItems--center flex1 flex-verticalCenter u-position--relative">
            <div className="HeaderLogo">
              <Link to={isClusterScope ? "/clusterscope" : "/"} tabIndex={-1}>
                {appLogo ? (
                  <span
                    className="nav-logo clickable"
                    style={{ backgroundImage: `url(${appLogo})` }}
                  />
                ) : !fetchingMetadata ? (
                  <span className="logo icon clickable" />
                ) : (
                  <span style={{ width: "30px", height: "30px" }} />
                )}
                {licenseType === "community" && (
                  <span className="flag flex">
                    {" "}
                    <span className="flagText">Community Edition</span>{" "}
                  </span>
                )}
              </Link>
            </div>
          </div>
          {Utilities.isLoggedIn() &&
            appsList?.length > 0 &&
            !isInitialEmbeddedInstall &&
            !isEmbeddedClusterWaitingForNodes && (
              <div className="flex flex-auto left-items">
                <div
                  className={classNames("NavItem u-position--relative flex", {
                    "is-active": selectedTab === "dashboard",
                  })}
                >
                  <span
                    onClick={this.redirectToDashboard}
                    className="flex flex1 u-cursor--pointer text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center"
                  >
                    Application
                  </span>
                </div>
                {isGitOpsSupported && (
                  <div
                    className={classNames("NavItem u-position--relative flex", {
                      "is-active": selectedTab === "gitops",
                    })}
                  >
                    <span
                      onClick={this.handleGoToGitOps}
                      className="flex flex1 u-cursor--pointer text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center"
                    >
                      GitOps
                    </span>
                  </div>
                )}
                {(isKurlEnabled || isEmbeddedClusterEnabled) &&
                  location.pathname !==
                    `${selectedApp?.slug}/cluster/manage` && (
                    <div
                      className={classNames(
                        "NavItem u-position--relative flex",
                        {
                          "is-active": selectedTab === "cluster_management",
                        }
                      )}
                    >
                      <span
                        onClick={this.handleGoToClusterManagement}
                        className="flex flex1 u-cursor--pointer text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center"
                      >
                        Cluster Management
                      </span>
                    </div>
                  )}
                {isSnapshotsSupported && (
                  <div
                    className={classNames("NavItem u-position--relative flex", {
                      "is-active": selectedTab === "snapshots",
                    })}
                  >
                    <span
                      onClick={this.handleGoToSnapshots}
                      className="flex flex1 u-cursor--pointer alignItems--center text u-fontSize--normal u-fontWeight--medium flex"
                    >
                      {isEmbeddedClusterEnabled
                        ? "Disaster Recovery"
                        : "Snapshots"}
                    </span>
                  </div>
                )}
                {isIdentityServiceSupported && isKurlEnabled && (
                  <div
                    className={classNames("NavItem u-position--relative flex", {
                      "is-active": selectedTab === "access",
                    })}
                  >
                    <span
                      onClick={this.handleGoToAccess}
                      className="flex flex1 u-cursor--pointer alignItems--center text u-fontSize--normal u-fontWeight--medium flex"
                    >
                      Access
                    </span>
                  </div>
                )}
              </div>
            )}
        </div>
        {Utilities.isLoggedIn() && !isEmbeddedClusterWaitingForNodes && (
          <>
            <NavBarDropdown
              handleLogOut={this.handleLogOut}
              isEmbeddedCluster={isEmbeddedClusterEnabled}
            />
          </>
        )}
        {this.props.errLoggingOut && this.props.errLoggingOut.length > 0 && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={this.props.errLoggingOut}
            tryAgain={this.handleLogOut}
            err="Failed to log out"
            loading={this.state.loggingOut}
          />
        )}
      </div>
    );
  }
}

// @ts-ignore
export default withRouter(NavBar);
