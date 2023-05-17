import React from "react";
import {
  useNavigate,
  useLocation,
  useParams,
  useOutletContext,
} from "react-router-dom";

/**
 * @deprecated The method should not be used on new components. New components should use the hooks directly.
 */
export function withRouter(Component: JSX.IntrinsicAttributes) {
  function ComponentWithRouterProp(props: JSX.IntrinsicAttributes) {
    let location = useLocation();
    let navigate = useNavigate();
    let params = useParams();
    const outletContext = useOutletContext();
    return (
      <Component
        {...props}
        router={{ location, navigate, params }}
        outletContext={outletContext}
      />
    );
  }

  return ComponentWithRouterProp;
}
