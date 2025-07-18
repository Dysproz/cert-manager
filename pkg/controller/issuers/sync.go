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

package issuers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/cert-manager/cert-manager/internal/controller/feature"
	internalissuers "github.com/cert-manager/cert-manager/internal/controller/issuers"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/cert-manager/cert-manager/pkg/controller/globals"
	logf "github.com/cert-manager/cert-manager/pkg/logs"
	utilfeature "github.com/cert-manager/cert-manager/pkg/util/feature"
)

const (
	errorInitIssuer = "ErrInitIssuer"

	messageErrorInitIssuer = "Error initializing issuer: "
)

func (c *controller) Sync(ctx context.Context, iss *cmapi.Issuer) (err error) {
	log := logf.FromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, globals.DefaultControllerContextTimeout)
	defer cancel()

	issuerCopy := iss.DeepCopy()
	defer func() {
		if saveErr := c.updateIssuerStatus(ctx, iss, issuerCopy); saveErr != nil {
			err = errors.NewAggregate([]error{saveErr, err})
		}
	}()

	i, err := c.issuerFactory.IssuerFor(issuerCopy)
	if err != nil {
		return err
	}

	err = i.Setup(ctx, issuerCopy)
	if err != nil {
		s := messageErrorInitIssuer + err.Error()
		log.V(logf.WarnLevel).Info(s)
		c.recorder.Event(issuerCopy, corev1.EventTypeWarning, errorInitIssuer, s)
		return err
	}

	return nil
}

func (c *controller) updateIssuerStatus(ctx context.Context, oldIssuer, newIssuer *cmapi.Issuer) error {
	if apiequality.Semantic.DeepEqual(oldIssuer.Status, newIssuer.Status) {
		return nil
	}

	if utilfeature.DefaultFeatureGate.Enabled(feature.ServerSideApply) {
		return internalissuers.ApplyIssuerStatus(ctx, c.cmClient, c.fieldManager, newIssuer)
	} else {
		_, err := c.cmClient.CertmanagerV1().Issuers(newIssuer.Namespace).UpdateStatus(ctx, newIssuer, metav1.UpdateOptions{})
		return err
	}
}
