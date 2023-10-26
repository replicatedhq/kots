import { createRef } from "react";
import Markdown from "react-remarkable";
import some from "lodash/some";
import isEmpty from "lodash/isEmpty";
import { ConfigService } from "../../services/ConfigService";

import ConfigInput from "./ConfigInput";
import ConfigTextarea from "./ConfigTextarea";
import ConfigSelectOne from "./ConfigSelectOne";
import ConfigItemTitle from "./ConfigItemTitle";
import ConfigCheckbox from "./ConfigCheckbox";
import ConfigFileInput from "./ConfigFileInput";
import { setOrder } from "./ConfigUtil";
import { ConfigWrapper } from "./ConfigComponents";
import Icon from "../Icon";

const ConfigGroup = (props) => {
  const markdownNode = createRef();

  const handleItemChange = (itemName, value, data) => {
    if (props.handleChange) {
      props.handleChange(itemName, value, data);
    }
  };

  const handleAddItem = (itemName) => {
    if (props.handleAddItem) {
      props.handleAddItem(itemName);
    }
  };

  const handleRemoveItem = (itemName, itemToRemove) => {
    if (props.handleRemoveItem) {
      props.handleRemoveItem(itemName, itemToRemove);
    }
  };

  const renderConfigItems = (items, readonly) => {
    if (!items) {
      return null;
    }

    return items.map((item, i) => {
      const isReadOnly = readonly || item.readonly;
      switch (item.type) {
        case "text":
          return (
            <ConfigInput
              key={`${i}-${item.name}`}
              handleOnChange={handleItemChange}
              handleAddItem={handleAddItem}
              handleRemoveItem={handleRemoveItem}
              inputType="text"
              groupName={props.item.name}
              hidden={item.hidden}
              when={item.when}
              {...item}
              readonly={isReadOnly}
              index={i + 1}
              validationErrorMessage={item?.validationError}
              showValidationError={item?.validationError}
            />
          );
        case "textarea":
          return (
            <ConfigTextarea
              key={`${i}-${item.name}`}
              handleOnChange={handleItemChange}
              handleAddItem={handleAddItem}
              handleRemoveItem={handleRemoveItem}
              hidden={item.hidden}
              groupName={props.item.name}
              when={item.when}
              {...item}
              readonly={isReadOnly}
              index={i + 1}
              validationErrorMessage={item?.validationError}
              showValidationError={item?.validationError}
            />
          );
        case "bool":
          return (
            <ConfigCheckbox
              key={`${i}-${item.name}`}
              handleOnChange={handleItemChange}
              handleAddItem={handleAddItem}
              handleRemoveItem={handleRemoveItem}
              hidden={item.hidden}
              groupName={props.item.name}
              when={item.when}
              {...item}
              readonly={isReadOnly}
              index={i + 1}
            />
          );
        case "label":
          return (
            <div
              key={`${i}-${item.name}`}
              className="field field-type-label"
              style={{
                margin: props.affix ? "0" : "15px",
                order: setOrder(i + 1, item.affix),
              }}
            >
              <ConfigItemTitle
                title={item.title}
                recommended={item.recommended}
                required={item.required}
                hidden={item.hidden}
                groupName={props.item.name}
                when={item.when}
                name={item.name}
                error={item.error}
                readonly={isReadOnly}
              />
            </div>
          );
        case "file":
          return (
            <ConfigWrapper
              key={`${i}-${item.name}`}
              className={"field-type-label "}
              marginTop={item.affix ? "0" : "35px"}
              order={setOrder(i + 1, item.affix)}
            >
              <ConfigFileInput
                {...item}
                title={item.title}
                recommended={item.recommended}
                groupName={props.item.name}
                required={item.required}
                handleChange={handleItemChange}
                handleRemoveItem={handleRemoveItem}
                hidden={item.hidden}
                when={item.when}
                configSequence={props.configSequence}
                appSlug={props.appSlug}
                readonly={isReadOnly}
                index={i + 1}
                validationErrorMessage={item?.validationError}
                showValidationError={item?.validationError}
              />
            </ConfigWrapper>
          );
        case "select_one":
          return (
            <ConfigSelectOne
              key={`${i}-${item.name}`}
              handleOnChange={handleItemChange}
              hidden={item.hidden}
              groupName={props.item.name}
              when={item.when}
              {...item}
              readonly={isReadOnly}
              index={i + 1}
            />
          );
        case "heading":
          return (
            <div
              key={`${i}-${item.name}`}
              className={`u-marginTop--15 u-marginBottom--15   ${
                item.hidden || item.when === "false" ? "hidden" : ""
              }`}
              style={{ order: setOrder(i + 1, item.affix) }}
            >
              <h3 className="header-color field-section-header">
                {item.title}
              </h3>
            </div>
          );
        case "password":
          return (
            <ConfigInput
              key={`${i}-${item.name}`}
              handleOnChange={handleItemChange}
              handleAddItem={handleAddItem}
              handleRemoveItem={handleRemoveItem}
              hidden={item.hidden}
              groupName={props.item.name}
              when={item.when}
              inputType="password"
              {...item}
              readonly={isReadOnly}
              index={i + 1}
              validationErrorMessage={item?.validationError}
              showValidationError={item?.validationError}
            />
          );
        default:
          return (
            <div key={`${i}-${item.name}`}>
              Unsupported config type{" "}
              <a
                href="https://help.replicated.com/docs/config-screen/config-yaml/"
                target="_blank"
                rel="noopener noreferrer"
              >
                Check out our docs
              </a>{" "}
              to see all the support config types.
            </div>
          );
      }
    });
  };

  const isAtLeastOneItemVisible = () => {
    const { item } = props;
    if (!item) {
      return false;
    }
    return some(props.item.items, (item) => {
      if (!isEmpty(item)) {
        return ConfigService.isVisible(props.items, item);
      }
    });
  };

  const { item, readonly } = props;
  const hidden = item && item.when === "false";
  if (hidden || !isAtLeastOneItemVisible()) {
    return null;
  }
  const hasAffix = item.items.every((option) => option.affix);
  return (
    <div className="flex-column flex-auto">
      {item && (
        <div
          id={`${item.name}`}
          className={`flex-auto config-item-wrapper card-item u-padding--15 observe-elements ${
            isAtLeastOneItemVisible() ? "u-marginBottom--20" : ""
          } config-groups`}
        >
          <h3 className="card-item-title">{item.title}</h3>
          {item.description !== "" ? (
            <div className="field-section-help-text help-text-color u-marginTop--10">
              <Markdown
                ref={markdownNode}
                options={{
                  linkTarget: "_blank",
                  linkify: true,
                }}
              >
                {item.description}
              </Markdown>
            </div>
          ) : null}
          <div
            className="config-items"
            style={{ display: hasAffix ? "grid" : "block" }}
          >
            {renderConfigItems(item.items, readonly)}
          </div>
          {item.repeatable && (
            <div className="u-marginTop--15">
              <button className="btn secondary blue rounded add-btn">
                <Icon icon="plus" size={14} />
                Add another {item.title}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ConfigGroup;
