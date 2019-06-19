import React from "react";
import classNames from "classnames";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import MonacoEditor from "react-monaco-editor";
import isEmpty from "lodash/isEmpty";
import { createNewWatch, updateStateJSON } from "../../mutations/WatchMutations";
import { getWatchJson } from "../../queries/WatchQueries";
import { userInfo } from "../../queries/UserQueries";
import Loader from "../shared/Loader";

import "../../scss/components/state/StateFileViewer.scss";

class StateFileViewer extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      specValue: "",
      savedJson: false,
      initialSpecValue: "",
      saving: false,
      specValueError: false,
      serverError: false,
      specValueMessage: "",
      serverErrorMessage: "",
    }
  }

  onSpecChange = (value) => {
    this.setState({
      specValue: value
    });
  }

  getWatchJson = (slug) => {
    this.props.client.query({
      query: getWatchJson,
      variables: { slug }
    })
      .then((res) => {
        const json = res.data.getWatch.stateJSON;
        this.setState({ specValue: json, initialSpecValue: json });
      })
      .catch();
  }

  clearSaved = () => {
    this.timeout = setTimeout(() => {
      this.setState({ savedJson: false });
    }, 3000);
  }

  savedValues = () => {
    this.setState({ savedJson: true, saving: false });
    this.clearSaved();
  }

  validateSpecValue(specValue) {
    let canSubmit = true;
    if (isEmpty(specValue)) {
      this.setState({
        specValueError: true,
        specValueMessage: "Can't save an empty json"
      })
      canSubmit = false;
    }
    return canSubmit;
  }

  handleSaveValues = () => {
    const { specValue } = this.state;
    const { owner, slug } = this.props.match.params;
    const watchSlug = `${owner}/${slug}`;
    this.setState({ saving: true, specValueError: false, serverEror: false });

    const isValid = this.validateSpecValue(specValue);

    if (this.props.isNew) {
      if (isValid) {
        this.props.createNewWatch(specValue)
          .then(() => {
            this.setState({ saving: false });
            this.savedValues();
            this.props.history.replace(`/watches`);
          })
          .catch((err) => {
            err.graphQLErrors.map(({ message }) => {
              if (message === "JSON is not valid") {
                this.setState({
                  saving: false,
                  serverEror: true,
                  serverErrorMessage: message
                })
              } else {
                this.setState({
                  saving: false,
                  serverEror: true,
                  serverErrorMessage: message
                })
              }
            });
          })
      } else {
        this.setState({ saving: false });
      }
    } else {
      if (isValid) {
        this.props.updateStateJSON(watchSlug, specValue)
          .then((res) => {
            this.setState({ specValue: res.data.updateStateJSON.stateJSON, initialSpecValue: res.data.updateStateJSON.stateJSON, saving: false });
            this.savedValues();
          })
          .catch((err) => {
            err.graphQLErrors.map(({ message }) => {
              if (message === "JSON is not valid") {
                this.setState({
                  saving: false,
                  serverEror: true,
                  serverErrorMessage: message
                })
              } else {
                this.setState({
                  saving: false,
                  serverEror: true,
                  serverErrorMessage: message
                })
              }
            });
          })
      } else {
        this.setState({ saving: false });
      }
    }
  }

  componentDidMount() {
    const { slug, owner } = this.props.match.params;
    if (slug) {
      this.getWatchJson(`${owner}/${slug}`);
    }
  }

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  render() {
    const {
      specValue,
      specValueError,
      specValueMessage,
      serverEror,
      serverErrorMessage,
      savedJson,
      saving
    } = this.state;

    if (saving) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center u-paddingTop--20">
          <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">We're moving as fast as we can but it may take a moment.</p>
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className={classNames("flex-column flex1 HelmValues--wrapper", {
        "u-paddingTop--30": this.props.isNew,
        "u-paddingTop--20": !this.props.isNew
        })}>
        <div className="flex-column flex-1-auto u-overflow--auto container">
          {this.props.headerText && <p className="u-color--tuna u-fontWeight--medium u-fontSize--large">{this.props.headerText}</p>}
          {this.props.subText && <p className="u-color--dustyGray u-fontSize--normal u-fontWeight--medium u-marginTop--10 u-lineHeight--normal">{this.props.subText}</p>}
          <div className="MonacoEditor--wrapper helm-values flex1 flex u-height--full u-width--full u-marginTop--20">
            <div className="flex1 flex-column u-width--half u-overflow--hidden">
              <MonacoEditor
                ref={(editor) => { this.monacoEditor = editor }}
                language="json"
                onChange={this.onSpecChange}
                value={specValue}
                height="100%"
                width="100%"
                options={{
                  minimap: {
                    enabled: false
                  },
                  scrollBeyondLastLine: false,
                }}
              />
            </div>
          </div>
          <div className="flex justifyContent--flexEnd u-marginBottom--30">
            <div className="flex-column flex-verticalCenter">
              {savedJson &&
                <p className="u-color--chateauGreen u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">Values saved</p>
              }
              {specValueError &&
                <p className=" u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">
                  {specValueMessage}</p>
              }
              {serverEror &&
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">
                  {serverErrorMessage}</p>
              }
            </div>
            <div className="flex justifyContent--flexEnd">
              <button className="btn primary u-marginRight--normal" onClick={this.handleSaveValues} disabled={saving}>{saving ? "Saving" : "Save"}</button>
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
  graphql(createNewWatch, {
    props: ({ mutate }) => ({
      createNewWatch: (stateJSON) => mutate({ variables: { stateJSON } })
    })
  }),
  graphql(updateStateJSON, {
    props: ({ mutate }) => ({
      updateStateJSON: (slug, stateJSON) => mutate({ variables: { slug, stateJSON } })
    })
  }),
  graphql(userInfo, {
    name: "getUserInfo"
  }),
)(StateFileViewer);
