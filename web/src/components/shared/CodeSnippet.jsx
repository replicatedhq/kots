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
    preText: PropTypes.node,
    language: PropTypes.string,
    copyDelay: PropTypes.number
  }

  static defaultProps = {
    language: "bash",
    copyText: "Copy command",
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
      copyText
    } = this.props;

    const { didCopy } = this.state;

    return (
      <div className={classNames("CodeSnippet", className)}>
        <div className="CodeSnippet-content">
          {preText && React.isValidElement(preText)
            ? preText
            : (
              <div className="u-fontSize--small u-fontWeight--bold u-marginBottom--5">{preText}</div>
            )
          }
          <Prism language={language}>
            {Array.isArray(children)
              ? children.join("\n")
              : children
            }
          </Prism>
          {canCopy && (
            <span className="CodeSnippet-copy u-fontWeight--bold" onClick={this.copySnippet}>
              {didCopy
                ? "Copied!"
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
