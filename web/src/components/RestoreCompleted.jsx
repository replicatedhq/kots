import { Component } from "react";
import { Link } from "react-router-dom";
import Icon from "./Icon";

export default class RestoreCompleted extends Component {
  render() {
    return (
      <div
        className="u-width--full flex-column flex1 justifyContent--center u-position--relative"
        data-testid="restore-completed-view"
      >
        <div className="u-flexTabletReflow flex1 alignItems--center">
          <div className="Text-wrapper flex justifyContent--center flex1">
            <div className="Text u-textAlign--center">
              <Icon
                icon="check-circle-filled"
                size={40}
                className="success-color"
              />
              <p className="u-fontWeight--bold u-textColor--primary u-fontSize--larger u-lineHeight--normal u-marginTop--15 u-marginBottom--10">
                Application has been restored
              </p>
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--normal">
                Your application has been restored with no warnings or errors.
                Log back in to view your application.
              </p>
              <Link
                to="/secure-console"
                className="btn primary blue u-marginTop--20"
              >
                {" "}
                Log in to dashboard{" "}
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
