import * as React from "react";
import { Ship as ShipInit } from "@replicatedhq/ship-init";
import PropTypes from "prop-types";

class Ship extends React.Component {
  static propTypes = {
    history: PropTypes.object.isRequired,
    rootURL: PropTypes.string.isRequired,
    initSessionId: PropTypes.string,
    onCompletion: PropTypes.func.isRequired,
  }

  render() {
    const { url } = this.props.match;
    const {
      history,
      rootURL,
      initSessionId,
      onCompletion,
    } = this.props;

    if (!initSessionId) {
      return "Unable to access Ship Cloud API instance, please try again";
    }

    const shipApiEndpoint = `${rootURL}${initSessionId}/api/v1`
    return (
      <ShipInit
        apiEndpoint={shipApiEndpoint}
        basePath={url}
        history={history}
        onCompletion={onCompletion}
        stepsEnabled={true}
      />
    );
  }
}

export default Ship;
