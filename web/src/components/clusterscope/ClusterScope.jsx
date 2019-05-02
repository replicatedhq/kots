import * as React from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import ClusterScopeBatchCreate from "./ClusterScopeBatchCreate";
import ClusterScopeBatch from "./ClusterScopeBatch";
import { uploadImageWatchBatch } from "../../mutations/ImageWatchMutations";

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
