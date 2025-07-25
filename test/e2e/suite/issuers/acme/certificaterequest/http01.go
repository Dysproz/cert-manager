/*
Copyright 2020 The cert-manager Authors.

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

package certificate

import (
	"context"
	"crypto/x509"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/cert-manager/cert-manager/e2e-tests/framework"
	"github.com/cert-manager/cert-manager/e2e-tests/framework/log"
	e2eutil "github.com/cert-manager/cert-manager/e2e-tests/util"
	cmacme "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/test/unit/gen"

	. "github.com/cert-manager/cert-manager/e2e-tests/framework/matcher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = framework.CertManagerDescribe("ACME CertificateRequest (HTTP01)", func() {
	f := framework.NewDefaultFramework("create-acme-certificate-request-http01")
	h := f.Helper()

	var acmeIngressDomain string
	issuerName := "test-acme-issuer"
	certificateRequestName := "test-acme-certificate-request"
	// fixedIngressName is the name of an ingress resource that is configured
	// with a challenge solve.
	// To utilise this solver, add the 'testing.cert-manager.io/fixed-ingress: "true"' label.
	fixedIngressName := "testingress"

	BeforeEach(func(testingCtx context.Context) {
		solvers := []cmacme.ACMEChallengeSolver{
			{
				HTTP01: &cmacme.ACMEChallengeSolverHTTP01{
					Ingress: &cmacme.ACMEChallengeSolverHTTP01Ingress{
						Class: &f.Config.Addons.IngressController.IngressClass,
					},
				},
			},
			{
				Selector: &cmacme.CertificateDNSNameSelector{
					MatchLabels: map[string]string{
						"testing.cert-manager.io/fixed-ingress": "true",
					},
				},
				HTTP01: &cmacme.ACMEChallengeSolverHTTP01{
					Ingress: &cmacme.ACMEChallengeSolverHTTP01Ingress{
						Name: fixedIngressName,
					},
				},
			},
		}
		acmeIssuer := gen.Issuer(issuerName,
			gen.SetIssuerNamespace(f.Namespace.Name),
			gen.SetIssuerACMEEmail(testingACMEEmail),
			gen.SetIssuerACMEURL(f.Config.Addons.ACMEServer.URL),
			gen.SetIssuerACMEPrivKeyRef(testingACMEPrivateKey),
			gen.SetIssuerACMESkipTLSVerify(true),
			gen.SetIssuerACMESolvers(solvers))
		By("Creating an Issuer")
		_, err := f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name).Create(testingCtx, acmeIssuer, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		By("Waiting for Issuer to become Ready")
		err = e2eutil.WaitForIssuerCondition(testingCtx, f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name),
			issuerName,
			v1.IssuerCondition{
				Type:   v1.IssuerConditionReady,
				Status: cmmeta.ConditionTrue,
			})
		Expect(err).NotTo(HaveOccurred())
		By("Verifying the ACME account URI is set")
		err = e2eutil.WaitForIssuerStatusFunc(testingCtx, f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name),
			issuerName,
			func(i *v1.Issuer) (bool, error) {
				if i.GetStatus().ACMEStatus().URI == "" {
					return false, nil
				}
				return true, nil
			})
		Expect(err).NotTo(HaveOccurred())
		By("Verifying ACME account private key exists")
		secret, err := f.KubeClientSet.CoreV1().Secrets(f.Namespace.Name).Get(testingCtx, testingACMEPrivateKey, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		if len(secret.Data) != 1 {
			Fail("Expected 1 key in ACME account private key secret, but there was %d", len(secret.Data))
		}
	})

	JustBeforeEach(func(testingCtx context.Context) {
		acmeIngressDomain = e2eutil.RandomSubdomain(f.Config.Addons.IngressController.Domain)
	})

	AfterEach(func(testingCtx context.Context) {
		By("Cleaning up")
		err := f.CertManagerClientSet.CertmanagerV1().Issuers(f.Namespace.Name).Delete(testingCtx, issuerName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = f.KubeClientSet.CoreV1().Secrets(f.Namespace.Name).Delete(testingCtx, testingACMEPrivateKey, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should obtain a signed certificate with a single CN from the ACME server", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		By("Creating a CertificateRequest")
		csr, key, err := gen.CSR(x509.RSA, gen.SetCSRCommonName(acmeIngressDomain), gen.SetCSRDNSNames(acmeIngressDomain))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying the Certificate is valid")
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Minute*5, key)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should obtain a signed ecdsa certificate with a single CN from the ACME server", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		By("Creating a CertificateRequest")
		csr, key, err := gen.CSR(x509.ECDSA, gen.SetCSRCommonName(acmeIngressDomain), gen.SetCSRDNSNames(acmeIngressDomain))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		By("Verifying the Certificate is valid and of type ECDSA")
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Minute*5, key)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should obtain a signed certificate for a long domain using http01 validation", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		// the maximum length of a single segment of the domain being requested
		const maxLengthOfDomainSegment = 63
		By("Creating a CertificateRequest")
		csr, key, err := gen.CSR(x509.ECDSA, gen.SetCSRCommonName(acmeIngressDomain), gen.SetCSRDNSNames(acmeIngressDomain, e2eutil.RandomSubdomainLength(acmeIngressDomain, maxLengthOfDomainSegment)))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Minute*5, key)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should obtain a signed certificate with a CN and single subdomain as dns name from the ACME server", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		By("Creating a CertificateRequest")
		dnsName := e2eutil.RandomSubdomain(acmeIngressDomain)
		csr, key, err := gen.CSR(x509.RSA, gen.SetCSRCommonName(dnsName), gen.SetCSRDNSNames(dnsName))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		By("Verifying the CertificateRequest is valid")
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Minute*5, key)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail to obtain a certificate for an invalid ACME dns name", func(testingCtx context.Context) {
		// create test fixture
		By("Creating a CertificateRequest")
		csr, _, err := gen.CSR(x509.RSA, gen.SetCSRCommonName("google.com"), gen.SetCSRDNSNames("google.com"))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		cr, err = f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name).Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		notReadyCondition := v1.CertificateRequestCondition{
			Type:   v1.CertificateRequestConditionReady,
			Status: cmmeta.ConditionFalse,
		}
		Eventually(cr, "30s", "1s").Should(HaveCondition(f, notReadyCondition))
		Consistently(cr, "1m", "10s").Should(HaveCondition(f, notReadyCondition))
	})

	It("should automatically recreate challenge pod and still obtain a certificate if it is manually deleted", func(testingCtx context.Context) {
		crClient := f.CertManagerClientSet.CertmanagerV1().CertificateRequests(f.Namespace.Name)

		By("Creating a CertificateRequest")
		csr, key, err := gen.CSR(x509.RSA, gen.SetCSRCommonName(acmeIngressDomain), gen.SetCSRDNSNames(acmeIngressDomain))
		Expect(err).NotTo(HaveOccurred())
		cr := gen.CertificateRequest(certificateRequestName,
			gen.SetCertificateRequestNamespace(f.Namespace.Name),
			gen.SetCertificateRequestIssuer(cmmeta.ObjectReference{Kind: v1.IssuerKind, Name: issuerName}),
			gen.SetCertificateRequestCSR(csr),
		)

		_, err = crClient.Create(testingCtx, cr, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("killing the solver pod")
		podClient := f.KubeClientSet.CoreV1().Pods(f.Namespace.Name)
		var pod corev1.Pod
		logf, done := log.LogBackoff()
		defer done()
		err = wait.PollUntilContextTimeout(testingCtx, 1*time.Second, time.Minute*3, true, func(ctx context.Context) (bool, error) {
			logf("Waiting for solver pod to exist")
			podlist, err := podClient.List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, err
			}

			for _, p := range podlist.Items {
				logf("solver pod %s", p.Name)
				// TODO(dmo): make this cleaner instead of just going by name
				if strings.Contains(p.Name, "http-solver") {
					pod = p
					return true, nil
				}
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred())

		err = podClient.Delete(testingCtx, pod.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// The pod should get remade and the certificate should be made valid.
		// Killing the pod could potentially make the validation invalid if pebble
		// were to ask us for the challenge after the pod was killed, but because
		// we kill it so early, we should always be in the self-check phase
		By("Verifying the CertificateRequest is valid")
		err = h.WaitCertificateRequestIssuedValid(testingCtx, f.Namespace.Name, certificateRequestName, time.Minute*5, key)
		Expect(err).NotTo(HaveOccurred())
	})
})
