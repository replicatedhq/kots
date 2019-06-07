import React, { PureComponent } from "react";
import classNames from "classnames";
import { Link, withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";

import { Utilities } from "@src/utilities/utilities";
import { userFeatures } from "@src/queries/WatchQueries";
import { listWatches } from "@src/queries/WatchQueries";
import { userInfo } from "@src/queries/UserQueries";
import { logout } from "@src/mutations/GitHubMutations";
import Avatar from "../shared/Avatar";

import "@src/scss/components/shared/NavBar.scss";

export class NavBar extends PureComponent {
  constructor() {
    super();
    this.state = {}
  }

  handleLogOut = async (e) => {
    e.preventDefault();
    await this.props.logout()
      .catch((err) => {
        console.log(err);
      })
    Utilities.logoutUser();
  }

  componentDidUpdate() {
    if (Utilities.isLoggedIn() && !this.state.user) {
      this.props.client.query({ query: userInfo })
        .then((res) => {
          this.setState({ user: res.data.userInfo });
        }).catch();
    }
  }

  componentDidMount() {
    if (Utilities.isLoggedIn()) {
      this.props.client.query({ query: userInfo })
        .then((res) => {
          this.setState({ user: res.data.userInfo });
        }).catch();
    }
  }

  handleGoToWatches = () => {
    if (this.props.location.pathname === "/watches") {
      this.props.client.query({
        query: listWatches,
        fetchPolicy: "network-only",
      });
    } else {
      this.props.history.push("/watches");
    }
  }

  handleGoToClusters = () => {
    if (this.props.location.pathname === "/clusters") {
      this.props.client.query({
        query: listWatches,
        fetchPolicy: "network-only",
      });
    } else {
      this.props.history.push("/clusters");
    }
  }

  render() {
    const { className, logo } = this.props;
    const { user } = this.state;

    const isClusterScope = this.props.location.pathname.includes("/clusterscope");
    return (
      <div className={classNames("NavBarWrapper flex flex-auto", className, {
        "cluster-scope": isClusterScope
      })}>
        <div className="container flex flex1">
          <div className="flex1 justifyContent--flexStart">
            <div className="flex1 flex u-height--full">
              <div className="flex flex-auto">
                <div className="HeaderLogo-wrapper flex alignItems--center flex1 flex-verticalCenter u-position--relative">
                  <div className="HeaderLogo">
                    <Link to={isClusterScope ? "/clusterscope" : "/"} tabIndex="-1">
                      {logo
                        ? <img className="watch-logo clickable" src={logo} />
                        : <span className="logo icon clickable" />
                      }
                      <span className="text u-color--tuna flex-column justifyContent--center">
                        <span>
                          {isClusterScope
                            ? "ClusterScope"
                            : "Replicated Ship"
                          }
                        </span>
                      </span>
                    </Link>
                  </div>
                </div>
                {Utilities.isLoggedIn() && (
                  <div className="flex flex-auto left-items">
                    <div className="NavItem u-position--relative flex">
                      <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToWatches}>
                        <span className="text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center">
                          <span>Clusters</span>
                        </span>
                      </span>
                    </div>
                    <div className="NavItem u-position--relative flex ${clustersEnabled">
                      <span className="HeaderLink flex flex1 u-cursor--pointer" onClick={this.handleGoToClusters}>
                        <span className="text u-fontSize--normal u-fontWeight--medium flex-column justifyContent--center">
                          <span>Team</span>
                        </span>
                      </span>
                    </div>
                  </div>
                  )
                }
              </div>
              {this.props.location.pathname === "/coming-soon" ?
                <div className="flex flex1 justifyContent--flexEnd right-items">
                  <div className="flex-column flex-auto justifyContent--center">
                    <p className="NavItem" onClick={this.handleLogOut}>Log out</p>
                  </div>
                </div>
                : null}
              {Utilities.isLoggedIn() ?
                <div className="flex flex1 justifyContent--flexEnd right-items">
                  <div className="flex-column flex-auto u-marginRight--5 justifyContent--center">
                    <Link className="NavBar-add-app u-color--chateauGreen u-fontSize--normal u-fontWeight--bold u-marginRight--10" to="/watch/create/init">
                      Add a new application
                    </Link>
                  </div>
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
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(logout, {
    props: ({ mutate }) => ({
      logout: () => mutate()
    })
  }),
  graphql(userFeatures, {
    name: "userFeaturesQuery",
    skip: !Utilities.isLoggedIn()
  })
)(NavBar);
