export default function DummyRbacRow() {
  return (
    <div className="flex flex-auto DummyRbacRow--wrapper alignItems--center">
      <div className="flex-column flex1">
        <div className="flex flex-column">
          <div className="EmptyRow name"></div>
        </div>
      </div>
      <div className="flex flex1">
        <div className="flex flex-auto alignItems--center u-marginTop--5">
          <div className="flex flex1 alignItems--center">
            <div className="EmptyRow role u-marginRight--20"></div>
            <div className="EmptyRow role"> </div>
          </div>
        </div>
      </div>
      <div className="flex flex-auto">
        <div className="EmptyCircle"></div>
      </div>
    </div>
  );
}
