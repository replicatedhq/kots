import * as React from "react";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardActions from "@mui/material/CardActions";
import CardContent from "@mui/material/CardContent";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import Grid from "@mui/material/Grid";
import { Settings as SettingsIcon } from "@mui/icons-material";
import Chip from "@mui/material/Chip";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import LaunchIcon from "@mui/icons-material/Launch";
import { Link } from "react-router-dom";
import Divider from "@mui/material/Divider";

import "@src/scss/components/shared/NavBar.scss";

import useApps from "@features/Gitops/hooks/useApps";

export function NewDashboard() {
  const { data } = useApps();
  let app = "";
  let appLink = "";
  if (data) {
    app = data[0]?.slug;
    appLink = data[0]?.downstream.links[0].uri;
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
          <SettingsIcon sx={{ marginLeft: "10px" }} />
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
            <Chip
              icon={<CheckCircleIcon />}
              label="Ready"
              variant="filled"
              color="success"
              sx={{ marginTop: "4px" }}
            />
          </Grid>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">Deployed Version</Typography>
            <Typography variant="h6">1.0.0</Typography>
          </Grid>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">Preflight Checks</Typography>
            <Chip
              icon={<ErrorIcon />}
              label="Failed"
              variant="filled"
              color="error"
              sx={{ marginTop: "4px" }}
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
              Does not expire
            </Typography>
          </Grid>
          <Divider orientation="vertical" flexItem sx={{ paddingTop: "2px" }} />
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">App Link</Typography>
            <Box display="flex" alignItems="center" sx={{ marginTop: "3px" }}>
              <Link to={appLink}>Open Link</Link>
              <LaunchIcon
                fontSize={"small"}
                color="primary"
                sx={{ marginLeft: "2px" }}
              />
            </Box>
          </Grid>
          <Grid item container direction="column" sx={{ maxWidth: "150px" }}>
            <Typography variant="caption">New version available</Typography>
            <Chip
              label="Update"
              variant="filled"
              color="primary"
              sx={{ marginTop: "4px" }}
            />
          </Grid>
          <Grid
            xs={12}
            sx={{ marginTop: "20px" }}
            className="details-subnav"
            container
            justifyContent="center"
          >
            {/* <Divider orientation="horizontal" flexItem /> */}
            <li className="subnav-item">
              <Link to={`app/${app}/version-history`}>Version history</Link>
            </li>
            <li className="subnav-item">
              <Link to={`app/${app}/config`}>Config</Link>
            </li>
            <li className="subnav-item">
              <Link to={`app/${app}/troubleshoot`}>Troubleshoot</Link>
            </li>
            <li className="subnav-item">
              <Link to={`app/${app}/license`}>License</Link>
            </li>
            <li className="subnav-item">
              <a>View files</a>
            </li>
            <li className="subnav-item">
              <Link to={`app/${app}/registry-settings`}>Registry settings</Link>
            </li>
          </Grid>
        </Grid>
      </Card>
    </Box>
  );
}
