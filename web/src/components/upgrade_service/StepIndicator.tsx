interface StepIndicatorProps {
  items: string[];
  value: number;
  className?: string;
}

const StepIndicator = ({ items, value, className }: StepIndicatorProps) => {
  return (
    <div className="tw-flex tw-justify-center">
      {items.map((item, index) => {
        const isActive = value === index;
        const isLast = index === items.length - 1;
        return (
          <div
            key={index}
            className={`${className} ${
              isLast ? "tw-ml-8 tw-w-12" : "tw-w-[180px]"
            }`}
          >
            <div className="tw-flex tw-flex-col tw-items-center tw-relative tw-mr-16 tw-text-sm">
              <div
                className={`${
                  isActive
                    ? "tw-text-gray-600 tw-font-semi-bold"
                    : "tw-text-[#BCBCBC] tw-font-normal"
                } tw-whitespace-nowrap`}
              >
                {item}
              </div>
              <div className="tw-flex tw-items-center">
                <div
                  className={`tw-h-5 tw-w-5 tw-flex tw-justify-center tw-items-center tw-my-2 tw-p-1 tw-rounded-full ${
                    isActive
                      ? "tw-bg-[#6A77FB] tw-text-white"
                      : "tw-bg-[#F0F1FF] tw-text-[#C2C7FD]"
                  }`}
                >
                  <div>{index + 1}</div>
                </div>
              </div>
              {!isLast && (
                <div className="tw-mb-5 tw-bg-[#F0F1FF] tw-h-0.5 tw-w-[143px] tw-absolute tw-left-[77px] tw-bottom-0"></div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default StepIndicator;
