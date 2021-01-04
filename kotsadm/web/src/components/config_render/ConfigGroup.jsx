import React from "react";
import Markdown from "react-remarkable";
import each from "lodash/each";
import some from "lodash/some";
import isEmpty from "lodash/isEmpty";
import { ConfigService } from "../../services/ConfigService";

import ConfigInput from "./ConfigInput";
import ConfigTextarea from "./ConfigTextarea";
import ConfigSelectOne from "./ConfigSelectOne";
import ConfigItemTitle from "./ConfigItemTitle";
import ConfigCheckbox from "./ConfigCheckbox";
import ConfigFileInput from "./ConfigFileInput";

export default class ConfigGroup extends React.Component {
  constructor() {
    super();
    this.markdownNode = React.createRef();
  }

  handleItemChange = (itemName, value, data) => {
    if (this.props.handleChange) {
      this.props.handleChange(itemName, value, data);
    }
  }

  renderConfigItems = (items, readonly) => {
    if (!items) return null;
    return items.map((item, i) => {
      switch (item.type) {
      case "text":
        return (
          <ConfigInput
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            inputType="text"
            hidden={item.hidden}
            when={item.when}
            {...item}
            readonly={readonly}
          />
        );
      case "textarea":
        return (
          <ConfigTextarea
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            when={item.when}
            {...item}
            readonly={readonly}
          />
        );
      case "bool":
        return (
          <ConfigCheckbox
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            when={item.when}
            {...item}
            readonly={readonly}
          />
        );
      case "label":
        return (
          <div key={`${i}-${item.name}`} className="field field-type-label u-marginTop--15">
            <ConfigItemTitle
              title={item.title}
              recommended={item.recommended}
              required={item.required}
              hidden={item.hidden}
              when={item.when}
              name={item.name}
              error={item.error}
              readonly={readonly}
            />
          </div>
        );
      case "file":
        return (
          <div key={`${i}-${item.name}`}>
            <ConfigFileInput
              {...item}
              title={item.title}
              recommended={item.recommended}
              required={item.required}
              handleChange={this.handleItemChange}
              hidden={item.hidden}
              when={item.when}
              readonly={readonly}
            />
          </div>
        );
      case "select_one":
        return (
          <ConfigSelectOne
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            when={item.when}
            {...item}
            readonly={readonly}
          />
        );
      case "heading":
        return (
          <div key={`${i}-${item.name}`} className={`u-marginTop--40 u-marginBottom--15 ${item.hidden || item.when === "false" ? "hidden" : ""}`}>
            <h3 className="header-color field-section-header">{item.title}</h3>
          </div>
        );
      case "password":
        return (
          <ConfigInput
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            when={item.when}
            inputType="password"
            {...item}
            readonly={readonly}
          />
        );
      default:
        return (
          <div key={`${i}-${item.name}`}>Unsupported config type <a href="https://help.replicated.com/docs/config-screen/config-yaml/" target="_blank" rel="noopener noreferrer">Check out our docs</a> to see all the support config types.</div>
        );
      }
    })
  }

  isAtLeastOneItemVisible = () => {
    const { item } = this.props;
    if (!item) return false;
    return some(this.props.item.items, (item) => {
      if (!isEmpty(item)) {
        return ConfigService.isVisible(this.props.items, item);
      }
    });
  }

  render() {
    const { item, readonly } = this.props;
    const hidden = item && (item.when === "false");
    if (hidden || !this.isAtLeastOneItemVisible()) return null;
    return (
      <div className="flex-column flex-auto">
        {item &&
          <div id={item.name} className={`flex-auto config-item-wrapper ${this.isAtLeastOneItemVisible() ? "u-marginBottom--40" : ""}`}>
            <h3 className="header-color field-section-header">{item.title}</h3>
            {item.description !== "" ?
              <div className="field-section-help-text u-marginTop--10">
                <Markdown
                  ref={this.markdownNode}
                  options={{
                    linkTarget: "_blank",
                    linkify: true,
                  }}>
                  {item.description}
                </Markdown>
              </div>
              : null}
            <div className="config-item">
              {this.renderConfigItems(item.items, readonly)}
            </div>
          </div>
        }
      </div>
    );
  }
}
