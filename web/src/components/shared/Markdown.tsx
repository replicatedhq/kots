import React from "react";
import MarkdownIt from "markdown-it";

interface MarkdownProps {
  children: string;
  options?: {
    linkTarget?: string;
    linkify?: boolean;
  };
}

const Markdown: React.FC<MarkdownProps> = ({ children, options = {} }) => {
  const { linkTarget = "_blank", linkify = true } = options;
  
  const md = new MarkdownIt({
    linkify,
    html: true,
    breaks: true,
  });

  // Configure link attributes if linkTarget is specified
  if (linkTarget) {
    md.renderer.rules.link_open = function (tokens, idx, opts, _env, renderer) {
      const token = tokens[idx];
      const hrefIndex = token.attrIndex('href');
      
      if (hrefIndex >= 0) {
        const href = token.attrGet('href');
        // Only add target="_blank" for external links
        if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
          token.attrSet('target', linkTarget);
          token.attrSet('rel', 'noopener noreferrer');
        }
      }
      
      return renderer.renderToken(tokens, idx, opts);
    };
  }

  const html = md.render(children || "");

  return <div dangerouslySetInnerHTML={{ __html: html }} />;
};

export default Markdown;