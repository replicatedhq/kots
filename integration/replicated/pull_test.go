package replicated

type replicatedPullTest struct {
	name    string
	testDir string
}

const endpoint = "http://localhost:3001"

// func Test_PullReplicated(t *testing.T) {
// 	namespace := "test_ns"
// 	tests := []replicatedPullTest{}

// 	testDirs, err := ioutil.ReadDir("tests")
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, testDir := range testDirs {
// 		if testDir.IsDir() {
// 			_, name := path.Split(testDir.Name())

// 			tests = append(tests, replicatedPullTest{
// 				name:    name,
// 				testDir: path.Join("tests", testDir.Name()),
// 			})
// 		}
// 	}

// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			req := require.New(t)

// 			archiveData, err := ioutil.ReadFile(path.Join(test.testDir, "archive.tar.gz"))
// 			req.NoError(err)

// 			licenseFilepath := path.Join(test.testDir, "license.yaml")
// 			licenseFile, err := ioutil.ReadFile(licenseFilepath)
// 			req.NoError(err)

// 			stopCh, err := StartMockServer(endpoint, "integration", "integration", archiveData, licenseFile)
// 			req.NoError(err)

// 			defer func() {
// 				stopCh <- true
// 			}()

// 			actualDir, err := ioutil.TempDir("", "integration")
// 			req.NoError(err)
// 			defer os.RemoveAll(actualDir)

// 			pullOptions := pull.PullOptions{
// 				RootDir:             actualDir,
// 				LicenseFile:         licenseFilepath,
// 				Namespace:           namespace,
// 				ExcludeAdminConsole: true,
// 				ExcludeKotsKinds:    true,
// 				Silent:              true,
// 			}
// 			_, err = pull.Pull("replicated://integration", pullOptions)
// 			req.NoError(err)

// 			// create an archive of the actual results
// 			actualFilesystemDir, err := ioutil.TempDir("", "kots")
// 			req.NoError(err)
// 			defer os.RemoveAll(actualFilesystemDir)

// 			paths := []string{
// 				path.Join(actualDir, "upstream"),
// 				path.Join(actualDir, "base"),
// 				path.Join(actualDir, "overlays"),
// 			}

// 			tarGz := archiver.TarGz{
// 				Tar: &archiver.Tar{
// 					ImplicitTopLevelFolder: false,
// 				},
// 			}
// 			err = tarGz.Archive(paths, path.Join(actualFilesystemDir, "archive.tar.gz"))
// 			req.NoError(err)

// 			actualFilesystemBytes, err := ioutil.ReadFile(path.Join(actualFilesystemDir, "archive.tar.gz"))
// 			req.NoError(err)

// 			// create an archive of the expected
// 			expectedFilesystemDir, err := ioutil.TempDir("", "kots")
// 			req.NoError(err)
// 			defer os.RemoveAll(expectedFilesystemDir)

// 			paths = []string{
// 				path.Join(test.testDir, "expected", "upstream"),
// 				path.Join(test.testDir, "expected", "base"),
// 				path.Join(test.testDir, "expected", "overlays"),
// 			}
// 			err = tarGz.Archive(paths, path.Join(expectedFilesystemDir, "archive.tar.gz"))
// 			req.NoError(err)

// 			expectedFilesystemBytes, err := ioutil.ReadFile(path.Join(expectedFilesystemDir, "archive.tar.gz"))
// 			req.NoError(err)

// 			compareOptions := util.CompareOptions{
// 				IgnoreFilesInActual: []string{path.Join("upstream", "userdata", "license.yaml")},
// 			}

// 			ok, err := util.CompareTars(expectedFilesystemBytes, actualFilesystemBytes, compareOptions)
// 			req.NoError(err)

// 			assert.Equal(t, true, ok)
// 		})
// 	}
// }
