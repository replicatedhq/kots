import { useState } from "react";
import { ConfigWrapper } from "./ConfigComponents";
import ConfigItemTitle from "./ConfigItemTitle";
import Icon from "@components/Icon";
import Markdown from "react-remarkable";
import { isEmpty } from "lodash";

const ConfigDropdown = (props) => {
  const [selectedValue, setSelectedValue] = useState(
    props.value || props.default || ""
  );

  let options = [{ value: "", label: "Select an option" }];

  props.items.map((item) => {
    if (isEmpty(item)) {
      return null;
    }
    options.push({ value: item.name, label: item.title });
  });

  const handleChange = (val) => {
    setSelectedValue(val);
    props.handleOnChange(props.name, val);
  };

  return (
    <ConfigWrapper
      id={`${props.name}-group`}
      className={`field-type-dropdown`}
      marginTop={props.hidden || props.affix ? "0" : "15px"}
      hidden={props.hidden}
    >
      {props.title !== "" ? (
        <ConfigItemTitle
          title={props.title}
          recommended={props.recommended}
          required={props.required}
          name={props.name}
          error={props.error}
        />
      ) : null}
      {props.help_text !== "" ? (
        <div className="field-section-help-text help-text-color">
          <Markdown
            options={{
              linkTarget: "_blank",
              linkify: true,
            }}
          >
            {props.help_text}
          </Markdown>
        </div>
      ) : null}
      <select
        className="Input tw-mt-4"
        value={selectedValue}
        onChange={(e) => handleChange(e.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {props.repeatable && (
        <div
          className="u-marginTop--10"
          onClick={() => props.handleAddItem(props.name)}
        >
          <span className="add-btn u-fontSize--small u-fontWeight--bold link">
            <Icon icon="plus" size={14} className="clickable" />
            Add another {props.title}
          </span>
        </div>
      )}
    </ConfigWrapper>
  );
};

export default ConfigDropdown;
