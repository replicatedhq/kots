import React, { Component } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import Prism from "@maji/react-prism";

import "@src/scss/components/shared/CodeSnippet.scss";

class CodeSnippet extends Component {
  state = {
    didCopy: false
  }

  static propTypes = {
    children: PropTypes.oneOfType([
      PropTypes.string,
      PropTypes.arrayOf(PropTypes.string)
    ]).isRequired,
    canCopy: PropTypes.bool,
    copyText: PropTypes.string,
    onCopyText: PropTypes.node,
    preText: PropTypes.node,
    language: PropTypes.string,
    copyDelay: PropTypes.number,
    variant: PropTypes.string
  }

  static defaultProps = {
    variant: "plain",
    language: "bash",
    copyText: "Copy command",
    onCopyText: "Copied!",
    copyDelay: 3000
  }

  copySnippet = () => {
    const { children, copyDelay } = this.props;
    const textToCopy = Array.isArray(children)
      ? children.join("\n")
      : children
    if (navigator.clipboard) {
      navigator.clipboard.writeText(textToCopy).then(() => {
        this.setState({ didCopy: true });

        setTimeout(() => {
          this.setState({ didCopy: false });
        }, copyDelay);
      });
    }
  }

  render() {
    const {
      className,
      children,
      language,
      preText,
      canCopy,
      copyText,
      onCopyText,
      variant
    } = this.props;

    const { didCopy } = this.state;
    const trimChild = text => {
      console.log(text);
      return text.trim();
    }
    return (
      <div className={classNames("CodeSnippet", `variant-${variant}`, className)}>
        <div className="CodeSnippet-content">
          {preText && React.isValidElement(preText)
            ? preText
            : (
              <div className="u-fontSize--small u-fontWeight--bold u-marginBottom--5">{preText}</div>
            )
          }
          <Prism language={language}>
            {Array.isArray(children)
              ? children.map(trimChild).filter(Boolean).join("\n")
              : children.trim()
            }
          </Prism>
          {canCopy && (
            <span
              className={classNames("CodeSnippet-copy u-fontWeight--bold", {
                "is-copied": didCopy
              })}
              onClick={this.copySnippet}
            >
              {didCopy
                ? onCopyText
                : copyText
              }
            </span>
          )}
        </div>
      </div>
    )
  }
}

export default CodeSnippet;
