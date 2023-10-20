import { Component } from "react";
import Markdown from "markdown-it";

import "@src/scss/components/shared/MarkdownRenderer.scss";

const md = Markdown();

export default class MarkdownRenderer extends Component {
  componentDidMount() {
    const anchors = document
      .getElementById(this.props.id)
      .getElementsByTagName("a");
    for (let i = 0; i < anchors.length; i++) {
      anchors[i].setAttribute("target", "_blank");
    }
  }

  render() {
    const { children = "", className } = this.props;

    return (
      <div className={className}>
        <div
          id={this.props.id}
          className={`${className || ""} markdown-wrapper`}
          dangerouslySetInnerHTML={{ __html: md.render(children) }}
        />
      </div>
    );
  }
}
