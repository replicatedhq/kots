import { Component, createRef } from "react";
import PropTypes from "prop-types";

/**
 * Component that performs an action if you click outside of it
 */
export default class ClickOutsideAction extends Component {
  constructor(props) {
    super(props);

    this.wrapperRef = createRef();
  }

  componentDidMount() {
    document.addEventListener("mousedown", this.handleClickOutside);
  }

  componentWillUnmount() {
    document.removeEventListener("mousedown", this.handleClickOutside);
  }

  /**
   * Alert if clicked on outside of element
   */
  handleClickOutside = (event) => {
    if (this.wrapperRef && !this.wrapperRef.current.contains(event.target)) {
      this.props.onOutsideClick(event);
    }
  };

  render() {
    return <div ref={this.wrapperRef}>{this.props.children}</div>;
  }
}

ClickOutsideAction.propTypes = {
  children: PropTypes.element.isRequired,
  onOutsideClick: PropTypes.func.isRequired,
};
