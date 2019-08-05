import React from "react";
import Helmet from "react-helmet";
import classNames from "classnames";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import MonacoEditor from "react-monaco-editor";
import { createNewWatch, updateStateJSON } from "../../mutations/WatchMutations";
import { getWatchJson } from "../../queries/WatchQueries";
import { userInfo } from "../../queries/UserQueries";

import "../../scss/components/state/StateFileViewer.scss";

class StateFileViewer extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      specValue: "",
      initialSpecValue: "",
      specValueError: false,
      serverError: false,
      specValueMessage: "",
      serverErrorMessage: "",
    }
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
    const { watch } = this.props;
    const {
      specValue,
      specValueError,
      specValueMessage,
      serverEror,
      serverErrorMessage
    } = this.state;


    return (
      <div className={classNames("flex-column flex1 HelmValues--wrapper", {
        "u-paddingTop--30": this.props.isNew,
        "u-paddingTop--20": !this.props.isNew
        })}>
        <Helmet>
          <title>{`${watch?.watchName || ""} State JSON`.trim()}</title>
        </Helmet>
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
                  readOnly: true,
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
              {specValueError &&
                <p className=" u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">
                  {specValueMessage}</p>
              }
              {serverEror &&
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--10 u-lineHeight--normal">
                  {serverErrorMessage}</p>
              }
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
