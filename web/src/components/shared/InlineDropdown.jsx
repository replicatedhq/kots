import { Component } from "react";
import { Link } from "react-router-dom";
import ClickOutsideAction from "./ClickOutsideAction";
import "../../scss/components/shared/InlineDropdown.scss";
import Icon from "../Icon";

export default class InlineDropdown extends Component {
  state = {
    showOptions: false,
  };

  toggleDropdownVisible = () => {
    this.setState({ showOptions: !this.state.showOptions });
  };

  buildOptions = () => {
    /* Option object
      {
        displayText: String,
        link | href | onClick: String | String | Func
      }
    */
    const { dropdownOptions } = this.props;
    if (!dropdownOptions || dropdownOptions.length === 0) {
      return null;
    }

    return dropdownOptions.map((opt, i) => {
      if (opt.link) {
        return (
          <Link className="option" key={i} to={opt.link}>
            {opt.displayText}
          </Link>
        );
      } else if (opt.href) {
        return (
          <a
            target="_blank"
            rel="noopener noreferrer"
            className="option"
            key={i}
            href={opt.href}
          >
            {opt.displayText}
          </a>
        );
      } else if (opt.onClick) {
        return (
          <div className="option" key={i} onClick={opt.onClick}>
            {opt.displayText}
          </div>
        );
      }
    });
  };

  render() {
    const { showOptions } = this.state;

    return (
      <ClickOutsideAction
        onOutsideClick={() => this.setState({ showOptions: false })}
      >
        <div
          className={`InlineDropdown--wrapper ${
            showOptions ? "show-options" : ""
          }`}
        >
          <div
            className="flex flex-auto alignItems--center"
            onClick={() => this.toggleDropdownVisible()}
          >
            <span className="display-text">
              {this.props.defaultDisplayText || ""}
            </span>
            <Icon
              icon="down-arrow"
              size={12}
              className="clickable u-marginLeft--5"
            />
          </div>
          <div className="Options--wrapper">{this.buildOptions()}</div>
        </div>
      </ClickOutsideAction>
    );
  }
}
