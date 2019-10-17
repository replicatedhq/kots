import * as React from "react";
import Helmet from "react-helmet";
import Modal from "react-modal";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";

import { listSupportBundles } from "../../queries/TroubleshootQueries";

import AddClusterModal from "../shared/modals/AddClusterModal";
import Loader from "../shared/Loader";
import SupportBundleRow from "./SupportBundleRow";
import GenerateSupportBundle from "./GenerateSupportBundle";
import "../../scss/components/troubleshoot/SupportBundleList.scss";

class SupportBundleList extends React.Component {
  state = {
    addNewClusterModal: false
  }

  openClusterModal = () => {
    this.setState({ addNewClusterModal: true });
  }

  addClusterToWatch = (clusterId, githubPath) => {
    const { watch } = this.props;
    localStorage.setItem("clusterRedirect", `/watch/${watch.slug}/troubleshoot`);
    const upstreamUrl = `ship://ship-cloud/${watch.slug}`;
    this.props.history.push(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}`);
  }

  closeAddClusterModal = () => {
    this.setState({ addNewClusterModal: false });
  }

  createDownstreamForCluster = () => {
    const { watch } = this.props;
    localStorage.setItem("clusterRedirect", `/watch/${watch.slug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  render() {
    const { addNewClusterModal } = this.state;
    const { watch } = this.props;
    const { loading, error, listSupportBundles } = this.props.listSupportBundles;

    const appTitle = watch.watchName || watch.name;
    const downstreams =
      watch.watches ||
      watch.downstreams ||
      [];

    if (error) {
      return <p>{error.message}</p>;
    }

    if (loading) {
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <Loader size="60" color="#44bb66" />
        </div>
      );
    }

    let bundlesNode;
    if (downstreams.length) {
      if (listSupportBundles?.length) {
        bundlesNode = (
          listSupportBundles.map(bundle => (
            <SupportBundleRow
              key={bundle.id}
              appType={watch.watchName ? "watch" : "kots"}
              bundle={bundle}
              watchSlug={watch.slug}
            />
          ))
        );
      } else {
        return (
          <GenerateSupportBundle
            watch={watch}
          />
        );
      }
    } else {
      bundlesNode = (
        <div className="flex-column flex1 u-marginTop--15">
          <div className="EmptyState--wrapper flex-column flex1">
            <div className="EmptyState flex-column flex1 alignItems--center justifyContent--center">
              <div className="flex alignItems--center justifyContent--center">
                <span className="icon ship-complete-icon-gh"></span>
                <span className="deployment-or-text">OR</span>
                <span className="icon ship-medium-size"></span>
              </div>
              <div className="u-textAlign--center u-marginTop--10">
                <p className="u-fontSize--largest u-color--tuna u-lineHeight--medium u-fontWeight--bold u-marginBottom--10">Deploy to a cluster</p>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium"> To troubleshoot {watch.watchName} you should first deploy your application to a cluster.</p>
              </div>
              <div className="u-marginTop--20">
                <button className="btn secondary" onClick={this.openClusterModal}>Add a deployment cluster</button>
              </div>
            </div>
          </div>
        </div>
      )
    }

    return (
      <div className="container u-paddingBottom--30 u-paddingTop--30 flex1 flex">
        <Helmet>
          <title>{`${appTitle} Troubleshoot`}</title>
        </Helmet>
        <div className="flex1 flex-column">
          <div className="flex flex1">
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-paddingBottom--10 flex">
                <div className="flex flex1">
                  <div className="flex1 u-flexTabletReflow">
                    <div className="flex flex1">
                      <div className="flex-auto alignSelf--center">
                        <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna flex alignContent--center">Support bundles</h2>
                      </div>
                    </div>
                    <div className="RightNode flex-auto flex-column flex-verticalCenter u-position--relative">
                      <Link to={`${this.props.match.url}/generate`} className="btn secondary">Generate a support bundle</Link>
                    </div>
                  </div>
                </div>
              </div>
              <div className={`${downstreams.length ? "flex1 flex-column u-overflow--auto" : ""}`}>
                {bundlesNode}
              </div>
            </div>
          </div>
        </div>
        {addNewClusterModal &&
          <Modal
            isOpen={addNewClusterModal}
            onRequestClose={this.closeAddClusterModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="AddNewClusterModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {appTitle} to a new downstream</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing downstreams to deploy to.</p>
              <AddClusterModal
                onAddCluster={this.addClusterToWatch}
                onRequestClose={this.closeAddClusterModal}
                createDownstreamForCluster={this.createDownstreamForCluster}
                existingDeploymentClusters={[]}
              />
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  graphql(listSupportBundles, {
    name: "listSupportBundles",
    options: props => {
      return {
        variables: {
          watchSlug: props.watch.slug
        },
        fetchPolicy: "no-cache",
      }
    }
  })
  // graphql(archiveSupportBundle, {
  //   props: ({ mutate }) => ({
  //     archiveSupportBundle: (id) => mutate({ variables: { id } })
  //   })
  // }),
)(SupportBundleList));