import * as React from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import CodeSnippet from "../shared/CodeSnippet";
import NodeRow from "./NodeRow";
import Loader from "../shared/Loader";

export class ClusterNodes extends React.Component {

  state = {
    loadingNodes: false,
    installCommand: `curl https://kurl.sh/sentry-pro/join.sh | sudo bash -s kubernetes-master-address=10.240.0.127 kubeadm-token=4u3q2l.y3gc68u9lp29noon kubeadm-token-ca-hash=sha256:338cce35a55a369b70d7df58768e95057bb99fbce49f673a6806e5d2ebc0a050 kubernetes-version=1.15.3  docker-registry-ip=10.110.176.226`,
    nodes: [
      {
        id: "23389asfkji289asf",
        hostname: "ip-10-0-0-15",
        status: "Connected",
        version: "v0.3.2",
        cores: 1,
        ram: 8
      },
      {
        id: "544545assfasdsd",
        hostname: "ip-10-0-0-14",
        status: "Disconnected",
        version: "v0.2.6",
        cores: 4,
        ram: 8
      },
    ],
  };

  drainNode = (nodeId) => {
    console.log(`drain node ${nodeId}`);
  }

  deleteNode = (nodeId) => {
    console.log(`delete node ${nodeId}`);
  }

  render() {
    const { loadingNodes, nodes } = this.state;

    if (loadingNodes) {
      return (
        <div className="container flex-column flex1 u-overflow--auto">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--50">
        <div className="flex-column flex1 alignItems--center">
          <div className="flex1 flex-column centered-container">
            <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal">Install your node</p>
            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--5">Run the curl command below to get the install script for your new node.</p>
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
            >
              {this.state.installCommand}
            </CodeSnippet>
            <div className="u-marginTop--40 u-paddingBottom--30">
              <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Your nodes</p>
              <div className="flex1 u-overflow--auto">
                {nodes && nodes.map((node, i) => (
                  <NodeRow key={i} node={node} drainNode={this.drainNode} deleteNode={this.deleteNode} />
                ))}
              </div>
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
)(ClusterNodes);
