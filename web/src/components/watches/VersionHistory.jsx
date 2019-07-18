import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getWatch } from "../../queries/WatchQueries";
import { deployWatchVersion } from "../../mutations/WatchMutations";
import { Link } from "react-router-dom";
import GitHubVersionCard from "./GitHubVersionCard";
import ShipVersionCard from "./ShipVersionCard";
import Loader from "../shared/Loader";
import "../../scss/components/watches/VersionCard.scss";

class VersionHistory extends React.Component {
  constructor() {
    super();
    this.state = {
      selected: false,
      checkedCards: [],
      versionHistory: []
    }
  }

  onReleasesToDiff = () => {
    this.setState({ selected: true });
  }

  onCancel = () => {
    this.setState({ selected: false });
  }

  onCardChecked = (sequence, isChecked) => {
    if (isChecked) {
      this.setState({
        checkedCards: [{ sequence, isChecked }].concat(this.state.checkedCards).slice(0, 2)
      })
    } else {
      this.setState({
        checkedCards: this.state.checkedCards.filter(card => card.sequence !== sequence)
      })
    }
  }

  makeCurrentRelease = async (sequence) => {
    const { getWatch } = this.props.getWatch;
    await this.props.deployWatchVersion(getWatch.id, sequence).then(() => {
      this.props.getWatch.refetch();
    })
  }

  componentDidMount() {
    const { getWatch } = this.props.getWatch;
    if (getWatch) {
      const pending = getWatch.pendingVersions;
      const current = getWatch.currentVersion ? [getWatch.currentVersion] : [];
      const past = getWatch.pastVersions;
      const versionHistory = pending.concat(current, past);
      this.setState({ versionHistory });
    }
  }

  componentDidUpdate(lastProps) {
    const { getWatch } = this.props.getWatch;
    if (getWatch !== lastProps.getWatch.getWatch && getWatch) {
      const pending = getWatch.pendingVersions;
      const current = getWatch.currentVersion ? [getWatch.currentVersion] : [];
      const past = getWatch.pastVersions;
      const versionHistory = pending.concat(current, past);
      this.setState({ versionHistory });
    }
  }

  render() {
    const { selected, checkedCards, versionHistory } = this.state;
    const { getWatch } = this.props;

    if (!getWatch || getWatch.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    const watch = getWatch.getWatch;
    const gitOpsRef = watch.cluster && watch.cluster.gitOpsRef;
    const historyTitle = watch.cluster ? watch.cluster.title : watch.watchName;

    if (versionHistory) {
      let firstSeqNumber, secondSeqNumber;
      if (checkedCards.length === 2) {
        firstSeqNumber = checkedCards[1].sequence;
        secondSeqNumber = checkedCards[0].sequence;
        // check which one is oldest
        const first = versionHistory.find((result) => result.sequence === firstSeqNumber);
        const second = versionHistory.find((result) => result.sequence === secondSeqNumber);
        const firstDate = first.createdOn
        const secondDate = second.createdOn;
        if (firstDate > secondDate) {
          firstSeqNumber = checkedCards[0].sequence;
          secondSeqNumber = checkedCards[1].sequence;
        } else {
          firstSeqNumber = checkedCards[1].sequence;
          secondSeqNumber = checkedCards[0].sequence;
        }
      }

      return (
        <div className="flex1 flex-column u-position--relative verison-history-wrapper u-overflow--auto">
          <div className="version-card-header-wrapper">
            <p onClick={this.props.history.goBack} className="u-fontSize--normal u-fontWeight--medium u-color--chateauGreen u-textDecoration--underlineOnHover">
              <span className="icon clickable backArrow-icon u-marginRight--10"></span>Watches
            </p>
            <div className="version-card-header flex-column flex1 u-marginTop--20">
              <div className="flex flex1">
                <div className="flex1 u-marginLeft--10">
                  <div className="flex flex-column">
                    <p className="u-fontSize--header2 u-fontWeight--bold u-color--tundora flex alignContent--center alignItems--center">
                      {watch.cluster ?
                        <span className={`icon u-marginRight--10 clusterType ${gitOpsRef ? "git" : "ship"}`}></span>
                      :
                        <div className="avatar-wrapper u-marginRight--10">
                          <span style={{ backgroundImage: `url(${watch.watchIcon})` }}></span>
                        </div>
                      }
                    {historyTitle}</p>
                    <span className="u-fontSize--large u-color--dustyGray u-fontWeight--medium u-marginTop--10">Version history for {gitOpsRef ? `${gitOpsRef.owner}/${gitOpsRef.repo}/${gitOpsRef.branch}${gitOpsRef.path}` : "your cluster"}</span>
                  </div>
                </div>
              </div>
            </div>
            <div className="u-marginTop--30 u-paddingLeft--10 u-marginBottom--10 u-fontSize--header u-fontWeight--bold u-color--tundora">{watch.currentVersion && watch.currentVersion.title}</div>
            <div className="icon-wrapper flex-auto flex flex1 justifyContent--spaceBetween u-paddingLeft--10 u-marginTop--10">
              <div className="flex1 flex">
                <div className="icon checkmark-icon flex-auto u-marginRight--10 u-marginTop--5"></div>
                <span className="flex1 flex-column flex-verticalCenter u-fontSize--normal u-color--dustyGray u-fontWeight--medium">Most recent version</span>
              </div>
              {!selected ?
                <div className="flex1 flex justifyContent--flexEnd">
                  <span className="u-fontSize--normal u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover alignSelf--center" onClick={() => this.onReleasesToDiff()}>Select releases to diff</span>
                </div>
                :
                <div className="flex1 flex justifyContent--flexEnd">
                  <span className="u-fontSize--normal u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover alignSelf--center u-marginRight--20" onClick={() => this.onCancel()}>Cancel</span>
                  {checkedCards.length === 2 ?
                    gitOpsRef ?
                      <Link to={`${this.props.match.url}/compare/${gitOpsRef.owner}/${gitOpsRef.repo}/${gitOpsRef.branch}${gitOpsRef.path}/${firstSeqNumber}/${secondSeqNumber}`} className="btn primary alignSelf--center">Diff releases</Link>
                    :
                      <Link to={`${this.props.match.url}/compare/${firstSeqNumber}/${secondSeqNumber}`} className="btn primary alignSelf--center">Diff releases</Link>
                    :
                    <Link to={`${this.props.match.url}/compare/${firstSeqNumber}/${secondSeqNumber}`} className="btn primary disabled-link alignSelf--center">Diff releases</Link>
                  }
                </div>
              }
            </div>
          </div>
          <div className="verison-card-wrapper">
            {versionHistory && versionHistory.map((gi, i) => {
              let card = null;
              if (gitOpsRef) {
                card = (
                  <GitHubVersionCard
                    key={`${gi.sequence}-${i}`}
                    {...gi}
                    selected={this.state.selected}
                    isPending={watch.pendingVersions.find(version => version.sequence === gi.sequence)}
                    versionHistory={gi}
                    pullRequestRootUrl={`${gitOpsRef.owner}/${gitOpsRef.repo}`}
                    onCardChecked={(sequence, isChecked) => this.onCardChecked(sequence, isChecked)}
                    isChecked={!!this.state.checkedCards.find(card => card.sequence === gi.sequence)}
                  />
                )
              } else {
                card = (
                  <ShipVersionCard
                    key={`${gi.sequence}-${i}`}
                    {...gi}
                    selected={this.state.selected}
                    makeCurrentVersion={this.makeCurrentRelease}
                    versionHistory={gi}
                    isPending={watch.pendingVersions.find(version => version.sequence === gi.sequence)}
                    onCardChecked={(sequence, isChecked) => this.onCardChecked(sequence, isChecked)}
                    isChecked={!!this.state.checkedCards.find(card => card.sequence === gi.sequence)}
                  />
                )
              }
              return card;
            }) }
          </div>
        </div>
      );
    } else {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <p>No version history was found</p>
        </div>
      )
    }
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(
    getWatch, {
      name: "getWatch",
      fetchPolicy: "no-cache",
      options: ({ match }) => ({
        variables: { slug: `${match.params.owner}/${match.params.slug}` }
      })
    }
  ),
  graphql(deployWatchVersion, {
    props: ({ mutate }) => ({
      deployWatchVersion: (watchId, sequence) => mutate({ variables: { watchId, sequence } })
    })
  })
)(VersionHistory);
