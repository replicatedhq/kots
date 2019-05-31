import * as React from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import { getWatch, getWatchVersion } from "../../queries/WatchQueries";
import Loader from "../shared/Loader";
import { MonacoDiffEditor } from "react-monaco-editor";
import { Utilities } from "../../utilities/utilities";
import "../../scss/components/watches/DiffReleases.scss";

class DiffGitHubReleases extends React.Component {
  constructor() {
    super();
    this.state = {
      editorValueOne: "",
      editorValueTwo: "",
    }
  }

  getYamlForSequence = async (sequence) => {
    const response = await fetch(`${window.env.SHIPDOWNLOAD_ENDPOINT}/${this.props.getWatch.getWatch.id}/${sequence}`, {
      headers: new Headers({
        "Authorization": Utilities.getToken(),
      }),
    })
    const body = await response.text();
    return body;
  }


  async componentDidMount() {
    if (this.props.getWatch.getWatch) {
      const params = this.props.match.params;
      const sequencesToDiff = [params.firstSeqNumber, params.secondSeqNumber];
      sequencesToDiff.forEach((sequence) => {
        this.props.client.query({
          query: getWatchVersion,
          variables: { id: this.props.getWatch.getWatch.id, sequence: parseInt(sequence) }
        })
          .then((res) => {
            const result = res.data.getWatchVersion;
            this.setState({ [`diff-${sequence}`]: result })
          })
          .catch();
      })
    }
    if (this.props.getWatch.getWatch && this.props.getWatch.getWatch.id) {
      this.setState({
        editorValueOne: await this.getYamlForSequence(this.props.match.params.firstSeqNumber),
        editorValueTwo: await this.getYamlForSequence(this.props.match.params.secondSeqNumber)
      })
    }
  }

  async componentDidUpdate(lastProps) {
    const { getWatch } = this.props.getWatch;
    if (getWatch !== lastProps.getWatch.getWatch && getWatch) {
      const params = this.props.match.params;
      const sequencesToDiff = [params.firstSeqNumber, params.secondSeqNumber];
      sequencesToDiff.forEach((sequence) => {
        this.props.client.query({
          query: getWatchVersion,
          variables: { id: this.props.getWatch.getWatch.id, sequence: parseInt(sequence) }
        })
          .then((res) => {
            const result = res.data.getWatchVersion;
            this.setState({ [`diff-${sequence}`]: result })
          })
          .catch();
      })
      this.setState({
        editorValueOne: await this.getYamlForSequence(params.firstSeqNumber),
        editorValueTwo: await this.getYamlForSequence(params.secondSeqNumber)
      })
    }
  }

  render() {
    const { getWatch } = this.props;
    const { editorValueOne, editorValueTwo } = this.state;
    const { params } = this.props.match;
    
    if (!getWatch || getWatch.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    const firstDiff = this.state[`diff-${params.firstSeqNumber}`] || {};
    const secondDiff = this.state[`diff-${params.secondSeqNumber}`] || {};
    const gitPath = getWatch.getWatch.cluster && getWatch.getWatch.cluster.gitOpsRef ? `${getWatch.getWatch.cluster.gitOpsRef.owner}/${getWatch.getWatch.cluster.gitOpsRef.repo}/pull` : "";

    return (
      <div className="flex1 flex-column u-position--relative container">
        <div className="diff-header-wrapper u-marginTop--30">
          <p onClick={this.props.history.goBack} className="u-fontSize--normal u-fontWeight--medium u-color--chateauGreen u-textDecoration--underlineOnHover">
            <span className="icon clickable backArrow-icon u-marginRight--10"></span>Versions
          </p>
          <div className="diff-header u-marginTop--10">
            <div className="flex flex1 u-textAlign--center">
              <div className="flex-auto flex-column justifyContent--center">
                <span className="normal icon clusterType git"></span>
              </div>
              <div className="flex1 u-marginLeft--10 alignSelf--center">
                <h2 className="u-fontSize--header2 u-fontWeight--bold u-color--tundora flex alignContent--center alignItems--center">{getWatch.getWatch.watchName}</h2>
              </div>
            </div>
          </div>
          <div className=" u-marginTop--10">
            <span className="u-fontSize--large u-color--dustyGray u-fontWeight--medium">Diff of <span className="u-fontSize--large u-color--doveGray u-fontWeight--bold ">{firstDiff.title}</span> (
              <a href={`https://github.com/${gitPath}/${firstDiff.pullrequestNumber}`} target="_blank" rel="noopener noreferrer" className="replicated-link u-fontSize--meidum">#{firstDiff.pullrequestNumber}</a>) from <span className="u-fontSize--large u-color--doveGray u-fontWeight--bold">{secondDiff.title}</span> (
                <a href={`https://github.com/${gitPath}/${secondDiff.pullrequestNumber}`} target="_blank" rel="noopener noreferrer" className="replicated-link u-fontSize--meidum">#{secondDiff.pullrequestNumber}</a>
              )</span>
          </div>
        </div>
        <div className="MonacoEditor--wrapper flex1 flex u-height--full u-width--full u-marginTop--20">
          <div className="flex1 flex-column u-width--full u-overflow--hidden">
            <MonacoDiffEditor
              ref={(editor) => { this.monacoDiffEditor = editor }}
              width="100%"
              height="100%"
              language="yaml"
              original={editorValueOne}
              value={editorValueTwo}
              options={{
                renderSideBySide: true,
                enableSplitViewResizing: true,
                scrollBeyondLastLine: false,
                readOnly: true
              }}
            />
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(
    getWatch, {
      name: "getWatch",
      options: ({ match }) => ({
        variables: { slug: `${match.params.owner}/${match.params.slug}` }
      })
    }
  )
)(DiffGitHubReleases);


