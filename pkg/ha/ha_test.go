package ha

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	k8sutiltypes "github.com/replicatedhq/kots/pkg/k8sutil/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

func TestCanRunHA(t *testing.T) {
	type args struct {
		ctx       context.Context
		clientset kubernetes.Interface
	}
	tests := []struct {
		name       string
		args       args
		want       bool
		wantReason string
		wantErr    bool
	}{
		{
			name: "1 node available",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
					},
				}),
			},
			want:       false,
			wantReason: REASON_NOT_ENOUGH_NODES,
			wantErr:    false,
		},
		{
			name: "2 nodes available",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
					},
				}),
			},
			want:       false,
			wantReason: REASON_NOT_ENOUGH_NODES,
			wantErr:    false,
		},
		{
			name: "3 amd64 nodes available",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node3",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
					},
				}),
			},
			want:       true,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "3 arm64 nodes available",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node3",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
					},
				}),
			},
			want:       true,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "> 3 nodes available",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node3",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node4",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
					},
				}),
			},
			want:       true,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "4 nodes available but 2 nodes don't match the label selector",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&corev1.NodeList{
					Items: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node1",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "amd64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node2",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "arm64",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node3",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "ppc64le",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "node4",
								Labels: map[string]string{
									"kubernetes.io/os":   "linux",
									"kubernetes.io/arch": "ppc64le",
								},
							},
						},
					},
				}),
			},
			want:       false,
			wantReason: REASON_NOT_ENOUGH_NODES,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotReason, err := CanRunHA(tt.args.ctx, tt.args.clientset)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanRunHA() error = %v, wantErr: %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CanRunHA() got = %v, want: %v", got, tt.want)
			}
			if gotReason != tt.wantReason {
				t.Errorf("CanRunHA() gotReason = %v, want: %v", gotReason, tt.wantReason)
			}
		})
	}
}

func TestEnableHA(t *testing.T) {
	type args struct {
		ctx        context.Context
		clientset  kubernetes.Interface
		namespace  string
		timeout    time.Duration
		readyAfter time.Duration
	}
	tests := []struct {
		name           string
		args           args
		wantReplicas   int32
		wantArgs       []string
		wantErr        bool
		wantTimeoutErr bool
	}{
		{
			name: "scales up rqlite, modifies its args, and waits for it to be ready",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kotsadm-rqlite",
								Namespace: "default",
							},
							Spec: appsv1.StatefulSetSpec{
								Replicas: pointer.Int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "rqlite",
												Args: []string{
													"-disco-mode=dns",
													"-disco-config={\"name\":\"kotsadm-rqlite-headless\"}",
													"-bootstrap-expect=1",
													"-auth=/auth/config.json",
													"-join-as=kotsadm",
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				namespace:  "default",
				timeout:    4 * time.Second,
				readyAfter: 2 * time.Second,
			},
			wantReplicas: 3,
			wantArgs: []string{
				"-disco-mode=dns",
				"-disco-config={\"name\":\"kotsadm-rqlite-headless\"}",
				"-bootstrap-expect=3",
				"-auth=/auth/config.json",
				"-join-as=kotsadm",
			},
			wantErr: false,
		},
		{
			name: "scales up rqlite, modifies its args, and times out if it doesn't become ready",
			args: args{
				ctx: context.Background(),
				clientset: fake.NewSimpleClientset(&appsv1.StatefulSetList{
					Items: []appsv1.StatefulSet{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kotsadm-rqlite",
								Namespace: "default",
							},
							Spec: appsv1.StatefulSetSpec{
								Replicas: pointer.Int32Ptr(1),
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name: "rqlite",
												Args: []string{
													"-disco-mode=dns",
													"-disco-config={\"name\":\"kotsadm-rqlite-headless\"}",
													"-bootstrap-expect=1",
													"-auth=/auth/config.json",
													"-join-as=kotsadm",
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				namespace:  "default",
				timeout:    2 * time.Second,
				readyAfter: 4 * time.Second,
			},
			wantReplicas: 3,
			wantArgs: []string{
				"-disco-mode=dns",
				"-disco-config={\"name\":\"kotsadm-rqlite-headless\"}",
				"-bootstrap-expect=3",
				"-auth=/auth/config.json",
				"-join-as=kotsadm",
			},
			wantErr:        true,
			wantTimeoutErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				time.Sleep(tt.args.readyAfter)

				sts, err := tt.args.clientset.AppsV1().StatefulSets(tt.args.namespace).Get(tt.args.ctx, "kotsadm-rqlite", metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get statefulset: %v", err)
				}

				sts.Status.ReadyReplicas = tt.wantReplicas
				sts.Status.ObservedGeneration = sts.Generation

				_, err = tt.args.clientset.AppsV1().StatefulSets(tt.args.namespace).UpdateStatus(tt.args.ctx, sts, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update statefulset: %v", err)
				}
			}()

			err := EnableHA(tt.args.ctx, tt.args.clientset, tt.args.namespace, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnableHA() error = %v, wantErr: %v", err, tt.wantErr)
				return
			}
			if tt.wantTimeoutErr {
				if _, ok := errors.Cause(err).(*k8sutiltypes.ErrorTimeout); !ok {
					t.Errorf("EnableHA() error = %v, want: timeout error", err)
				}
			}

			sts, err := tt.args.clientset.AppsV1().StatefulSets(tt.args.namespace).Get(tt.args.ctx, "kotsadm-rqlite", metav1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get statefulset: %v", err)
			}
			if *sts.Spec.Replicas != tt.wantReplicas {
				t.Errorf("replicas = %v, want: %v", *sts.Spec.Replicas, tt.wantReplicas)
			}
			if !reflect.DeepEqual(sts.Spec.Template.Spec.Containers[0].Args, tt.wantArgs) {
				t.Errorf("args = %v, want: %v", sts.Spec.Template.Spec.Containers[0].Args, tt.wantArgs)
			}
		})
	}
}
