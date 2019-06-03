import * as React from "react";
import { compose, withApollo } from "react-apollo";
import debounce from "lodash/debounce";
import { searchWatches, searchPendingInitSessions } from "../../queries/WatchQueries";
import "../../scss/components/shared/SearchBar.scss";
import Loader from "./Loader";

export class SearchWatchesBar extends React.Component {
  constructor() {
    super();
    this.state = {
      isLoading: false
    };
    this.handleWatchSearch = debounce(this.handleWatchSearch, 400);
  }

  handleWatchSearch = async (value) => {
    this.setState({ isLoading: true });
    let watches = [];
    let pendingWatches = [];
    await this.props.client.query({
      query: searchWatches,
      variables: { watchName: value }
    })
    .then(({ data }) => {
      watches = data.searchWatches;
    })
    .catch((err) => {
      console.log(err);
      this.setState({ isLoading: false });
    });
    await this.props.client.query({
      query: searchPendingInitSessions,
      variables: { title: value }
    })
    .then(({ data }) => {
      pendingWatches = data.searchPendingInitSessions;
    })
    .catch((err) => {
      console.log(err);
      this.setState({ isLoading: false });
    })

    this.setState({ isLoading: false });
    if (typeof this.props.searchCallback === "function") {
      this.props.searchCallback(watches, pendingWatches);
    }

  }

  onSearch = (e) => {
    const { value } = e.target;
    this.handleWatchSearch(value);
  }

  render() {
    return (
      <div className="SearchBar--wrapper flex flex1 flex-column u-position--relative">
        <input 
          type="text" 
          name="watch-search" 
          className="Input"
          placeholder="Search watches"
          onChange={(e) => this.onSearch(e)} 
        />
        {this.state.isLoading && <Loader size="25" />}
      </div>
    );
  }
}

export default compose(
  withApollo,
)(SearchWatchesBar);
