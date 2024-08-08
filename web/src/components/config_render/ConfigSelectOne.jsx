import { Component } from "react";
import map from "lodash/map";
import isEmpty from "lodash/isEmpty";

import ConfigItemTitle from "./ConfigItemTitle";
import ConfigRadio from "./ConfigRadio";
import Markdown from "react-remarkable";
import { setOrder } from "./ConfigUtil";
import { ConfigWrapper } from "./ConfigComponents";
import Icon from "../Icon";

export default class ConfigSelectOne extends Component {
  handleOnChange = (itemName, val) => {
    if (
      this.props.handleOnChange &&
      typeof this.props.handleOnChange === "function"
    ) {
      this.props.handleOnChange(itemName, val);
    }
  };

  render() {
    let options = [];
    map(this.props.items, (childItem, i) => {
      if (isEmpty(childItem)) {
        return null;
      }
      options.push(
        <ConfigRadio
          key={`${childItem.name}-${i}`}
          name={childItem.name}
          title={childItem.title}
          id={childItem.name}
          default={this.props.default}
          group={this.props.name}
          value={this.props.value}
          readOnly={this.props.readonly}
          handleChange={(itemName, val) => this.handleOnChange(itemName, val)}
        />
      );
    });

    var hidden = this.props.hidden || this.props.when === "false";

    return (
      <ConfigWrapper
        id={`${this.props.name}-group`}
        className={`field-type-select-one`}
        marginTop={hidden || this.props.affix ? "0" : "15px"}
        hidden={hidden}
        order={setOrder(this.props.index, this.props.affix)}
      >
        {this.props.title !== "" ? (
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
            error={this.props.error}
            deprecated={this.props.deprecated}
          />
        ) : null}
        {this.props.help_text !== "" ? (
          <div className="field-section-help-text help-text-color">
            <Markdown
              options={{
                linkTarget: "_blank",
                linkify: true,
              }}
            >
              {this.props.help_text}
            </Markdown>
          </div>
        ) : null}
        <div className="field-input-wrapper u-marginTop--5 flex flexWrap--wrap">
          {options}
        </div>
        {this.props.repeatable && (
          <div
            className="u-marginTop--10"
            onClick={() => this.props.handleAddItem(this.props.name)}
          >
            <span className="add-btn u-fontSize--small u-fontWeight--bold link">
              <Icon icon="plus" size={14} className="clickable" />
              Add another {this.props.title}
            </span>
          </div>
        )}
      </ConfigWrapper>
    );
  }
}
