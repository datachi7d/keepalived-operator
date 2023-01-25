package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	redhatcopv1alpha1 "github.com/redhat-cop/keepalived-operator/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var _ = Describe("Keepalived controller", func() {

	const (
		KeepalivedGroupName      = "test-keepalived"
		KeepalivedGroupNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	ctx := context.Background()
	keepalivedLookupKey := types.NamespacedName{Name: KeepalivedGroupName, Namespace: KeepalivedGroupNamespace}
	keepalivedConfigMap := &corev1.ConfigMap{}

	service1 := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeepalivedGroupName + "1",
			Namespace: KeepalivedGroupNamespace,
			Annotations: map[string]string{
				"keepalived-operator.redhat-cop.io/keepalivedgroup": fmt.Sprintf("%s/%s", KeepalivedGroupNamespace, KeepalivedGroupName),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(6443),
				},
			},
			ExternalIPs: []string{
				"1.1.1.1",
			},
		},
	}

	Context("Creating a KeepalivedGroup and Service", func() {
		It("Should create a configmap", func() {

			keepalivedgroup := &redhatcopv1alpha1.KeepalivedGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "redhatcop.redhat.io/v1alpha1",
					Kind:       "KeepalivedGroup",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      KeepalivedGroupName,
					Namespace: KeepalivedGroupNamespace,
				},
				Spec: redhatcopv1alpha1.KeepalivedGroupSpec{
					Image:              "registry.redhat.io/openshift4/ose-keepalived-ipfailover",
					Interface:          "eno1",
					BlacklistRouterIDs: []int{1, 2, 4, 5},
				},
			}

			Expect(k8sClient.Create(ctx, keepalivedgroup)).Should(Succeed())

			Expect(k8sClient.Create(ctx, service1)).Should(Succeed())

			Eventually(func() bool {
				return k8sClient.Get(ctx, keepalivedLookupKey, keepalivedConfigMap) == nil
			}, timeout, interval).Should(BeTrue())

			Expect(keepalivedConfigMap.Data["keepalived.conf"]).Should(ContainSubstring("virtual_router_id 3"))

		})
	})

	Context("Creating a second service", func() {
		It("Should add another VRRP Router ID to the configmap", func() {
			service2 := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      KeepalivedGroupName + "2",
					Namespace: KeepalivedGroupNamespace,
					Annotations: map[string]string{
						"keepalived-operator.redhat-cop.io/keepalivedgroup": fmt.Sprintf("%s/%s", KeepalivedGroupNamespace, KeepalivedGroupName),
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "https",
							Port:       443,
							Protocol:   "TCP",
							TargetPort: intstr.FromInt(6443),
						},
					},
					ExternalIPs: []string{
						"1.1.1.2",
					},
				},
			}

			Expect(k8sClient.Create(ctx, service2)).Should(Succeed())

			Eventually(func() string {
				if k8sClient.Get(ctx, keepalivedLookupKey, keepalivedConfigMap) == nil {
					if configData, ok := keepalivedConfigMap.Data["keepalived.conf"]; ok {
						return configData
					}
				}
				return ""
			}, timeout, interval).Should(And(ContainSubstring("virtual_router_id 3"), ContainSubstring("virtual_router_id 6")))
		})
	})

	Context("Removing a service", func() {
		It("Should remove the VRRP router ID from the configmap", func() {

			Expect(k8sClient.Delete(ctx, service1)).Should(Succeed())

			Eventually(func() string {
				if k8sClient.Get(ctx, keepalivedLookupKey, keepalivedConfigMap) == nil {
					if configData, ok := keepalivedConfigMap.Data["keepalived.conf"]; ok {
						return configData
					}
				}
				return ""
			}, timeout, interval).Should(And(Not(ContainSubstring("virtual_router_id 3")), ContainSubstring("virtual_router_id 6")))
		})
	})

	Context("Adding another service", func() {
		It("Should add a VRRP router ID to the configmap", func() {
			service3 := &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      KeepalivedGroupName + "3",
					Namespace: KeepalivedGroupNamespace,
					Annotations: map[string]string{
						"keepalived-operator.redhat-cop.io/keepalivedgroup": fmt.Sprintf("%s/%s", KeepalivedGroupNamespace, KeepalivedGroupName),
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Ports: []corev1.ServicePort{
						{
							Name:       "https",
							Port:       443,
							Protocol:   "TCP",
							TargetPort: intstr.FromInt(6443),
						},
					},
					ExternalIPs: []string{
						"1.1.1.3",
					},
				},
			}

			Expect(k8sClient.Create(ctx, service3)).Should(Succeed())

			Eventually(func() string {
				if k8sClient.Get(ctx, keepalivedLookupKey, keepalivedConfigMap) == nil {
					if configData, ok := keepalivedConfigMap.Data["keepalived.conf"]; ok {
						return configData
					}
				}
				return ""
			}, timeout, interval).Should(And(ContainSubstring("virtual_router_id 3"), ContainSubstring("virtual_router_id 6")))
		})
	})
})
