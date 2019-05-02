import * as React from "react";
import PropTypes from "prop-types";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom";
import { createUnforkSession } from "../mutations/WatchMutations";
import { validateUpstreamURL } from "../queries/GitHubQueries";
import ShipLoading from "./ShipLoading";
import { Utilities } from "../utilities/utilities";

const ShipUnforkError = ({ handleTryAgain }) => (
  <div className="Form flex-column flex1 alignItems--center justifyContent--center">
    <span className="icon u-superWarning--large"></span>
    <h2 className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginTop--20 u-marginBottom--10">We were unable to unfork your application</h2>
    <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30 u-textAlign--center">
      We weren't able to unfork this application. We're aware that not all applications can be unforked currently and are working on solutions. <br />
      In the meantime, you can either try again or prepare your application for deployment with <span className="u-fontWeight--medium u-color--chateauGreen">Ship Init</span>.
    </p>
    <div className="flex">
      <button className="btn secondary u-marginRight--10" onClick={handleTryAgain}>Try again</button>
      <Link className="btn primary" to="/watch/create/init">Use Ship Init</Link>
    </div>
  </div>
)

export class ShipUnfork extends React.Component {
  static propTypes = {
    onActiveInitSession: PropTypes.func.isRequired,
  };

  state = {
    url: "",
    forkUrlError: false,
    upstreamUrlError: false,
    fork: "",
    saving: false,
    fetchError: false,
  };

  componentDidMount() {
    const { search } = this.props.location;
    const searchParams = new URLSearchParams(search);
    const upstream = searchParams.get("upstream");
    this.setState({ url: upstream || "" });
  }

  onUrlChange(value) {
    this.setState({ url: value });
  }

  onForkChange(value) {
    this.setState({ fork: value });
  }

  onShipInitUrlSubmitted = async() => {
    const { onActiveInitSession } = this.props;
    const { url, fork } = this.state;

    if(url === "" || fork === "") {
      this.setState({
        forkUrlError: fork === "",
        upstreamUrlError: url === ""
      });
    } else {
      this.setState({
        saving: true,
        url,
        fork
      });
  
      const validUpstream = await this.validateUpstream(url).catch(this.handleInvalidUpstream);
  
      if (validUpstream) {
        this.props.createUnforkSession(url, fork)
          .then(({ data }) => {
            if(!window.location.pathname.includes("/watch/create/unfork")) {return;} // Prevent redirect if component is no longer mounted
            const { createUnforkSession } = data;
            const { id: initSessionId, result } = createUnforkSession;
            if(result === "error unforking application") {
              this.handleInvalidUpstream();
            } else {
              onActiveInitSession(initSessionId);
              this.props.history.push("/ship/update")
            }
          }).catch(this.handleInvalidUpstream);
      } else {
        this.handleInvalidUpstream();
      }
    }
  }

  validateUpstream = (url) => (
    this.props.client.query({
      query: validateUpstreamURL,
      variables: { upstream: url },
    })
  )

  handleInvalidUpstream = () => this.setState({ saving: false, fetchError: true })

  handleTryAgain = () => {
    this.setState({ fetchError: false });
  }

  render() {
    const { url, fetchError, forkUrlError, upstreamUrlError } = this.state;
    const n = url.lastIndexOf("/");
    const appString = url.substring(n + 1);

    if (this.state.saving) {
      return (
        <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
          <ShipLoading headerText={`Unforking ${Utilities.toTitleCase(appString) || ""}`} subText="We're moving as fast as we can but it may take a moment." />
        </div>
      );
    }

    if(fetchError) {
      return (
        <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
          <ShipUnforkError handleTryAgain={this.handleTryAgain} />
        </div>
      )
    }

    return (
      <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
        <div className="Form flex-column flex1 alignItems--center justifyContent--center">
          <div className="init-unfork-wrapper flex-auto">
            <div className="flex1 flex-column u-textAlign--center">
              <div className="u-marginBottom--10"><span className="icon u-betaBadge u-marginRight--10"></span></div>
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal">What is the URL of your forked application?</p>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">This could be a URL to any Kubernetes application or Helm chart</p>
              {(upstreamUrlError || forkUrlError) &&
                <p className="u-fontSize--small u-color--chestnut u-marginBottom--5 u-paddingBottom--20">Please enter valid upstream and fork urls to continue</p>
              }
              <div className="flex flex1 alignItems--flexEnd">
                <div className="flex-column flex1 u-marginRight--10 alignItems--flexStart">
                  <label className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Upstream URL</label>
                  <input defaultValue={this.state.url} value={this.state.url} onChange={(e) => { this.onUrlChange(e.target.value) }} type="text" className="Input jumbo flex1" placeholder="https://github.com/helm/charts/stable/grafana" />
                </div>
                <div className="flex-column flex1 alignItems--flexStart">
                  <label className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Fork URL</label>
                  <input defaultValue={this.state.fork} value={this.state.fork} onChange={(e) => { this.onForkChange(e.target.value) }} type="text" className="Input jumbo flex1" placeholder="https://github.com/username/fork" />
                </div>
                <div className="flex-auto u-marginLeft--10">
                  <button onClick={this.onShipInitUrlSubmitted} className="btn primary large">Ship Unfork</button>
                </div>
              </div>
            </div>
            <div className="u-marginTop--30 u-textAlign--center">
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--30">If you don't have a fork, <Link to="/watch/create/init" className="replicated-link">create a new watch here</Link></p>
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
  graphql(createUnforkSession, {
    props: ({ mutate }) => ({
      createUnforkSession: (upstreamUri, forkUri) => mutate({ variables: { upstreamUri, forkUri }})
    })
  }),
)(ShipUnfork);
