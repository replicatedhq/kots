import React from "react";
import { useOutletContext } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";

class dumbComponent extends React.Component {
  render() {
    console.log(this.props);
    return <div>dumbComponent </div>;
  }
}

export default withRouter(dumbComponent);
