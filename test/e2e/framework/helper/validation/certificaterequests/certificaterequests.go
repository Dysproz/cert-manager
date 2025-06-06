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

package certificaterequests

import (
	"crypto"
	"fmt"
	"time"

	apiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/cert-manager/cert-manager/pkg/util/pki"
)

// ValidationFunc describes a CertificateRequest validation helper function
type ValidationFunc func(certificaterequest *cmapi.CertificateRequest, key crypto.Signer) error

// ExpectDuration checks if the issued certificate matches the CertificateRequest's duration
func ExpectDurationToMatch(certificaterequest *cmapi.CertificateRequest, key crypto.Signer) error {
	certDuration := apiutil.DefaultCertDuration(certificaterequest.Spec.Duration)
	return ExpectDuration(certDuration, 30*time.Second)(certificaterequest, key)
}

func ExpectDuration(expectedDuration, fuzz time.Duration) func(certificaterequest *cmapi.CertificateRequest, key crypto.Signer) error {
	return func(certificaterequest *cmapi.CertificateRequest, key crypto.Signer) error {
		certBytes := certificaterequest.Status.Certificate
		if len(certBytes) == 0 {
			return fmt.Errorf("no certificate data found in CertificateRequest.Status.Certificate")
		}
		cert, err := pki.DecodeX509CertificateBytes(certBytes)
		if err != nil {
			return err
		}

		// Here we ensure that the requested duration is what is signed on the
		// certificate. We tolerate fuzz either way.
		certDuration := cert.NotAfter.Sub(cert.NotBefore)
		if certDuration > (expectedDuration+fuzz) || certDuration < (expectedDuration-fuzz) {
			return fmt.Errorf(
				"expected duration of %s, got %s (fuzz: %s) [NotBefore: %s, NotAfter: %s]",
				expectedDuration, certDuration, fuzz,
				cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339),
			)
		}

		return nil
	}
}
