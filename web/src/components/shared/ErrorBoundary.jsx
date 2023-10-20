import { Component } from "react";
import Icon from "../Icon";

export default class ErrorBoundaryComponent extends Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      hasError: false,
      message: "",
      sendingBugsnagMessage: false,
    };
  }

  componentDidCatch(error, errorInfo) {
    this.setState({
      error,
      errorInfo,
      hasError: true,
    });
  }

  handleFormChange = (field, e) => {
    let nextState = {};
    nextState[field] = e.target.value;
    this.setState(nextState);
  };

  backToAdminConsole = () => {
    window.location.replace("/");
  };

  render() {
    const { info } = this.props;

    if (this.state.hasError || info) {
      return (
        <div className="flex1 flex-column u-overflow--auto">
          <div className="flex1 flex-column justifyContent--center alignItems--center">
            <div className="flex-column u-width--half">
              <Icon
                icon="warning"
                size={40}
                className="warning-color u-marginBottom--20 u-textAlign--center alignSelf--center"
              />
              <p className="u-textAlign--center alignItems--center u-fontSize--header2 u-fontWeight--bold u-textColor--secondary">
                Oops, something went wrong.
              </p>
              <p className=" u-textAlign--center u-marginTop--20 u-fontWeight--medium u-textColor--bodyCopy u-fontSize--normal u-lineHeight--normal">
                {" "}
                Click the button below to try again.{" "}
              </p>
              <div className="flex alignItems--center alignSelf--center u-marginTop--20">
                <button
                  className="btn secondary"
                  onClick={() => this.backToAdminConsole()}
                >
                  {" "}
                  Back to Admin Console{" "}
                </button>
              </div>
            </div>
          </div>
        </div>
      );
    } else {
      return <div className="flex-column flex1">{this.props.children}</div>;
    }
  }
}
