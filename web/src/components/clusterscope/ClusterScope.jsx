import React, { Suspense, lazy } from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { uploadImageWatchBatch } from "../../mutations/ImageWatchMutations";

const ClusterScopeBatchCreate = lazy(() => import("./ClusterScopeBatchCreate"));
const ClusterScopeBatch = lazy(() => import("./ClusterScopeBatch"));

import "../../scss/components/state/StateFileViewer.scss";

class ClusterScope extends React.Component {
  constructor() {
    super();
  }

  componentDidMount() {
    document.title = "ClusterScope - Discover outdated containers in your Kubernetes cluster"
  }

  render() {
    return (
      <div className="WatchDetailPage--wrapper flex-column flex1">
        <div className="flex-column flex1 HelmValues--wrapper">
          <Suspense fallback={<div className="flex-column flex1 alignItems--center justifyContent--center"><Loader size="60" color="#44bb66" /></div>}>
            <Switch>
              <Route exact path="/clusterscope" render={() =>
                <ClusterScopeBatchCreate
                  history={this.props.history}
                  uploadImageWatchBatch={this.props.uploadImageWatchBatch} />
              }/>
              <Route exact path="/clusterscope/:batchId" render={() =>
                <ClusterScopeBatch
                  getImageWatch={this.props.getImageWatch} />
              }/>
            </Switch>
          </Suspense>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(uploadImageWatchBatch, {
    props: ({ mutate }) => ({
      uploadImageWatchBatch: (imageList) => mutate({ variables: { imageList } })
    })
  }),
)(ClusterScope);
