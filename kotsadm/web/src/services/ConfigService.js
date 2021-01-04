// @TODO: Refactor this so its not using so much lodash
// the "without" module throws if we import every lodash util piecemeal
import _, { get, isEmpty, some, has } from "lodash";

export const ConfigService = {
  getItems(groups) {
    return _(groups)
      .map((group) => {
        return _(get(group, "items", []))
          .map((item) => {
            if (!isEmpty(item)) {
              if (item.type === "select_many") {
                return _(get(item, "items", []))
                  .map((childItem) => {
                    if (!isEmpty(childItem)) {
                      return childItem;
                    }
                  })
                  .value();
              }
              return item;
            }
          })
          .value();
      })
      .flattenDeep()
      .without(null)
      .value();
  },

  getItem(groups, itemName) {
    let item = null;
    some(ConfigService.getItems(groups), (otherItem) => {
      if (otherItem.name === itemName) {
        item = otherItem;
        return true;
      }
    });
    return item;
  },

  evaluateWhen(groups, when) {
    const expanded = ConfigService.expandWhen(when);
    if (!expanded.key) {
      return true;
    }
    const theItem = ConfigService.getItem(groups, expanded.key);
    if (!theItem) {
      return true;
    }
    // recursively evaluate whens
    if (theItem.when && !ConfigService.evaluateWhen(groups, theItem.when)) {
      return false;
    }
    let value = get(theItem, "value");
    value = isEmpty(value) ? theItem.default : value;
    return (value === expanded.value) !== expanded.negate;
  },

  filterGroups(groups, filters) {
    return _(groups)
      .map((group) => {
        if (!ConfigService.evaluateFilters(get(group, "filters"), filters)) {
          return null;
        }
        group.items = _(get(group, "items", []))
          .map((item) => {
            if (!ConfigService.evaluateFilters(get(item, "filters"), filters)) {
              return null;
            }
            return item;
          })
          .without(null)
          .value();
        return group;
      })
      .without(null)
      .value();
  },

  evaluateFilters(assertions, filters) {
    return !some(assertions, (when) => {
      const expanded = ConfigService.expandWhen(when);
      if (has(filters, expanded.key)) {
        const values = expanded.value.split(",");
        return !(expanded.negate !== some(values, (value) => {
          return filters[expanded.key] === value;
        }));
      }
      return false;
    });
  },

  expandWhen(when) {
    let expanded = {
      key: "",
      value: "",
      negate: false,
    };
    if (!when || typeof when !== "string") {
      return expanded;
    }
    const parts = when.split("=");
    if (parts.length < 2) {
      return expanded;
    }
    expanded.key = parts.shift();
    expanded.value = parts.join("=");
    if (expanded.key.substr(expanded.key.length - 1) === "!") {
      expanded.key = expanded.key.substr(0, expanded.key.length - 1);
      expanded.negate = true;
    }
    return expanded;
  },

  isVisible(groups, obj) {
    return !obj.hidden && obj.when !== "false" && ConfigService.isEnabled(groups, obj);
  },

  isEnabled(groups, obj) {
    const when = get(obj, "when");
    return when ? ConfigService.evaluateWhen(groups, when) : true;
  },
};
