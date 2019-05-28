import * as React from "react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import sortBy from "lodash/sortBy";
dayjs.extend(relativeTime);

const Point = ({ point, multi }) => (
  <div className="point flex flex-column alignItems--center">
    <div className="details">
      <p className="flex flex-column alignItems--center u-fontSize--small u-fontWeight--medium u-color--dustyGray">
        {multi ? `${multi.length} more versions` : point.version}
        {multi ? null : <span className="u-fontSize--smaller u-lineHeight--normal">{dayjs(point.date).fromNow()}</span>}
      </p>
    </div>
    { multi ? 
      <div className="dot-wrapper flex">
        <span></span>
        <span></span>
        <span></span>
      </div> :
      <div className="dot-wrapper flex">
        <span></span>
      </div> 
    }
  </div>
);

const ClusterScopeBatchPath = ({ path, loading }) => {
  const _path = sortBy(path, ["sort"]);
  const oldest = _path[0];
  const newest = _path[_path.length-1];
  const middle = _path.slice(1, _path.length - 1);
  const points = _path.length < 5 ? 
    middle.length ? middle.map((point, i) => (
      <Point point={point} key={i} />
    )) : null 
    : <Point multi={middle} />
  return (
    <div className="ClusterScopeBatchPath--wrapper flex flex1">
      <div className="version oldest-version">
        <p className="flex flex-column alignItems--center u-color--tuna u-fontSize--larger u-fontWeight--bold">
          {loading ? "---" : oldest.version}
          {loading ? null : <span className="u-fontSize--smaller u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">{dayjs(oldest.date).fromNow()}</span>}
        </p>
        <p className="u-position--relative u-fontSize--small u-color--dustyGray">
        Your version 
          <span className={`version--point ${loading ? "gray" : "red"}`}></span>
          <span className="version--line"></span>
        </p>
      </div>
      <div className={`path flex flex1 ${loading ? "loading" : "checked"}`}>
        <div className="path--line flex1"></div>
        <div className="path--points flex flex1 justifyContent--spaceBetween">
          { loading ? null : points }
        </div>
      </div>
      <div className="version newest-version">
        <p  className="flex flex-column alignItems--center u-color--tuna u-fontSize--larger u-fontWeight--bold">
          {loading ? "---" : newest.version}
          {loading ? null : <span className="u-fontSize--smaller u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">{dayjs(newest.date).fromNow()}</span>}
        </p>
        <p className="u-position--relative u-fontSize--small u-color--dustyGray">
          <span className="version--line"></span>
          <span className={`version--point ${loading ? "gray" : "green"}`}></span> 
        Latest version 
        </p>
      </div>
    </div>
  )
}

export default ClusterScopeBatchPath;