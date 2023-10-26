import { Component } from "react";
export default class UnsupportedBrowser extends Component {
  render() {
    return (
      <div className="u-width--full flex-column flex1 justifyContent--center u-position--relative">
        <div className="u-flexTabletReflow flex1 alignItems--center">
          <div className="Text-wrapper flex justifyContent--center flex1">
            <div className="Text u-textAlign--center">
              <p className="u-fontSize--largest u-fontWeight--light u-textColor--primary u-lineHeight--default">
                Unsupported browser
              </p>
              <p className="u-marginTop--30 u-textColor--bodyCopy u-fontSize--large u-lineHeight--normal">
                Oops, this browser is either unsupported or you need to enable
                cookies and web storage.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
