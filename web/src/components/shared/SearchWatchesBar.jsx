import * as React from "react";
import { compose, withApollo } from "react-apollo";
import debounce from "lodash/debounce";
import { searchWatches } from "../../queries/WatchQueries";
import "../../scss/components/shared/SearchBar.scss";

export class SearchWatchesBar extends React.Component {
  constructor() {
    super();
    this.state = {};
    this.handleWatchSearch = debounce(this.handleWatchSearch, 500);
  }

  handleWatchSearch = (value) => {
    this.props.client.query({
      query: searchWatches,
      variables: { watchName: value }
    })
      .then(({ data }) => {
        if(typeof this.props.searchCallback === "function") {
          this.props.searchCallback(data.searchWatches);
        }
      })
      .catch((err) => {
        console.log(err);
      });
  }

  onSearch = (e) => {
    const { value } = e.target;
    this.handleWatchSearch(value);
  }

  render() {
    return (
      <div className="SearchBar--wrapper flex flex1 flex-column">
        <input 
          type="text" 
          name="watch-search" 
          className="Input"
          placeholder="Search watches"
          onChange={(e) => this.onSearch(e)} 
        />
      </div>
    );
  }
}

export default compose(
  withApollo,
)(SearchWatchesBar);
