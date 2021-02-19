import React from "react";
import map from "lodash/map";
import isEmpty from "lodash/isEmpty";

import ConfigItemTitle from "./ConfigItemTitle";
import ConfigRadio from "./ConfigRadio";
import Markdown from "react-remarkable";

export default class ConfigSelectOne extends React.Component {

  handleOnChange = (itemName, val) => {
    if (this.props.handleOnChange && typeof this.props.handleOnChange === "function") {
      this.props.handleOnChange(itemName, val);
    }
  }

  render() {
    let options = [];
    map(this.props.items, (childItem, i) => {
      if (isEmpty(childItem)) return null;
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
      )
    });

    var hidden = this.props.hidden || this.props.when === "false";

    return (
      <div id={this.props.name} className={`field field-type-select-one ${hidden ? "hidden" : "u-marginTop--15"}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
            error={this.props.error}
          />
          : null}
        {this.props.help_text !== "" ? 
          <div className="field-section-help-text u-marginTop--10">
            <Markdown
              options={{
                linkTarget: "_blank",
                linkify: true,
              }}>
              {this.props.help_text}
            </Markdown>
          </div>
        : null}
        <div className="field-input-wrapper u-marginTop--5 flex flexWrap--wrap">
          {options}
        </div>
      </div>
    );
  }
}
