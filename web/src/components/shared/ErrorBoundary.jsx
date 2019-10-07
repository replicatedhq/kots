import React from "react";

export default class ErrorBoundaryComponent extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      hasError: false,
      message: "",
      sendingBugsnagMessage: false
    }
  }

  componentDidCatch(error, errorInfo) {
    this.setState({
      error,
      errorInfo,
      hasError: true
    });
  }

  handleFormChange = (field, e) => {
    let nextState = {};
    nextState[field] = e.target.value;
    this.setState(nextState);
  }


  backToShip = () => {
    window.location.replace("/");
  }

  render() {
    const { info } = this.props;

    if (this.state.hasError || info) {
      return (
        <div className="flex1 flex-column u-overflow--auto">
          <div className="flex1 flex-column justifyContent--center alignItems--center">
            <div className="flex-column u-width--half">
              <span
                className="icon errorWarningIcon u-marginBottom--20 u-textAlign--center alignSelf--center"
              ></span>
              <p className="u-textAlign--center alignItems--center u-fontSize--header2 u-fontWeight--bold u-color--tundora">Oops, something went wrong.</p>
              <p className=" u-textAlign--center u-marginTop--20 u-fontWeight--medium u-color--dustyGray u-fontSize--normal u-lineHeight--normal"> We’ve notified our team of the error that occurred and will be working to resolve the issue.
              We apologize for any inconvenience this may have caused. </p>
              <div className="flex alignItems--center alignSelf--center u-marginTop--20">
                <button className="btn secondary gray" onClick={() => this.backToShip()}> Back to Ship </button>
              </div>
            </div>
          </div>
        </div>
      );
    } else {
      return (
        <div className="flex-column flex1">
          {this.props.children}
        </div>
      );
    }
  }
}