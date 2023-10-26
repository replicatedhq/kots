import { Component } from "react";
import ConfigGroup from "./ConfigGroup";

export default class ConfigGroups extends Component {
  handleGroupChange = (groupName, itemName, value, data) => {
    if (this.props.handleChange) {
      this.props.handleChange(groupName, itemName, value, data);
    }
  };

  handleAddItem = (groupName, itemName) => {
    if (this.props.handleAddItem) {
      this.props.handleAddItem(groupName, itemName);
    }
  };

  handleRemoveItem = (groupName, itemName, itemToRemove) => {
    if (this.props.handleRemoveItem) {
      this.props.handleRemoveItem(groupName, itemName, itemToRemove);
    }
  };

  render() {
    const { fieldsList, fields, readonly } = this.props;
    return (
      <div className="flex-column flex1">
        {fieldsList &&
          fieldsList.map((fieldName, i) => (
            <ConfigGroup
              key={`${i}-${fieldName}`}
              items={fields}
              handleAddItem={(itemName) =>
                this.handleAddItem(fieldName, itemName)
              }
              handleRemoveItem={(itemName, itemToRemove) =>
                this.handleRemoveItem(fieldName, itemName, itemToRemove)
              }
              item={fields[fieldName]}
              handleChange={(itemName, value, data) =>
                this.handleGroupChange(fieldName, itemName, value, data)
              }
              readonly={readonly}
              configSequence={this.props.configSequence}
              appSlug={this.props.appSlug}
            />
          ))}
      </div>
    );
  }
}
