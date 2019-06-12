import * as React from "react";
import classNames from "classnames";

export const WatchContributorCheckbox = ({ handleCheckboxChange, contributors, githubLogin, item }) => (
  <div data-qa={`Checkbox--${item.login}`} className="BoxedCheckbox-wrapper flex-column">
    <div className={
      classNames("BoxedCheckbox flex1 flex", {
        "is-active": contributors[item.login] && contributors[item.login].isActive,
        "is-disabled": githubLogin && githubLogin.toLowerCase() === item.login
      })}>
      <div className="flex-column flex-verticalCenter input-wrapper">
        <input
          type="checkbox"
          className="u-cursor--pointer"
          id={`${item.login}`}
          checked={contributors[item.login] && contributors[item.login].isActive}
          value={item.login}
          onChange={(e) => { handleCheckboxChange(item.login, e) }}
          disabled={githubLogin && githubLogin.toLowerCase() === item.login}
        />
      </div>
      <label htmlFor={`${item.login}`} className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
        <div className="flex-column flex-verticalCenter specs-icon-wrapper alignItems--center">
          <span className="user-photo u-pointerEvents--none" style={{ backgroundImage: `url(${item.avatar_url})` }}></span>
        </div>
        <div className="flex-column u-marginLeft--small flex-verticalCenter">
          <span className="u-fontWeight--medium u-color--tuna u-fontSize--normal u-lineHeight--default">{item.login}</span>
        </div>
      </label>
    </div>
  </div>
);
