import { Children, Component, Fragment } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";

import "@src/scss/components/shared/TabView.scss";

export default class TabView extends Component {
  constructor(props) {
    super(props);
    const { children, initialTab } = props;
    const tabToDisplay = initialTab || children[0].props.name;

    this.state = {
      currentTab: tabToDisplay,
    };
  }

  static propTypes = {
    children: PropTypes.oneOfType([
      PropTypes.element,
      PropTypes.arrayOf(PropTypes.element),
    ]),
    separator: PropTypes.oneOfType([PropTypes.string, PropTypes.element]),
    onTabChange: PropTypes.func,
  };

  static defaultProps = {
    separator: "|",
    onTabChange: () => {},
  };

  setTab = (name) => {
    const { onTabChange } = this.props;

    this.setState(
      {
        currentTab: name,
      },
      () => onTabChange(name)
    );
  };

  render() {
    const { className, children, separator } = this.props;
    const { currentTab } = this.state;
    const childToRender = Children.toArray(children).find(
      (child) => child.props.name === currentTab
    );
    return (
      <div className={classNames("tabview", className)}>
        <div className="tabview-tabwrapper">
          {Children.map(children, (child, idx) => {
            const { displayText, name } = child.props;
            return (
              <Fragment key={name}>
                <span
                  className={classNames(
                    "tabview-tabname u-cursor--pointer u-fontSize--small",
                    {
                      selected: name === currentTab,
                    }
                  )}
                  onClick={() => {
                    this.setTab(name);
                  }}
                >
                  {displayText}
                </span>
                {idx + 1 !== children.length && separator}
              </Fragment>
            );
          })}
        </div>
        {childToRender}
      </div>
    );
  }
}
