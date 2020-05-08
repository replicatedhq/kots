import * as React from "react";

export default class UnsupportedBrowser extends React.Component {
  render() {
    return (
      <div className="u-width--full flex-column flex1 justifyContent--center u-position--relative">
        <div className="u-flexTabletReflow flex1 alignItems--center">
          <div className="Text-wrapper flex justifyContent--center flex1">
            <div className="Text u-textAlign--center">
              <p className="u-fontSize--largest u-fontWeight--light u-color--tuna u-lineHeight--default">
                Unsupported browser
              </p>
              <p className="u-marginTop--30 u-color--dustyGray u-fontSize--large u-lineHeight--normal">
                Oops, this browser is either unsupported or you need to enable cookies and web storage.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
