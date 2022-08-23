import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";
import { Link, withRouter } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import ErrorModal from "../modals/ErrorModal";
import NavBarDropdown from "./NavBarDropdown";

import "@src/scss/components/shared/NavBar.scss";
import styled from "styled-components";

const StyledThemeButton = styled.button`
  background: ${(props) => props.theme.colors.primary};
`;

export class NavBar extends PureComponent {
  constructor(props) {
    super(props);

    this.state = {
      selectedTab: "",
      loggingOut: false,
      displayErrorModal: false,
    };
  }

  static propTypes = {
    refetchAppsList: PropTypes.func.isRequired,
    history: PropTypes.object.isRequired,
  };

  handleLogOut = async (e) => {
    const { onLogoutError } = this.props;
    e.preventDefault();
    try {
      this.setState({ loggingOut: true, displayErrorModal: false });
      const res = await fetch(`${process.env.API_ENDPOINT}/logout`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
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
      this.setState({ loggingOut: false, displayErrorModal: true });
      onLogoutError(
        err ? err.message : "Something went wrong, please try again."
      );
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  componentDidUpdate(lastProps) {
    const { pathname } = this.props.location;
    if (pathname !== lastProps.location.pathname) {
      this.setSelectedTab();
    }
  }

  componentDidMount() {
    this.setSelectedTab();
  }

  setSelectedTab = () => {
    const { pathname } = this.props.location;
    let selectedTab = "";
    if (pathname === "/gitops") {
      selectedTab = "gitops";
    } else if (pathname === "/cluster/manage") {
      selectedTab = "cluster_management";
    } else if (pathname.startsWith("/app")) {
      selectedTab = "dashboard";
    } else if (pathname.startsWith("/snapshots")) {
      selectedTab = "snapshots";
    } else if (pathname.startsWith("/access")) {
      selectedTab = "access";
    }
    this.setState({ selectedTab });
  };

  handleGoToGitOps = () => {
    if (this.props.location.pathname !== "/gitops") {
      this.props.history.push("/gitops");
    }
  };

  handleGoToClusterManagement = () => {
    this.props.history.push("/cluster/manage");
  };

  handleAddNewApplication = () => {
    this.props.history.push("/upload-license");
  };

  handleGoToSnapshots = () => {
    this.props.history.push("/snapshots");
  };

  handleGoToAccess = () => {
    this.props.history.push("/access");
  };

  redirectToDashboard = () => {
    const { refetchAppsList, history } = this.props;
    refetchAppsList().then(() => {
      history.push("/");
    });
  };

  render() {
    const {
      className,
      fetchingMetadata,
      isKurlEnabled,
      isGitOpsSupported,
      isIdentityServiceSupported,
      appsList,
      logo,
      location,
      isSnapshotsSupported,
    } = this.props;
    const { selectedTab } = this.state;

    const pathname = location.pathname.split("/");
    let selectedApp;
    let appLogo;
    let licenseType;
    if (pathname.length > 2 && pathname[1] === "app") {
      selectedApp = appsList.find((app) => app.slug === pathname[2]);
      appLogo = selectedApp?.iconUri;
      licenseType = selectedApp?.licenseType;
    } else {
      appLogo = logo;
      licenseType = "";
    }

    const isClusterScope =
      this.props.location.pathname.includes("/clusterscope");
    return (
      <div
        className={classNames("NavBarWrapper", className, {
          "cluster-scope": isClusterScope,
        })}
      >
        <StyledThemeButton>Hello!!</StyledThemeButton>
        <div className="flex flex-auto u-height--full">
          <div className="flex alignItems--center flex1 flex-verticalCenter u-position--relative">
            <div className="HeaderLogo">
              <Link to={isClusterScope ? "/clusterscope" : "/"} tabIndex="-1">
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
          {Utilities.isLoggedIn() && appsList.length > 0 && (
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
              {isKurlEnabled && (
                <div
                  className={classNames("NavItem u-position--relative flex", {
                    "is-active": selectedTab === "cluster_management",
                  })}
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
                    Snapshots
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
        {Utilities.isLoggedIn() && (
          <>
            <NavBarDropdown
              handleLogOut={this.handleLogOut}
              isHelmManaged={this.props.isHelmManaged}
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

export default withRouter(NavBar);
