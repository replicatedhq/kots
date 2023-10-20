import { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";

class dumbComponent extends Component {
  render() {
    console.log(this.props);
    return <div>dumbComponent </div>;
  }
}

export default withRouter(dumbComponent);
