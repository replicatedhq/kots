import { Component } from "react";
import Icon from "../Icon";

export default class RBACGroupPolicyRow extends Component {
  render() {
    const {
      groupName,
      index,
      handleFormChange,
      onRemoveGroupRow,
      roles,
      groupRoles,
      onAddGroup,
      onEdit,
      showRoleDetails,
      onShowRoleDetails,
      onHideRoleDetails,
      isApplicationSettings,
      checkedRoles,
      handleRoleCheckboxChange,
      onCancelGroupRow,
    } = this.props;

    return (
      <div
        className="flex-1-auto u-borderBottom--gray darker"
        style={{ padding: "8px 10px" }}
        key={index}
      >
        <div className="flex flex1 alignItems--center justifyContent--spaceBetween">
          <div className="flex1 flex-column justifyContent--flexStart u-paddingRight--10">
            <div className="flex alignItems--center">
              {!this.props.isEditing ? (
                <span
                  className="u-fontSize--normal u-lineHeight--normal u-textColor--secondary"
                  style={{ maxWidth: "690px" }}
                >
                  {" "}
                  {groupName}{" "}
                </span>
              ) : (
                <input
                  type="text"
                  className="Input darker"
                  placeholder="Group name"
                  value={groupName}
                  onChange={(e) => {
                    handleFormChange("groupName", index, e);
                  }}
                />
              )}
              <div className="u-marginLeft--10 flex alignItems--center">
                {isApplicationSettings && (
                  <span className="RoleNum--wrapper">
                    {" "}
                    {`${
                      checkedRoles?.length === 1
                        ? "1 role"
                        : `${checkedRoles?.length} roles`
                    }`}
                  </span>
                )}
                <span
                  className="link u-fontSize--small u-marginLeft--5"
                  onClick={() =>
                    showRoleDetails
                      ? onHideRoleDetails(index)
                      : onShowRoleDetails(index)
                  }
                >
                  {" "}
                  {showRoleDetails ? "Hide Roles" : "Show Roles"}{" "}
                </span>
              </div>
            </div>
            {showRoleDetails ? (
              isApplicationSettings ? (
                <div className="Roles--wrapper flex flex1 u-marginTop--7">
                  {roles?.map((role, i) => {
                    const gRole = groupRoles?.find((r) => r?.id === role?.id);
                    return (
                      <div
                        className="flex u-marginRight--20 alignItems--center"
                        key={`${role.id}-${index}-${i}`}
                      >
                        <input
                          type="checkbox"
                          id={`checkbox-${index}=${role.id}`}
                          checked={gRole ? gRole.isChecked : false}
                          onChange={(e) => {
                            handleRoleCheckboxChange(index, i, e);
                          }}
                        />
                        <label
                          htmlFor={`checkbox-${index}=${role.id}`}
                          className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                        >
                          <p className="u-textColor--accent u-fontSize--small u-fontWeight--medium">
                            {role.name}
                          </p>
                        </label>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="Roles--wrapper flex flex1 u-marginTop--7">
                  {roles?.map((role, i) => {
                    const gRole = groupRoles?.find((r) => r?.id === role?.id);
                    return (
                      <div
                        className="flex u-marginRight--20 alignItems--center"
                        key={`${role.id}-${index}-${i}`}
                      >
                        <input
                          type="radio"
                          id={`radio-${index}=${role.id}`}
                          checked={gRole ? gRole?.id === role?.id : false}
                          onChange={(e) => {
                            handleFormChange(
                              `${role.id}-${index}-${i}`,
                              index,
                              e
                            );
                          }}
                        />
                        <label
                          htmlFor={`radio-${index}=${role.id}`}
                          className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                          style={{ marginTop: "3px" }}
                        >
                          <p className="u-textColor--accent u-fontSize--small u-fontWeight--medium">
                            {role.name}
                          </p>
                        </label>
                      </div>
                    );
                  })}
                </div>
              )
            ) : null}
          </div>
          <div className="flex fle1 justifyContent--flexEnd">
            {this.props.isEditing ? (
              <div className="flex flex1">
                <button
                  className="btn primary blue"
                  onClick={() => onAddGroup(index)}
                >
                  Add group
                </button>
                <button
                  className="btn secondary blue u-marginLeft--20"
                  onClick={() => onCancelGroupRow(index)}
                >
                  Cancel
                </button>
              </div>
            ) : (
              <div className="flex flex1">
                <Icon
                  icon="edit"
                  size={20}
                  className="gray-color u-marginRight--10 clickable"
                  onClick={() => onEdit(index)}
                />
                <Icon
                  icon="trash"
                  size={20}
                  className="gray-color clickable"
                  onClick={() => onRemoveGroupRow(index)}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
}
