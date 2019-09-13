import React from "react";
import { compose, withApollo, graphql} from "react-apollo";
import { getAirgapBundleProgress } from "../queries/AppsQueries";
import Loader from "./shared/Loader";
function AirgapUploadProgress(props) {

  const { loading, getAirgapBundleProgress } = props.data;
  if (loading) {
    return (
      <div className="flex1 flex-column alignItems--center justifyContent--center">
        <Loader size={60} />
      </div>
    );
  }
  return (
    <div className="flex1 flex-column alignItems--center justifyContent--center">
      Checking in on your airgap bundle right now...
      Progress: {getAirgapBundleProgress?.value}
    </div>
  );
}

export default compose(
  withApollo,
  graphql(getAirgapBundleProgress, {
    options: () => {
      return {
        pollInterval: 2000
      }
    }
  })
)(AirgapUploadProgress);
