import * as React from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";

import { listSupportBundles } from "../../queries/TroubleshootQueries";

import Loader from "../shared/Loader";
import SupportBundleRow from "./SupportBundleRow";
import GenerateSupportBundle from "./GenerateSupportBundle";
import "../../scss/components/troubleshoot/SupportBundleList.scss";

class SupportBundleList extends React.Component {
  render() {
    const { watch } = this.props;
    const { loading, error, listSupportBundles } = this.props.listSupportBundles;

    const appTitle = watch.watchName || watch.name;
    const downstreams = watch.downstreams || [];

    if (error) {
      return <p>{error.message}</p>;
    }

    if (loading) {
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
      );
    }

    let bundlesNode;
    if (downstreams.length) {
      if (listSupportBundles?.length) {
        bundlesNode = (
          listSupportBundles.map(bundle => (
            <SupportBundleRow
              key={bundle.id}
              bundle={bundle}
              watchSlug={watch.slug}
            />
          ))
        );
      } else {
        return (
          <GenerateSupportBundle
            watch={watch}
          />
        );
      }
    }

    return (
      <div className="container u-paddingBottom--30 u-paddingTop--30 flex1 flex">
        <Helmet>
          <title>{`${appTitle} Troubleshoot`}</title>
        </Helmet>
        <div className="flex1 flex-column">
          <div className="flex flex1">
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-paddingBottom--10 flex">
                <div className="flex flex1">
                  <div className="flex1 u-flexTabletReflow">
                    <div className="flex flex1">
                      <div className="flex-auto alignSelf--center">
                        <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna flex alignContent--center">Support bundles</h2>
                      </div>
                    </div>
                    <div className="RightNode flex-auto flex-column flex-verticalCenter u-position--relative">
                      <Link to={`${this.props.match.url}/generate`} className="btn secondary">Generate a support bundle</Link>
                    </div>
                  </div>
                </div>
              </div>
              <div className={`${downstreams.length ? "flex1 flex-column u-overflow--auto" : ""}`}>
                {bundlesNode}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  graphql(listSupportBundles, {
    name: "listSupportBundles",
    options: props => {
      return {
        variables: {
          watchSlug: props.watch.slug
        },
        fetchPolicy: "no-cache",
      }
    }
  })
  // graphql(archiveSupportBundle, {
  //   props: ({ mutate }) => ({
  //     archiveSupportBundle: (id) => mutate({ variables: { id } })
  //   })
  // }),
)(SupportBundleList));