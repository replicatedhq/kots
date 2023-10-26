export default function DummySnapshotRow() {
  return (
    <div className="flex flex-auto DummySnapshotRow--wrapper alignItems--center">
      <div className="flex-column flex1">
        <div className="flex flex-column">
          <div className="EmptyRow name"></div>
          <div className="flex flex1 alignItems--center u-marginTop--10">
            <p className="EmptyRow created u-marginRight--20"></p>
            <p className="EmptyRow finished"></p>
          </div>
        </div>
      </div>
      <div className="flex flex1">
        <div className="flex flex-auto alignItems--center u-marginTop--5">
          <div className="flex flex1 alignItems--center">
            <div className="EmptyRow volumeSize u-marginRight--20"></div>
            <div className="EmptyRow volumes"> </div>
          </div>
        </div>
      </div>
      <div className="flex flex-auto">
        <div className="EmptyCircle u-marginRight--10"></div>
        <div className="EmptyCircle u-marginRight--10"></div>
        <div className="EmptyCircle"></div>
      </div>
    </div>
  );
}
