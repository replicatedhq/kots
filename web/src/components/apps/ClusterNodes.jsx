import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import CodeSnippet from "../shared/CodeSnippet";
import NodeRow from "./NodeRow";
import Loader from "../shared/Loader";
import { kurl } from "../../queries/KurlQueries";
import { drainNode, deleteNode, generateWorkerAddNodeCommand } from "../../mutations/KurlMutations"

export class ClusterNodes extends React.Component {
  state = {
    generating: false,
    command: "",
    expiry: null,
  }

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

  generateWorkerAddNodeCommand = (ev) => {
    ev.preventDefault();
    this.setState({ generating: true, command: "", expiry: null });

    this.props.generateWorkerAddNodeCommand()
      .then((resp) => {
        const data = resp.data.generateWorkerAddNodeCommand;
        this.setState({ generating: false, command: data.command, expiry: data.expiry });
      })
      .catch((error) => {
        this.setState({ generating: false });
        console.log(error);
      });
  }

  generateMasterAddNodeCommand = (ev) => {
    ev.preventDefault();
    this.setState({ generating: false, command: "", expiry: null });
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
        <Helmet>
          <title>{`${this.props.appName ? `${this.props.appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="flex-column flex1 alignItems--center">
          <div className="flex1 flex-column centered-container">
            <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal">Install your node</p>
            <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">
              Generate node join command:
              <span> </span><a href="#" disabled={this.state.generating} onClick={this.generateWorkerAddNodeCommand}>worker</a>
              <span> </span><a href="#" disabled={this.state.generating} onClick={this.generateMasterAddNodeCommand}>master</a>
            </p>
            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--5">Run the curl command below to get the install script for your new node.</p>
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
            >
              { !this.state.generating && this.state.command && this.state.command.length > 0 ?
                this.state.command.join("\n  ") :
                "" }
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
  }),
  graphql(generateWorkerAddNodeCommand, {
    props: ({ mutate }) => ({
      generateWorkerAddNodeCommand: () => mutate()
    })
  })
)(ClusterNodes);
