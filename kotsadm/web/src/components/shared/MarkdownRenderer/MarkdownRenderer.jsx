import React from "react";
import Markdown from "markdown-it";

import "@src/scss/components/shared/MarkdownRenderer.scss";

const md = Markdown();

export default function MarkdownRenderer(props) {
  const { children = "", className } = props;

  return (
    <div className={className}>
      <div
        className="is-kotsadm markdown-wrapper"
        dangerouslySetInnerHTML={{ __html: md.render(children)}}
      />
    </div>
  );
}
