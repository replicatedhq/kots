import Dropzone from "react-dropzone";
import isEmpty from "lodash/isEmpty";
import { getFileContent } from "../../utilities/utilities";

const UploadCACertificate = ({ certificate, handleSetCACert }) => {
  const handleDrop = async (files) => {
    let binary = "";
    const bytes = new Uint8Array(await getFileContent(files[0]));
    const len = bytes.byteLength;

    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode(bytes[i]);
    }

    const content = window.btoa(binary);

    handleSetCACert({
      name: files[0].name,
      data: content,
    });
  };

  const clearFile = () => {
    handleSetCACert({
      name: "",
      data: [],
    });
  };

  return (
    <>
      <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
        Upload a CA Certificate
      </p>
      {!isEmpty(certificate.name) && (
        <div className="ca-cert-file-wrapper u-marginBottom--30">
          <div className="icon cert-file-icon" />
          <div>
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--medium">
              {certificate.name}
            </p>
            <span className="link u-fontSize--small" onClick={clearFile}>
              Select a different file
            </span>
          </div>
        </div>
      )}
      {isEmpty(certificate.name) && (
        <Dropzone
          className="Dropzone-wrapper u-marginBottom--30"
          accept={[".pem", ".cer", ".crt", ".ca", ".key"]}
          onDropAccepted={handleDrop}
          multiple={false}
        >
          <div className="cert-dropzone-wrapper">
            <div className="icon cert-file-icon" />
            <div>
              <p className="u-fontSize--normal u-textColor--secondary u-lineHeight--normal">
                Drag your cert here or
                <span
                  className="link u-textDecoration--underlineOnHover"
                  style={{ paddingLeft: "4px" }}
                >
                  choose a file
                </span>
              </p>
              <p className="u-fontSize--small u-textColor--info u-lineHeight--normal">
                Supported file types are .pem, .cer, .crt, .ca, and .key
              </p>
            </div>
          </div>
        </Dropzone>
      )}
    </>
  );
};

export default UploadCACertificate;
