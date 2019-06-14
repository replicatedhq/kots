import React from "react";
import GraphiQL from "graphiql";
import fetch from "isomorphic-fetch";
import { parse, print } from "graphql";
import { Utilities } from "../utilities/utilities";
import { userInfoRaw } from "../queries/UserQueries";
import "../../node_modules/graphiql/graphiql.css";

export default class ReplicatedGraphiQL extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      fetcher: this.fetcher.bind(this),
      query: userInfoRaw,
    };
  }

  fetcher(params) {
    return fetch(window.env.GRAPHQL_ENDPOINT, {
      method: "post",
      headers: {
        "Content-Type": "application/json",
        "Authorization": Utilities.getToken(),
      },
      body: JSON.stringify(params),
    }).then(response => response.json());
  }

  handleClickPrettifyButton() {
    const editor = this.graphiql.getQueryEditor();
    const currentText = editor.getValue();
    const prettyText = print(parse(currentText));
    editor.setValue(prettyText);
  }

  render() {
    return (
      <GraphiQL ref={c => { this.graphiql = c; }} {...this.state}>
        <GraphiQL.Toolbar>

          <GraphiQL.Button
            onClick={this.handleClickPrettifyButton}
            label="Prettify"
            title="Prettify Query (Shift-Ctrl-P)"
          />

        </GraphiQL.Toolbar>
      </GraphiQL>
    );
  }
}