import { Component } from "react";
import PropTypes from "prop-types";
import { formatNumber } from "accounting";
import Loader from "./Loader";

import "../../scss/components/shared/Pager.scss";
import Icon from "../Icon";

class Pager extends Component {
  pageCount() {
    return Math.ceil(this.props.totalCount / this.props.pageSize);
  }

  offset() {
    return this.props.currentPage * this.props.pageSize;
  }

  handleGoToPage(page, e) {
    this.props.goToPage(page, e);
  }

  render() {
    if (
      this.props.currentPage === 0 &&
      this.props.totalCount <= this.props.pageSize
    ) {
      return null;
    }

    return (
      <div
        className="flex flex-auto Pager alignItems--center justifyContent--center"
        style={{ backgroundColor: "white" }}
      >
        <div className="flex-column justifyContent--center u-marginRight--50">
          {this.props.currentPage > 0 ? (
            <div className="flex arrow-wrapper">
              <p
                className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-cursor--pointer u-display--inlineBlock"
                onClick={
                  !this.props.loading
                    ? (e) => this.handleGoToPage(this.props.currentPage - 1, e)
                    : null
                }
                data-testid="pager-prev"
              >
                <Icon
                  icon="prev-arrow"
                  size={10}
                  className="clickable gray-color u-marginRight--5"
                />
                Prev
              </p>
            </div>
          ) : null}
        </div>
        <div className="flex-auto">
          {this.props.loading ? (
            <Loader size="24" />
          ) : this.props.currentPageLength ? (
            <p className="u-fontSize--normal u-lineHeight--normal u-textAlign--center">
              <span className="u-color--dustyGray">
                Showing {this.props.pagerType}{" "}
              </span>
              <span className="u-textColor--primary u-fontWeight--medium">{`${
                this.offset() + 1
              } - ${this.offset() + this.props.currentPageLength}`}</span>
              <span className="u-color--dustyGray"> of </span>
              <span className="u-textColor--primary u-fontWeight--medium">
                {formatNumber(this.props.totalCount)}
              </span>
            </p>
          ) : null}
        </div>
        <div className="flex-column justifyContent--center u-marginLeft--50">
          {this.props.currentPage < this.pageCount() - 1 ? (
            <div className="flex arrow-wrapper">
              <p
                className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-cursor--pointer u-display--inlineBlock"
                onClick={
                  !this.props.loading
                    ? (e) => this.handleGoToPage(this.props.currentPage + 1, e)
                    : null
                }
                data-testid="pager-next"
              >
                Next{" "}
                <Icon
                  icon="next-arrow"
                  size={10}
                  className="clickable gray-color u-marginLeft--5"
                />
              </p>
            </div>
          ) : null}
        </div>
      </div>
    );
  }
}

Pager.propTypes = {
  pagerType: PropTypes.string,
  currentPage: PropTypes.number.isRequired,
  pageSize: PropTypes.number.isRequired,
  totalCount: PropTypes.number.isRequired,
  loading: PropTypes.bool.isRequired,
  currentPageLength: PropTypes.number.isRequired,
};

export default Pager;
