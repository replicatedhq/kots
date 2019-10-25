import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import CodeSnippet from "../shared/CodeSnippet";
import NodeRow from "./NodeRow";
import Loader from "../shared/Loader";
import { kurl } from "../../queries/KurlQueries";
import { drainNode, deleteNode } from "../../mutations/KurlMutations"

export class ClusterNodes extends React.Component {

  drainNode = (name) => {
    try {
      this.props.drainNode(name);
      // feedback here showing the node was drained?
    } catch (error) {
      console.log(error)
    }
  }

  deleteNode = (name) => {
    try {
      this.props.deleteNode(name);
      // refecth nodes so deleted node is from the list?
    } catch (error) {
      console.log(error);
    }
  }

  render() {
    const { kurl, loading } = this.props.data;

    if (loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
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
              {kurl?.addNodeCommand || ""}
            </CodeSnippet>
            <div className="u-marginTop--40 u-paddingBottom--30">
              <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Your nodes</p>
              <div className="flex1 u-overflow--auto">
                {kurl?.nodes && kurl?.nodes.map((node, i) => (
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
  graphql(kurl, {
    options: {
      pollInterval: 2000,
      fetchPolicy: "no-cache",
    },
  }),
  graphql(drainNode, {
    props: ({ mutate }) => ({
      drainNode: (name) => mutate({ variables: { name } })
    })
  }),
  graphql(deleteNode, {
    props: ({ mutate }) => ({
      deleteNode: (name) => mutate({ variables: { name } })
    })
  })
)(ClusterNodes);
