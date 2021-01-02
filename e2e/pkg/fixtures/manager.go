package fixtures

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
)

const (
	ServiceAccountName   = "manager-sa"
	ManagerName          = "manager"
	ManagerPodLabelKey   = "operator.h3poteto.dev"
	ManagerPodLabelValue = "control-plane"
)

var ManagerPodLabels = map[string]string{
	ManagerPodLabelKey: ManagerPodLabelValue,
}

func NewManagerManifests(ns, clusterRoleName, image string) (*corev1.ServiceAccount, *rbacv1.ClusterRoleBinding, *rbacv1.Role, *rbacv1.RoleBinding, *appsv1.Deployment) {
	leName := "leader-election"
	return serviceAccount(ns), roleBinding(ns, clusterRoleName), leaderElectionRole(ns, leName), leaderElectionRoleBinding(ns, leName), deployment(ns, image)
}

func serviceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: ns,
		},
	}
}

func roleBinding(ns, roleName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "manager-role-binding",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}
}

func deployment(ns, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: ManagerName,
			Labels: map[string]string{
				"operator.h3poteto.dev": "control-plane",
			},
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utilpointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: ManagerPodLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ManagerPodLabels,
				},
				Spec: corev1.PodSpec{
					Volumes:        nil,
					InitContainers: nil,
					Containers: []corev1.Container{
						{
							Name:    "manager",
							Image:   image,
							Command: nil,
							Args: []string{
								"/fluentd-sidecar-injector",
								"controller",
								"sidecar-injector",
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "CONTAINER_IMAGE",
									Value: image,
								},
							},
							TerminationMessagePath:   "",
							TerminationMessagePolicy: "",
							ImagePullPolicy:          corev1.PullAlways,
						},
					},
					ServiceAccountName: ServiceAccountName,
					HostNetwork:        false,
					HostPID:            false,
					HostIPC:            false,
				},
			},
		},
	}
}

func leaderElectionRole(ns, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
			},
			{
				Verbs: []string{
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{""},
				Resources: []string{"configmaps/status"},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
			},
		},
	}
}

func leaderElectionRoleBinding(ns, roleName string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "manager-leader-election-role-binding",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      ServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
}
