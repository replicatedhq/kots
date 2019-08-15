import React from "react";
import Helmet from "react-helmet";

import Loader from "../shared/Loader";
import { Link } from "react-router-dom";

import CodeSnippet from "@src/components/shared/CodeSnippet";
import "../../scss/components/image_check/ImageWatchBatch.scss";

export default class ClusterScope extends React.Component {

  componentDidMount() {
    const script = document.createElement("script");

    script.id = "asciicast-262264";
    script.src = "https://asciinema.org/a/262264.js";
    script.async = true;
    script.setAttribute("data-autoplay", true);
    script.setAttribute("data-loop", "1");

    document.getElementById("asciinema-player").appendChild(script);

  }

  render() {
    return (
      <div className="Login-wrapper  flex-column flex1 u-overflow--hidden justifyContent--center">
        <Helmet>
          <title>kubectl outdated - A plugin to show out-of-date images running in a cluster</title>
        </Helmet>
        <div className="ClusterScopePage--wrapper u-overflow--auto">
          <div className="flex1 flex-column container">
            <div className="u-flexTabletReflow flex1 justifyContent--center u-paddingTop--30 u-paddingBottom--30">
              <div className="flex-column flex1 left-block-wrapper">
                <div className="flex-column">
                  <div className="icon kub-logo u-marginBottom--20"></div>
                  <div className="flex">
                    <p className="u-fontSize--header2 u-fontWeight--bold u-fontFamily--monaco u-color--tuna u-lineHeight--more">kubectl outdated</p>
                    <span className="u-marginLeft--10 flex-column justifyContent--center">
                      <a className="github-button" href="https://github.com/replicatedhq/outdated" data-size="large" data-show-count="true" aria-label="Star replicatedhq/outdated on GitHub">Star</a>
                    </span>
                  </div>
                  <p className="u-fontSize--larger u-color--tundora u-fontWeight--normal u-lineHeight--more u-marginTop--10">A kubectl plugin to show out-of-date images running in a cluster. Simply run the following commands from your workstation.</p>
                  <div className="u-marginTop--20">
                    <CodeSnippet language="bash" canCopy={false}>
                      {`kubectl krew install outdated`}
                      {`kubectl outdated`}
                    </CodeSnippet>
                  </div>
                  <p className="u-fontSize--large u-color--dustyGray u-fontWeight--normal u-lineHeight--more u-marginTop--20">
                    The plugin will scan for all pods in all namespaces that you have at least read access to. It will then connect to the registry that hosts the image, and (if there's permission), it will analyze your tag to the list of current tags.
                  </p>
                  <p className="u-fontSize--large u-color--dustyGray u-fontWeight--normal u-lineHeight--more u-marginTop--20">
                    The output is a list of all images, with the most out-of-date images in red, slightly outdated in yellow, and up-to-date in green.
                  </p>
                  <div className="u-marginTop--20">
                    <Link to="/login" className="btn primary">Update images using kotsadm</Link>
                  </div>
                </div>
              </div>
              <div className="flex-column flex1 justifyContent--center right-block-wrapper">
                <div className="iframe-placeholder">
                  <Loader className="ascii-loader" color="#ffffff" size="60" />
                  <div id="asciinema-player" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
