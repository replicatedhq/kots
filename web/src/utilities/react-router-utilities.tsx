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
  const wrappedMatch = useRouteMatch();

  return children({ history, location, wrappedMatch });
};

/**
 * @deprecated The method should not be used on new components
 */
function withRouter(
  WrappedComponent: React.ComponentType<
    RouteComponentProps | { wrappedMatch: match }
  >
) {
  return class extends React.Component {
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
}

export { withRouter };
