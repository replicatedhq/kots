import { useNavigate, useParams } from "react-router-dom";

const PreflightCheck = () => {
  const navigate = useNavigate();
  const { slug } = useParams();
  return (
    <div>
      PreflightCheck
      <button onClick={() => navigate(`/upgrade-service/app/${slug}/config`)}>
        Back: Config
      </button>
      <button onClick={() => navigate(`/upgrade-service/app/${slug}/deploy`)}>
        Next: Confirm and deploy
      </button>
    </div>
  );
};

export default PreflightCheck;
