import React, { ComponentType } from "react";
import {
  useNavigate,
  useLocation,
  useParams,
  useOutletContext,
} from "react-router-dom";
export interface RouterProps {
  location: ReturnType<typeof useLocation>;
  navigate: ReturnType<typeof useNavigate>;
  params: ReturnType<typeof useParams>;
  outletContext: ReturnType<typeof useOutletContext>;
}

export function withRouter<TProps extends RouterProps>(
  Component: ComponentType<TProps>
) {
  function ComponentWithRouterProp(props: TProps) {
    let location = useLocation();
    let navigate = useNavigate();
    let params = useParams();
    const outletContext = useOutletContext();
    return (
      <Component
        {...props}
        location={location}
        navigate={navigate}
        params={params}
        outletContext={outletContext}
      />
    );
  }

  return ComponentWithRouterProp;
}
