import React, { Component } from "react";

export default class RBACGroupPolicyRow extends Component {
  render() {
    const { groupName, index, handleFormChange, onRemoveGroupRow, roles, groupRole, onAddGroup, onEdit, showRoleDetails, onShowRoleDetails, onHideRoleDetails } = this.props;

    return (
      <div className="flex flex-column u-borderBottom--gray darker" style={{ padding: "8px 10px" }} key={index}>
        <div className="flex flex1 alignItems--center justifyContent--spaceBetween">
          <div className="flex flex-column justifyContent--flexStart">
            <div className="flex alignItems--center">
              {!this.props.isEditing ?
                <span className="u-fontSize--normal u-lineHeight--normal u-color--tundora" style={{ width: "140px" }}> {groupName} </span> :
                <input type="text"
                  className="Input darker"
                  placeholder="Group name"
                  value={groupName}
                  onChange={(e) => { handleFormChange("groupName", index, e) }} />}
              <div className="u-marginLeft--10 flex alignItems--center">
                <span className="replicated-link u-fontSize--small u-marginLeft--5" onClick={() => showRoleDetails ? onHideRoleDetails(index) : onShowRoleDetails(index)}> {showRoleDetails ? "Hide Roles" : "Show Roles"} </span>
              </div>
            </div>
            {showRoleDetails &&
              <div className="Roles--wrapper flex flex1 u-marginTop--7">
                {roles?.map((role, i) => {
                  return (
                    <div className="flex u-marginRight--20 alignItems--center" key={`${role.id}-${index}-${i}`}>
                      <input
                        type="radio"
                        id={role.id}
                        checked={groupRole?.id === role?.id}
                        onChange={(e) => { handleFormChange(`${role.id}-${index}-${i}`, index, e) }}
                      />
                      <label htmlFor={role.id} className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none" style={{marginTop: "3px"}}>
                        <p className="u-color--doveGray u-fontSize--small u-fontWeight--medium">{role.name}</p>
                      </label>
                    </div>
                  )
                })
                }
              </div>}
          </div>
          <div className="flex fle1 justifyContent--flexEnd">
            {this.props.isEditing ?
              <div className="flex flex1">
                <button className="btn primary blue" onClick={() => onAddGroup(index)}>Add group</button>
                <button className="btn secondary blue u-marginLeft--20" onClick={() => onRemoveGroupRow(index)}>Cancel</button>
              </div>
              :
              <div className="flex flex1">
                <span className="icon gray-edit u-cursor--pointer u-marginRight--10" onClick={() => onEdit(index)} />
                <span className="icon gray-trash u-cursor--pointer" onClick={() => onRemoveGroupRow(index)} />
              </div>
            }
          </div>
        </div>
      </div>
    )
  }
}

