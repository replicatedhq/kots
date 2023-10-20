import { Component } from "react";
import get from "lodash/get";

export default class ConfigRadio extends Component {
  handleOnChange = (e) => {
    const { group } = this.props;
    if (
      this.props.handleChange &&
      typeof this.props.handleChange === "function"
    ) {
      this.props.handleChange(group, e.target.value);
    }
  };

  render() {
    let val = get(this.props, "value");
    if (!val || val.length === 0) {
      val = this.props.default;
    }
    const checked = val === this.props.name;

    return (
      <div
        id={`${this.props.name}-group`}
        className="flex alignItems--center u-marginRight--20 u-marginTop--15"
      >
        <input
          type="radio"
          name={this.props.group}
          id={`${this.props.group}-${this.props.name}`}
          value={this.props.name}
          checked={checked}
          disabled={this.props.readOnly}
          onChange={(e) => this.handleOnChange(e)}
          className={`${this.props.className || ""} ${
            this.props.readOnly ? "readonly" : ""
          }`}
        />
        <label
          htmlFor={`${this.props.group}-${this.props.name}`}
          className={`u-marginLeft--5 card-item-title field-section-sub-header u-userSelect--none ${
            this.props.readOnly ? "u-cursor--default" : "u-cursor--pointer"
          }`}
        >
          {this.props.title}
        </label>
      </div>
    );
  }
}
