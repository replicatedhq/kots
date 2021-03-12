import React from "react";
import keyBy from "lodash/keyBy";
import debounce from "lodash/debounce";
import _ from "lodash/core";

import ConfigGroups from "./ConfigGroups";
import { ConfigService } from "../../services/ConfigService";

export default class ConfigRender extends React.Component {

  constructor(props) {
    super(props);
    this.state= {
      groups: this.props.fields
    }
    this.triggerChange = debounce(this.triggerChange, 300);
  }

  triggerChange = (data) => {
    if (this.props.handleChange) {
      this.props.handleChange(data);
    }
  }

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
    let groups = _.map(this.props.fields, (group) => {
      if (group.name === groupName) {
        group.items = _.map(group.items, (item) => {
          if (!(item.type in ["select_many", "label", "heading"]) && item.name === itemName) {
            if (item.multiple) {
              item.multi_value = getValues(value);
              if (item.type === "file") {
                item.multi_data = getValues(data);
              }
            } else {
              item.value = getValue(value);
              if (item.type === "file") {
                item.data = getValue(data);
              }
            }
          } else {
            if (item.type !== "select_one") {
              item.items = _.map(item.items, (childItem) => {
                if (childItem.name === itemName) {
                  if (childItem.multiple) {
                    childItem.multi_value = getValues(value);
                    if (childItem.type === "file") {
                      childItem.multi_data = getValues(data);
                    }
                  } else {
                    childItem.value = getValue(value);
                    if (childItem.type === "file") {
                      childItem.data = getValue(data);
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

    this.setState({groups: keyBy(groups, "name")});

    // TODO: maybe this should only be on submit
    this.triggerChange(this.props.getData(groups));
  }

  componentDidUpdate(lastProps) {
    if (this.props.fields !== lastProps.fields) {
      this.setState({
        groups: keyBy(ConfigService.filterGroups(this.props.fields, this.props.filters), "name"),
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
          readonly={readonly}
        />
      </div>
    );
  }
}
