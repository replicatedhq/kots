import { Component } from "react";
import { Link } from "react-router-dom";

export default class NotFound extends Component {
  render() {
    return (
      <div className="u-width--full flex-column flex1 justifyContent--center u-position--relative">
        <div className="u-flexTabletReflow flex1 alignItems--center">
          <div className="Text-wrapper flex justifyContent--center flex1">
            <div className="Text u-textAlign--center">
              <p className="u-fontSize--largest u-fontWeight--light u-textColor--primary u-lineHeight--default">
                Error 404
              </p>
              <p className="u-marginTop--30 u-textColor--bodyCopy u-fontSize--large u-lineHeight--normal">
                Oops, we couldn't find the page you were looking for
              </p>
              <div className="u-marginTop--30">
                <Link to="/" className="btn primary large">
                  Take me home
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
