import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";
import { Link, withRouter } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import Avatar from "../shared/Avatar";
import ErrorModal from "../modals/ErrorModal";

import "@src/scss/components/shared/NavBar.scss";
import toJson from "enzyme-to-json";

export class NavBar extends PureComponent {
  constructor(props) {
    super(props);

    this.state = {
      selectedTab: "",
      loggingOut: false,
      displayErrorModal: false
    };
  }

  static propTypes = {
    refetchAppsList: PropTypes.func.isRequired,
    history: PropTypes.object.isRequired
  }

  handleLogOut = async (e) => {
    const { onLogoutError } = this.props;
    e.preventDefault();
    try {
      this.setState({ loggingOut: true, displayErrorModal: false })
      const res = await fetch(`${window.env.API_ENDPOINT}/logout`, {
        headers: {
          "Authorization": Utilities.getToken(),
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
      console.log(err)
      this.setState({ loggingOut: false, displayErrorModal: true });
      onLogoutError(err ? err.message : "Something went wrong, please try again.")
    }
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

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
      selectedTab = "dashboard"
    } else if (pathname.startsWith("/snapshots")) {
      selectedTab = "snapshots";
    } else if (pathname.startsWith("/access")) {
      selectedTab = "access";
    } else {
      for(var extension in this.props.extensions){
        if (pathname.startsWith("/"+extension.name)) {
          selectedTab = extension.name
        }
      }
    }
    this.setState({ selectedTab });
  }

  handleGoToGitOps = () => {
    if (this.props.location.pathname !== "/gitops") {
      this.props.history.push("/gitops");
    }
  }

  handleGoToClusterManagement = () => {
    this.props.history.push("/cluster/manage");
  }

  handleAddNewApplication = () => {
    this.props.history.push("/upload-license");
  }

  handleGoToSnapshots = () => {
    this.props.history.push("/snapshots");
  }

  handleGoToAccess = () => {
    this.props.history.push("/access");
  }

  handleGoToExtension(name) {
    this.props.history.push("/" + name);
  }

  redirectToDashboard = () => {
    const { refetchAppsList, history } = this.props;
    refetchAppsList().then(() => {
      history.push("/");
    });
  }

  render() {
    const { className, fetchingMetadata, isKurlEnabled, isGitOpsSupported, isIdentityServiceSupported, appsList, logo, location, isSnapshotsSupported, extensions } = this.props;
    const { user, selectedTab } = this.state;

    const pathname = location.pathname.split("/");
    let selectedApp;
    let appLogo;
    let licenseType;
    if (pathname.length > 2 && pathname[1] === "app") {
      selectedApp = appsList.find(app => app.slug === pathname[2]);
      appLogo = selectedApp?.iconUri;
      licenseType = selectedApp?.licenseType;
    } else {
      appLogo = logo;
      licenseType = "";
    }

    const isClusterScope = this.props.location.pathname.includes("/clusterscope");
    return (
      <div className={classNames("NavBarWrapper flex flex-auto", className, {
        "cluster-scope": isClusterScope
      })}>
        <div className="container flex flex1">
          <div className="flex1 justifyContent--flexStart">
            <div className="flex1 flex u-height--full">
              <div className="flex flex-auto">
                <div className="flex alignItems--center flex1 flex-verticalCenter u-position--relative u-marginRight--20">
                  <div className="HeaderLogo">
                    <Link to={isClusterScope ? "/clusterscope" : "/"} tabIndex="-1">
                      {appLogo
                        ? <span className="nav-logo clickable" style={{ backgroundImage: `url(${appLogo})` }} />
                        : !fetchingMetadata ? <span className="logo icon clickable" />
                          : <span style={{ width: "30px", height: "30px" }} />
                      }
                      {licenseType === "community" && <span className="flag flex"> <span className="flagText">Community Edition</span> </span>}
                    </Link>
                  </div>
                </div>
                {Utilities.isLoggedIn() && appsList.length > 0 && (
                  <div className="flex flex-auto left-items">
                    <div className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === "dashboard" })}>
                      <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.redirectToDashboard}>
                        <span className="text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center">
                          <span>Application</span>
                        </span>
                      </span>
                    </div>
                    {isGitOpsSupported &&
                      <div className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === "gitops" })}>
                        <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToGitOps}>
                          <span className="text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center">
                            <span>GitOps</span>
                          </span>
                        </span>
                      </div>
                    }
                    {isKurlEnabled &&
                      <div className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === "cluster_management" })}>
                        <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToClusterManagement}>
                          <span className="text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center">
                            <span>Cluster Management</span>
                          </span>
                        </span>
                      </div>
                    }
                    {isSnapshotsSupported &&
                      <div className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === "snapshots" })}>
                        <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToSnapshots}>
                          <div className="flex flex1 alignItems--center">
                            <span className="text u-fontSize--normal u-fontWeight--medium flex"> Snapshots </span>
                          </div>
                        </span>
                      </div>
                    }
                    {isIdentityServiceSupported && isKurlEnabled &&
                      <div className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === "access" })}>
                        <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToAccess}>
                          <div className="flex flex1 alignItems--center">
                            <span className="text u-fontSize--normal u-fontWeight--medium flex"> Access </span>
                          </div>
                        </span>
                      </div>
                    }
                    {this.props.extensions?.map((extension) => (
                      <div key={extension.name} className={classNames("NavItem u-position--relative flex", { "is-active": selectedTab === extension.name } )}>
                        <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={() => this.handleGoToExtension(extension.name)}>
                          <div className="flex flex1 alignItems--center">
                            <span className="text u-fontSize--normal u-fontWeight--medium flex"> {extension.name} </span>
                          </div>
                        </span>
                      </div>
                    ))}
                    
                  </div>
                )}
              </div>
              {Utilities.isLoggedIn() ?
                <div className="flex flex1 justifyContent--flexEnd right-items">
                  {pathname[1] === "upload-license" || pathname[2] === "airgap" ?
                    null :
                    <div className="flex-column flex-auto u-marginRight--20 justifyContent--center">
                      <Link className="btn secondary blue rounded" to="/upload-license">
                        Add a new application
                    </Link>
                    </div>}
                  <div className="flex-column flex-auto justifyContent--center">
                    <p data-qa="Navbar--logOutButton" className="NavItem" onClick={this.handleLogOut}>Log out</p>
                  </div>
                  {user && user.avatarUrl !== "" ?
                    <div className="flex-column flex-auto justifyContent--center u-marginLeft--10">
                      <Avatar imageUrl={this.state.user && this.state.user.avatarUrl} />
                    </div>
                    : null}
                </div>
                : null}
            </div>
          </div>
        </div>
        {this.props.errLoggingOut && this.props.errLoggingOut.length > 0 &&
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={this.props.errLoggingOut}
            tryAgain={this.handleLogOut}
            err="Failed to log out"
            loading={this.state.loggingOut}
          />}
      </div>
    );
  }
}

export default withRouter(NavBar);
