/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"

	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/ingress-nginx/internal/ingress"
	"testing"
)

func TestMergeAlternativeBackends(t *testing.T) {
	testCases := map[string]struct {
		ingress      *ingress.Ingress
		upstreams    map[string]*ingress.Backend
		servers      map[string]*ingress.Server
		expUpstreams map[string]*ingress.Backend
		expServers   map[string]*ingress.Server
	}{
		"alternative backend has no server and embeds into matching real backend": {
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "example",
					},
					Spec: extensions.IngressSpec{
						Rules: []extensions.IngressRule{
							{
								Host: "example.com",
								IngressRuleValue: extensions.IngressRuleValue{
									HTTP: &extensions.HTTPIngressRuleValue{
										Paths: []extensions.HTTPIngressPath{
											{
												Path: "/",
												Backend: extensions.IngressBackend{
													ServiceName: "http-svc-canary",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-http-svc-80": {
					Name:     "example-http-svc-80",
					NoServer: false,
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"example.com": {
					Hostname: "example.com",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-http-svc-80": {
					Name:                "example-http-svc-80",
					NoServer:            false,
					AlternativeBackends: []string{"example-http-svc-canary-80"},
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"example.com": {
					Hostname: "example.com",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
		},
		"alternative backend merges with the correct real backend when multiple are present": {
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "example",
					},
					Spec: extensions.IngressSpec{
						Rules: []extensions.IngressRule{
							{
								Host: "foo.bar",
								IngressRuleValue: extensions.IngressRuleValue{
									HTTP: &extensions.HTTPIngressRuleValue{
										Paths: []extensions.HTTPIngressPath{
											{
												Path: "/",
												Backend: extensions.IngressBackend{
													ServiceName: "foo-http-svc-canary",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: 80,
													},
												},
											},
										},
									},
								},
							},
							{
								Host: "example.com",
								IngressRuleValue: extensions.IngressRuleValue{
									HTTP: &extensions.HTTPIngressRuleValue{
										Paths: []extensions.HTTPIngressPath{
											{
												Path: "/",
												Backend: extensions.IngressBackend{
													ServiceName: "http-svc-canary",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-foo-http-svc-80": {
					Name:     "example-foo-http-svc-80",
					NoServer: false,
				},
				"example-foo-http-svc-canary-80": {
					Name:     "example-foo-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
				"example-http-svc-80": {
					Name:                "example-http-svc-80",
					NoServer:            false,
					AlternativeBackends: []string{"example-http-svc-canary-80"},
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"foo.bar": {
					Hostname: "foo.bar",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-foo-http-svc-80",
						},
					},
				},
				"example.com": {
					Hostname: "example.com",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-foo-http-svc-80": {
					Name:                "example-foo-http-svc-80",
					NoServer:            false,
					AlternativeBackends: []string{"example-foo-http-svc-canary-80"},
				},
				"example-foo-http-svc-canary-80": {
					Name:     "example-foo-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
				"example-http-svc-80": {
					Name:                "example-http-svc-80",
					NoServer:            false,
					AlternativeBackends: []string{"example-http-svc-canary-80"},
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"example.com": {
					Hostname: "example.com",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
		},
		"alternative backend does not merge into itself": {
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "example",
					},
					Spec: extensions.IngressSpec{
						Rules: []extensions.IngressRule{
							{
								Host: "example.com",
								IngressRuleValue: extensions.IngressRuleValue{
									HTTP: &extensions.HTTPIngressRuleValue{
										Paths: []extensions.HTTPIngressPath{
											{
												Path: "/",
												Backend: extensions.IngressBackend{
													ServiceName: "http-svc-canary",
													ServicePort: intstr.IntOrString{
														Type:   intstr.Int,
														IntVal: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{},
			map[string]*ingress.Backend{},
			map[string]*ingress.Server{},
		},
		"catch-all alternative backend has no server and embeds into matching real backend": {
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "example",
					},
					Spec: extensions.IngressSpec{
						Backend: &extensions.IngressBackend{
							ServiceName: "http-svc-canary",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-http-svc-80": {
					Name:     "example-http-svc-80",
					NoServer: false,
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"_": {
					Hostname: "_",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"example-http-svc-80": {
					Name:                "example-http-svc-80",
					NoServer:            false,
					AlternativeBackends: []string{"example-http-svc-canary-80"},
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"_": {
					Hostname: "_",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "example-http-svc-80",
						},
					},
				},
			},
		},
		"catch-all alternative backend does not merge into itself": {
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "example",
					},
					Spec: extensions.IngressSpec{
						Backend: &extensions.IngressBackend{
							ServiceName: "http-svc-canary",
							ServicePort: intstr.IntOrString{
								IntVal: 80,
							},
						},
					},
				},
			},
			map[string]*ingress.Backend{
				"upstream-default-backend": {
					Name:     "upstream-default-backend",
					NoServer: false,
				},
				"example-http-svc-canary-80": {
					Name:     "example-http-svc-canary-80",
					NoServer: true,
					TrafficShapingPolicy: ingress.TrafficShapingPolicy{
						Weight: 20,
					},
				},
			},
			map[string]*ingress.Server{
				"_": {
					Hostname: "_",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "upstream-default-backend",
						},
					},
				},
			},
			map[string]*ingress.Backend{},
			map[string]*ingress.Server{
				"_": {
					Hostname: "_",
					Locations: []*ingress.Location{
						{
							Path:    "/",
							Backend: "upstream-default-backend",
						},
					},
				},
			},
		},
	}

	for title, tc := range testCases {
		t.Run(title, func(t *testing.T) {
			mergeAlternativeBackends(tc.ingress, tc.upstreams, tc.servers)

			for upsName, expUpstream := range tc.expUpstreams {
				actualUpstream, ok := tc.upstreams[upsName]
				if !ok {
					t.Errorf("expected upstream %s to exist but it did not", upsName)
				}

				if !actualUpstream.Equal(expUpstream) {
					t.Logf("actual upstream %s alternative backends: %s", actualUpstream.Name, actualUpstream.AlternativeBackends)
					t.Logf("expected upstream %s alternative backends: %s", expUpstream.Name, expUpstream.AlternativeBackends)
					t.Errorf("upstream %s was not equal to what was expected: ", upsName)
				}
			}

			for serverName, expServer := range tc.expServers {
				actualServer, ok := tc.servers[serverName]
				if !ok {
					t.Errorf("expected server %s to exist but it did not", serverName)
				}

				if !actualServer.Equal(expServer) {
					t.Errorf("server %s was not equal to what was expected", serverName)
				}
			}
		})
	}
}

func TestExtractTLSSecretName(t *testing.T) {
	testCases := map[string]struct {
		host    string
		ingress *ingress.Ingress
		fn      func(string) (*ingress.SSLCert, error)
		expName string
	}{
		"nil ingress": {
			"foo.bar",
			nil,
			func(string) (*ingress.SSLCert, error) {
				return nil, nil
			},
			"",
		},
		"empty ingress": {
			"foo.bar",
			&ingress.Ingress{},
			func(string) (*ingress.SSLCert, error) {
				return nil, nil
			},
			"",
		},
		"ingress tls, nil secret": {
			"foo.bar",
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: extensions.IngressSpec{
						TLS: []extensions.IngressTLS{
							{SecretName: "demo"},
						},
						Rules: []extensions.IngressRule{
							{
								Host: "foo.bar",
							},
						},
					},
				},
			},
			func(string) (*ingress.SSLCert, error) {
				return nil, nil
			},
			"",
		},
		"ingress tls, no host, matching cert cn": {
			"foo.bar",
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: extensions.IngressSpec{
						TLS: []extensions.IngressTLS{
							{SecretName: "demo"},
						},
						Rules: []extensions.IngressRule{
							{
								Host: "foo.bar",
							},
						},
					},
				},
			},
			func(string) (*ingress.SSLCert, error) {
				return &ingress.SSLCert{
					Certificate: fakeX509Cert([]string{"foo.bar", "example.com"}),
				}, nil
			},
			"demo",
		},
		"ingress tls, no host, wildcard cert with matching cn": {
			"foo.bar",
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: extensions.IngressSpec{
						TLS: []extensions.IngressTLS{
							{
								SecretName: "demo",
							},
						},
						Rules: []extensions.IngressRule{
							{
								Host: "test.foo.bar",
							},
						},
					},
				},
			},
			func(string) (*ingress.SSLCert, error) {
				return &ingress.SSLCert{
					Certificate: fakeX509Cert([]string{"*.foo.bar", "foo.bar"}),
				}, nil
			},
			"demo",
		},
		"ingress tls, hosts, matching cert cn": {
			"foo.bar",
			&ingress.Ingress{
				Ingress: extensions.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: extensions.IngressSpec{
						TLS: []extensions.IngressTLS{
							{
								Hosts:      []string{"foo.bar", "example.com"},
								SecretName: "demo",
							},
						},
						Rules: []extensions.IngressRule{
							{
								Host: "foo.bar",
							},
						},
					},
				},
			},
			func(string) (*ingress.SSLCert, error) {
				return nil, nil
			},
			"demo",
		},
	}

	for title, tc := range testCases {
		t.Run(title, func(t *testing.T) {
			name := extractTLSSecretName(tc.host, tc.ingress, tc.fn)
			if name != tc.expName {
				t.Errorf("Expected Secret name %q (got %q)", tc.expName, name)
			}
		})
	}
}

var oidExtensionSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}

func fakeX509Cert(dnsNames []string) *x509.Certificate {
	return &x509.Certificate{
		DNSNames: dnsNames,
		Extensions: []pkix.Extension{
			{Id: oidExtensionSubjectAltName},
		},
	}
}
