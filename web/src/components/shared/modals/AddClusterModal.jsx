import * as React from "react";
import { withRouter } from "react-router-dom";
import { compose, graphql, withApollo } from "react-apollo";
import { listClusters } from "../../../queries/ClusterQueries";
import Select from "react-select";
import Loader from "../../shared/Loader";

const NEW_DOWNSTREAM = "Create a new downstream cluster";

class AddNewClusterModal extends React.Component {

  state = {
    selectedCluster: {
      value: "",
      label: "Select a cluster"
    },
    githubPath: ""
  };

  onSubmit = () => {
    const { watch, history } = this.props;
    const { selectedCluster, githubPath } = this.state;

    if (watch?.hasPreflight) {
      history.push(`/preflight/${watch.slug}/${selectedCluster.slug}${githubPath
        ? `?path=${encodeURI(githubPath)}`
        : ""
      }`);
    } else {
      this.props.onAddCluster(selectedCluster.value, githubPath);
    }

  }

  handleEnterPress = (e) => {
    if (e.charCode === 13) {
      this.onSubmit();
    }
  }

  onClusterChange = (selectedOption) => {
    if (selectedOption.label === NEW_DOWNSTREAM) {
      return this.handleCreateDownstream();
    }
    this.setState({ selectedCluster: selectedOption });
  }

  handleCancel = () => {
    if (this.props.onRequestClose && typeof this.props.onRequestClose === "function") {
      this.props.onRequestClose();
    }
  }

  handleCreateDownstream = () => {
    if (this.props.createDownstreamForCluster && typeof this.props.createDownstreamForCluster === "function") {
      this.props.createDownstreamForCluster();
    }
  }

  renderIcons = (type) => {
    if (type === "ship") {
      return <span className="icon clusterType ship"></span>
    } else if (type === "git") {
      return <span className="icon clusterType git"></span>
    } else {
      return;
    }
  }

  getLabel = ({ type, label }) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "0.5em" }}>{this.renderIcons(type)}</span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  }

  render() {
    const {
      existingDeploymentClusters,
      listClustersQuery
    } = this.props;

    let options = [];
    if (listClustersQuery?.listClusters) {
      options = this.props.listClustersQuery.listClusters.filter((cluster) => !existingDeploymentClusters.includes(cluster.id)).map((cluster) => {
        return ({
          value: cluster.id,
          label: cluster.title,
          slug: cluster.slug,
          type: cluster.gitOpsRef ? "git" : "ship"
        })
      });
      options.unshift({ label: NEW_DOWNSTREAM, id: "new-downstream" });
    }
    const buttonDisabled = this.state.selectedCluster.value === "" || (this.state.selectedCluster.type === "git" && this.state.githubPath === "");
    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    let modalContent = null;
    if (options?.length) {
      modalContent = (
        <div>
          <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora">Downstream deployment clusters</p>
          <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">Select a downstream you would like to add for deployments.</p>
          <div className="u-marginTop--10">
            <Select
              className="replicated-select-container"
              classNamePrefix="replicated-select"
              options={options}
              getOptionLabel={this.getLabel}
              getOptionValue={(option) => option.label}
              value={this.state.selectedCluster}
              onChange={this.onClusterChange}
              isOptionSelected={(option) => {option.value === this.state.selectedCluster.value}}
            />
          </div>
        {this.state.selectedCluster.type === "git" ?
          <div className="u-marginTop--10">
            <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">What is the root path for this cluster</p>
            <input type="text" className="Input" placeholder="/my-path" onKeyPress={(e => { this.handleEnterPress(e) })} defaultValue={this.state.githubPath} onChange={(e) => { this.setState({ githubPath: e.target.value }); }}/>
          </div>
          : null}
        <div className="u-marginTop--10 u-paddingTop--5 flex">
          <button onClick={this.handleCancel} className="btn secondary u-marginRight--10">Cancel</button>
          <button disabled={buttonDisabled} onClick={this.onSubmit} className="btn green primary">Add cluster</button>
        </div>
      </div>
      );
    } else {
      modalContent = (
        <div>
          <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora">Downstream deployment clusters</p>
          <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">You have no downstreams available to deploy to.</p>
          <div className="u-marginTop--10">
            <button type="button" className="btn primary" onClick={this.handleCreateDownstream}>Create a new downstream cluster</button>
          </div>
        </div>
      );
    }

    return (
      <div className="flex flex1">
        <div className="flex-column flex1">
          {listClustersQuery.loading ?
            centeredLoader
          :
            modalContent
          }
        </div>
      </div>
    )
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(listClusters, {
    name: "listClustersQuery",
    options: {
      fetchPolicy: "network-only"
    }
  })
)(AddNewClusterModal);