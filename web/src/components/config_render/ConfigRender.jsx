import { Component } from "react";
import keyBy from "lodash/keyBy";
import find from "lodash/find";
import debounce from "lodash/debounce";
import _ from "lodash/core";

import ConfigGroups from "./ConfigGroups";
import { ConfigService } from "../../services/ConfigService";

export default class ConfigRender extends Component {
  constructor(props) {
    super(props);
    this.state = {
      groups: this.props.fields,
    };
    this.triggerChange = debounce(this.triggerChange, 300);
  }

  triggerChange = async (groups) => {
    // this actually updates the config state- it doesn't just get data.
    // also think it almost always returns undefined
    const data = await this.props.getData(groups);
    if (this.props.handleChange) {
      this.props.handleChange(data);
    }
  };

  handleGroupsChange = (groupName, itemName, value, data) => {
    const getValues = (val) => {
      if (!val) {
        return [];
      }
      return _.isArray(val) ? val : [val];
    };
    const getValue = (val) => {
      if (!val) {
        return val;
      }
      return _.isArray(val) ? _.first(val) : val;
    };
    const groups = _.map(this.props.fields, (group) => {
      if (group.name === groupName) {
        group.items = _.map(group.items, (item) => {
          if (
            !(item.type in ["select_many", "label", "heading"]) &&
            item.name === itemName
          ) {
            if (item.valuesByGroup && item.type === "file") {
              const multi_values = getValues(value);
              item.valuesByGroup[groupName] = {};
              if (multi_values.length > 0) {
                multi_values.map((file) => {
                  item.valuesByGroup[groupName][file.value] = file.filename;
                });
              }
              item.countByGroup[groupName] = multi_values.length;
            } else {
              item.value = getValue(value);
              if (item.type === "file") {
                item.filename = getValue(data);
              }
              if (item.valuesByGroup) {
                // Variadic config value
                item.valuesByGroup[groupName][data] = item.value;
              }
            }
          } else {
            if (item.type !== "select_one") {
              item.items = _.map(item.items, (childItem) => {
                if (childItem.name === itemName) {
                  if (childItem.multiple) {
                    childItem.multi_value = getValues(value);
                    if (childItem.type === "file") {
                      childItem.multi_filename = getValues(data);
                    }
                  } else {
                    childItem.value = getValue(value);
                    if (childItem.type === "file") {
                      childItem.filename = getValue(data);
                    }
                  }
                }
                return childItem;
              });
            }
          }
          return item;
        });
      }
      return group;
    });

    this.setState({
      rawGroups: groups,
      groups: keyBy(groups, "name"),
    });

    // TODO: maybe this should only be on submit
    this.triggerChange(groups);
  };

  handleAddItem = (groupName, itemName) => {
    const groups = this.props.rawGroups;
    const groupToEdit = find(groups, ["name", groupName]);
    let itemToEdit = find(groupToEdit.items, ["name", itemName]);
    if (itemToEdit.countByGroup) {
      itemToEdit.countByGroup[groupName] =
        itemToEdit.countByGroup[groupName] + 1;
    } else {
      itemToEdit["valuesByGroup"] = {
        [`${groupName}`]: {},
      };
    }
    this.setState({ rawGroups: groups });
    this.triggerChange(groups);
  };

  handleRemoveItem = (groupName, itemName, itemToRemove) => {
    const groups = this.props.rawGroups;
    const groupToEdit = find(groups, ["name", groupName]);
    let itemToEdit = find(groupToEdit.items, ["name", itemName]);
    itemToEdit.countByGroup[groupName] = itemToEdit.countByGroup[groupName] - 1;
    delete itemToEdit.valuesByGroup[`${groupName}`][`${itemToRemove}`];
    this.setState({ rawGroups: groups });
    this.triggerChange(groups);
  };

  componentDidUpdate(lastProps) {
    if (this.props.fields !== lastProps.fields) {
      this.setState({
        groups: keyBy(
          ConfigService.filterGroups(this.props.fields, this.props.filters),
          "name"
        ),
      });
    }
  }

  render() {
    const { fieldsList, readonly } = this.props;

    return (
      <div className="flex-column flex1">
        <ConfigGroups
          fieldsList={fieldsList}
          fields={this.state.groups}
          handleChange={this.handleGroupsChange}
          handleAddItem={this.handleAddItem}
          handleRemoveItem={this.handleRemoveItem}
          readonly={readonly}
          configSequence={this.props.configSequence}
          appSlug={this.props.appSlug}
        />
      </div>
    );
  }
}
