import * as React from "react";
import { compose, graphql, withApollo } from "react-apollo";
import { listClusters } from "../../../queries/ClusterQueries";
import Select from "react-select";

class AddNewClusterModal extends React.Component {

  state = {
    selectedCluster: {
      value: "",
      label: "Select a cluster"
    },
    githubPath: ""
  };

  onSubmit = () => {
    this.props.onAddCluster(this.state.selectedCluster.value, this.state.githubPath);
  }

  handleEnterPress = (e) => {
    if (e.charCode === 13) {
      this.onSubmit();
    }
  }

  onClusterChange = (selectedOption) => {
    this.setState({ selectedCluster: selectedOption });
  }

  handleCancel = () => {
    if (this.props.onRequestClose && typeof this.props.onRequestClose === "function") {
      this.props.onRequestClose();
    }
  }

  render() {
    const { existingDeploymentClusters } = this.props;
    let options = [];
    if (this.props.listClustersQuery && this.props.listClustersQuery.listClusters) {
      options = this.props.listClustersQuery.listClusters.filter((cluster) => !existingDeploymentClusters.includes(cluster.id)).map((cluster) => {
        return ({
          value: cluster.id,
          label: cluster.title,
          type: cluster.gitOpsRef ? "git" : "ship"
        })
      });
    }
    const buttonDisabled = this.state.selectedCluster.value === "" || (this.state.selectedCluster.type === "git" && this.state.githubPath === "");
    return (
      <div className="flex flex1">
        <div className="flex-column">
          <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora">Deployment clusters</p>
          <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">Select a cluster you would like to add for deployments.</p>
          <div className="u-marginTop--10">
            <Select
              className="replicated-select-container"
              classNamePrefix="replicated-select"
              options={options}
              getOptionLabel={(selectedCluster) => selectedCluster.label}
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
      </div>
    )
  }
}

export default compose(
  withApollo,
  graphql(listClusters, { 
    name: "listClustersQuery",
    options: {
      fetchPolicy: "network-only"
    }
  })
)(AddNewClusterModal);