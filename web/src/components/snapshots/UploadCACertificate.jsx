import React from "react";
import Dropzone from "react-dropzone";
import { getFileContent } from "../../utilities/utilities";


const UploadCACertificate = () => {

  const handleDrop = async (files) => {
    const content = await getFileContent(files[0]);
    const parsedCert = (new TextDecoder("utf-8")).decode(content);
    let certificate;
    // try {
    //   certificate = yaml.loadAll(parsedCert);
    // } catch (e) {
    //   console.log(e);
    //   this.setState({ errorMessage: "Faild to parse license file" });
    //   return;
    // }
    // const hasMultiApp = certificate.length > 1;
    // if (hasMultiApp) {
    //   this.setAvailableAppOptions(certificate);
    // }
    // this.setState({
    //   licenseFile: files[0],
    //   licenseFileContent: hasMultiApp ? keyBy(certificate, (option) => { return option.spec.appSlug }) : certificate[0],
    //   errorMessage: "",
    //   hasMultiApp,
    // });
  }

  return (
    <>
      <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
        Upload a CA Certificate
      </p>
      <Dropzone
        className="Dropzone-wrapper"
        accept={[".pem", ".cer", ".crt", ".ca", ".key"]}
        onDropAccepted={handleDrop}
        multiple={false}
      >
        <div className="cert-dropzone-wrapper">
          <div className="icon cert-file-icon" />
          <div>
            <p className="u-fontSize--normal u-textColor--secondary u-lineHeight--normal">
              Drag your cert here or
              <span className="u-linkColor u-textDecoration--underlineOnHover" style={{paddingLeft: "4px"}}>
                choose a file
              </span>
            </p>
            <p className="u-fontSize--small u-textColor--info u-lineHeight--normal">
              Supported file types are .pem, .cer, .crt, .a, and .key
            </p>
          </div>
        </div>
      </Dropzone>

      {/* TODO: Update this link to the docs!! */}
      <p className="u-fontSize--small u-lineHeight--normal u-textColor--bodyCopy u-marginTop--15 u-marginBottom--30">
        For more information about uploading CA certificates, including details on how to upload multiple certificates,
        <a
          href="https://docs.replicated.com/"
          target="_blank"
          rel="noopener noreferrer"
          className="replicated-link"
          style={{ paddingLeft: "4px"}}
        >
          check out our docs
        </a>
        .
      </p>

    </>
  )
}

export default UploadCACertificate;