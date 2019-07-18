import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Link } from "react-router-dom";
import { pullRequestHistory } from "../../queries/WatchQueries";
import { Utilities } from "../../utilities/utilities";

class GitHubIntegration extends React.Component {
  constructor() {
    super();
    this.state = {
      currentVersion: "",
      countVersions: 0,
      prNumber: "",
      prUri: ""
    }
  }

  componentDidMount() {
    if (this.props.id && !this.props.pending) {
      this.props.client.query({
        query: pullRequestHistory,
        variables: { notificationId: this.props.id },
        fetchPolicy: "network-only"
      })
        .then((res) => {
          const result = res.data.pullRequestHistory;
          const deployed = result.filter(({ status }) => status === "deployed")
          const pending = result.filter(({ status }) =>  status === "pending");
          if (deployed.length !== 0) {
            const current = deployed[0].title.slice(8);

            const index = result.findIndex(obj => obj.status === "deployed")
            this.setState({ currentVersion: current === "" ? `---` : current, countVersions: index, currentPr: deployed[0].number, prUri: deployed[0].uri });
          }
          if (pending.length !== 0 && result[0].status === "pending") {
            this.setState({ pendingPr: pending[0].number, pendingUri: pending[0].uri });
          }
        })
        .catch();
    }
  }


  render() {
    const { currentVersion, countVersions, currentPr, prUri, pendingPr, pendingUri } = this.state;
    const { id, org, repo, branch, rootPath, enabled, pending, handleEnableToggle, slug, handleManage, toggleDeleteIntegrationModal } = this.props;

    return (
      <div className="integration flex-column flex1 flex">
        <div className="flex">
          <span className={`${pending ? "disabled" : "normal"} u-marginRight--5 icon integration-card-icon-github`}></span>
          <div className="flex1 justifyContent--center">
            <p className={`path u-fontWeight--bold u-fontSize--large ${pending ? "u-color--dustyGray" : "u-color--tundora"}`}>{`${org}/${repo}/${branch}${Utilities.trimLeadingSlash(rootPath)}`}</p>
          </div>
        </div>
        <div>
          {!pending && <p className="u-fontSize--small u-color--dustyGray u-marginBottom--10 u-fontWeight--medium u-marginTop--15">Current version</p>}
          <div className="flex flex1">
            {currentVersion ?
              <h2 className="u-fontSize--jumbo2 alignSelf--center u-fontWeight--bold u-color--tuna">{currentVersion} </h2>
              : pending ? 
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginTop--20">Your integration will be enabled after you <br/><a href={`https://github.com/${org}/${repo}`} target="_blank" rel="noreferrer noopener" className="u-color--astral u-fontWeight--medium">merge the PR</a> created by Ship.</p> :
                <h2 className="u-fontSize--jumbo2 alignSelf--center u-fontWeight--bold u-color--tuna">---</h2>
            }
            {currentVersion && countVersions === 1 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">One version behind</p>
              </div>
            }
            {currentVersion && countVersions >= 2 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">Two or more versions behind</p>
              </div>
            }
            {currentVersion && countVersions === 0 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon checkmark-icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">Up to date</p>
              </div>
            }
            {currentVersion && <a href={`${prUri}`} target="_blank" rel="noopener noreferrer" className="u-color--astral u-fontWeight--medium u-fontSize--large alignSelf--center"> #{currentPr}</a>}
          </div>
          {!pending && <Link to={`/watch/${slug}/${org}/${repo}/${branch}${Utilities.trimLeadingSlash(rootPath)}`} className="u-color--astral u-fontWeight--medium u-fontSize--normal u-lineHeight--normal">See version history</Link>}
        </div>
        <div className="flex justifyContent--spaceBetween alignItems--center  u-marginTop--20">
          <div className="flex">
            <div className={`flex flex-auto Checkbox--switch ${enabled === 1 ? "is-checked" : ""}`}>
              <input
                type="checkbox"
                className="Checkbox-toggle flex-auto"
                name={id}
                checked={enabled}
                disabled={pending}
                onChange={(e) => { handleEnableToggle(e, enabled) }} />
            </div>
            <label htmlFor={id} className={`flex1 flex-column flex-verticalCenter ${pending ? "u-color--dustyGray" : "u-color--tundora"} u-marginLeft--5 u-fontSize--normal u-fontWeight--medium u-cursor--pointer`}>Enable Integration</label>
          </div>
          {pendingPr && <a href={pendingUri} className="btn green secondary" target="_blank" rel="noopener noreferrer">Review PR</a>}
        </div>
        <div className="flex flex1 alignItems--flexEnd">
          <div className="flex u-marginTop--20 u-borderTop--gray u-width--full">
            <div className={`flex1 flex card-action-wrapper u-cursor--pointer`}>
              <span className="flex1 u-marginRight--5 u-color--red card-action u-fontSize--small u-fontWeight--medium u-textAlign--center" onClick={() => { toggleDeleteIntegrationModal(id, `${org}/${repo}/${branch}${Utilities.trimLeadingSlash(rootPath)}`, pending) }}>Delete integration</span>
            </div>
            {pending ? null : 
              <div className="flex1 flex card-action-wrapper u-cursor--pointer">
                <span onClick={(e) => handleManage(e, { id, org, repo, branch, rootPath })} className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center">Manage integration</span>
              </div>
            }
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
)(GitHubIntegration);


