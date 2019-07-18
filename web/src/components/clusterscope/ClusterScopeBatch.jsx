import * as React from "react";
import truncateMiddle from "truncate-middle";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { getImageWatch } from "../../queries/ImageWatchQueries";
import { Utilities } from "../../utilities/utilities";
import ContentHeader from "../shared/ContentHeader";
import ClusterScopeBatchPath from "./ClusterScopeBatchPath";
import "../../scss/components/image_check/ImageWatchBatch.scss";
import Modal from "react-modal";
import reverse from "lodash/reverse";
import sortBy from "lodash/sortBy";
import map from "lodash/map";
import some from "lodash/some";

class ClusterScopeBatch extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      loggedIn: Utilities.isLoggedIn(),
      displaySignUpModal: false,
      imageWatches: []
    }
  }

  componentDidMount() {
    const { loggedIn } = this.state;
    const showSignUpModal = sessionStorage.getItem("showSignUpModal");
    if(!loggedIn && showSignUpModal) {
      setTimeout(this.toggleSignUpModal, 30000); // 30 seconds to modal pop
      sessionStorage.removeItem("showSignUpModal");
    }
  }

  componentDidUpdate(lastProps) {
    if(this.props.data.imageWatches !== lastProps.data.imageWatches) {
      this.setState({ imageWatches: this.props.data.imageWatches });
      if(!some(this.props.data.imageWatches, { lastCheckedOn: null })) {this.props.data.stopPolling();}
    }
  }

  toggleSignUpModal = () => {
    const { displaySignUpModal } = this.state;
    this.setState({ displaySignUpModal: !displaySignUpModal });
  }

  handleLogIn = () => {
    this.props.history.push("/auth/github");
  }

  render() {
    const { displaySignUpModal } = this.state;
    const items = reverse(sortBy(this.state.imageWatches, ["versionsBehind"]));
    const rows = map(sortBy(items, ["isPrivate"]), (item) => {
      const warningClass = item.versionsBehind > 0 && item.versionsBehind <= 9 ? "warning" : "";
      const superWarningClass = item.versionsBehind >= 10 ? "super-warning" : "";
      const upToDateClass = item.versionsBehind === 0 ? "up-to-date" : "";
      const isPrivateClass = item.isPrivate ? "private-warning" : "";
      const path = item.path ? JSON.parse(item.path) : [];
      const pending = item.lastCheckedOn === null;

      if(pending) {
        return (
          <div key={item.id} className="flex cluster-scope-row pending alignItems--center">
            <div className="left flex1 flex-column">
              <div className="loading-wrapper">
                <div className="flex">
                  <div className="loading-bar flex1"></div>
                  <span className="versions-behind flex-auto pending"><span className="icon warn-icon"></span>Checking image</span>
                </div>
                <div className="loading-bar"></div>
              </div>
            </div>
            <div className="flex1 flex justifyContent--flexEnd">
              <ClusterScopeBatchPath path={[]} loading={true} />
            </div>
          </div>
        )
      }

      if (item.isPrivate) {
        return (
          <div key={item.id} className="flex cluster-scope-row private-image">
            <div className="left flex1 flex-column">
              <div className="flex">
                <p className="u-fontSize--larger u-lineHeight--normal u-fontWeight--bold u-color--dustyGray" title={item.name}>{truncateMiddle(item.name, 30, 40, "...")}</p>
                <span className={`versions-behind flex-auto ${isPrivateClass}`}><span className="icon warn-icon"></span>Unknown</span>
              </div>
              <p className="u-marginTop--10 u-fontSize--normal u-fontWeight--medium u-lineHeight--normal">This is a private image so we can't read when it was last updated.</p>
            </div>
            <div className="flex1 flex-column"></div>
          </div>
        );
      }

      return (
        <div key={item.id} className="flex cluster-scope-row alignItems--center">
          <div className="left flex1 flex-column">
            <div className="flex">
              <p className="u-fontSize--larger u-lineHeight--normal u-fontWeight--bold u-color--tuna" title={item.name}>{truncateMiddle(item.name, 30, 40, "...")}</p>
              <span className={`versions-behind flex-auto ${upToDateClass} ${warningClass} ${superWarningClass}`}><span className="icon warn-icon"></span>{item.versionsBehind  === 0 ? "Up to date" : `${item.versionsBehind} versions behind`}</span>
            </div>
            { item.versionsBehind === 0 ?
              <p className="u-marginTop--10 u-fontSize--normal u-fontWeight--medium u-lineHeight--normal">
                Use <Link className="u-color--astral u-fontWeight--medium u-cursor--pointer" to="/watch/create/init">Replicated Ship</Link> to receive a pull request when this image has a new version available.
              </p>
              :
              <p className="u-marginTop--10 u-fontSize--normal u-fontWeight--medium u-lineHeight--normal">
                Use <Link className="u-color--astral u-fontWeight--medium u-cursor--pointer" to="/watch/create/init">Replicated Ship</Link> to receive a pull request when this image has a new version available.
              </p>
            }
          </div>
          <div className="flex1 flex justifyContent--flexEnd">
            { path.length >=2 ?
              <ClusterScopeBatchPath path={path} />
              :
              <div className="flex flex-column">
                <p className="u-fontSize--large u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">Nice work, <span className="u-fontWeight--bold u-color--tundora" title={item.name}>{truncateMiddle(item.name, 30, 40, "...")}</span> is up to date</p>
              </div> }
          </div>
        </div>
      );
    });

    return (
      <div className="ClusterScoperBatch--wrapper flex-column flex1">
        <ContentHeader title="Images running in your Kubernetes cluster" />
        <div className="flex1 u-overflow--auto">
          {rows}
        </div>
        <Modal
          isOpen={displaySignUpModal}
          onRequestClose={this.toggleSignUpModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Sign Up Modal"
          ariaHideApp={false}
          className="ClusterScopeSignUpModal--wrapper Modal DefaultSize"
        >
          <div className="LoginBox-wrapper image-batch u-textAlign--center">
            <span className="icon ship-login-icon"></span>
            <p className="u-lineHeight--normal u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginBottom--30">Ready to update your images? <br /><span className="u-fontWeight--medium u-color--dustyGray">Connect your GitHub account to get started using Replicated Ship</span></p>
            <button type="button" className="btn auth github" onClick={this.handleLogIn}>
              <span className="icon clickable github-button-icon"></span> Login with Octo
            </button>
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(getImageWatch, {
    options: ({ match }) => ({
      variables: { batchId: match.params.batchId },
      pollInterval: 500
    })
  }),
)(ClusterScopeBatch);
