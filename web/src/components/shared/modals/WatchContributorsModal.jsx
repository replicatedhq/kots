import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Modal from "react-modal";
import Loader from "../Loader";
import { WatchContributorCheckbox } from "./WatchContributorCheckbox";
import { getWatchContributors } from "../../../queries/WatchQueries";
import { githubUserOrgs, getOrgMembers } from "../../../queries/GitHubQueries";
import { userInfo } from "../../../queries/UserQueries";
import map from "lodash/map";
import { addWatchContributor, removeWatchContributor } from "../../../mutations/WatchMutations";
import Select from "react-select";
import keyBy from "lodash/keyBy";
import merge from "lodash/merge";
import omitBy from "lodash/omitBy";
import get from "lodash/get";
import isEmpty from "lodash/isEmpty";

import "../../../scss/components/watches/WatchContributorsModal.scss";

export class WatchContributorsModal extends React.Component {
  constructor() {
    super();
    this.state = {
      org: null,
      orgs: [],
      orgsPage: 1,
      loadingMembers: false,
      contributors: {},
      activeContributors: {},

      orgMembersPage: 1,
      nextPageMembers: [],
    };
  }

  setInitialState = (contributors) => {
    const contributorsForState = contributors.map((obj) => {
      return {
        ...obj,
        login: obj.login.toLowerCase(),
        isActive: true
      };
    })
    this.setState({
      contributors: keyBy(contributorsForState, "login"),
      activeContributors: keyBy(contributorsForState, "login")
    })
  }

  componentDidMount() {
    const { getGithubUserOrgs, getWatchContributorsQuery } = this.props;
    if (getWatchContributorsQuery && getWatchContributorsQuery.watchContributors) {
      const { watchContributors } = getWatchContributorsQuery;
      this.setInitialState(watchContributors);
    }
    if (getGithubUserOrgs && getGithubUserOrgs !== getGithubUserOrgs.installationOrganizations) {
      if (getGithubUserOrgs.installationOrganizations) {
        const filteredOrgs = getGithubUserOrgs.installationOrganizations.installations.filter((org) => this.props.getUserInfo.userInfo.login !== org.login)
        this.setState({ orgs: filteredOrgs });
      }
    }
  }

  componentDidUpdate(lastProps) {
    const { getGithubUserOrgs, getWatchContributorsQuery } = this.props;
    if (getWatchContributorsQuery !== lastProps.getWatchContributorsQuery && getWatchContributorsQuery.watchContributors) {
      const { watchContributors } = getWatchContributorsQuery;
      this.setInitialState(watchContributors);
    }
    if (getGithubUserOrgs !== lastProps.getGithubUserOrgs && getGithubUserOrgs.installationOrganizations) {
      if (getGithubUserOrgs.installationOrganizations) {
        const filteredOrgs = getGithubUserOrgs.installationOrganizations.installations.filter((org) => this.props.getUserInfo.userInfo.login !== org.login)
        this.setState({ orgs: filteredOrgs });
      }
    }
  }

  getActiveContributorsArr() {
    const { activeContributors } = this.state;
    const contributorsArr = Object.keys(activeContributors).map((key) => {
      const { githubId, avatar_url, login } = activeContributors[key];
      return {
        githubId,
        avatar_url,
        login
      };
    });
    return contributorsArr;
  }

  updateActiveContributors() {
    const { activeContributors, contributors } = this.state;
    const _nextActive = omitBy(merge(activeContributors, contributors), { isActive: false });
    this.setState({ activeContributors: _nextActive });
  }

  makeStateContributors = (orgMembers) => {
    const { activeContributors } = this.state;
    const contributorsForState = orgMembers.map(obj => {
      const { login, avatar_url, id } = obj;
      const _login = login.toLowerCase();
      return {
        login: login.toLowerCase(),
        avatar_url,
        githubId: id,
        isActive: !!activeContributors[_login]
      };
    });
    return keyBy(contributorsForState, "login");
  }

  handleCheckboxChange = (field, e) => {
    const { checked } = e.target;
    const { contributors } = this.state;
    let nextContributors = contributors
    nextContributors[field].isActive = checked;

    this.setState({ contributors: nextContributors });
    this.updateActiveContributors();
  }

  onOrgChange = async(org) => {
    const { login: orgName } = org;
    const firstPage = 1;
    const secondPage = 2;

    if (orgName !== "") {
      const firstPageOrgMembers = await this.fetchOrgMembers(orgName, firstPage);
      const contributors = this.makeStateContributors(firstPageOrgMembers);
      const secondPageOrgMembers = await this.fetchOrgMembers(orgName, secondPage);
      this.setState({
        contributors,
        org,
        nextPageMembers: secondPageOrgMembers,
        orgMembersPage: secondPage,
      });
    } else {
      // Set state to active contributors on empty org
      this.setState({ contributors: this.state.activeContributors });
    }
  }

  fetchOrgMembers = (orgName, pageToFetch) => {
    this.setState({ loadingMembers: true });
    return this.props.client.query({
      query: getOrgMembers,
      variables: { org: orgName, page: pageToFetch },
    }).then(({ data }) => {
      this.setState({ loadingMembers: false });
      return data.orgMembers;
    }).catch((err) => {
      this.setState({ loadingMembers: false });
      console.log(err);
    });
  }

  onSaveContributors = async () => {
    const { id } = this.props.watchBeingEdited;

    const desiredContributors = this.getActiveContributorsArr();
    const desiredContributorGithubIds = map(desiredContributors, "githubId");

    const currentContributors = this.props.getWatchContributorsQuery.watchContributors;
    const currentContributorGithubIds = map(currentContributors, "githubId");

    const addContributors = [];
    const removeContributors = [];

    for (const desiredContributor of desiredContributors) {
      if (currentContributorGithubIds.indexOf(desiredContributor.githubId) === -1) {
        addContributors.push(desiredContributor);
      }
    }

    for (const currentContributor of currentContributors) {
      if (desiredContributorGithubIds.indexOf(currentContributor.githubId) === -1) {
        removeContributors.push(currentContributor);
      }
    }

    this.setState({ savingContributors: true });

    try {
      for (const addContributor of addContributors) {
        await this.props.client.mutate({
          mutation: addWatchContributor,
          variables: {
            watchId: id,
            githubId: addContributor.githubId,
            login: addContributor.login,
            avatarUrl: addContributor.avatar_url,
          }
        });
      }

      for (const removeContributor of removeContributors) {
        await this.props.client.mutate({
          mutation: removeWatchContributor,
          variables: {
            watchId: id,
            contributorId: removeContributor.id,
          },
        });
      }
    } catch (err) {
      console.log(err);
    } finally {
      this.setState({ savingContributors: false });
    }
  }

  handleMenuScrollToBottomOrgs = async() => {
    const { orgs, orgsPage } = this.state;
    const { getGithubUserOrgs } = this.props;

    // subtract one for yourself
    if (orgs.length < getGithubUserOrgs.installationOrganizations.totalCount - 1) {
      const newOrgsPage = orgsPage + 1;
      const { data } = await this.props.client.query({
        query: githubUserOrgs,
        variables: { page: newOrgsPage },
      });
      this.setState({
        orgsPage: newOrgsPage,
        orgs: [...orgs, ...data.installationOrganizations.installations],
      });
    }
  }

  goNextOrgMemberPage = async () => {
    const { contributors, org, orgMembersPage, nextPageMembers } = this.state;
    const { login: orgName } = org;

    const newContributors = this.makeStateContributors(nextPageMembers);
    const nextOrgMembersPage = await this.fetchOrgMembers(orgName, orgMembersPage + 1);
    this.setState({
      nextPageMembers: nextOrgMembersPage,
      orgMembersPage: orgMembersPage + 1,
      contributors: {
        ...contributors,
        ...newContributors,
      },
    });
  }

  handleModalClose = () => {
    this.props.toggleContributorsModal();
  }

  render() {
    const {
      displayContributorsModal,
      toggleContributorsModal,
      watchBeingEdited,
      getUserInfo,
      getWatchContributorsQuery
    } = this.props;
    const {
      org,
      orgs,
      contributors,
      loadingMembers,
      savingContributors,
      showSaved,
      nextPageMembers,
    } = this.state;
    const loading = getUserInfo && this.props.getUserInfo.loading || getWatchContributorsQuery && getWatchContributorsQuery.loading;

    if (loading) {
      return null;
    }

    return (
      <Modal
        isOpen={displayContributorsModal}
        onRequestClose={toggleContributorsModal}
        shouldReturnFocusAfterClose={false}
        contentLabel="Modal"
        ariaHideApp={false}
        className="WatchContributorsModal--wrapper Modal SmallSize">
        <div className="Modal-body flex flex-column flex1 u-overflow--auto">
          <h2 className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--10">Edit contributors for {watchBeingEdited.name}</h2>
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">Select the users that are allowed to make changes to this application. If you don't see who your looking for, head over to GitHub and add the desired user(s) to your org and they will be visible here.</p>
          <div className="Form">
            <div className="flex flex1 u-position--relative u-marginBottom--20">
              <div>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Organization</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Select the organization you want to add users from.</p>
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  options={orgs}
                  getOptionLabel={(org) => org.login}
                  onMenuScrollToBottom={this.handleMenuScrollToBottomOrgs}
                  placeholder="Please select an organization"
                  onChange={this.onOrgChange}
                  value={org}
                  isOptionSelected={(option) => {option.login === get(org, "login")}}
                />
              </div>
              <div className="flex1"></div>
            </div>
            <div>
              <p className="u-fontWeight--medium u-fontSize--small">Organization Members</p>
              <div className="flex flex-column u-borderTop--gray u-marginTop--10 u-position--relative">
                <div className="contributer-wrapper">
                  {!isEmpty(contributors) && Object.keys(contributors).map((key, i) =>
                    <WatchContributorCheckbox
                      key={i}
                      item={contributors[key]}
                      contributors={contributors}
                      handleCheckboxChange={(field, e) => this.handleCheckboxChange(field, e)}
                      githubLogin={getUserInfo && getUserInfo.userInfo.username.toLowerCase()}
                    />
                  )}
                </div>
                <div className="more-contributors flex justifyContent--center u-paddingBottom--10 u-borderTop--gray">
                  { nextPageMembers.length > 0 ?
                    <button className="btn secondary green u-marginTop--20" disabled={savingContributors} onClick={this.goNextOrgMemberPage}>Load more</button>
                    : null
                  }
                </div>
                {loadingMembers &&
                  <div className="flex flex1 alignItems--center justifyContent--center contributors-loading">
                    <Loader size="50" />
                  </div>
                }
              </div>
            </div>
            <div className="flex flex1 justifyContent--flexEnd alignItems--center u-marginTop--20">
              { showSaved && <p className="u-fontSize--small u-color--chateauGreen u-marginRight--20 u-fontWeight--medium">Contributors have been updated</p> }
              <button onClick={this.handleModalClose} className="btn secondary u-marginRight--10">Close</button>
              <button className="btn primary" disabled={savingContributors} onClick={this.onSaveContributors}>{savingContributors ? "Saving" : "Save contributors"}</button>
            </div>
          </div>
        </div>
      </Modal>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(getWatchContributors, {
    name: "getWatchContributorsQuery",
    options: ({ watchBeingEdited }) => ({
      variables: { id: watchBeingEdited.id }
    })
  }),
  graphql(githubUserOrgs, {
    name: "getGithubUserOrgs",
    options: () => ({
      variables: { page: 1 },
    })
  }),
  graphql(userInfo, {
    name: "getUserInfo"
  }),
)(WatchContributorsModal);
