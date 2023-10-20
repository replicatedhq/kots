import { Component } from "react";
import Icon from "../Icon";

export default class AnnotationRow extends Component {
  state = {
    key: "",
    value: "",
  };

  handleFormChange = (field, e) => {
    let nextState = {};
    nextState[field] = e.target.value;
    this.setState(nextState);
  };

  render() {
    return (
      <div
        className="flex flex-column u-borderBottom--gray darker"
        style={{ padding: "8px 10px" }}
      >
        <div className="flex flex1 alignItems--center justifyContent--spaceBetween">
          <div className="flex justifyContent--flexStart">
            <div className="flex alignItems--center">
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--accent">
                Key
              </span>
              <input
                type="text"
                className="Input darker"
                style={{ marginLeft: "12px" }}
                placeholder="key"
                value={this.state.key}
                onChange={(e) => {
                  this.handleFormChange("key", e);
                }}
              />
            </div>
            <div className="flex alignItems--center u-marginLeft--20">
              <span className="u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-textColor--accent">
                Value
              </span>
              <input
                type="text"
                className="Input darker"
                style={{ marginLeft: "12px" }}
                placeholder="value"
                value={this.state.value}
                onChange={(e) => {
                  this.handleFormChange("value", e);
                }}
              />
            </div>
          </div>
          <div className="flex fle1 justifyContent--flexEnd">
            <Icon
              icon="trash"
              size={20}
              className="clickable gray-color"
              onClick={this.props.removeAnnotation}
            />
          </div>
        </div>
      </div>
    );
  }
}
