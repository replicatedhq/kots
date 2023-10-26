import { Component, Fragment } from "react";
import classNames from "classnames";

import Loader from "@src/components/shared/Loader";
import "@src/scss/components/shared/SideBar.scss";

interface Props {
  className?: string;
  items: (JSX.Element | undefined)[];
  loading?: boolean;
}
class SideBar extends Component<Props> {
  static defaultProps = {
    items: [],
  };

  render() {
    const { className, items, loading } = this.props;

    if (loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center u-minHeight--full sidebar">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div
        className={classNames(
          "sidebar flex-column flex-auto u-overflow--auto",
          className
        )}
      >
        <div className="flex-column u-width--full">
          {items?.map((jsx, idx) => {
            return <Fragment key={idx}>{jsx}</Fragment>;
          })}
        </div>
      </div>
    );
  }
}

export default SideBar;
