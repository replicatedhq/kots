import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import MonacoEditor from "react-monaco-editor"; 
import CodeSnippet from "./shared/CodeSnippet";
import { getPreflightCommand } from "@src/queries/AppsQueries";
import "../scss/components/PreflightCheckPage.scss";

class PreflightResultErrors extends Component {
  state = {
    showErrorDetails: false,
  };

  toggleShowErrorDetails = () => {
    this.setState({
      showErrorDetails: !this.state.showErrorDetails
    })
  }

  render() {
    const { valueFromAPI, logo } = this.props;
  
    return (
      <div className="flex flex1 flex-column">
        <div className="flex flex1 u-height--full u-width--full u-marginTop--5 u-marginBottom--20">
          <div className="flex-column u-width--full u-overflow--hidden u-paddingTop--30 u-paddingBottom--5 alignItems--center justifyContent--center">
            <div className="PreChecksBox-wrapper flex-column u-padding--20">
              <div className="flex">
                {logo &&
                  <div className="flex-auto u-marginRight--10">
                    <div className="watch-icon" style={{ backgroundImage: `url(${logo})`, width: "36px", height: "36px" }}></div>
                  </div>
                }
                <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Unable to automatically run preflight checks</h2>
              </div>
              <p className="u-marginTop--10 u-marginBottom--10 u-fontSize--normal u-lineHeight--normal u-color--dustyGray u-fontWeight--normal">
                The Kubernetes RBAC policy that the Admin Console is running with does not have access to complete the Preflight Checks. It’s recommended that you run these manually before proceeding.
              </p>
              <p className="replicated-link u-fontSize--normal u-marginBottom--10" onClick={this.toggleShowErrorDetails}>{this.state.showErrorDetails ? "Hide details" : "Show details"}</p>
              {this.state.showErrorDetails &&
                <div className="flex-column flex flex1 monaco-editor-wrapper u-border--gray">
                  <MonacoEditor
                    language="bash"
                    value={valueFromAPI}
                    height="300"
                    width="100%"
                    options={{
                      readOnly: true,
                      contextmenu: false,
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
                </div>
              }
              <div className="u-marginTop--20">
                <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Run Preflight Checks Manually</h2>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Run the commands below from your workstation to complete the Preflight Checks.</p>
                <CodeSnippet
                  language="bash"
                  canCopy={true}
                  onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                >
                  {this.props.getPreflightCommand.getPreflightCommand || ""}
                </CodeSnippet>
              </div>
              <div className="u-marginTop--30 flex justifyContent--flexEnd">
                <span className="replicated-link u-fontSize--normal" onClick={this.props.retryResults}>Try again</span>
                <span className="replicated-link u-marginLeft--20 u-fontSize--normal" onClick={this.props.ignorePermissionErrors}>
                  Proceed with limited Preflights
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(getPreflightCommand, {
    name: "getPreflightCommand",
    options: props => {
      const { match, preflightResultData } = props
      const sequence = match.params.sequence ? parseInt(match.params.sequence, 10) : 0;
      return {
        variables: {
          appSlug: preflightResultData.appSlug,
          clusterSlug: preflightResultData.clusterSlug,
          sequence,
        }
      }
    }
  }),
)(PreflightResultErrors);
