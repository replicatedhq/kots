import React, { Component } from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter, Link } from "react-router-dom"
import Helmet from "react-helmet";
import dayjs from "dayjs";
import Select from "react-select";

import RedactorRow from "./RedactorRow";
import DeleteRedactorModal from "../modals/DeleteRedactorModal";

import { Utilities } from "../../utilities/utilities";

const redactors = [
  {
    id: "1",
    name: "my-demo-redactor",
    createdAt: "2020-05-10T21:17:37.002Z",
    updatedOn: "2020-05-18T22:17:37.002Z",
    details: "Redact all AWS secrets",
    status: "enabled"
  },
  {
    id: "2",
    name: "my-other-redactor",
    createdAt: "2020-05-11T21:20:37.002Z",
    updatedOn: "2020-05-16T19:17:30.002Z",
    details: "Redact ip address’s from 10.0.0.0 - 10.255.255.255",
    status: "disabled"
  },
]

class Redactors extends Component {
  state = {
    sortedRedactors: [],
    selectedOption: {
      value: "createdAt",
      label: "Sort by: Created At"
    },
    deleteRedactorModal: false,
    redactorToDelete: {},
    isLoadingRedactors: false,
    redactorsErrMsg: ""
  };

  getRedactors = () => {
    this.setState({
      isLoadingRedactors: true,
      redactorsErrMsg: ""
    });

    fetch(`${window.env.API_ENDPOINT}/troubleshoot/redacts`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        console.log(result)
        this.setState({
          isLoadingRedactors: false,
          redactorsErrMsg: "",
        })
      })
      .catch(err => {
        this.setState({
          isLoadingRedactors: false,
          redactorsErrMsg: err,
        })
      })
  }

  handleSortChange = selectedOption => {
    this.setState({ selectedOption }, () => {
      this.sortRedactors(this.state.selectedOption.value);
    });
  }

  componentDidMount() {
    this.getRedactors();
    if (this.state.selectedOption) {
      this.sortRedactors(this.state.selectedOption.value);
    }
  }

  sortRedactors = value => {
    if (value === "createdAt") {
      this.setState({ sortedRedactors: redactors.sort((a, b) => dayjs(b.createdAt) - dayjs(a.createdAt)) });
    } else {
      this.setState({ sortedRedactors: redactors.sort((a, b) => dayjs(b.updatedOn) - dayjs(a.updatedOn)) });
    }
  }

  toggleConfirmDeleteModal = redactor => {
    if (this.state.deleteRedactorModal) {
      this.setState({ deleteRedactorModal: false, redactorToDelete: "", deleteErr: false, deleteErrorMsg: "" });
    } else {
      this.setState({ deleteRedactorModal: true, redactorToDelete: redactor, deleteErr: false, deleteErrorMsg: "" });
    }
  };

  handleDeleteRedactor = redactor => {
    console.log("deleting", redactor)
  }


  render() {
    const { sortedRedactors, selectedOption, deleteRedactorModal } = this.state;

    console.log(1111)

    const selectOptions = [
      {
        value: "createdAt",
        label: "Sort by: Created At"
      },
      {
        value: "updatedOn",
        label: "Sort by: Updated on"
      }
    ]

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
        <Helmet>
          <title>Redactors </title>
        </Helmet>
        <div className="Redactors--wrapper flex1 flex-column u-width--full">
          <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween">
            <div className="flex flex1">
              <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginRight--10"> Redactors </p>
              <div style={{ width: "220px" }}>
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  options={selectOptions}
                  value={selectedOption}
                  getOptionValue={(option) => option.label}
                  isOptionSelected={(option) => { option.value === selectedOption }}
                  onChange={this.handleSortChange}
                />
              </div>
            </div>
            <div className="flex justifyContent--flexEnd">
              <Link to="/redactors/new" className="btn primary blue"> Create new redactor </Link>
            </div>
          </div>
          <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginTop--small u-marginBottom--30">Define custom rules for sensitive values you need to be redacted when gathering a support bundle. This might include things like Secrets or IP addresses. For help with creating custom redactors,
          <a href="" target="_blank" rel="noopener noreferrer" className="replicated-link"> check out our docs</a>.</p>
          {sortedRedactors?.map((redactor) => (
            <RedactorRow
              key={`redactor-${redactor.id}`}
              redactor={redactor}
              toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
            />
          ))}
        </div>
        {deleteRedactorModal &&
          <DeleteRedactorModal
            deleteRedactorModal={deleteRedactorModal}
            toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
            handleDeleteRedactor={this.handleDeleteRedactor}
            redactorToDelete={this.state.redactorToDelete}
            deleteErr={this.state.deleteErr}
            deleteErrorMsg={this.state.deleteErrorMsg}
          />
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
)(Redactors);
