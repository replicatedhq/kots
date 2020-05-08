import * as React from "react";
import { Link, withRouter } from "react-router-dom";
import { getBuildVersion } from "../../utilities/utilities";
import "../../scss/components/shared/Footer.scss";

export class Footer extends React.Component {

  getItems() {
    return [
      {
        label: "Terms",
        href: "https://www.replicated.com/terms",
      },
      {
        label: "Privacy",
        href: "https://www.replicated.com/privacy",
      },
      {
        label: getBuildVersion()
      }
    ];
  }

  render() {
    const footerItems = this.getItems();
    return (
      <div className={`FooterContent-wrapper flex flex-auto justifyContent--center ${this.props.className || ""}`}>
        <div className="container flex1 flex">
          <div className="flex flex1 justifyContent--center">
            {footerItems.filter(item => item).map((item, i) => {
              let node = (
                <span className="FooterItem">{item.label}</span>
              );
              if (item.linkTo) {
                node = (
                  <Link to={item.linkTo} target="_blank" rel="noopener noreferrer" className="FooterItem">{item.label}</Link>
                );
              } else if (item.href) {
                node = (
                  <a href={item.href} target="_blank" rel="noopener noreferrer" className="FooterItem">{item.label}</a>
                );
              }
              return (
                <div key={i} className="FooterItem-wrapper">
                  {node}
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
