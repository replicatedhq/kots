package pull

const testTemplate = `package pull

var {{ .Name}} ReplicatedPullTest = ReplicatedPullTest{
	Name:                 "{{ .Name }}",
        LicenseData:          ` + "`{{ .LicenseData }}`" + `,
        ReplicatedAppArchive: ` + "`{{ .ReplicatedAppArchive }}`" + `,
        ExpectedFilesystem:   ` + "`{{ .ExpectedFilesystem }}`" + `,
}
`
