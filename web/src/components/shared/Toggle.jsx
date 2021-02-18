import React from "react"
import "../../scss/components/shared/Toggle.scss"

export default class Tooltip extends React.Component {

  render() {
    const { items } = this.props;

    return (
      <div className="Toggle flex flex-auto alignItems--center">
        {items?.map((item, i) => {
          return (
            <div key={i} className={`Toggle-item ${item.isActive ? "is-active" : ""}`} onClick={item.onClick}>
              {item.title}
            </div>
          );
        })}
      </div>
    );
  }
}
