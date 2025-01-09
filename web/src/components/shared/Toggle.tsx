import "../../scss/components/shared/Toggle.scss";

interface Props {
  items: {
    onClick: () => void;
    isActive: boolean;
    title: string;
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
          </div>
        );
      })}
    </div>
  );
}
