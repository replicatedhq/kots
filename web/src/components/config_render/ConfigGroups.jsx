import React from "react";
import ConfigGroup from "./ConfigGroup";

export default class ConfigGroups extends React.Component {

  handleGroupChange = (groupName, itemName, value, data) => {
    if (this.props.handleChange) {
      this.props.handleChange(groupName, itemName, value, data);
    }
  }

  render() {
    const { fieldsList, fields, readonly } = this.props;
    return (
      <div className="flex-column flex1">
        {fieldsList && fieldsList.map((fieldName, i) => (
          <ConfigGroup
            key={`${i}-${fieldName}`}
            items={fields}
            item={fields[fieldName]}
            handleChange={(itemName, value, data) => this.handleGroupChange(fieldName, itemName, value, data)}
            readonly={readonly}
          />
        ))
        }
      </div>
    );
  }
}
