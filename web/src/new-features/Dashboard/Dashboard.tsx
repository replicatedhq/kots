import React, { useState } from "react";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Grid from "@mui/material/Grid";
import { Settings as SettingsIcon } from "@mui/icons-material";
import Chip from "@mui/material/Chip";
import LaunchIcon from "@mui/icons-material/Launch";
import { useParams } from "react-router-dom";
import Divider from "@mui/material/Divider";
import Modal from "react-modal";
import sortBy from "lodash/sortBy";
import { KotsParams, ResourceStates, Version } from "@types";
import { Paragraph } from "@src/styles/common";
import { Utilities, getPreflightResultState } from "@src/utilities/utilities";
import { PreflightStatusChip, StatusChip } from "./Chips";
import { getAppsList } from "./utils";

import { Link as MuiLink } from "@mui/material";
// import AppLicense from "@components/apps/AppLicense";

// import "@src/scss/components/shared/NavBar.scss";

import { useSelectedAppClusterDashboardWithIntercept } from "@features/Dashboard/api/useSelectedAppClusterDashboard";
import { useApps, useSelectedApp } from "@features/App";
// import GenerateSupportBundle from "@components/troubleshoot/GenerateSupportBundle";
// import AppVersionHistory from "@components/apps/AppVersionHistory";
// import PreflightResultPage from "@components/PreflightResultPage";
// import AppConfig from "../../features/AppConfig/components/AppConfig";

// interface TabPanelProps {
//   children?: React.ReactNode;
//   index: string;
//   value: string;
// }

function Dashboard() {
  const [value, setValue] = useState("");
  const [version, setVersion] = useState(1);
  const [isNewVersionAvailable, setIsNewVersionAvailable] = useState(true);

  const onUpdateVersion = () => {
    setTimeout(() => {
      setVersion(version + 1);
      setIsNewVersionAvailable(false);
    }, 2000);

    setTimeout(() => {
      setIsNewVersionAvailable(true);
    }, 10000);
  };
  const handleChange = (newValue: string) => {
    setValue(newValue);
  };
  const { refetch: refetchApps } = useApps();
  const params = useParams<KotsParams>();

  // function TabPanel(props: TabPanelProps) {
  //   const { children, value, index, ...other } = props;
  //   return (
  //     <div
  //       role="tabpanel"
  //       hidden={value !== index}
  //       id={`simple-tabpanel-${index}`}
  //       aria-labelledby={`simple-tab-${index}`}
  //       {...other}
  //     >
  //       {value === index && (
  //         <Box sx={{ p: 3 }}>
  //           <Typography>{children}</Typography>
  //         </Box>
  //       )}
  //     </div>
  //   );
  // }
  const getAppResourcesByState = () => {
    if (!appStatus?.resourceStates?.length) {
      return {};
    }

    const resourceStates = appStatus?.resourceStates;
    const statesMap: {
      [key: string]: ResourceStates[];
    } = {};

    for (let i = 0; i < resourceStates.length; i++) {
      const resourceState = resourceStates[i];
      if (!statesMap.hasOwnProperty(resourceState.state)) {
        statesMap[resourceState.state] = [];
      }
      statesMap[resourceState.state].push(resourceState);
    }

    // sort resources so that the order doesn't change while polling (since we show live data)
    Object.keys(statesMap).forEach((stateKey) => {
      statesMap[stateKey] = sortBy(statesMap[stateKey], (resource) => {
        const fullResourceName = `${resource?.namespace}/${resource?.kind}/${resource?.name}`;
        return fullResourceName;
      });
    });

    // sort the available states to show them in the correct order
    const allStates = Object.keys(statesMap);
    const sortedStates = sortBy(allStates, (s) => {
      if (s === "failed") {
        return 1;
      }
      if (s === "unavailable") {
        return 2;
      }
      if (s === "degraded") {
        return 3;
      }
      if (s === "updating") {
        return 4;
      }
      if (s === "success") {
        return 5;
      }
    });

    return {
      statesMap,
      sortedStates,
    };
  };
  const getPreflightState = (version: Version) => {
    let preflightsFailed = false;
    let preflightState = "";
    if (version?.preflightResult) {
      const preflightResult = JSON.parse(version.preflightResult);
      preflightState = getPreflightResultState(preflightResult);
      preflightsFailed = preflightState === "fail";
    }
    return {
      preflightsFailed,
      preflightState,
      preflightSkipped: version?.preflightSkipped,
    };
  };
  const createDashboardActionLink = (uri: string) => {
    try {
      const parsedUrl = new URL(uri);
      if (parsedUrl.hostname === "localhost") {
        parsedUrl.hostname = window.location.hostname;
      }
      return parsedUrl.href;
    } catch (error) {
      return "";
    }
  };

  const selectedApp = useSelectedApp();
  const { data: selectedAppClusterDashboardResponse } =
    useSelectedAppClusterDashboardWithIntercept();
  const [isDetailsOpen, setIsDetailsOpen] = React.useState(false);
  const appStatus = selectedAppClusterDashboardResponse?.appStatus;
  const appResourcesByState = getAppResourcesByState();

  let app = "";
  let appLink = "";
  if (selectedApp) {
    app = selectedApp.slug;
    appLink = createDashboardActionLink(selectedApp.downstream.links[0].uri);
  }

  let checksStatusText;
  if (selectedApp) {
    const preflightState = getPreflightState(
      selectedApp?.downstream?.currentVersion
    );
    if (preflightState.preflightsFailed) {
      checksStatusText = "Failed";
    } else if (preflightState.preflightState === "warn") {
      checksStatusText = "Warning";
    } else {
      checksStatusText = "Passed";
    }
  }

  return (
    <Box sx={{ margin: "2rem 6rem" }}>
      <Box display="flex" alignItems="center" justifyContent="space-between">
        <Box display="flex" alignItems="center">
          <img
            src="https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/icon/color/kubernetes-icon-color.png"
            width="50"
          />
          <Typography variant="h4" sx={{ marginLeft: "10px" }}>
            App Name
          </Typography>
        </Box>
        <Box display="flex" alignItems="center">
          <Button variant="contained">Redeploy</Button>
          <MuiLink onClick={() => handleChange("config")}>
            <SettingsIcon sx={{ marginLeft: "10px" }} />
          </MuiLink>
        </Box>
      </Box>

      <Card
        sx={{
          paddingLeft: "1rem",
          paddingBottom: 0,
          marginTop: "1rem",
          display: "flex",
          alignItems: "center",
        }}
      >
        <Grid container spacing={2} sx={{ marginTop: 0 }}>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">App Status</Typography>
            <StatusChip
              label={appStatus?.state}
              onClick={() => setIsDetailsOpen(!isDetailsOpen)}
            />
          </Grid>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">Deployed Version</Typography>
            <MuiLink onClick={() => handleChange("version-history")}>
              <Typography variant="h6">0.0.{version}</Typography>
            </MuiLink>
          </Grid>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">Preflight Checks</Typography>
            <PreflightStatusChip
              status={checksStatusText}
              onClick={() => handleChange("preflight")}
            />
          </Grid>
          <Grid
            item
            container
            direction="column"
            sx={{ marginRight: "10px", maxWidth: "150px" }}
          >
            <Typography variant="caption">License</Typography>
            <Typography variant="body1" sx={{ marginTop: "3px" }}>
              <MuiLink onClick={() => handleChange("license")}>
                Does not expire
              </MuiLink>
            </Typography>
          </Grid>
          <Divider orientation="vertical" flexItem sx={{ paddingTop: "2px" }} />
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">App Link</Typography>
            <Box display="flex" alignItems="center" sx={{ marginTop: "3px" }}>
              <a target="_blank" href={appLink}>
                Open Link
              </a>
              <LaunchIcon
                fontSize={"small"}
                color="primary"
                sx={{ marginLeft: "2px" }}
              />
            </Box>
          </Grid>
          {isNewVersionAvailable && (
            <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
              <Typography variant="caption">New version available</Typography>
              <Chip
                label="Deploy"
                variant="filled"
                color="primary"
                sx={{ marginTop: "4px" }}
                onClick={() => {
                  onUpdateVersion();
                }}
              />
            </Grid>
          )}
          <Box sx={{ width: "100%", marginTop: "20px" }}>
            {/* <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
              <Tabs
                value={value}
                onChange={handleChange}
                aria-label="basic tabs example"
              >
                <Tab label="Item One" {...a11yProps(0)} />
                <Tab label="Item Two" {...a11yProps(1)} />
                <Tab label="Item Three" {...a11yProps(2)} />
              </Tabs>
            </Box> */}
          </Box>
        </Grid>
      </Card>
      <Box>
        {/* {value === "config" && (
          <AppConfig
            //   {...props}
            app={selectedApp}
            refreshAppData={refetchApps}
            fromLicenseFlow={true}
            refetchAppsList={getAppsList}
          />
        )} */}
        {/* {value === "version-history" && (
          <AppVersionHistory
            app={selectedApp}
            match={{ match: { params: params } }}
            refreshAppData={refetchApps}
          />
        )} */}
        {/* {value === "troubleshoot" && (
          <GenerateSupportBundle watch={selectedApp} />
        )} */}
        {/* {value === "preflight" && (
          <PreflightResultPage
            logo={""}
            fromLicenseFlow={true}
            refetchAppsList={getAppsList}
          />
        )} */}
        {/* <TabPanel value={value} index={"license"}> */}
        {/* {value === "license" && (
          <AppLicense
            app={selectedApp}
            // syncCallback={refetchData}
            // changeCallback={refetchData}
            //isHelmManaged={props.isHelmManaged}
          />
        )} */}
        {/* </TabPanel> */}
      </Box>
      <Modal
        isOpen={isDetailsOpen}
        onRequestClose={() => setIsDetailsOpen(!isDetailsOpen)}
        ariaHideApp={false}
        className="Modal DefaultSize"
      >
        <div className="Modal-body">
          <Paragraph size="16" weight="bold">
            Resource status
          </Paragraph>
          <div
            className="u-marginTop--10 u-marginBottom--10 u-overflow--auto"
            style={{ maxHeight: "50vh" }}
          >
            {appResourcesByState?.sortedStates?.map((sortedState, i) => (
              <div key={i}>
                <p className="u-fontSize--normal u-color--mutedteal u-fontWeight--bold u-marginTop--20">
                  {Utilities.toTitleCase(sortedState)}
                </p>
                {appResourcesByState?.statesMap[sortedState]?.map(
                  (resource, j) => (
                    <div key={`${resource?.name}-${j}`}>
                      <p
                        className={`ResourceStateText status-tag u-fontSize--normal ${resource.state}`}
                      >
                        {resource?.namespace}/{resource?.kind}/{resource?.name}
                      </p>
                    </div>
                  )
                )}
              </div>
            ))}
          </div>
          <div className="flex alignItems--center u-marginTop--30">
            <button
              type="button"
              className="btn primary"
              onClick={() => setIsDetailsOpen(!isDetailsOpen)}
            >
              Ok, got it!
            </button>
            <button
              type="button"
              className="btn secondary blue u-marginLeft--10"
              onClick={() => {
                handleChange("troubleshoot");
                setIsDetailsOpen(!isDetailsOpen);
              }}
            >
              Troubleshoot
            </button>
          </div>
        </div>
      </Modal>
    </Box>
  );
}

export { Dashboard };