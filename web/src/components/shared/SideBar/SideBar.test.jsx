import React from "react";
import { shallow } from "enzyme";

import SideBar from "./SideBar";

describe("<SideBar> tests", () => {
  const dummyData = [
    {
      name: "Jeff",
      occupation: "Professional Jeff"
    },
    {
      name: "Rick",
      occupation: "Professional Jeff Impersonator"
    }
  ];

  it("Renders without crashing", () => {
    const wrapper = shallow(<SideBar />);

    expect(wrapper).toBeTruthy();
  });

  it("Renders a few items", () => {
    const wrapper = shallow(
      <SideBar
        items={dummyData.map( (person, idx) => (
          <div key={idx} className="card">
            <span className="card-name">{person.name}</span>
            <span className="card-job">{person.occupation}</span>
          </div>
        ))}
      />
    );

    expect(wrapper.find(".card")).toHaveLength(2);
    expect(wrapper.find(".card-job")).toHaveLength(2);
  });

  it("Displays a loader if the loader prop is passed in", () => {
    const wrapper = shallow(
      <SideBar
      loading={true}
        items={dummyData.map((person, idx) => (
          <div key={idx} className="card">
            <span className="card-name">{person.name}</span>
            <span className="card-job">{person.occupation}</span>
          </div>
        ))}
      />
    );

    expect(wrapper.find("Loader")).toHaveLength(1);
  });
});
