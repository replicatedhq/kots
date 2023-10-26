import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

// import NotFound from "../static/NotFound";
// import SubNavBar from "@src/components/shared/SubNavBar";
// import ConfigureIngress from "@src/components/identity/ConfigureIngress";
import IdentityProviders from "@src/components/identity/IdentityProviders";

import "@src/scss/components/identity/IdentityManagement.scss";

type Props = {
  isKurlEnabled: boolean;
  isGeoaxisSupported: boolean;
};

const Access = (props: Props) => {
  const navigate = useNavigate();
  // TODO: move this into a redirect route or update links to default to /identity-providers
  useEffect(() => {
    if (location.pathname === "/access") {
      navigate(`/access/identity-providers`, { replace: true });
      return;
    }
  }, []);

  return (
    <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
      <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
        {/* TODO ===> THIS WILL COME LATER */}
        {/* <Fragment>
            <SubNavBar
              className="flex"
              isAccess={true}
              activeTab={match.params.tab}
            />
            <Switch>
              <Route exact path="/access/configure-ingress" render={() =>
                <ConfigureIngress />
              } />
              <Route exact path="/access/identity-providers" render={() =>
                <IdentityProviders isKurlEnabled={this.props.isKurlEnabled} />
              } />
              <Route component={NotFound} />
            </Switch>
          </Fragment> */}
        <IdentityProviders
          isKurlEnabled={props.isKurlEnabled}
          isGeoaxisSupported={props.isGeoaxisSupported}
        />
      </div>
    </div>
  );
};

export default Access;
