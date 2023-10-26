import { Component } from "react";
import * as PropTypes from "prop-types";
import "../../scss/components/shared/Tooltip.scss";

export default class Tooltip extends Component {
  static propTypes = {
    className: PropTypes.string,
    visible: PropTypes.bool,
    text: PropTypes.string,
    content: PropTypes.node,
    position: PropTypes.string,
    minWidth: PropTypes.string,
  };

  static defaultProps = {
    position: "top-center",
    minWidth: "80",
  };

  render() {
    const { className, visible, text, content, position, minWidth } =
      this.props;

    const wrapperClass = `Tooltip-wrapper tooltip-${position} ${
      className || ""
    } ${visible ? "is-active" : ""}`;

    return (
      <span className={wrapperClass} style={{ minWidth: `${minWidth}px` }}>
        <span className="Tooltip-content">{content || text}</span>
      </span>
    );
  }
}
