import { Component } from "react";
import "../../scss/components/shared/Avatar.scss";

export default class Avatar extends Component {
  render() {
    return (
      <div
        className="avatar-wrapper"
        style={{ backgroundImage: `url(${this.props.imageUrl || ""})` }}
      ></div>
    );
  }
}
