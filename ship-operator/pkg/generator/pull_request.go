package generator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func (g *Generator) generatePullRequestContainer(prSpec *shipv1beta1.PullRequestActionSpec) (corev1.Container, []corev1.Volume) {
	debug := level.Debug(log.With(g.logger, "method", "generatePullRequestContainer"))

	toolsImageName, toolsImagePullPolicy := g.shipToolsImage()
	// TODO check if action was valid

	branchName := strconv.Itoa(int(time.Now().Unix()))
	basePath := strings.TrimLeft(prSpec.BasePath, "/")
	basePath = strings.TrimRight(basePath, "/")

	debug.Log("event", "construct container")
	container := corev1.Container{
		Image:           toolsImageName,
		ImagePullPolicy: toolsImagePullPolicy,
		Name:            fmt.Sprintf("ship-pullrequest-%s", GenerateID(5)),
		Env: []corev1.EnvVar{
			{
				Name: "GITHUB_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: prSpec.GitHub.Token.ValueFrom.SecretKeyRef.Name,
						},
						Key: prSpec.GitHub.Token.ValueFrom.SecretKeyRef.Key,
					},
				},
			},
		},
		Command: []string{
			"/bin/sh",
			"-c",
			`echo 'github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==' >> ~/.ssh/known_hosts
				echo "github.com" > /root/hub &&
				echo "- user: ` + prSpec.GitHub.Owner + `" >> /root/hub &&
				echo "  oauth_token: $GITHUB_TOKEN" >> /root/hub &&
				echo "  protocol: https" >> /root/hub &&
				mkdir -p ~/.config &&
				cp -r /root/hub ~/.config/hub &&
				mkdir -p ~/repo &&
				cd ~/repo &&
				git config --global user.email "ship@replicated.com" &&
				git config --global user.name "Replicated Ship" &&
				git clone git@github.com:` + prSpec.GitHub.Owner + `/` + prSpec.GitHub.Repo + ` . &&
				git checkout -b ` + branchName + ` &&
				cp -r /out/* ./` + basePath + ` &&
				rm -rf ./` + basePath + `/chart &&
				sleep 1 &&
				git add . &&
				sleep 1 &&
				git commit -m "Update to better db" &&
				sleep 1 &&
				git log --decorate &&
				git push origin ` + branchName + ` &&
				sleep 5 &&
				hub pull-request -b master -m "Update to better db"`,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/out",
				ReadOnly:  true,
			},
			{
				Name:      "key",
				MountPath: "/root/.ssh/id_rsa",
				SubPath:   "id_rsa",
				ReadOnly:  true,
			},
		},
	}

	secretMode := int32(0400)
	volumes := []corev1.Volume{
		{
			Name: "key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: prSpec.GitHub.Key.ValueFrom.SecretKeyRef.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  prSpec.GitHub.Key.ValueFrom.SecretKeyRef.Key,
							Path: "id_rsa",
							Mode: &secretMode,
						},
					},
				},
			},
		},
	}

	return container, volumes
}
