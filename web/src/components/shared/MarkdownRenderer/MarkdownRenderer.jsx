import React from "react";
import Markdown from "markdown-it";

import "@src/scss/components/shared/MarkdownRenderer.scss";

const md = Markdown();

export default class MarkdownRenderer extends React.Component {

  componentDidMount() {
    const anchors = document.getElementById("markdown-wrapper").getElementsByTagName("a");

    for (let i=0; i < anchors.length; i++) {
      anchors[i].setAttribute("target", "_blank");
    }
  }

  render () {
    const { children = "", className } = this.props;
  
    return (
      <div className={className}>
        <div
          id="markdown-wrapper"
          className="is-kotsadm markdown-wrapper"
          dangerouslySetInnerHTML={{ __html: md.render(children)}}
        />
      </div>
    );
  }
}
