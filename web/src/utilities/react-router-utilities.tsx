import { useSelectedApp } from "@features/App";
import useApps from "@features/Gitops/hooks/useApps";
import React, { ComponentType } from "react";
import {
  useNavigate,
  useLocation,
  useParams,
  useOutletContext,
} from "react-router-dom";
import { KotsParams } from "@types";
export interface RouterProps {
  location: ReturnType<typeof useLocation>;
  navigate: ReturnType<typeof useNavigate>;
  params: KotsParams;
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
    const selectedApp = useSelectedApp();
    const { refetch: refetchApps } = useApps();
    return (
      <Component
        {...props}
        location={location}
        navigate={navigate}
        params={params}
        outletContext={outletContext}
        app={selectedApp}
        refetchApps={refetchApps}
      />
    );
  }

  return ComponentWithRouterProp;
}
