import * as React from "react";
import PropTypes from "prop-types";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom";
import { createInitSession } from "../mutations/WatchMutations";
import { validateUpstreamURL } from "../queries/GitHubQueries";
import ShipLoading from "./ShipLoading";
import { Utilities } from "../utilities/utilities";
import Modal from "react-modal";
import "../scss/components/Login.scss";

export class ShipInitPre extends React.Component {
  static propTypes = {
    onActiveInitSession: PropTypes.func.isRequired,
  };

  state = {
    url: "",
    urlError: false,
    saving: false,
    fetchError: false
  };

  componentDidMount() {
    window.addEventListener("keydown", this.initOnEnterKey);
    const { search } = this.props.location;
    const searchParams = new URLSearchParams(search);
    const upstream = searchParams.get("upstream");
    const pendingInitId = searchParams.get("pendingInitId");
    const licenseId = searchParams.get("license_id");
    const clusterId = searchParams.get("cluster_id");
    const githubPath = searchParams.get("path");
    const autoStart = searchParams.get("start");

    const url = `${upstream ? upstream : ""}${licenseId ? `?license_id=${licenseId}`: ""}`;
    this.setState({
      pendingInitId: pendingInitId || "",
      url: url || "",
      clusterId: clusterId || "",
      githubPath: githubPath || ""
    });
    if (autoStart === "1") {
      this.onShipInitUrlSubmitted(url);
    }
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.initOnEnterKey);
  }

  componentDidUpdate(lastProps, lastState) {
    const { search } = this.props.location;
    const searchParams = new URLSearchParams(search);
    const autoStart = searchParams.get("start");
    if (this.state.clusterId !== lastState.clusterId && this.state.clusterId.length) {
      if (autoStart === "1") {
        this.onShipInitUrlSubmitted(this.state.url);
      }
    }
  }

  onUrlChange(value) {
    this.setState({ url: value });
  }

  onLicenseIdChange(value) {
    const { url } = this.state;
    const arr = url.split("?");
    const newUrl = value === "" ? arr[0] : `${arr[0]}?license_id=${value}`;
    this.setState({ url: newUrl });
  }

  onShipInitUrlSubmitted = async (url) => {
    const { onActiveInitSession } = this.props;

    if (url.indexOf("?license_id=") === -1 && url.indexOf("replicated.app") !== -1) { // Prompt user to input license ID before trying to fetch app
      return this.setState({ displayLicenseIdModal: true });
    }

    this.setState({
      saving: true,
      urlError: false,
      displayLicenseIdModal: false,
      url,
    });
    const validUpstream = await this.validateUpstream(url).catch(this.handleInvalidUpstream);

    if (validUpstream) {
      const { clusterId, githubPath, pendingInitId } = this.state;
      this.props.createInitSession(pendingInitId, url, clusterId, githubPath)
        .then(({ data }) => {
          if (!window.location.pathname.includes("/watch/create/init")) {
            // Prevent redirect if component is no longer mounted
            return;
          }

          const { createInitSession } = data;
          const { id: initSessionId } = createInitSession;
          onActiveInitSession(initSessionId);
          this.props.history.push("/ship/init")
        }).catch(this.handleInvalidUpstream);
    } else {
      this.handleInvalidUpstream();
    }
  }

  initOnEnterKey = (e) => {
    const enterKey = e.keyCode === 13;
    if (enterKey) {
      e.preventDefault();
      e.stopPropagation();
      this.onShipInitUrlSubmitted(this.state.url);
    }
  }

  validateUpstream = (url) => (
    this.props.client.query({
      query: validateUpstreamURL,
      variables: { upstream: url },
    })
  )

  handleInvalidUpstream = () => this.setState({ saving: false, fetchError: true })

  render() {
    const { url, fetchError, urlError } = this.state;
    const n = url.lastIndexOf("/");
    const appString = url.substring(n + 1);
    const readableAppString = appString.split(/[?#]/)[0];

    if (this.state.saving) {
      return (
        <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
          <ShipLoading headerText={`Fetching ${Utilities.toTitleCase(readableAppString) || ""}`} subText="We're moving as fast as we can but it may take a moment." />
        </div>
      );
    }


    return (
      <div className="Login-wrapper container flex-column flex1 u-overflow--auto">
        <div className="Form flex-column flex1 alignItems--center justifyContent--center">
          <div className="init-pre-wrapper flex-auto">
            <div className="flex1 flex-column u-textAlign--center">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal">What is the URL of the application you want to install?</p>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">This could be a URL to any Helm Chart, Kubernetes or Replicated application</p>
              {fetchError &&
                <p className="u-fontSize--small u-color--chestnut u-marginBottom--5 u-paddingBottom--20">There's a problem with fetching {Utilities.toTitleCase(appString)}. Verify you have the correct URL and try again</p>
              }
              {urlError &&
                <p className="u-fontSize--small u-color--chestnut u-marginBottom--5 u-paddingBottom--20">Please enter a valid url to the kubernetes application to continue</p>
              }
              <div className="flex flex1">
                <input value={this.state.url} onChange={(e) => { this.onUrlChange(e.target.value) }} type="text" className="Input jumbo flex1" placeholder="https://github.com/helm/charts/stable/grafana" />
                <div className="flex-auto u-marginLeft--10">
                  <button onClick={this.onShipInitUrlSubmitted.bind(this, this.state.url)} className="btn primary large">Ship init</button>
                </div>
              </div>
            </div>
            <div className="u-marginTop--10">
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--30">If you already have a <code>state.json</code> file you can <Link to="/watch/create/state" className="replicated-link">upload it here</Link></p>
            </div>

            <div className="flex flex1">
              <div className="unfork-callout flex-column flex1 flex">
                <div className="flex alignItems--center">
                  <span className="icon u-unforkIcon"></span>
                  <div className="flex1 justifyContent--center">
                    <p className="u-marginLeft--10 u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--tuna"> Unfork existing apps.</p>
                  </div>
                </div>
                <p className="u-marginTop--10 u-color--dustyGray  u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Have you already forked a 3rd-party Helm chart or k8s yaml?</p>
                <div className="flex alignItems--center">
                  <Link to="/watch/create/unfork" className="unforkLink u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Let us unfork it for you <span className="arrow icon clickable u-arrow u-marginLeft--5"></span></Link>
                </div>
              </div>

              <div className="unfork-callout flex-column flex1 flex">
                <div className="flex alignItems--center">
                  <span className="icon u-manageQuestionIcon"></span>
                  <div className="flex1 justifyContent--center">
                    <p className="u-marginLeft--10 u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-color--tuna">Not sure what to manage?</p>
                  </div>
                </div>
                <p className="u-marginTop--10 u-color--dustyGray u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Try ClusterScope to discover the outdated images in your cluster.</p>
                <div className="flex alignItems--center">
                  <Link to="/clusterscope" className="unforkLink u-marginTop--10 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Try ClusterScope today <span className="arrow icon clickable u-arrow u-marginLeft--5"></span></Link>
                </div>
              </div>
            </div>
          </div>
        </div>
        <Modal
          isOpen={this.state.displayLicenseIdModal}
          onRequestClose={() => this.setState({ displayLicenseIdModal: false })}
          shouldReturnFocusAfterClose={false}
          contentLabel="License ID Prompt Modal"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="flex-column licenseId-prompt-modal u-modalPadding">
            <p className="u-lineHeight--normal u-fontSize--larger u-color--tuna u-fontWeight--bold">License ID</p>
            <p className="u-lineHeight--normal u-fontSize--normal u-color--dustyGray u-fontWeight--normal u-marginTop--5">You must provide a license ID to fetch this app</p>
            <div className="u-marginTop--15">
              <input value={this.state.licenseId} onChange={(e) => { this.onLicenseIdChange(e.target.value) }} type="text" className="Input flex1" />
            </div>
            <div className="u-marginTop--15">
              <button type="button" className="btn primary green" onClick={this.onShipInitUrlSubmitted.bind(this, this.state.url)}>Fetch application</button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
  graphql(createInitSession, {
    props: ({ mutate }) => ({
      createInitSession: (pendingInitId, upstreamUri, clusterID, githubPath) => mutate({ variables: { pendingInitId, upstreamUri, clusterID, githubPath } })
    })
  })
)(ShipInitPre);
