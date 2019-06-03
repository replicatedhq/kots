import * as React from "react";
import { Link, withRouter } from "react-router-dom";

import "../../scss/components/shared/Footer.scss";

export class Footer extends React.Component {

  getItems() {
    return [
      {
        label: "Terms",
        linkTo: "/terms",
      },
      {
        label: "Privacy",
        linkTo: "/privacy",
      }
    ];
  }

  render() {
    const footerItems = this.getItems();
    return (
      <div className={`FooterContent-wrapper flex flex-auto justifyContent--center ${this.props.className || ""}`}>
        <div className="container flex1 flex">
          <div className="flex flex-auto">
            <div className="FooterItem-wrapper">
              <span className="FooterItem">&#169; {new Date().getFullYear()}, Replicated, Inc. All Rights Reserved.</span>
            </div>
          </div>
          <div className="flex flex1 justifyContent--flexEnd">
            {footerItems.filter(item => item).map((item, i) => {
              return (
                <div key={i} className="FooterItem-wrapper">
                  {item.linkTo
                    ? <Link to={item.linkTo} target="_blank" rel="noopener noreferrer" className="FooterItem">{item.label}</Link>
                    : item.href ?
                      <a href={item.href} target="_blank" rel="noopener noreferrer" className="FooterItem">{item.label}</a>
                      : <span className="FooterItem">{item.label}</span>
                  }
                </div>
              );
            })}
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(Footer);
