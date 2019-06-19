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

  it("Does not render if less than 2 items exist", () => {
    const wrapper = shallow(
      <SideBar
        items={dummyData.slice(1).map((person, idx) => (
          <div key={idx} className="card">
            <span className="card-name">{person.name}</span>
            <span className="card-job">{person.occupation}</span>
          </div>
        ))}
      />
    );

    expect(wrapper.find("div")).toHaveLength(0);
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

  it("Does not display the loader subsequently if aggressive mode is turned on", () =>{
    const wrapper = shallow(
      <SideBar
        loading={true}
        aggressive={true}
        items={dummyData.map((person, idx) => (
          <div key={idx} className="card">
            <span className="card-name">{person.name}</span>
            <span className="card-job">{person.occupation}</span>
          </div>
        ))}
      />
    );
    expect(wrapper.find("Loader")).toHaveLength(1);

    wrapper.setProps({ loading: false });
    expect(wrapper.find(".card")).toHaveLength(2);

    wrapper.setProps({ loading: true });

    expect(wrapper.find("Loader")).toHaveLength(0);
    expect(wrapper.find(".card")).toHaveLength(2);
  });
});
