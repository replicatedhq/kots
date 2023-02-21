import { KotsParams } from "@types";
import React from "react";
import {
  match,
  RouteComponentProps,
  useHistory,
  useLocation,
  useRouteMatch,
} from "react-router";

// @ts-ignore
const RouterWrapper = ({ children }) => {
  const history = useHistory();
  const location = useLocation();
  const wrappedMatch = useRouteMatch<KotsParams>();

  return children({ history, location, wrappedMatch });
};

/**
 * @deprecated The method should not be used on new components. New components should use the hooks directly.
 */
const withRouter = <P extends object>(
  WrappedComponent: React.ComponentType<
    RouteComponentProps | { wrappedMatch: match } | P
  >
) => {
  return class extends React.Component<P> {
    render() {
      return (
        <RouterWrapper
          // @ts-ignore
          children={({ history, location, wrappedMatch }) => {
            return (
              <WrappedComponent
                history={history}
                location={location}
                match={wrappedMatch}
                wrappedMatch={wrappedMatch}
                {...this.props}
              />
            );
          }}
        />
      );
    }
  };
};

export type withRouterType = RouteComponentProps<KotsParams> & {
  wrappedMatch: match<KotsParams>;
};

export { withRouter };
