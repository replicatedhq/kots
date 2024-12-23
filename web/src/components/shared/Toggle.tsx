import "../../scss/components/shared/Toggle.scss";

interface Props {
  items: {
    onClick: () => void;
    isActive: boolean;
    title: string;
    status?: string;
  }[];
}

export default function Tooltip(props: Props) {
  return (
    <div className="Toggle flex flex-auto alignItems--center">
      {props.items?.map((item, i) => {
        return (
          <div
            key={i}
            className={`Toggle-item ${item.isActive ? "is-active" : ""}`}
            onClick={item.onClick}
          >
            {item.title}

            {item.status && (
              <span
                className={`status-indicator ${item.status?.toLowerCase()} u-marginLeft--5`}
              ></span>
            )}
          </div>
        );
      })}
    </div>
  );
}
