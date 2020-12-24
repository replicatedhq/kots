import React from "react";


export default function AddRoleGroup(props) {
  const { addGroup, isApplicationSettings } = props;

  return (
    <div className={`flex flex-column AddRoleGroup--wrapper ${!isApplicationSettings && "bigger"} alignItems--center`}>
      <p className="u-fontSize--jumbo2 u-fontWeight--bold u-lineHeight--more u-color--tundora"> Add Role Based Access Control groups </p>
      <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-color--dustyGray">
        If you do not define groups, everyone will have admin access to your application. Once one group is configured, all other members will not have access unless assigned to a configured group.
        </p>
      <div className="flex justifyContent--cenyer u-marginTop--20">
        <button className="btn secondary blue" onClick={addGroup}> Add a group </button>
      </div>
    </div>
  );
}